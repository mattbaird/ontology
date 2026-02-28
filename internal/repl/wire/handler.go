package wire

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/matthewbaird/ontology/internal/repl/autocomplete"
	"github.com/matthewbaird/ontology/internal/repl/executor"
	"github.com/matthewbaird/ontology/internal/repl/meta"
	"github.com/matthewbaird/ontology/internal/repl/planner"
	"github.com/matthewbaird/ontology/internal/repl/pql"
	"github.com/matthewbaird/ontology/internal/repl/session"
)

const (
	// rowBatchSize controls how many rows are sent per "rows" message.
	rowBatchSize = 50
)

// Handler manages WebSocket connections for the REPL.
type Handler struct {
	sessions     *session.Manager
	planner      *planner.Planner
	executor     *executor.Executor
	autocomplete *autocomplete.Engine
	meta         *meta.Handler
}

// NewHandler creates a WebSocket handler with all dependencies.
func NewHandler(
	sessions *session.Manager,
	pl *planner.Planner,
	exec *executor.Executor,
	ac *autocomplete.Engine,
	metaHandler *meta.Handler,
) *Handler {
	return &Handler{
		sessions:     sessions,
		planner:      pl,
		executor:     exec,
		autocomplete: ac,
		meta:         metaHandler,
	}
}

// ServeHTTP upgrades to WebSocket and runs the message loop.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("repl: websocket accept: %v", err)
		return
	}
	defer conn.CloseNow()

	// Create session
	sess := h.sessions.Create()
	ctx := r.Context()

	// Send session info
	h.send(ctx, conn, ServerMessage{
		Type: "session",
		Data: SessionData{
			SessionID: sess.ID,
			Mode:      string(sess.Mode),
		},
	})

	// Message loop
	for {
		var msg ClientMessage
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				log.Printf("repl: connection closed: %v", websocket.CloseStatus(err))
			}
			return
		}

		switch msg.Type {
		case "execute":
			h.handleExecute(ctx, conn, sess, msg)
		case "autocomplete":
			h.handleAutocomplete(ctx, conn, msg)
		case "ping":
			h.send(ctx, conn, ServerMessage{Type: "pong", RequestID: msg.ID})
		case "cancel":
			// Phase 1: queries are synchronous, cancel is a no-op
		default:
			h.sendError(ctx, conn, msg.ID, "unknown_type", fmt.Sprintf("unknown message type: %s", msg.Type))
		}
	}
}

func (h *Handler) handleExecute(ctx context.Context, conn *websocket.Conn, sess *session.Session, msg ClientMessage) {
	start := time.Now()

	var data ExecuteData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		h.sendError(ctx, conn, msg.ID, "invalid_data", "invalid execute data")
		return
	}

	if data.PQL == "" {
		h.sendError(ctx, conn, msg.ID, "empty_query", "empty PQL query")
		return
	}

	sess.AddHistory(data.PQL)

	// Lex
	lexer := pql.NewLexer(data.PQL)
	tokens, lexErrors := lexer.Tokenize()
	if len(lexErrors) > 0 {
		h.sendError(ctx, conn, msg.ID, "lex_error", lexErrors[0].Error())
		return
	}

	// Parse
	parser := pql.NewParser(tokens)
	stmts, parseErrors := parser.Parse()
	if len(parseErrors) > 0 {
		h.sendError(ctx, conn, msg.ID, "parse_error", parseErrors[0].Error())
		return
	}
	if len(stmts) == 0 {
		h.sendError(ctx, conn, msg.ID, "empty_query", "no statements found")
		return
	}

	// Execute each statement (Phase 1: typically just one)
	for _, stmt := range stmts {
		// Plan
		plan, err := h.planner.Plan(stmt)
		if err != nil {
			h.sendError(ctx, conn, msg.ID, "plan_error", err.Error())
			return
		}

		// Meta-commands handled specially
		if plan.Type == planner.PlanMeta {
			result, err := h.meta.Execute(sess, plan.MetaCommand, plan.MetaArgs)
			if err != nil {
				h.sendError(ctx, conn, msg.ID, "meta_error", err.Error())
				return
			}
			h.send(ctx, conn, ServerMessage{
				Type:      "meta",
				RequestID: msg.ID,
				Data:      result,
			})
			continue
		}

		// Execute query
		result, err := h.executor.Execute(ctx, plan)
		if err != nil {
			h.sendError(ctx, conn, msg.ID, "exec_error", err.Error())
			return
		}

		// Send meta
		if result.Meta != nil {
			h.send(ctx, conn, ServerMessage{
				Type:      "meta",
				RequestID: msg.ID,
				Data: MetaData{
					Entity: result.Meta.Entity,
					Total:  result.Meta.Total,
				},
			})
		}

		// Send rows in batches
		if result.Rows != nil {
			for i := 0; i < len(result.Rows); i += rowBatchSize {
				end := i + rowBatchSize
				if end > len(result.Rows) {
					end = len(result.Rows)
				}
				h.send(ctx, conn, ServerMessage{
					Type:      "rows",
					RequestID: msg.ID,
					Data:      RowsData{Rows: result.Rows[i:end]},
				})
			}
		}

		// Send count
		if result.Count != nil {
			h.send(ctx, conn, ServerMessage{
				Type:      "rows",
				RequestID: msg.ID,
				Data: map[string]int{
					"count": *result.Count,
				},
			})
		}

		// Send done
		elapsed := time.Since(start)
		h.send(ctx, conn, ServerMessage{
			Type:      "done",
			RequestID: msg.ID,
			Data: DoneData{
				Total:   result.Meta.Total,
				Elapsed: elapsed.String(),
			},
		})
	}
}

func (h *Handler) handleAutocomplete(ctx context.Context, conn *websocket.Conn, msg ClientMessage) {
	var data AutocompleteData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		h.sendError(ctx, conn, msg.ID, "invalid_data", "invalid autocomplete data")
		return
	}

	items := h.autocomplete.Complete(data.PQL, data.Cursor)
	h.send(ctx, conn, ServerMessage{
		Type:      "completions",
		RequestID: msg.ID,
		Data:      CompletionsData{Items: items},
	})
}

func (h *Handler) send(ctx context.Context, conn *websocket.Conn, msg ServerMessage) {
	if err := wsjson.Write(ctx, conn, msg); err != nil {
		log.Printf("repl: write error: %v", err)
	}
}

func (h *Handler) sendError(ctx context.Context, conn *websocket.Conn, requestID, code, message string) {
	h.send(ctx, conn, ServerMessage{
		Type:      "error",
		RequestID: requestID,
		Data: ErrorData{
			Code:    code,
			Message: message,
		},
	})
}

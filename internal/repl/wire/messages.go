// Package wire defines the WebSocket protocol for the REPL.
package wire

import (
	"encoding/json"

	"github.com/matthewbaird/ontology/internal/repl/autocomplete"
)

// ── Client → Server messages ────────────────────────────────────────────────

// ClientMessage is the envelope for all client-to-server WebSocket messages.
type ClientMessage struct {
	Type string          `json:"type"` // "execute", "cancel", "autocomplete", "ping"
	ID   string          `json:"id"`   // Client-assigned request ID
	Data json.RawMessage `json:"data,omitempty"`
}

// ExecuteData is the payload for "execute" messages.
type ExecuteData struct {
	PQL string `json:"pql"`
}

// AutocompleteData is the payload for "autocomplete" messages.
type AutocompleteData struct {
	PQL    string `json:"pql"`
	Cursor int    `json:"cursor"`
}

// ── Server → Client messages ────────────────────────────────────────────────

// ServerMessage is the envelope for all server-to-client WebSocket messages.
type ServerMessage struct {
	Type      string `json:"type"`                // "meta", "rows", "done", "error", "completions", "session", "pong"
	RequestID string `json:"request_id,omitempty"` // Echoes client ID
	Data      any    `json:"data,omitempty"`
}

// MetaData is sent before results to describe the schema and expected count.
type MetaData struct {
	Entity string   `json:"entity"`
	Fields []string `json:"fields,omitempty"`
	Total  int      `json:"total"`
}

// RowsData carries a batch of result rows.
type RowsData struct {
	Rows []json.RawMessage `json:"rows"`
}

// DoneData signals completion of a query.
type DoneData struct {
	Total   int    `json:"total"`
	Elapsed string `json:"elapsed"`
}

// ErrorData carries an error message.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CompletionsData carries autocomplete suggestions.
type CompletionsData struct {
	Items []autocomplete.CompletionItem `json:"items"`
}

// SessionData carries session information.
type SessionData struct {
	SessionID string `json:"session_id"`
	Mode      string `json:"mode"` // "dev" or "operator"
}

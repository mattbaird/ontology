// Package meta handles REPL meta-commands (:help, :clear, :env, :history).
package meta

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matthewbaird/ontology/internal/repl/schema"
	"github.com/matthewbaird/ontology/internal/repl/session"
)

// Handler dispatches meta-commands.
type Handler struct {
	registry *schema.Registry
}

// New creates a meta-command handler.
func New(registry *schema.Registry) *Handler {
	return &Handler{registry: registry}
}

// Result is the output of a meta-command execution.
type Result struct {
	Output string `json:"output"`
	Clear  bool   `json:"clear,omitempty"` // Signal frontend to clear screen
}

// Execute runs a meta-command and returns the result.
func (h *Handler) Execute(sess *session.Session, command string, args []string) (*Result, error) {
	switch command {
	case "help":
		return h.help(args)
	case "clear":
		return &Result{Clear: true}, nil
	case "env":
		return h.env(sess)
	case "history":
		return h.history(sess)
	case "schema":
		return h.schemaCmd(args)
	default:
		return nil, fmt.Errorf("unknown meta-command ':%s'. Type :help for available commands", command)
	}
}

func (h *Handler) help(args []string) (*Result, error) {
	if len(args) > 0 {
		return h.helpTopic(args[0])
	}

	help := `PQL — Propeller Query Language

Queries:
  find <entity> [clauses]  Search for entities
  get <entity> "<id>"      Fetch a single entity by ID
  count <entity> [where]   Count matching entities

Mutations:
  create <entity> set <field> = <value> [, ...]
  update <entity> "<id>" set <field> = <value> [, ...]
  delete <entity> "<id>"

Clauses (any order):
  where <field> <op> <value>   Filter results
  select <field>, ...          Project specific fields
  include <edge>, ...          Eager-load relationships
  order by <field> [asc|desc]  Sort results
  limit <n>                    Limit result count
  offset <n>                   Skip first n results

Operators: =, !=, >, <, >=, <=, like, in
Logic: and, or, not

Meta-commands:
  :help [topic]    Show help
  :clear           Clear the screen
  :env             Show session info
  :history         Show command history
  :schema [entity] Show entity schema

Examples:
  find lease where status = "active" limit 10
  find person where first_name like "J%"
  get person "550e8400-e29b-41d4-a716-446655440000"
  count space where status in ["vacant", "available"]
  create portfolio set name = "Main Portfolio"
  update property "550e..." set name = "Updated Name"
  delete building "550e..."`

	return &Result{Output: help}, nil
}

func (h *Handler) helpTopic(topic string) (*Result, error) {
	switch topic {
	case "find":
		return &Result{Output: "find <entity> [where ...] [select ...] [include ...] [order by ...] [limit N] [offset N]"}, nil
	case "get":
		return &Result{Output: "get <entity> \"<uuid>\"\n\nFetches a single entity by its UUID."}, nil
	case "count":
		return &Result{Output: "count <entity> [where ...]\n\nReturns the number of matching entities."}, nil
	case "create":
		return &Result{Output: "create <entity> set <field> = <value> [, <field> = <value> ...]\n\nCreates a new entity with the specified field values."}, nil
	case "update":
		return &Result{Output: "update <entity> \"<uuid>\" set <field> = <value> [, <field> = <value> ...]\n\nUpdates an existing entity's fields."}, nil
	case "delete":
		return &Result{Output: "delete <entity> \"<uuid>\"\n\nDeletes an entity by its UUID."}, nil
	case "where":
		return &Result{Output: "where <field> <op> <value> [and|or <field> <op> <value> ...]\n\nOperators: =, !=, >, <, >=, <=, like, in\n\nLIKE uses SQL wildcards: % = any characters, _ = single character\n  Example: find person where first_name like \"J%\""}, nil
	default:
		return &Result{Output: fmt.Sprintf("No help available for '%s'", topic)}, nil
	}
}

func (h *Handler) env(sess *session.Session) (*Result, error) {
	out := fmt.Sprintf("Session: %s\nMode: %s\nCreated: %s\nLast active: %s\nHistory entries: %d\nVariables: %d",
		sess.ID, sess.Mode,
		sess.CreatedAt.Format("2006-01-02 15:04:05"),
		sess.LastActiveAt.Format("2006-01-02 15:04:05"),
		len(sess.History), len(sess.Variables))
	return &Result{Output: out}, nil
}

func (h *Handler) history(sess *session.Session) (*Result, error) {
	if len(sess.History) == 0 {
		return &Result{Output: "(no history)"}, nil
	}

	var b strings.Builder
	for i, entry := range sess.History {
		fmt.Fprintf(&b, "%3d  %s\n", i+1, entry)
	}
	return &Result{Output: b.String()}, nil
}

func (h *Handler) schemaCmd(args []string) (*Result, error) {
	if len(args) == 0 {
		// List all entities
		names := h.registry.EntityNames()
		return &Result{Output: fmt.Sprintf("Entities (%d):\n  %s", len(names), strings.Join(names, "\n  "))}, nil
	}

	entityName := strings.ToLower(args[0])
	es := h.registry.Entity(entityName)
	if es == nil {
		return nil, fmt.Errorf("unknown entity '%s'", entityName)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Entity: %s (%s)\n", es.Name, es.EntName)
	if es.HasStateMachine {
		fmt.Fprintf(&b, "State Machine: yes\n")
	}
	if es.Immutable {
		fmt.Fprintf(&b, "Immutable: yes\n")
	}

	fmt.Fprintf(&b, "\nFields:\n")
	for _, fname := range es.FieldOrder {
		fm := es.Fields[fname]
		opt := ""
		if fm.Optional {
			opt = " (optional)"
		}
		extra := ""
		if fm.Type == schema.FieldEnum && len(fm.EnumValues) > 0 {
			data, _ := json.Marshal(fm.EnumValues)
			extra = " values=" + string(data)
		}
		fmt.Fprintf(&b, "  %-30s %s%s%s\n", fname, fm.Type, opt, extra)
	}

	if len(es.EdgeOrder) > 0 {
		fmt.Fprintf(&b, "\nEdges:\n")
		for _, ename := range es.EdgeOrder {
			em := es.Edges[ename]
			fmt.Fprintf(&b, "  %-30s -> %s (%s)\n", ename, em.Target, em.Cardinality)
		}
	}

	if es.HasStateMachine && es.StateMachine != nil {
		fmt.Fprintf(&b, "\nState Machine:\n")
		for from, targets := range es.StateMachine {
			fmt.Fprintf(&b, "  %s -> %s\n", from, strings.Join(targets, ", "))
		}
	}

	return &Result{Output: b.String()}, nil
}

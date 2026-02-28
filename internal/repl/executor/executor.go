package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/internal/repl/planner"
)

// Result holds the output of a query execution.
type Result struct {
	Rows  []json.RawMessage `json:"rows,omitempty"`
	Count *int              `json:"count,omitempty"`
	Meta  *ResultMeta       `json:"meta,omitempty"`
}

// ResultMeta provides metadata about the result.
type ResultMeta struct {
	Entity string `json:"entity"`
	Total  int    `json:"total"`
}

// Executor runs QueryPlans against the Ent client.
type Executor struct {
	client     *ent.Client
	dispatchers *DispatchRegistry
}

// New creates an executor backed by the given Ent client and dispatch registry.
func New(client *ent.Client, dispatchers *DispatchRegistry) *Executor {
	return &Executor{
		client:     client,
		dispatchers: dispatchers,
	}
}

// Execute runs a query plan and returns the result.
func (e *Executor) Execute(ctx context.Context, plan *planner.QueryPlan) (*Result, error) {
	switch plan.Type {
	case planner.PlanFind:
		return e.execFind(ctx, plan)
	case planner.PlanGet:
		return e.execGet(ctx, plan)
	case planner.PlanCount:
		return e.execCount(ctx, plan)
	case planner.PlanCreate:
		return e.execCreate(ctx, plan)
	case planner.PlanUpdate:
		return e.execUpdate(ctx, plan)
	case planner.PlanDelete:
		return e.execDelete(ctx, plan)
	case planner.PlanMeta:
		// Meta-commands handled externally
		return nil, fmt.Errorf("meta-commands should be handled by the meta-command handler")
	default:
		return nil, fmt.Errorf("unsupported plan type: %d", plan.Type)
	}
}

// execFind executes a find query.
func (e *Executor) execFind(ctx context.Context, plan *planner.QueryPlan) (*Result, error) {
	d := e.dispatchers.Get(plan.Entity)
	if d == nil {
		return nil, fmt.Errorf("no dispatcher for entity '%s'", plan.Entity)
	}

	qh := d.Query(e.client)

	// Apply predicates
	if len(plan.Predicates) > 0 {
		qh = qh.Where(plan.Predicates...)
	}

	// Apply edge loading
	for _, edge := range plan.Edges {
		qh = qh.WithEdge(edge)
	}

	// Apply ordering
	for _, o := range plan.OrderBy {
		qh = qh.OrderBy(o.Field, o.Desc)
	}

	// Apply limit
	if plan.Limit > 0 {
		qh = qh.Limit(plan.Limit)
	}

	// Apply offset
	if plan.Offset > 0 {
		qh = qh.Offset(plan.Offset)
	}

	// Execute
	entities, err := qh.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Serialize to JSON
	rows, err := serializeResults(entities, plan.Fields)
	if err != nil {
		return nil, fmt.Errorf("serialization failed: %w", err)
	}

	total := len(rows)
	return &Result{
		Rows: rows,
		Meta: &ResultMeta{
			Entity: plan.Entity,
			Total:  total,
		},
	}, nil
}

// execGet executes a get-by-ID query.
func (e *Executor) execGet(ctx context.Context, plan *planner.QueryPlan) (*Result, error) {
	d := e.dispatchers.Get(plan.Entity)
	if d == nil {
		return nil, fmt.Errorf("no dispatcher for entity '%s'", plan.Entity)
	}

	id, err := uuid.Parse(plan.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %s", plan.ID)
	}

	entity, err := d.Get(ctx, e.client, id)
	if err != nil {
		return nil, fmt.Errorf("get failed: %w", err)
	}

	data, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("serialization failed: %w", err)
	}

	return &Result{
		Rows: []json.RawMessage{data},
		Meta: &ResultMeta{
			Entity: plan.Entity,
			Total:  1,
		},
	}, nil
}

// execCount executes a count query.
func (e *Executor) execCount(ctx context.Context, plan *planner.QueryPlan) (*Result, error) {
	d := e.dispatchers.Get(plan.Entity)
	if d == nil {
		return nil, fmt.Errorf("no dispatcher for entity '%s'", plan.Entity)
	}

	qh := d.Query(e.client)

	if len(plan.Predicates) > 0 {
		qh = qh.Where(plan.Predicates...)
	}

	count, err := qh.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count failed: %w", err)
	}

	return &Result{
		Count: &count,
		Meta: &ResultMeta{
			Entity: plan.Entity,
			Total:  count,
		},
	}, nil
}

// execCreate inserts a new entity.
func (e *Executor) execCreate(ctx context.Context, plan *planner.QueryPlan) (*Result, error) {
	md, err := e.dispatchers.GetMutation(plan.Entity)
	if err != nil {
		return nil, err
	}

	entity, err := md.Create(ctx, e.client, plan.Assignments)
	if err != nil {
		return nil, fmt.Errorf("create failed: %w", err)
	}

	data, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("serialization failed: %w", err)
	}

	return &Result{
		Rows: []json.RawMessage{data},
		Meta: &ResultMeta{
			Entity: plan.Entity,
			Total:  1,
		},
	}, nil
}

// execUpdate modifies an existing entity by ID.
func (e *Executor) execUpdate(ctx context.Context, plan *planner.QueryPlan) (*Result, error) {
	md, err := e.dispatchers.GetMutation(plan.Entity)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(plan.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %s", plan.ID)
	}

	entity, err := md.Update(ctx, e.client, id, plan.Assignments)
	if err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	data, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("serialization failed: %w", err)
	}

	return &Result{
		Rows: []json.RawMessage{data},
		Meta: &ResultMeta{
			Entity: plan.Entity,
			Total:  1,
		},
	}, nil
}

// execDelete removes an entity by ID.
func (e *Executor) execDelete(ctx context.Context, plan *planner.QueryPlan) (*Result, error) {
	md, err := e.dispatchers.GetMutation(plan.Entity)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(plan.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %s", plan.ID)
	}

	if err := md.Delete(ctx, e.client, id); err != nil {
		return nil, fmt.Errorf("delete failed: %w", err)
	}

	count := 1
	return &Result{
		Count: &count,
		Meta: &ResultMeta{
			Entity: plan.Entity,
			Total:  1,
		},
	}, nil
}

// serializeResults converts entity values to JSON, optionally projecting fields.
func serializeResults(entities []any, fields []string) ([]json.RawMessage, error) {
	rows := make([]json.RawMessage, 0, len(entities))

	for _, ent := range entities {
		data, err := json.Marshal(ent)
		if err != nil {
			return nil, err
		}

		// Field projection: if specific fields requested, filter the JSON
		if len(fields) > 0 {
			var full map[string]json.RawMessage
			if err := json.Unmarshal(data, &full); err != nil {
				return nil, err
			}
			projected := make(map[string]json.RawMessage)
			// Always include id
			if v, ok := full["id"]; ok {
				projected["id"] = v
			}
			for _, f := range fields {
				if v, ok := full[f]; ok {
					projected[f] = v
				}
			}
			data, err = json.Marshal(projected)
			if err != nil {
				return nil, err
			}
		}

		rows = append(rows, data)
	}

	return rows, nil
}

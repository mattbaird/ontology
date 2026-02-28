package planner

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/internal/repl/pql"
	"github.com/matthewbaird/ontology/internal/repl/schema"
)

// DefaultLimit is applied when no explicit limit is specified.
const DefaultLimit = 100

// Planner transforms PQL AST nodes into QueryPlans using the schema registry.
type Planner struct {
	registry *schema.Registry
}

// New creates a planner backed by the given schema registry.
func New(registry *schema.Registry) *Planner {
	return &Planner{registry: registry}
}

// Plan converts a PQL statement AST node into a validated QueryPlan.
func (p *Planner) Plan(stmt pql.Statement) (*QueryPlan, error) {
	switch s := stmt.(type) {
	case *pql.FindStmt:
		return p.planFind(s)
	case *pql.GetStmt:
		return p.planGet(s)
	case *pql.CountStmt:
		return p.planCount(s)
	case *pql.CreateStmt:
		return p.planCreate(s)
	case *pql.UpdateStmt:
		return p.planUpdate(s)
	case *pql.DeleteStmt:
		return p.planDelete(s)
	case *pql.MetaCmdStmt:
		return p.planMeta(s)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// ── find ─────────────────────────────────────────────────────────────────────

func (p *Planner) planFind(stmt *pql.FindStmt) (*QueryPlan, error) {
	es, err := p.resolveEntity(stmt.Entity)
	if err != nil {
		return nil, err
	}

	plan := &QueryPlan{
		Type:   PlanFind,
		Entity: es.Name,
		Limit:  DefaultLimit,
	}

	// WHERE
	if stmt.Where != nil {
		preds, err := p.resolvePredicates(es, stmt.Where.Expr)
		if err != nil {
			return nil, err
		}
		plan.Predicates = preds
	}

	// SELECT
	if stmt.Select != nil {
		for _, fr := range stmt.Select.Fields {
			colName, err := p.resolveField(es, fr)
			if err != nil {
				return nil, err
			}
			plan.Fields = append(plan.Fields, colName)
		}
	}

	// INCLUDE
	if stmt.Include != nil {
		for _, ep := range stmt.Include.Edges {
			edgeName, err := p.resolveEdge(es, ep)
			if err != nil {
				return nil, err
			}
			plan.Edges = append(plan.Edges, edgeName)
		}
	}

	// ORDER BY
	if stmt.OrderBy != nil {
		for _, item := range stmt.OrderBy.Items {
			colName, err := p.resolveField(es, item.Field)
			if err != nil {
				return nil, err
			}
			plan.OrderBy = append(plan.OrderBy, OrderSpec{Field: colName, Desc: item.Desc})
		}
	}

	// LIMIT
	if stmt.Limit != nil {
		plan.Limit = stmt.Limit.Value
	}

	// OFFSET
	if stmt.Offset != nil {
		plan.Offset = stmt.Offset.Value
	}

	return plan, nil
}

// ── get ──────────────────────────────────────────────────────────────────────

func (p *Planner) planGet(stmt *pql.GetStmt) (*QueryPlan, error) {
	es, err := p.resolveEntity(stmt.Entity)
	if err != nil {
		return nil, err
	}

	plan := &QueryPlan{
		Type:   PlanGet,
		Entity: es.Name,
		ID:     stmt.ID,
	}

	if stmt.Include != nil {
		for _, ep := range stmt.Include.Edges {
			edgeName, err := p.resolveEdge(es, ep)
			if err != nil {
				return nil, err
			}
			plan.Edges = append(plan.Edges, edgeName)
		}
	}

	return plan, nil
}

// ── count ────────────────────────────────────────────────────────────────────

func (p *Planner) planCount(stmt *pql.CountStmt) (*QueryPlan, error) {
	es, err := p.resolveEntity(stmt.Entity)
	if err != nil {
		return nil, err
	}

	plan := &QueryPlan{
		Type:   PlanCount,
		Entity: es.Name,
	}

	if stmt.Where != nil {
		preds, err := p.resolvePredicates(es, stmt.Where.Expr)
		if err != nil {
			return nil, err
		}
		plan.Predicates = preds
	}

	return plan, nil
}

// ── create ────────────────────────────────────────────────────────────────────

func (p *Planner) planCreate(stmt *pql.CreateStmt) (*QueryPlan, error) {
	es, err := p.resolveEntity(stmt.Entity)
	if err != nil {
		return nil, err
	}

	if es.Immutable {
		return nil, fmt.Errorf("entity '%s' is immutable and cannot be created via REPL", es.Name)
	}

	assignments, err := p.resolveAssignments(es, stmt.Assignments)
	if err != nil {
		return nil, err
	}

	return &QueryPlan{
		Type:        PlanCreate,
		Entity:      es.Name,
		Assignments: assignments,
	}, nil
}

// ── update ────────────────────────────────────────────────────────────────────

func (p *Planner) planUpdate(stmt *pql.UpdateStmt) (*QueryPlan, error) {
	es, err := p.resolveEntity(stmt.Entity)
	if err != nil {
		return nil, err
	}

	if es.Immutable {
		return nil, fmt.Errorf("entity '%s' is immutable and cannot be updated", es.Name)
	}

	assignments, err := p.resolveAssignments(es, stmt.Assignments)
	if err != nil {
		return nil, err
	}

	return &QueryPlan{
		Type:        PlanUpdate,
		Entity:      es.Name,
		ID:          stmt.ID,
		Assignments: assignments,
	}, nil
}

// ── delete ────────────────────────────────────────────────────────────────────

func (p *Planner) planDelete(stmt *pql.DeleteStmt) (*QueryPlan, error) {
	es, err := p.resolveEntity(stmt.Entity)
	if err != nil {
		return nil, err
	}

	if es.Immutable {
		return nil, fmt.Errorf("entity '%s' is immutable and cannot be deleted", es.Name)
	}

	return &QueryPlan{
		Type:   PlanDelete,
		Entity: es.Name,
		ID:     stmt.ID,
	}, nil
}

// ── assignment resolution ────────────────────────────────────────────────────

func (p *Planner) resolveAssignments(es *schema.EntitySchema, assignments []pql.Assignment) (map[string]any, error) {
	result := make(map[string]any, len(assignments))

	for _, a := range assignments {
		colName, err := p.resolveField(es, a.Field)
		if err != nil {
			return nil, err
		}

		// Prevent setting computed/immutable fields
		if colName == "id" || colName == "created_at" || colName == "updated_at" {
			return nil, fmt.Errorf("field '%s' is computed and cannot be set", a.Field.String())
		}

		// Get field metadata for type coercion
		var fm *schema.FieldMeta
		if len(a.Field.Parts) == 1 {
			fm = es.Fields[a.Field.Parts[0]]
		}

		val, err := coerceLiteral(a.Value, fm)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", a.Field.String(), err)
		}

		result[colName] = val
	}

	return result, nil
}

// ── meta ─────────────────────────────────────────────────────────────────────

func (p *Planner) planMeta(stmt *pql.MetaCmdStmt) (*QueryPlan, error) {
	return &QueryPlan{
		Type:        PlanMeta,
		MetaCommand: stmt.Command,
		MetaArgs:    stmt.Args,
	}, nil
}

// ── Resolution helpers ──────────────────────────────────────────────────────

func (p *Planner) resolveEntity(name string) (*schema.EntitySchema, error) {
	es := p.registry.Entity(name)
	if es != nil {
		return es, nil
	}

	// Try fuzzy match
	suggestion := pql.SuggestFrom(name, p.registry.EntityNames(), 3)
	if suggestion != "" {
		return nil, fmt.Errorf("unknown entity '%s' (%s)", name, suggestion)
	}
	return nil, fmt.Errorf("unknown entity '%s'", name)
}

func (p *Planner) resolveField(es *schema.EntitySchema, fr pql.FieldRef) (string, error) {
	if len(fr.Parts) != 1 {
		// Phase 1: only simple field references (no dotted traversal)
		return "", fmt.Errorf("dotted field references not supported in Phase 1: %s", fr.String())
	}

	fieldName := fr.Parts[0]

	// Check for "id" specially
	if fieldName == "id" {
		return "id", nil
	}

	fm := es.Fields[fieldName]
	if fm != nil {
		return fm.EntColumn, nil
	}

	// Try fuzzy match
	fieldNames := es.FieldOrder
	suggestion := pql.SuggestFrom(fieldName, fieldNames, 3)
	if suggestion != "" {
		return "", fmt.Errorf("unknown field '%s' on entity '%s' (%s)", fieldName, es.Name, suggestion)
	}
	return "", fmt.Errorf("unknown field '%s' on entity '%s'", fieldName, es.Name)
}

func (p *Planner) resolveEdge(es *schema.EntitySchema, ep pql.EdgePath) (string, error) {
	if len(ep.Parts) != 1 {
		return "", fmt.Errorf("nested edge traversal not supported in Phase 1: %s", ep.String())
	}

	edgeName := ep.Parts[0]
	em := es.Edges[edgeName]
	if em != nil {
		return em.Name, nil
	}

	edgeNames := es.EdgeOrder
	suggestion := pql.SuggestFrom(edgeName, edgeNames, 3)
	if suggestion != "" {
		return "", fmt.Errorf("unknown edge '%s' on entity '%s' (%s)", edgeName, es.Name, suggestion)
	}
	return "", fmt.Errorf("unknown edge '%s' on entity '%s'", edgeName, es.Name)
}

// ── Predicate resolution ────────────────────────────────────────────────────

func (p *Planner) resolvePredicates(es *schema.EntitySchema, expr pql.Expr) ([]PredicateSpec, error) {
	// Flatten the expression tree into a list of AND-connected predicates.
	// For OR expressions, we wrap in a single predicate with OpIn where possible,
	// otherwise return an error (Phase 1 doesn't support full OR trees at execution).
	switch e := expr.(type) {
	case *pql.ComparisonExpr:
		spec, err := p.resolveComparison(es, e)
		if err != nil {
			return nil, err
		}
		return []PredicateSpec{spec}, nil

	case *pql.InExpr:
		spec, err := p.resolveInExpr(es, e)
		if err != nil {
			return nil, err
		}
		return []PredicateSpec{spec}, nil

	case *pql.BinaryLogicExpr:
		if e.Op == pql.LogicAnd {
			left, err := p.resolvePredicates(es, e.Left)
			if err != nil {
				return nil, err
			}
			right, err := p.resolvePredicates(es, e.Right)
			if err != nil {
				return nil, err
			}
			return append(left, right...), nil
		}
		// OR: not supported as flat predicates in Phase 1
		return nil, fmt.Errorf("OR expressions are not supported in Phase 1")

	case *pql.NotExpr:
		return nil, fmt.Errorf("NOT expressions are not supported in Phase 1")

	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func (p *Planner) resolveComparison(es *schema.EntitySchema, expr *pql.ComparisonExpr) (PredicateSpec, error) {
	colName, err := p.resolveField(es, expr.Field)
	if err != nil {
		return PredicateSpec{}, err
	}

	// Get field metadata for type checking
	var fm *schema.FieldMeta
	if len(expr.Field.Parts) == 1 && expr.Field.Parts[0] != "id" {
		fm = es.Fields[expr.Field.Parts[0]]
	}

	val, err := coerceLiteral(expr.Value, fm)
	if err != nil {
		return PredicateSpec{}, fmt.Errorf("field '%s': %w", expr.Field.String(), err)
	}

	op := mapCompOp(expr.Op)

	// Type-check: comparison operators only on comparable types
	if fm != nil && (op == OpGT || op == OpLT || op == OpGTE || op == OpLTE) {
		if !fm.Type.Comparable() {
			return PredicateSpec{}, fmt.Errorf("field '%s' (type %s) does not support operator %s",
				expr.Field.String(), fm.Type, op)
		}
	}

	return PredicateSpec{
		Field: colName,
		Op:    op,
		Value: val,
	}, nil
}

func (p *Planner) resolveInExpr(es *schema.EntitySchema, expr *pql.InExpr) (PredicateSpec, error) {
	colName, err := p.resolveField(es, expr.Field)
	if err != nil {
		return PredicateSpec{}, err
	}

	var fm *schema.FieldMeta
	if len(expr.Field.Parts) == 1 && expr.Field.Parts[0] != "id" {
		fm = es.Fields[expr.Field.Parts[0]]
	}

	var values []any
	for _, lit := range expr.Values {
		val, err := coerceLiteral(lit, fm)
		if err != nil {
			return PredicateSpec{}, fmt.Errorf("field '%s': %w", expr.Field.String(), err)
		}
		values = append(values, val)
	}

	return PredicateSpec{
		Field:  colName,
		Op:     OpIn,
		Values: values,
	}, nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func mapCompOp(op pql.CompOp) PredicateOp {
	switch op {
	case pql.CompEQ:
		return OpEQ
	case pql.CompNEQ:
		return OpNEQ
	case pql.CompGT:
		return OpGT
	case pql.CompLT:
		return OpLT
	case pql.CompGTE:
		return OpGTE
	case pql.CompLTE:
		return OpLTE
	case pql.CompLike:
		return OpLike
	default:
		return OpEQ
	}
}

// coerceLiteral converts a PQL literal to a Go value appropriate for the
// target field type. If fm is nil, minimal coercion is applied.
func coerceLiteral(lit pql.Literal, fm *schema.FieldMeta) (any, error) {
	switch lit.Type {
	case pql.LitString:
		if fm != nil && fm.Type == schema.FieldEnum {
			// Validate enum value
			valid := false
			for _, ev := range fm.EnumValues {
				if strings.EqualFold(lit.Raw, ev) {
					valid = true
					break
				}
			}
			if !valid {
				return nil, fmt.Errorf("invalid enum value '%s', valid values: %v", lit.Raw, fm.EnumValues)
			}
		}
		// Parse UUID strings for UUID-typed fields
		if fm != nil && fm.Type == schema.FieldUUID {
			id, err := uuid.Parse(lit.Raw)
			if err != nil {
				return nil, fmt.Errorf("invalid UUID: %s", lit.Raw)
			}
			return id, nil
		}
		return lit.Raw, nil

	case pql.LitInt:
		n, err := strconv.ParseInt(lit.Raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %s", lit.Raw)
		}
		if fm != nil && fm.Type == schema.FieldInt {
			return int(n), nil
		}
		return n, nil

	case pql.LitFloat:
		f, err := strconv.ParseFloat(lit.Raw, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float: %s", lit.Raw)
		}
		return f, nil

	case pql.LitBool:
		return strings.ToLower(lit.Raw) == "true", nil

	case pql.LitNull:
		return nil, nil

	default:
		return lit.Raw, nil
	}
}

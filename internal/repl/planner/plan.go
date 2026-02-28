// Package planner transforms PQL AST nodes into validated QueryPlans
// using the schema registry.
package planner

// PlanType identifies the kind of query plan.
type PlanType int

const (
	PlanFind PlanType = iota
	PlanGet
	PlanCount
	PlanMeta
	PlanCreate
	PlanUpdate
	PlanDelete
)

// QueryPlan is the validated, resolved plan ready for the executor.
type QueryPlan struct {
	Type   PlanType
	Entity string // PQL entity name (snake_case)

	// For PlanFind
	Predicates []PredicateSpec
	Fields     []string // Ent column names for select (nil = all)
	Edges      []string // Edge names to eager-load
	OrderBy    []OrderSpec
	Limit      int // 0 = use default
	Offset     int

	// For PlanGet
	ID string // UUID string

	// For PlanMeta
	MetaCommand string
	MetaArgs    []string

	// For PlanCreate / PlanUpdate
	Assignments map[string]any // Ent column name -> coerced Go value
}

// PredicateSpec is a resolved predicate for the executor.
type PredicateSpec struct {
	Field  string      // Ent column name
	Op     PredicateOp // Comparison operator
	Value  any         // Coerced Go value
	Values []any       // For OpIn
}

// PredicateOp enumerates comparison operators at the plan level.
type PredicateOp int

const (
	OpEQ PredicateOp = iota
	OpNEQ
	OpGT
	OpLT
	OpGTE
	OpLTE
	OpIn
	OpLike
	OpNotIn
)

// String returns the SQL-like operator symbol.
func (op PredicateOp) String() string {
	switch op {
	case OpEQ:
		return "="
	case OpNEQ:
		return "!="
	case OpGT:
		return ">"
	case OpLT:
		return "<"
	case OpGTE:
		return ">="
	case OpLTE:
		return "<="
	case OpIn:
		return "IN"
	case OpLike:
		return "LIKE"
	case OpNotIn:
		return "NOT IN"
	default:
		return "?"
	}
}

// OrderSpec is a resolved ordering specification.
type OrderSpec struct {
	Field string // Ent column name
	Desc  bool
}

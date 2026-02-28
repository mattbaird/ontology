package planner

import (
	"testing"

	"github.com/matthewbaird/ontology/internal/repl/pql"
	"github.com/matthewbaird/ontology/internal/repl/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRegistry builds a minimal registry for testing.
func testRegistry() *schema.Registry {
	r := schema.NewRegistry()
	r.Register(&schema.EntitySchema{
		Name:    "lease",
		EntName: "Lease",
		Fields: map[string]*schema.FieldMeta{
			"status": {
				Name:       "status",
				EntColumn:  "status",
				Type:       schema.FieldEnum,
				EnumValues: []string{"draft", "active", "expired", "terminated"},
			},
			"lease_type": {
				Name:       "lease_type",
				EntColumn:  "lease_type",
				Type:       schema.FieldEnum,
				EnumValues: []string{"fixed_term", "month_to_month"},
			},
			"description": {
				Name:      "description",
				EntColumn: "description",
				Type:      schema.FieldString,
				Optional:  true,
			},
			"base_rent_amount_cents": {
				Name:      "base_rent_amount_cents",
				EntColumn: "base_rent_amount_cents",
				Type:      schema.FieldInt64,
			},
		},
		FieldOrder: []string{"status", "lease_type", "description", "base_rent_amount_cents"},
		Edges: map[string]*schema.EdgeMeta{
			"lease_spaces": {
				Name:        "lease_spaces",
				Target:      "lease_space",
				Cardinality: "O2M",
			},
			"tenant_roles": {
				Name:        "tenant_roles",
				Target:      "person_role",
				Cardinality: "M2M",
			},
		},
		EdgeOrder:       []string{"lease_spaces", "tenant_roles"},
		HasStateMachine: true,
		StateMachine: map[string][]string{
			"draft":  {"active", "terminated"},
			"active": {"expired", "terminated"},
		},
	})

	r.Register(&schema.EntitySchema{
		Name:    "person",
		EntName: "Person",
		Fields: map[string]*schema.FieldMeta{
			"first_name": {
				Name:      "first_name",
				EntColumn: "first_name",
				Type:      schema.FieldString,
			},
			"last_name": {
				Name:      "last_name",
				EntColumn: "last_name",
				Type:      schema.FieldString,
			},
		},
		FieldOrder: []string{"first_name", "last_name"},
		Edges:      map[string]*schema.EdgeMeta{},
		EdgeOrder:  nil,
	})

	return r
}

func planPQL(t *testing.T, registry *schema.Registry, input string) *QueryPlan {
	t.Helper()
	lexer := pql.NewLexer(input)
	tokens, lexErrs := lexer.Tokenize()
	require.Empty(t, lexErrs)

	parser := pql.NewParser(tokens)
	stmts, parseErrs := parser.Parse()
	require.Empty(t, parseErrs)
	require.Len(t, stmts, 1)

	planner := New(registry)
	plan, err := planner.Plan(stmts[0])
	require.NoError(t, err)
	return plan
}

func TestPlanner_FindBasic(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, "find lease")

	assert.Equal(t, PlanFind, plan.Type)
	assert.Equal(t, "lease", plan.Entity)
	assert.Equal(t, DefaultLimit, plan.Limit)
	assert.Empty(t, plan.Predicates)
}

func TestPlanner_FindWithWhere(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, `find lease where status = "active"`)

	require.Len(t, plan.Predicates, 1)
	assert.Equal(t, "status", plan.Predicates[0].Field)
	assert.Equal(t, OpEQ, plan.Predicates[0].Op)
	assert.Equal(t, "active", plan.Predicates[0].Value)
}

func TestPlanner_FindWithAndWhere(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, `find lease where status = "active" and lease_type = "fixed_term"`)

	require.Len(t, plan.Predicates, 2)
	assert.Equal(t, "status", plan.Predicates[0].Field)
	assert.Equal(t, "lease_type", plan.Predicates[1].Field)
}

func TestPlanner_FindWithSelect(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, "find lease select status, lease_type")

	require.Len(t, plan.Fields, 2)
	assert.Equal(t, "status", plan.Fields[0])
	assert.Equal(t, "lease_type", plan.Fields[1])
}

func TestPlanner_FindWithInclude(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, "find lease include lease_spaces, tenant_roles")

	require.Len(t, plan.Edges, 2)
	assert.Equal(t, "lease_spaces", plan.Edges[0])
	assert.Equal(t, "tenant_roles", plan.Edges[1])
}

func TestPlanner_FindWithOrderBy(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, "find lease order by status desc")

	require.Len(t, plan.OrderBy, 1)
	assert.Equal(t, "status", plan.OrderBy[0].Field)
	assert.True(t, plan.OrderBy[0].Desc)
}

func TestPlanner_FindWithLimit(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, "find lease limit 25")

	assert.Equal(t, 25, plan.Limit)
}

func TestPlanner_FindWithOffset(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, "find lease offset 50")

	assert.Equal(t, 50, plan.Offset)
}

func TestPlanner_GetBasic(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, `get lease "550e8400-e29b-41d4-a716-446655440000"`)

	assert.Equal(t, PlanGet, plan.Type)
	assert.Equal(t, "lease", plan.Entity)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", plan.ID)
}

func TestPlanner_CountBasic(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, "count lease")

	assert.Equal(t, PlanCount, plan.Type)
	assert.Equal(t, "lease", plan.Entity)
}

func TestPlanner_Meta(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, ":help")

	assert.Equal(t, PlanMeta, plan.Type)
	assert.Equal(t, "help", plan.MetaCommand)
}

func TestPlanner_UnknownEntity(t *testing.T) {
	reg := testRegistry()
	lexer := pql.NewLexer("find unicorn")
	tokens, _ := lexer.Tokenize()
	parser := pql.NewParser(tokens)
	stmts, _ := parser.Parse()

	planner := New(reg)
	_, err := planner.Plan(stmts[0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown entity")
}

func TestPlanner_FuzzyEntitySuggestion(t *testing.T) {
	reg := testRegistry()
	lexer := pql.NewLexer("find lese")
	tokens, _ := lexer.Tokenize()
	parser := pql.NewParser(tokens)
	stmts, _ := parser.Parse()

	planner := New(reg)
	_, err := planner.Plan(stmts[0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did you mean")
}

func TestPlanner_UnknownField(t *testing.T) {
	reg := testRegistry()
	lexer := pql.NewLexer(`find lease where nonexistent = "foo"`)
	tokens, _ := lexer.Tokenize()
	parser := pql.NewParser(tokens)
	stmts, _ := parser.Parse()

	planner := New(reg)
	_, err := planner.Plan(stmts[0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestPlanner_UnknownEdge(t *testing.T) {
	reg := testRegistry()
	lexer := pql.NewLexer("find lease include nonexistent")
	tokens, _ := lexer.Tokenize()
	parser := pql.NewParser(tokens)
	stmts, _ := parser.Parse()

	planner := New(reg)
	_, err := planner.Plan(stmts[0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown edge")
}

func TestPlanner_InvalidEnumValue(t *testing.T) {
	reg := testRegistry()
	lexer := pql.NewLexer(`find lease where status = "bogus"`)
	tokens, _ := lexer.Tokenize()
	parser := pql.NewParser(tokens)
	stmts, _ := parser.Parse()

	planner := New(reg)
	_, err := planner.Plan(stmts[0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid enum value")
}

func TestPlanner_ComparisonOnNonComparable(t *testing.T) {
	reg := testRegistry()
	lexer := pql.NewLexer(`find lease where status > "active"`)
	tokens, _ := lexer.Tokenize()
	parser := pql.NewParser(tokens)
	stmts, _ := parser.Parse()

	planner := New(reg)
	_, err := planner.Plan(stmts[0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support operator")
}

func TestPlanner_ComparisonOnComparable(t *testing.T) {
	reg := testRegistry()
	// Int64 fields should support > operator
	plan := planPQL(t, reg, `find lease where base_rent_amount_cents > 100000`)

	require.Len(t, plan.Predicates, 1)
	assert.Equal(t, OpGT, plan.Predicates[0].Op)
}

func TestPlanner_IdField(t *testing.T) {
	reg := testRegistry()
	plan := planPQL(t, reg, `find lease where id = "some-uuid"`)

	require.Len(t, plan.Predicates, 1)
	assert.Equal(t, "id", plan.Predicates[0].Field)
}

package pql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, input string) []Statement {
	t.Helper()
	lexer := NewLexer(input)
	tokens, lexErrs := lexer.Tokenize()
	require.Empty(t, lexErrs, "lex errors")

	parser := NewParser(tokens)
	stmts, parseErrs := parser.Parse()
	require.Empty(t, parseErrs, "parse errors")
	return stmts
}

func TestParser_FindBasic(t *testing.T) {
	stmts := parse(t, "find lease")
	require.Len(t, stmts, 1)

	find, ok := stmts[0].(*FindStmt)
	require.True(t, ok)
	assert.Equal(t, "lease", find.Entity)
	assert.Nil(t, find.Where)
	assert.Nil(t, find.Select)
	assert.Nil(t, find.Include)
	assert.Nil(t, find.OrderBy)
	assert.Nil(t, find.Limit)
	assert.Nil(t, find.Offset)
}

func TestParser_FindWithWhere(t *testing.T) {
	stmts := parse(t, `find lease where status = "active"`)
	require.Len(t, stmts, 1)

	find := stmts[0].(*FindStmt)
	assert.Equal(t, "lease", find.Entity)
	require.NotNil(t, find.Where)

	comp, ok := find.Where.Expr.(*ComparisonExpr)
	require.True(t, ok)
	assert.Equal(t, "status", comp.Field.Parts[0])
	assert.Equal(t, CompEQ, comp.Op)
	assert.Equal(t, "active", comp.Value.Raw)
}

func TestParser_FindWithAndWhere(t *testing.T) {
	stmts := parse(t, `find property where status = "active" and year_built >= 2000`)
	find := stmts[0].(*FindStmt)

	logic, ok := find.Where.Expr.(*BinaryLogicExpr)
	require.True(t, ok)
	assert.Equal(t, LogicAnd, logic.Op)

	left := logic.Left.(*ComparisonExpr)
	assert.Equal(t, "status", left.Field.Parts[0])
	assert.Equal(t, CompEQ, left.Op)

	right := logic.Right.(*ComparisonExpr)
	assert.Equal(t, "year_built", right.Field.Parts[0])
	assert.Equal(t, CompGTE, right.Op)
}

func TestParser_FindWithOrWhere(t *testing.T) {
	stmts := parse(t, `find space where status = "vacant" or status = "available"`)
	find := stmts[0].(*FindStmt)

	logic, ok := find.Where.Expr.(*BinaryLogicExpr)
	require.True(t, ok)
	assert.Equal(t, LogicOr, logic.Op)
}

func TestParser_FindWithIn(t *testing.T) {
	stmts := parse(t, `find lease where status in ["active", "draft"]`)
	find := stmts[0].(*FindStmt)

	inExpr, ok := find.Where.Expr.(*InExpr)
	require.True(t, ok)
	assert.Equal(t, "status", inExpr.Field.Parts[0])
	assert.Len(t, inExpr.Values, 2)
	assert.Equal(t, "active", inExpr.Values[0].Raw)
	assert.Equal(t, "draft", inExpr.Values[1].Raw)
}

func TestParser_FindWithLike(t *testing.T) {
	stmts := parse(t, `find person where first_name like "John"`)
	find := stmts[0].(*FindStmt)

	comp, ok := find.Where.Expr.(*ComparisonExpr)
	require.True(t, ok)
	assert.Equal(t, CompLike, comp.Op)
	assert.Equal(t, "John", comp.Value.Raw)
}

func TestParser_FindWithSelect(t *testing.T) {
	stmts := parse(t, "find lease select status, lease_type")
	find := stmts[0].(*FindStmt)

	require.NotNil(t, find.Select)
	assert.Len(t, find.Select.Fields, 2)
	assert.Equal(t, "status", find.Select.Fields[0].String())
	assert.Equal(t, "lease_type", find.Select.Fields[1].String())
}

func TestParser_FindWithInclude(t *testing.T) {
	stmts := parse(t, "find lease include lease_spaces, tenant_roles")
	find := stmts[0].(*FindStmt)

	require.NotNil(t, find.Include)
	assert.Len(t, find.Include.Edges, 2)
	assert.Equal(t, "lease_spaces", find.Include.Edges[0].String())
	assert.Equal(t, "tenant_roles", find.Include.Edges[1].String())
}

func TestParser_FindWithOrderBy(t *testing.T) {
	stmts := parse(t, "find lease order by created_at desc")
	find := stmts[0].(*FindStmt)

	require.NotNil(t, find.OrderBy)
	assert.Len(t, find.OrderBy.Items, 1)
	assert.Equal(t, "created_at", find.OrderBy.Items[0].Field.String())
	assert.True(t, find.OrderBy.Items[0].Desc)
}

func TestParser_FindWithLimit(t *testing.T) {
	stmts := parse(t, "find lease limit 25")
	find := stmts[0].(*FindStmt)

	require.NotNil(t, find.Limit)
	assert.Equal(t, 25, find.Limit.Value)
}

func TestParser_FindWithOffset(t *testing.T) {
	stmts := parse(t, "find lease offset 50")
	find := stmts[0].(*FindStmt)

	require.NotNil(t, find.Offset)
	assert.Equal(t, 50, find.Offset.Value)
}

func TestParser_FindWithAllClauses(t *testing.T) {
	stmts := parse(t, `find lease where status = "active" select status, lease_type include lease_spaces order by created_at desc limit 10 offset 20`)
	find := stmts[0].(*FindStmt)

	assert.NotNil(t, find.Where)
	assert.NotNil(t, find.Select)
	assert.NotNil(t, find.Include)
	assert.NotNil(t, find.OrderBy)
	assert.NotNil(t, find.Limit)
	assert.Equal(t, 10, find.Limit.Value)
	assert.NotNil(t, find.Offset)
	assert.Equal(t, 20, find.Offset.Value)
}

func TestParser_FindClausesAnyOrder(t *testing.T) {
	// Clauses should work in any order
	stmts := parse(t, `find lease limit 10 where status = "active" order by created_at desc`)
	find := stmts[0].(*FindStmt)

	assert.NotNil(t, find.Where)
	assert.NotNil(t, find.OrderBy)
	assert.NotNil(t, find.Limit)
	assert.Equal(t, 10, find.Limit.Value)
}

func TestParser_GetBasic(t *testing.T) {
	stmts := parse(t, `get lease "550e8400-e29b-41d4-a716-446655440000"`)
	require.Len(t, stmts, 1)

	getStmt, ok := stmts[0].(*GetStmt)
	require.True(t, ok)
	assert.Equal(t, "lease", getStmt.Entity)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", getStmt.ID)
}

func TestParser_GetWithInclude(t *testing.T) {
	stmts := parse(t, `get lease "some-id" include lease_spaces`)
	getStmt := stmts[0].(*GetStmt)

	require.NotNil(t, getStmt.Include)
	assert.Len(t, getStmt.Include.Edges, 1)
}

func TestParser_CountBasic(t *testing.T) {
	stmts := parse(t, "count lease")
	require.Len(t, stmts, 1)

	countStmt, ok := stmts[0].(*CountStmt)
	require.True(t, ok)
	assert.Equal(t, "lease", countStmt.Entity)
	assert.Nil(t, countStmt.Where)
}

func TestParser_CountWithWhere(t *testing.T) {
	stmts := parse(t, `count lease where status = "active"`)
	countStmt := stmts[0].(*CountStmt)

	require.NotNil(t, countStmt.Where)
}

func TestParser_MetaCommand(t *testing.T) {
	stmts := parse(t, ":help find")
	require.Len(t, stmts, 1)

	meta, ok := stmts[0].(*MetaCmdStmt)
	require.True(t, ok)
	assert.Equal(t, "help", meta.Command)
	assert.Equal(t, []string{"find"}, meta.Args)
}

func TestParser_MetaClear(t *testing.T) {
	stmts := parse(t, ":clear")
	meta := stmts[0].(*MetaCmdStmt)
	assert.Equal(t, "clear", meta.Command)
	assert.Empty(t, meta.Args)
}

func TestParser_NotExpression(t *testing.T) {
	stmts := parse(t, `find lease where not status = "terminated"`)
	find := stmts[0].(*FindStmt)

	notExpr, ok := find.Where.Expr.(*NotExpr)
	require.True(t, ok)
	assert.NotNil(t, notExpr.Expr)
}

func TestParser_ParenthesizedExpression(t *testing.T) {
	stmts := parse(t, `find lease where (status = "active" or status = "draft") and lease_type = "fixed_term"`)
	find := stmts[0].(*FindStmt)

	logic, ok := find.Where.Expr.(*BinaryLogicExpr)
	require.True(t, ok)
	assert.Equal(t, LogicAnd, logic.Op)

	// Left should be OR expression (parenthesized)
	leftLogic, ok := logic.Left.(*BinaryLogicExpr)
	require.True(t, ok)
	assert.Equal(t, LogicOr, leftLogic.Op)
}

func TestParser_DuplicateClauseError(t *testing.T) {
	lexer := NewLexer(`find lease where status = "active" where status = "draft"`)
	tokens, _ := lexer.Tokenize()
	parser := NewParser(tokens)
	_, errs := parser.Parse()
	assert.NotEmpty(t, errs)
}

func TestParser_FutureVerbError(t *testing.T) {
	lexer := NewLexer("run MoveInTenant")
	tokens, _ := lexer.Tokenize()
	parser := NewParser(tokens)
	_, errs := parser.Parse()
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "not yet implemented")
}

func TestParser_EntityNameNormalized(t *testing.T) {
	stmts := parse(t, "find Lease")
	find := stmts[0].(*FindStmt)
	assert.Equal(t, "lease", find.Entity)
}

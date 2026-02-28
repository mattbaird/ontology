package pql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLexer_Keywords(t *testing.T) {
	input := "find lease where status = \"active\" limit 10"
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{TokenFind, "find"},
		{TokenIdent, "lease"},
		{TokenWhere, "where"},
		{TokenIdent, "status"},
		{TokenEQ, "="},
		{TokenString, "active"},
		{TokenLimit, "limit"},
		{TokenInt, "10"},
		{TokenEOF, ""},
	}

	require.Len(t, tokens, len(expected))
	for i, exp := range expected {
		assert.Equal(t, exp.typ, tokens[i].Type, "token %d type", i)
		assert.Equal(t, exp.lit, tokens[i].Literal, "token %d literal", i)
	}
}

func TestLexer_CaseInsensitiveKeywords(t *testing.T) {
	input := "FIND Lease WHERE Status"
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	assert.Equal(t, TokenFind, tokens[0].Type)
	assert.Equal(t, TokenIdent, tokens[1].Type)
	assert.Equal(t, TokenWhere, tokens[2].Type)
	assert.Equal(t, TokenIdent, tokens[3].Type)
}

func TestLexer_Operators(t *testing.T) {
	input := `= != > < >= <=`
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	expected := []TokenType{TokenEQ, TokenNEQ, TokenGT, TokenLT, TokenGTE, TokenLTE, TokenEOF}
	for i, exp := range expected {
		assert.Equal(t, exp, tokens[i].Type, "token %d", i)
	}
}

func TestLexer_StringLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`'world'`, "world"},
		{`"with \"escape\""`, `with "escape"`},
		{`"line\nbreak"`, "line\nbreak"},
	}

	for _, tt := range tests {
		lexer := NewLexer(tt.input)
		tokens, errs := lexer.Tokenize()
		require.Empty(t, errs)
		require.GreaterOrEqual(t, len(tokens), 2)
		assert.Equal(t, TokenString, tokens[0].Type)
		assert.Equal(t, tt.expected, tokens[0].Literal)
	}
}

func TestLexer_Numbers(t *testing.T) {
	input := "42 3.14 0 100"
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	assert.Equal(t, TokenInt, tokens[0].Type)
	assert.Equal(t, "42", tokens[0].Literal)

	assert.Equal(t, TokenFloat, tokens[1].Type)
	assert.Equal(t, "3.14", tokens[1].Literal)

	assert.Equal(t, TokenInt, tokens[2].Type)
	assert.Equal(t, "0", tokens[2].Literal)
}

func TestLexer_MetaCommand(t *testing.T) {
	input := ":help find"
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	assert.Equal(t, TokenMetaCmd, tokens[0].Type)
	assert.Equal(t, ":help", tokens[0].Literal)
	assert.Equal(t, TokenFind, tokens[1].Type)
}

func TestLexer_Flag(t *testing.T) {
	input := "--dry-run --confirm"
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	assert.Equal(t, TokenFlag, tokens[0].Type)
	assert.Equal(t, "--dry-run", tokens[0].Literal)
	assert.Equal(t, TokenFlag, tokens[1].Type)
	assert.Equal(t, "--confirm", tokens[1].Literal)
}

func TestLexer_Comment(t *testing.T) {
	input := "find lease -- this is a comment\ncount person"
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	// Comments should be skipped
	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	assert.Equal(t, TokenFind, tokens[0].Type)
	assert.Equal(t, TokenIdent, tokens[1].Type)
	assert.Equal(t, TokenCount, tokens[2].Type)
}

func TestLexer_BoolAndNull(t *testing.T) {
	input := "true false null"
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	assert.Equal(t, TokenBool, tokens[0].Type)
	assert.Equal(t, "true", tokens[0].Literal)
	assert.Equal(t, TokenBool, tokens[1].Type)
	assert.Equal(t, "false", tokens[1].Literal)
	assert.Equal(t, TokenNull, tokens[2].Type)
}

func TestLexer_ArraySyntax(t *testing.T) {
	input := `status in ["active", "draft"]`
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)

	expected := []TokenType{
		TokenIdent, TokenIn, TokenLBrack, TokenString, TokenComma, TokenString, TokenRBrack, TokenEOF,
	}
	for i, exp := range expected {
		assert.Equal(t, exp, tokens[i].Type, "token %d", i)
	}
}

func TestLexer_ComplexQuery(t *testing.T) {
	input := `find property where status = "active" and year_built >= 2000 select status, property_type order by created_at desc limit 25 offset 50`
	lexer := NewLexer(input)
	tokens, errs := lexer.Tokenize()
	require.Empty(t, errs)
	// Just verify no errors and reasonable token count
	assert.Greater(t, len(tokens), 15)
}

func TestLexer_LinePositions(t *testing.T) {
	input := "find\nlease"
	lexer := NewLexer(input)
	tokens, _ := lexer.Tokenize()

	assert.Equal(t, 1, tokens[0].Line)
	assert.Equal(t, 2, tokens[1].Line)
}

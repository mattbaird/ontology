package pql

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser implements a recursive descent parser for PQL.
type Parser struct {
	tokens []Token
	pos    int
	errors []*ParseError
}

// NewParser creates a parser from a token slice (typically from Lexer.Tokenize).
func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens}
}

// Parse parses the token stream into a list of statements.
func (p *Parser) Parse() ([]Statement, []*ParseError) {
	var stmts []Statement
	for !p.atEnd() {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	return stmts, p.errors
}

// ── Token navigation ────────────────────────────────────────────────────────

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	tok := p.peek()
	if tok.Type != TokenEOF {
		p.pos++
	}
	return tok
}

func (p *Parser) atEnd() bool {
	return p.peek().Type == TokenEOF
}

func (p *Parser) check(t TokenType) bool {
	return p.peek().Type == t
}

func (p *Parser) match(types ...TokenType) (Token, bool) {
	for _, t := range types {
		if p.check(t) {
			return p.advance(), true
		}
	}
	return Token{}, false
}

func (p *Parser) expect(t TokenType) (Token, bool) {
	if p.check(t) {
		return p.advance(), true
	}
	tok := p.peek()
	p.addError(tok, fmt.Sprintf("expected %s, got %s", t, tok.Type))
	return tok, false
}

func (p *Parser) addError(tok Token, msg string) {
	p.errors = append(p.errors, &ParseError{
		Message: msg,
		Line:    tok.Line,
		Col:     tok.Col,
		Pos:     tok.Pos,
	})
}

func (p *Parser) addErrorWithSuggestion(tok Token, msg, suggestion string) {
	p.errors = append(p.errors, &ParseError{
		Message:    msg,
		Line:       tok.Line,
		Col:        tok.Col,
		Pos:        tok.Pos,
		Suggestion: suggestion,
	})
}

// synchronize skips tokens until a statement boundary (verb or meta-command).
func (p *Parser) synchronize() {
	for !p.atEnd() {
		tok := p.peek()
		if tok.Type.IsVerb() || tok.Type == TokenMetaCmd {
			return
		}
		p.advance()
	}
}

// ── Statement parsing ───────────────────────────────────────────────────────

func (p *Parser) parseStatement() Statement {
	tok := p.peek()

	switch tok.Type {
	case TokenFind:
		return p.parseFind()
	case TokenGet:
		return p.parseGet()
	case TokenCount:
		return p.parseCount()
	case TokenCreate:
		return p.parseCreate()
	case TokenUpdate:
		return p.parseUpdate()
	case TokenDelete:
		return p.parseDelete()
	case TokenMetaCmd:
		return p.parseMetaCmd()

	// Future verbs — produce clear errors
	case TokenRun, TokenDescribe, TokenExplain,
		TokenHistory, TokenDiff, TokenAggregate, TokenWatch:
		p.addError(tok, fmt.Sprintf("'%s' is not yet implemented", tok.Literal))
		p.advance()
		p.synchronize()
		return nil

	default:
		p.addError(tok, fmt.Sprintf("expected a PQL verb (find, get, count, create, update, delete) or meta-command, got %s", tok.Type))
		p.advance()
		p.synchronize()
		return nil
	}
}

// ── find ─────────────────────────────────────────────────────────────────────

func (p *Parser) parseFind() *FindStmt {
	tok := p.advance() // consume 'find'
	stmt := &FindStmt{TokenPos: tok.Pos}

	// Entity name
	entTok, ok := p.expect(TokenIdent)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.Entity = strings.ToLower(entTok.Literal)

	// Clauses in any order
	for !p.atEnd() && !p.peek().Type.IsVerb() && p.peek().Type != TokenMetaCmd {
		switch p.peek().Type {
		case TokenWhere:
			if stmt.Where != nil {
				p.addError(p.peek(), "duplicate 'where' clause")
				p.advance()
				p.synchronize()
				return stmt
			}
			stmt.Where = p.parseWhere()
		case TokenSelect:
			if stmt.Select != nil {
				p.addError(p.peek(), "duplicate 'select' clause")
				p.advance()
				p.synchronize()
				return stmt
			}
			stmt.Select = p.parseSelect()
		case TokenInclude:
			if stmt.Include != nil {
				p.addError(p.peek(), "duplicate 'include' clause")
				p.advance()
				p.synchronize()
				return stmt
			}
			stmt.Include = p.parseInclude()
		case TokenOrder:
			if stmt.OrderBy != nil {
				p.addError(p.peek(), "duplicate 'order by' clause")
				p.advance()
				p.synchronize()
				return stmt
			}
			stmt.OrderBy = p.parseOrderBy()
		case TokenLimit:
			if stmt.Limit != nil {
				p.addError(p.peek(), "duplicate 'limit' clause")
				p.advance()
				p.synchronize()
				return stmt
			}
			stmt.Limit = p.parseLimit()
		case TokenOffset:
			if stmt.Offset != nil {
				p.addError(p.peek(), "duplicate 'offset' clause")
				p.advance()
				p.synchronize()
				return stmt
			}
			stmt.Offset = p.parseOffset()
		default:
			// Unknown token in clause position
			p.addError(p.peek(), fmt.Sprintf("unexpected %s in find statement", p.peek().Type))
			p.advance()
			return stmt
		}
	}

	return stmt
}

// ── get ──────────────────────────────────────────────────────────────────────

func (p *Parser) parseGet() *GetStmt {
	tok := p.advance() // consume 'get'
	stmt := &GetStmt{TokenPos: tok.Pos}

	// Entity name
	entTok, ok := p.expect(TokenIdent)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.Entity = strings.ToLower(entTok.Literal)

	// ID (string literal)
	idTok, ok := p.expect(TokenString)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.ID = idTok.Literal

	// Optional include
	if p.check(TokenInclude) {
		stmt.Include = p.parseInclude()
	}

	return stmt
}

// ── count ────────────────────────────────────────────────────────────────────

func (p *Parser) parseCount() *CountStmt {
	tok := p.advance() // consume 'count'
	stmt := &CountStmt{TokenPos: tok.Pos}

	// Entity name
	entTok, ok := p.expect(TokenIdent)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.Entity = strings.ToLower(entTok.Literal)

	// Optional where
	if p.check(TokenWhere) {
		stmt.Where = p.parseWhere()
	}

	return stmt
}

// ── meta-command ─────────────────────────────────────────────────────────────

func (p *Parser) parseMetaCmd() *MetaCmdStmt {
	tok := p.advance() // consume meta-command token
	stmt := &MetaCmdStmt{
		TokenPos: tok.Pos,
		Command:  strings.TrimPrefix(tok.Literal, ":"),
	}

	// Collect all remaining tokens as args (meta-commands consume the rest of the input)
	for !p.atEnd() && p.peek().Type != TokenMetaCmd {
		arg := p.advance()
		stmt.Args = append(stmt.Args, arg.Literal)
	}

	return stmt
}

// ── create ────────────────────────────────────────────────────────────────────

func (p *Parser) parseCreate() *CreateStmt {
	tok := p.advance() // consume 'create'
	stmt := &CreateStmt{TokenPos: tok.Pos}

	// Entity name
	entTok, ok := p.expect(TokenIdent)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.Entity = strings.ToLower(entTok.Literal)

	// Expect 'set' keyword
	if _, ok := p.expect(TokenSet); !ok {
		p.synchronize()
		return nil
	}

	// Parse assignments
	stmt.Assignments = p.parseAssignments()
	if len(stmt.Assignments) == 0 {
		p.addError(p.peek(), "expected at least one field assignment after 'set'")
		return nil
	}

	return stmt
}

// ── update ────────────────────────────────────────────────────────────────────

func (p *Parser) parseUpdate() *UpdateStmt {
	tok := p.advance() // consume 'update'
	stmt := &UpdateStmt{TokenPos: tok.Pos}

	// Entity name
	entTok, ok := p.expect(TokenIdent)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.Entity = strings.ToLower(entTok.Literal)

	// ID (string literal)
	idTok, ok := p.expect(TokenString)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.ID = idTok.Literal

	// Expect 'set' keyword
	if _, ok := p.expect(TokenSet); !ok {
		p.synchronize()
		return nil
	}

	// Parse assignments
	stmt.Assignments = p.parseAssignments()
	if len(stmt.Assignments) == 0 {
		p.addError(p.peek(), "expected at least one field assignment after 'set'")
		return nil
	}

	return stmt
}

// ── delete ────────────────────────────────────────────────────────────────────

func (p *Parser) parseDelete() *DeleteStmt {
	tok := p.advance() // consume 'delete'
	stmt := &DeleteStmt{TokenPos: tok.Pos}

	// Entity name
	entTok, ok := p.expect(TokenIdent)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.Entity = strings.ToLower(entTok.Literal)

	// ID (string literal)
	idTok, ok := p.expect(TokenString)
	if !ok {
		p.synchronize()
		return nil
	}
	stmt.ID = idTok.Literal

	return stmt
}

// ── assignments ──────────────────────────────────────────────────────────────

func (p *Parser) parseAssignments() []Assignment {
	var assignments []Assignment

	for {
		if !p.check(TokenIdent) {
			break
		}

		field := p.parseFieldRef()

		if _, ok := p.expect(TokenEQ); !ok {
			break
		}

		val := p.parseLiteral()
		assignments = append(assignments, Assignment{Field: field, Value: val})

		// Optional comma between assignments
		if !p.check(TokenComma) {
			break
		}
		p.advance() // consume ','
	}

	return assignments
}

// ── WHERE clause ─────────────────────────────────────────────────────────────

func (p *Parser) parseWhere() *WhereClause {
	p.advance() // consume 'where'
	expr := p.parseOrExpr()
	if expr == nil {
		return nil
	}
	return &WhereClause{Expr: expr}
}

func (p *Parser) parseOrExpr() Expr {
	left := p.parseAndExpr()
	if left == nil {
		return nil
	}
	for p.check(TokenOr) {
		opTok := p.advance()
		right := p.parseAndExpr()
		if right == nil {
			return left
		}
		left = &BinaryLogicExpr{
			TokenPos: opTok.Pos,
			Op:       LogicOr,
			Left:     left,
			Right:    right,
		}
	}
	return left
}

func (p *Parser) parseAndExpr() Expr {
	left := p.parseUnaryExpr()
	if left == nil {
		return nil
	}
	for p.check(TokenAnd) {
		opTok := p.advance()
		right := p.parseUnaryExpr()
		if right == nil {
			return left
		}
		left = &BinaryLogicExpr{
			TokenPos: opTok.Pos,
			Op:       LogicAnd,
			Left:     left,
			Right:    right,
		}
	}
	return left
}

func (p *Parser) parseUnaryExpr() Expr {
	if tok, ok := p.match(TokenNot); ok {
		expr := p.parseUnaryExpr()
		if expr == nil {
			return nil
		}
		return &NotExpr{TokenPos: tok.Pos, Expr: expr}
	}

	if p.check(TokenLParen) {
		p.advance() // consume '('
		expr := p.parseOrExpr()
		p.expect(TokenRParen) // consume ')'
		return expr
	}

	return p.parseComparison()
}

func (p *Parser) parseComparison() Expr {
	if !p.check(TokenIdent) {
		p.addError(p.peek(), fmt.Sprintf("expected field name, got %s", p.peek().Type))
		return nil
	}

	field := p.parseFieldRef()
	startPos := p.peek().Pos

	// IN expression
	if p.check(TokenIn) {
		p.advance() // consume 'in'
		values := p.parseArrayLiteral()
		return &InExpr{
			TokenPos: startPos,
			Field:    field,
			Values:   values,
		}
	}

	// LIKE expression
	if p.check(TokenLike) {
		p.advance() // consume 'like'
		val := p.parseLiteral()
		return &ComparisonExpr{
			TokenPos: startPos,
			Field:    field,
			Op:       CompLike,
			Value:    val,
		}
	}

	// Comparison operators
	op, ok := p.parseCompOp()
	if !ok {
		p.addError(p.peek(), fmt.Sprintf("expected comparison operator (=, !=, >, <, >=, <=, like, in), got %s", p.peek().Type))
		return nil
	}

	val := p.parseLiteral()
	return &ComparisonExpr{
		TokenPos: startPos,
		Field:    field,
		Op:       op,
		Value:    val,
	}
}

func (p *Parser) parseCompOp() (CompOp, bool) {
	switch p.peek().Type {
	case TokenEQ:
		p.advance()
		return CompEQ, true
	case TokenNEQ:
		p.advance()
		return CompNEQ, true
	case TokenGT:
		p.advance()
		return CompGT, true
	case TokenLT:
		p.advance()
		return CompLT, true
	case TokenGTE:
		p.advance()
		return CompGTE, true
	case TokenLTE:
		p.advance()
		return CompLTE, true
	default:
		return 0, false
	}
}

func (p *Parser) parseLiteral() Literal {
	tok := p.peek()
	switch tok.Type {
	case TokenString:
		p.advance()
		return Literal{TokenPos: tok.Pos, Type: LitString, Raw: tok.Literal}
	case TokenInt:
		p.advance()
		return Literal{TokenPos: tok.Pos, Type: LitInt, Raw: tok.Literal}
	case TokenFloat:
		p.advance()
		return Literal{TokenPos: tok.Pos, Type: LitFloat, Raw: tok.Literal}
	case TokenBool:
		p.advance()
		return Literal{TokenPos: tok.Pos, Type: LitBool, Raw: tok.Literal}
	case TokenNull:
		p.advance()
		return Literal{TokenPos: tok.Pos, Type: LitNull, Raw: tok.Literal}
	default:
		p.addError(tok, fmt.Sprintf("expected literal value, got %s", tok.Type))
		p.advance()
		return Literal{TokenPos: tok.Pos, Type: LitNull, Raw: "null"}
	}
}

func (p *Parser) parseArrayLiteral() []Literal {
	if _, ok := p.expect(TokenLBrack); !ok {
		return nil
	}

	var values []Literal
	for !p.check(TokenRBrack) && !p.atEnd() {
		val := p.parseLiteral()
		values = append(values, val)
		if !p.check(TokenRBrack) {
			if _, ok := p.expect(TokenComma); !ok {
				break
			}
		}
	}

	p.expect(TokenRBrack) // consume ']'
	return values
}

// ── SELECT clause ───────────────────────────────────────────────────────────

func (p *Parser) parseSelect() *SelectClause {
	p.advance() // consume 'select'
	clause := &SelectClause{}

	// First field
	if !p.check(TokenIdent) && !p.check(TokenStar) {
		p.addError(p.peek(), "expected field name after 'select'")
		return nil
	}

	if p.check(TokenStar) {
		p.advance()
		// select * is default behavior — return nil to signal "all fields"
		return nil
	}

	clause.Fields = append(clause.Fields, p.parseFieldRef())

	// Additional fields
	for p.check(TokenComma) {
		p.advance() // consume ','
		clause.Fields = append(clause.Fields, p.parseFieldRef())
	}

	return clause
}

// ── INCLUDE clause ──────────────────────────────────────────────────────────

func (p *Parser) parseInclude() *IncludeClause {
	p.advance() // consume 'include'
	clause := &IncludeClause{}

	if !p.check(TokenIdent) {
		p.addError(p.peek(), "expected edge name after 'include'")
		return nil
	}

	clause.Edges = append(clause.Edges, p.parseEdgePath())

	for p.check(TokenComma) {
		p.advance() // consume ','
		clause.Edges = append(clause.Edges, p.parseEdgePath())
	}

	return clause
}

// ── ORDER BY clause ─────────────────────────────────────────────────────────

func (p *Parser) parseOrderBy() *OrderByClause {
	p.advance() // consume 'order'
	if _, ok := p.expect(TokenBy); !ok {
		return nil
	}

	clause := &OrderByClause{}
	clause.Items = append(clause.Items, p.parseOrderItem())

	for p.check(TokenComma) {
		p.advance() // consume ','
		clause.Items = append(clause.Items, p.parseOrderItem())
	}

	return clause
}

func (p *Parser) parseOrderItem() OrderItem {
	field := p.parseFieldRef()
	item := OrderItem{Field: field}

	if p.check(TokenDesc) {
		p.advance()
		item.Desc = true
	} else if p.check(TokenAsc) {
		p.advance()
		// asc is default
	}

	return item
}

// ── LIMIT / OFFSET ──────────────────────────────────────────────────────────

func (p *Parser) parseLimit() *LimitClause {
	p.advance() // consume 'limit'
	tok, ok := p.expect(TokenInt)
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(tok.Literal)
	if err != nil {
		p.addError(tok, fmt.Sprintf("invalid limit value: %s", tok.Literal))
		return nil
	}
	return &LimitClause{Value: n}
}

func (p *Parser) parseOffset() *OffsetClause {
	p.advance() // consume 'offset'
	tok, ok := p.expect(TokenInt)
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(tok.Literal)
	if err != nil {
		p.addError(tok, fmt.Sprintf("invalid offset value: %s", tok.Literal))
		return nil
	}
	return &OffsetClause{Value: n}
}

// ── Field and edge references ───────────────────────────────────────────────

func (p *Parser) parseFieldRef() FieldRef {
	ref := FieldRef{}
	tok := p.advance() // first identifier
	ref.Parts = append(ref.Parts, strings.ToLower(tok.Literal))

	for p.check(TokenDot) {
		p.advance() // consume '.'
		if !p.check(TokenIdent) {
			p.addError(p.peek(), "expected field name after '.'")
			break
		}
		tok = p.advance()
		ref.Parts = append(ref.Parts, strings.ToLower(tok.Literal))
	}

	return ref
}

func (p *Parser) parseEdgePath() EdgePath {
	ep := EdgePath{}
	tok := p.advance() // first identifier
	ep.Parts = append(ep.Parts, strings.ToLower(tok.Literal))

	for p.check(TokenDot) {
		p.advance() // consume '.'
		if !p.check(TokenIdent) {
			p.addError(p.peek(), "expected edge name after '.'")
			break
		}
		tok = p.advance()
		ep.Parts = append(ep.Parts, strings.ToLower(tok.Literal))
	}

	return ep
}

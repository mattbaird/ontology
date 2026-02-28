// Package pql implements the lexer, parser, and AST for PQL
// (Propeller Query Language).
package pql

import "strings"

// TokenType identifies the kind of lexical token.
type TokenType int

const (
	// Literals and identifiers
	TokenEOF     TokenType = iota
	TokenIdent             // unquoted identifier (entity name, field name)
	TokenString            // "quoted string"
	TokenInt               // 123
	TokenFloat             // 1.23
	TokenBool              // true / false
	TokenNull              // null

	// Operators
	TokenEQ    // =
	TokenNEQ   // !=
	TokenGT    // >
	TokenLT    // <
	TokenGTE   // >=
	TokenLTE   // <=
	TokenDot   // .
	TokenComma // ,
	TokenStar  // *

	// Grouping
	TokenLParen // (
	TokenRParen // )
	TokenLBrack // [
	TokenRBrack // ]

	// Keywords — PQL verbs (Phase 1)
	TokenFind
	TokenGet
	TokenCount

	// Keywords — PQL verbs (Phase 2: mutations)
	TokenCreate
	TokenUpdate
	TokenDelete
	TokenSet

	// Keywords — PQL verbs (future phases, recognized but produce clear errors)
	TokenRun
	TokenDescribe
	TokenExplain
	TokenHistory
	TokenDiff
	TokenAggregate
	TokenWatch

	// Keywords — clauses
	TokenWhere
	TokenSelect
	TokenInclude
	TokenOrder
	TokenBy
	TokenLimit
	TokenOffset
	TokenAsc
	TokenDesc

	// Keywords — logical operators
	TokenAnd
	TokenOr
	TokenNot
	TokenIn
	TokenLike

	// Special
	TokenMetaCmd // :help, :clear, etc.
	TokenFlag    // --dry-run, --confirm
	TokenComment // -- comment text
)

// String returns a human-readable name for the token type.
func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "identifier"
	case TokenString:
		return "string"
	case TokenInt:
		return "integer"
	case TokenFloat:
		return "float"
	case TokenBool:
		return "boolean"
	case TokenNull:
		return "null"
	case TokenEQ:
		return "="
	case TokenNEQ:
		return "!="
	case TokenGT:
		return ">"
	case TokenLT:
		return "<"
	case TokenGTE:
		return ">="
	case TokenLTE:
		return "<="
	case TokenDot:
		return "."
	case TokenComma:
		return ","
	case TokenStar:
		return "*"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenLBrack:
		return "["
	case TokenRBrack:
		return "]"
	case TokenFind:
		return "find"
	case TokenGet:
		return "get"
	case TokenCount:
		return "count"
	case TokenCreate:
		return "create"
	case TokenUpdate:
		return "update"
	case TokenDelete:
		return "delete"
	case TokenSet:
		return "set"
	case TokenRun:
		return "run"
	case TokenDescribe:
		return "describe"
	case TokenExplain:
		return "explain"
	case TokenHistory:
		return "history"
	case TokenDiff:
		return "diff"
	case TokenAggregate:
		return "aggregate"
	case TokenWatch:
		return "watch"
	case TokenWhere:
		return "where"
	case TokenSelect:
		return "select"
	case TokenInclude:
		return "include"
	case TokenOrder:
		return "order"
	case TokenBy:
		return "by"
	case TokenLimit:
		return "limit"
	case TokenOffset:
		return "offset"
	case TokenAsc:
		return "asc"
	case TokenDesc:
		return "desc"
	case TokenAnd:
		return "and"
	case TokenOr:
		return "or"
	case TokenNot:
		return "not"
	case TokenIn:
		return "in"
	case TokenLike:
		return "like"
	case TokenMetaCmd:
		return "meta-command"
	case TokenFlag:
		return "flag"
	case TokenComment:
		return "comment"
	default:
		return "unknown"
	}
}

// Token represents a single lexical token in a PQL statement.
type Token struct {
	Type    TokenType
	Literal string // raw text of the token
	Pos     int    // byte offset in source
	Line    int    // 1-based line number
	Col     int    // 1-based column number
}

// keywords maps lowercase keyword strings to their token types.
var keywords = map[string]TokenType{
	"find":      TokenFind,
	"get":       TokenGet,
	"count":     TokenCount,
	"create":    TokenCreate,
	"update":    TokenUpdate,
	"delete":    TokenDelete,
	"set":       TokenSet,
	"run":       TokenRun,
	"describe":  TokenDescribe,
	"explain":   TokenExplain,
	"history":   TokenHistory,
	"diff":      TokenDiff,
	"aggregate": TokenAggregate,
	"watch":     TokenWatch,
	"where":     TokenWhere,
	"select":    TokenSelect,
	"include":   TokenInclude,
	"order":     TokenOrder,
	"by":        TokenBy,
	"limit":     TokenLimit,
	"offset":    TokenOffset,
	"asc":       TokenAsc,
	"desc":      TokenDesc,
	"and":       TokenAnd,
	"or":        TokenOr,
	"not":       TokenNot,
	"in":        TokenIn,
	"like":      TokenLike,
	"true":      TokenBool,
	"false":     TokenBool,
	"null":      TokenNull,
}

// LookupKeyword returns the keyword token type for an identifier, or
// TokenIdent if the identifier is not a keyword. Lookup is case-insensitive.
func LookupKeyword(ident string) TokenType {
	if tok, ok := keywords[strings.ToLower(ident)]; ok {
		return tok
	}
	return TokenIdent
}

// IsVerb returns true if the token type is a PQL verb keyword.
func (t TokenType) IsVerb() bool {
	switch t {
	case TokenFind, TokenGet, TokenCount,
		TokenCreate, TokenUpdate, TokenDelete,
		TokenRun, TokenDescribe, TokenExplain,
		TokenHistory, TokenDiff, TokenAggregate, TokenWatch:
		return true
	}
	return false
}

// IsClause returns true if the token type begins a clause.
func (t TokenType) IsClause() bool {
	switch t {
	case TokenWhere, TokenSelect, TokenInclude,
		TokenOrder, TokenLimit, TokenOffset:
		return true
	}
	return false
}

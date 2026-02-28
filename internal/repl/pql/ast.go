package pql

import "strings"

// Node is the interface implemented by all AST nodes.
type Node interface {
	nodeType() string
	Pos() int // byte offset in source
}

// Statement is the interface for top-level PQL statements.
type Statement interface {
	Node
	stmtNode()
}

// ── Top-level statements ────────────────────────────────────────────────────

// FindStmt represents: find <entity> [where ...] [select ...] [include ...] [order by ...] [limit N] [offset N]
type FindStmt struct {
	TokenPos int
	Entity   string
	Where    *WhereClause
	Select   *SelectClause
	Include  *IncludeClause
	OrderBy  *OrderByClause
	Limit    *LimitClause
	Offset   *OffsetClause
}

func (s *FindStmt) nodeType() string { return "FindStmt" }
func (s *FindStmt) Pos() int         { return s.TokenPos }
func (s *FindStmt) stmtNode()        {}

// GetStmt represents: get <entity> <id>
type GetStmt struct {
	TokenPos int
	Entity   string
	ID       string
	Include  *IncludeClause
}

func (s *GetStmt) nodeType() string { return "GetStmt" }
func (s *GetStmt) Pos() int         { return s.TokenPos }
func (s *GetStmt) stmtNode()        {}

// CountStmt represents: count <entity> [where ...]
type CountStmt struct {
	TokenPos int
	Entity   string
	Where    *WhereClause
}

func (s *CountStmt) nodeType() string { return "CountStmt" }
func (s *CountStmt) Pos() int         { return s.TokenPos }
func (s *CountStmt) stmtNode()        {}

// MetaCmdStmt represents: :<command> [args...]
type MetaCmdStmt struct {
	TokenPos int
	Command  string   // e.g. "help", "clear", "env", "history"
	Args     []string // remaining tokens as raw strings
}

func (s *MetaCmdStmt) nodeType() string { return "MetaCmdStmt" }
func (s *MetaCmdStmt) Pos() int         { return s.TokenPos }
func (s *MetaCmdStmt) stmtNode()        {}

// CreateStmt represents: create <entity> set field = value [, field = value ...]
type CreateStmt struct {
	TokenPos    int
	Entity      string
	Assignments []Assignment
}

func (s *CreateStmt) nodeType() string { return "CreateStmt" }
func (s *CreateStmt) Pos() int         { return s.TokenPos }
func (s *CreateStmt) stmtNode()        {}

// UpdateStmt represents: update <entity> "<id>" set field = value [, field = value ...]
type UpdateStmt struct {
	TokenPos    int
	Entity      string
	ID          string
	Assignments []Assignment
}

func (s *UpdateStmt) nodeType() string { return "UpdateStmt" }
func (s *UpdateStmt) Pos() int         { return s.TokenPos }
func (s *UpdateStmt) stmtNode()        {}

// DeleteStmt represents: delete <entity> "<id>"
type DeleteStmt struct {
	TokenPos int
	Entity   string
	ID       string
}

func (s *DeleteStmt) nodeType() string { return "DeleteStmt" }
func (s *DeleteStmt) Pos() int         { return s.TokenPos }
func (s *DeleteStmt) stmtNode()        {}

// Assignment represents a field = value pair in create/update statements.
type Assignment struct {
	Field FieldRef
	Value Literal
}

// ── Clauses ─────────────────────────────────────────────────────────────────

// WhereClause holds the predicate expression tree.
type WhereClause struct {
	Expr Expr
}

// SelectClause holds the list of fields to project.
type SelectClause struct {
	Fields []FieldRef
}

// IncludeClause holds edge paths to eager-load.
type IncludeClause struct {
	Edges []EdgePath
}

// OrderByClause holds ordering specifications.
type OrderByClause struct {
	Items []OrderItem
}

// OrderItem is a single ordering specification.
type OrderItem struct {
	Field FieldRef
	Desc  bool
}

// LimitClause holds the result limit.
type LimitClause struct {
	Value int
}

// OffsetClause holds the result offset.
type OffsetClause struct {
	Value int
}

// ── Field and Edge references ───────────────────────────────────────────────

// FieldRef is a possibly-dotted field reference (e.g. "status" or "property.address.state").
type FieldRef struct {
	Parts []string
}

// String returns the dotted field reference.
func (fr FieldRef) String() string {
	return strings.Join(fr.Parts, ".")
}

// EdgePath is a possibly-dotted edge path (e.g. "tenant_roles" or "lease_spaces.space").
type EdgePath struct {
	Parts []string
}

// String returns the dotted edge path.
func (ep EdgePath) String() string {
	return strings.Join(ep.Parts, ".")
}

// ── Expressions (predicate tree) ────────────────────────────────────────────

// Expr is implemented by all expression nodes in a WHERE clause.
type Expr interface {
	Node
	exprNode()
}

// BinaryLogicExpr represents "expr AND expr" or "expr OR expr".
type BinaryLogicExpr struct {
	TokenPos int
	Op       LogicOp
	Left     Expr
	Right    Expr
}

// LogicOp is AND or OR.
type LogicOp int

const (
	LogicAnd LogicOp = iota
	LogicOr
)

func (e *BinaryLogicExpr) nodeType() string { return "BinaryLogicExpr" }
func (e *BinaryLogicExpr) Pos() int         { return e.TokenPos }
func (e *BinaryLogicExpr) exprNode()        {}

// NotExpr represents "not expr".
type NotExpr struct {
	TokenPos int
	Expr     Expr
}

func (e *NotExpr) nodeType() string { return "NotExpr" }
func (e *NotExpr) Pos() int         { return e.TokenPos }
func (e *NotExpr) exprNode()        {}

// ComparisonExpr represents "field op value".
type ComparisonExpr struct {
	TokenPos int
	Field    FieldRef
	Op       CompOp
	Value    Literal
}

// CompOp is a comparison operator.
type CompOp int

const (
	CompEQ CompOp = iota
	CompNEQ
	CompGT
	CompLT
	CompGTE
	CompLTE
	CompLike
)

// String returns the PQL operator symbol.
func (op CompOp) String() string {
	switch op {
	case CompEQ:
		return "="
	case CompNEQ:
		return "!="
	case CompGT:
		return ">"
	case CompLT:
		return "<"
	case CompGTE:
		return ">="
	case CompLTE:
		return "<="
	case CompLike:
		return "like"
	default:
		return "?"
	}
}

func (e *ComparisonExpr) nodeType() string { return "ComparisonExpr" }
func (e *ComparisonExpr) Pos() int         { return e.TokenPos }
func (e *ComparisonExpr) exprNode()        {}

// InExpr represents "field in [val1, val2, ...]".
type InExpr struct {
	TokenPos int
	Field    FieldRef
	Values   []Literal
}

func (e *InExpr) nodeType() string { return "InExpr" }
func (e *InExpr) Pos() int         { return e.TokenPos }
func (e *InExpr) exprNode()        {}

// ── Literal values ──────────────────────────────────────────────────────────

// Literal represents a constant value in PQL.
type Literal struct {
	TokenPos int
	Type     LiteralType
	Raw      string // raw token text
}

// LiteralType classifies a literal value.
type LiteralType int

const (
	LitString LiteralType = iota
	LitInt
	LitFloat
	LitBool
	LitNull
)

func (l Literal) nodeType() string { return "Literal" }
func (l Literal) Pos() int         { return l.TokenPos }

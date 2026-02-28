package pql

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes PQL source text.
type Lexer struct {
	input   string
	pos     int // current byte position
	line    int // 1-based
	col     int // 1-based
	tokens  []Token
	errors  []error
}

// NewLexer creates a lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		pos:   0,
		line:  1,
		col:   1,
	}
}

// Tokenize scans the entire input and returns all tokens plus any errors.
func (l *Lexer) Tokenize() ([]Token, []error) {
	for {
		tok := l.next()
		if tok.Type == TokenComment {
			continue // skip comments
		}
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return l.tokens, l.errors
}

// peek returns the current rune without advancing.
func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

// peekAt returns the rune at offset from current position.
func (l *Lexer) peekAt(offset int) rune {
	p := l.pos + offset
	if p >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[p:])
	return r
}

// advance moves forward by one rune and returns it.
func (l *Lexer) advance() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

// skipWhitespace advances past spaces, tabs, and newlines.
func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		r := l.peek()
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			l.advance()
		} else {
			break
		}
	}
}

// next scans and returns the next token.
func (l *Lexer) next() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.pos, Line: l.line, Col: l.col}
	}

	startPos := l.pos
	startLine := l.line
	startCol := l.col
	r := l.peek()

	// Meta-command: colon at start of statement (after whitespace)
	if r == ':' && l.isStatementStart() {
		return l.scanMetaCmd(startPos, startLine, startCol)
	}

	// String literal
	if r == '"' || r == '\'' {
		return l.scanString(startPos, startLine, startCol)
	}

	// Number
	if r >= '0' && r <= '9' {
		return l.scanNumber(startPos, startLine, startCol)
	}

	// Identifier or keyword
	if isIdentStart(r) {
		return l.scanIdent(startPos, startLine, startCol)
	}

	// Two-character operators
	if r == '-' && l.peekAt(1) == '-' {
		// Could be comment or flag
		if isIdentStart(l.peekAt(2)) {
			return l.scanFlag(startPos, startLine, startCol)
		}
		return l.scanComment(startPos, startLine, startCol)
	}
	if r == '!' && l.peekAt(1) == '=' {
		l.advance()
		l.advance()
		return Token{Type: TokenNEQ, Literal: "!=", Pos: startPos, Line: startLine, Col: startCol}
	}
	if r == '>' && l.peekAt(1) == '=' {
		l.advance()
		l.advance()
		return Token{Type: TokenGTE, Literal: ">=", Pos: startPos, Line: startLine, Col: startCol}
	}
	if r == '<' && l.peekAt(1) == '=' {
		l.advance()
		l.advance()
		return Token{Type: TokenLTE, Literal: "<=", Pos: startPos, Line: startLine, Col: startCol}
	}

	// Single-character operators
	l.advance()
	switch r {
	case '=':
		return Token{Type: TokenEQ, Literal: "=", Pos: startPos, Line: startLine, Col: startCol}
	case '>':
		return Token{Type: TokenGT, Literal: ">", Pos: startPos, Line: startLine, Col: startCol}
	case '<':
		return Token{Type: TokenLT, Literal: "<", Pos: startPos, Line: startLine, Col: startCol}
	case '.':
		return Token{Type: TokenDot, Literal: ".", Pos: startPos, Line: startLine, Col: startCol}
	case ',':
		return Token{Type: TokenComma, Literal: ",", Pos: startPos, Line: startLine, Col: startCol}
	case '*':
		return Token{Type: TokenStar, Literal: "*", Pos: startPos, Line: startLine, Col: startCol}
	case '(':
		return Token{Type: TokenLParen, Literal: "(", Pos: startPos, Line: startLine, Col: startCol}
	case ')':
		return Token{Type: TokenRParen, Literal: ")", Pos: startPos, Line: startLine, Col: startCol}
	case '[':
		return Token{Type: TokenLBrack, Literal: "[", Pos: startPos, Line: startLine, Col: startCol}
	case ']':
		return Token{Type: TokenRBrack, Literal: "]", Pos: startPos, Line: startLine, Col: startCol}
	}

	l.errors = append(l.errors, fmt.Errorf("line %d col %d: unexpected character %q", startLine, startCol, r))
	return Token{Type: TokenIdent, Literal: string(r), Pos: startPos, Line: startLine, Col: startCol}
}

// scanString reads a quoted string literal.
func (l *Lexer) scanString(startPos, startLine, startCol int) Token {
	quote := l.advance() // consume opening quote
	var b strings.Builder
	for l.pos < len(l.input) {
		r := l.advance()
		if r == quote {
			return Token{Type: TokenString, Literal: b.String(), Pos: startPos, Line: startLine, Col: startCol}
		}
		if r == '\\' {
			next := l.advance()
			switch next {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case '\'':
				b.WriteByte('\'')
			default:
				b.WriteByte('\\')
				b.WriteRune(next)
			}
			continue
		}
		b.WriteRune(r)
	}
	l.errors = append(l.errors, fmt.Errorf("line %d col %d: unterminated string", startLine, startCol))
	return Token{Type: TokenString, Literal: b.String(), Pos: startPos, Line: startLine, Col: startCol}
}

// scanNumber reads an integer or float literal.
func (l *Lexer) scanNumber(startPos, startLine, startCol int) Token {
	start := l.pos
	isFloat := false
	for l.pos < len(l.input) {
		r := l.peek()
		if r >= '0' && r <= '9' {
			l.advance()
		} else if r == '.' && !isFloat {
			// Check that next char is also a digit (avoid consuming trailing dot)
			if l.peekAt(1) >= '0' && l.peekAt(1) <= '9' {
				isFloat = true
				l.advance()
			} else {
				break
			}
		} else {
			break
		}
	}
	lit := l.input[start:l.pos]
	if isFloat {
		return Token{Type: TokenFloat, Literal: lit, Pos: startPos, Line: startLine, Col: startCol}
	}
	return Token{Type: TokenInt, Literal: lit, Pos: startPos, Line: startLine, Col: startCol}
}

// scanIdent reads an identifier or keyword.
func (l *Lexer) scanIdent(startPos, startLine, startCol int) Token {
	start := l.pos
	for l.pos < len(l.input) {
		r := l.peek()
		if isIdentPart(r) {
			l.advance()
		} else {
			break
		}
	}
	lit := l.input[start:l.pos]
	tokType := LookupKeyword(lit)
	return Token{Type: tokType, Literal: lit, Pos: startPos, Line: startLine, Col: startCol}
}

// scanMetaCmd reads a meta-command (e.g., :help, :clear).
func (l *Lexer) scanMetaCmd(startPos, startLine, startCol int) Token {
	l.advance() // consume ':'
	start := l.pos
	for l.pos < len(l.input) {
		r := l.peek()
		if isIdentPart(r) {
			l.advance()
		} else {
			break
		}
	}
	lit := ":" + l.input[start:l.pos]
	return Token{Type: TokenMetaCmd, Literal: lit, Pos: startPos, Line: startLine, Col: startCol}
}

// scanFlag reads a --flag token.
func (l *Lexer) scanFlag(startPos, startLine, startCol int) Token {
	l.advance() // first '-'
	l.advance() // second '-'
	start := l.pos
	for l.pos < len(l.input) {
		r := l.peek()
		if isIdentPart(r) || r == '-' {
			l.advance()
		} else {
			break
		}
	}
	lit := "--" + l.input[start:l.pos]
	return Token{Type: TokenFlag, Literal: lit, Pos: startPos, Line: startLine, Col: startCol}
}

// scanComment reads a -- comment to end of line.
func (l *Lexer) scanComment(startPos, startLine, startCol int) Token {
	start := l.pos
	for l.pos < len(l.input) && l.peek() != '\n' {
		l.advance()
	}
	return Token{Type: TokenComment, Literal: l.input[start:l.pos], Pos: startPos, Line: startLine, Col: startCol}
}

// isStatementStart returns true if the colon is at the beginning of a statement
// (i.e., no prior non-whitespace tokens on this logical line, or it's the first token).
func (l *Lexer) isStatementStart() bool {
	// Simple heuristic: if no tokens have been emitted yet, or the last token
	// was on a different line, or we're at position 0 after whitespace.
	if len(l.tokens) == 0 {
		return true
	}
	// Check if previous token was a statement-ending token
	last := l.tokens[len(l.tokens)-1]
	if last.Type == TokenEOF {
		return true
	}
	// For simplicity in Phase 1, treat colon as meta-cmd whenever it appears
	// at the start of input or after the previous statement is complete.
	// Since PQL statements don't use colon, this works.
	return true
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

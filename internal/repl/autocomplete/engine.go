// Package autocomplete provides context-aware completions for PQL.
package autocomplete

import (
	"strings"

	"github.com/matthewbaird/ontology/internal/repl/pql"
	"github.com/matthewbaird/ontology/internal/repl/schema"
)

// CompletionItem is a single autocomplete suggestion.
type CompletionItem struct {
	Label      string `json:"label"`
	Kind       string `json:"kind"` // "verb", "entity", "field", "edge", "operator", "value"
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insert_text,omitempty"`
}

// Engine provides <50ms autocomplete from the in-memory schema registry.
type Engine struct {
	registry *schema.Registry
}

// New creates an autocomplete engine backed by the given registry.
func New(registry *schema.Registry) *Engine {
	return &Engine{registry: registry}
}

// verbs is the list of available PQL verbs.
var verbs = []string{"find", "get", "count", "create", "update", "delete"}

// clauses is the list of PQL clause keywords.
var clauses = []string{"where", "select", "include", "order", "limit", "offset"}

// operators is the list of comparison operators.
var operators = []string{"=", "!=", ">", "<", ">=", "<=", "like", "in"}

// metaCommands is the list of available meta-commands.
var metaCommands = []string{":help", ":clear", ":env", ":history"}

// Complete returns autocomplete suggestions for the given PQL text and cursor position.
func (e *Engine) Complete(text string, cursor int) []CompletionItem {
	// Use the text up to the cursor position
	if cursor > len(text) {
		cursor = len(text)
	}
	prefix := text[:cursor]

	// Tokenize the prefix to understand context
	lexer := pql.NewLexer(prefix)
	tokens, _ := lexer.Tokenize()

	// Remove EOF token
	if len(tokens) > 0 && tokens[len(tokens)-1].Type == pql.TokenEOF {
		tokens = tokens[:len(tokens)-1]
	}

	// Check if the cursor is right after the last token (partial typing)
	// or past it (trailing whitespace = token is complete).
	cursorAtEndOfLastToken := false
	if len(tokens) > 0 {
		last := tokens[len(tokens)-1]
		tokenEnd := last.Pos + len(last.Literal)
		cursorAtEndOfLastToken = cursor <= tokenEnd
	}

	// Detect if the user is still "inside" a string value — either the string
	// is unterminated (typing inside quotes) or the string is terminated but
	// the cursor is right at the closing quote with no trailing whitespace.
	// In both cases we suppress after-value keyword suggestions.
	suppressAfterString := false
	if len(tokens) > 0 && tokens[len(tokens)-1].Type == pql.TokenString {
		if isUnterminatedString(prefix, tokens[len(tokens)-1]) {
			suppressAfterString = true
		} else if len(prefix) > 0 && prefix[len(prefix)-1] != ' ' && prefix[len(prefix)-1] != '\t' {
			// Terminated string but cursor is right at closing quote — no space yet
			suppressAfterString = true
		}
	}

	// Detect context and provide suggestions
	return e.contextualComplete(tokens, cursorAtEndOfLastToken, suppressAfterString)
}

func (e *Engine) contextualComplete(tokens []pql.Token, cursorAtEndOfLastToken, suppressAfterString bool) []CompletionItem {
	if len(tokens) == 0 {
		return e.completeVerbs("")
	}

	last := tokens[len(tokens)-1]

	// Only treat the last identifier as a partial if the cursor is right at the
	// end of it (i.e., the user is still typing). If there's trailing whitespace
	// the token is complete and we should suggest what comes next.
	partial := ""
	if cursorAtEndOfLastToken && (last.Type == pql.TokenIdent || last.Type == pql.TokenMetaCmd) {
		partial = strings.ToLower(last.Literal)
		tokens = tokens[:len(tokens)-1]
	}

	if len(tokens) == 0 {
		// Typing the first word
		items := e.completeVerbs(partial)
		items = append(items, e.completeMetaCmds(partial)...)
		return items
	}

	first := tokens[0]

	// After a verb, expect entity name
	if len(tokens) == 1 && first.Type.IsVerb() {
		return e.completeEntities(partial)
	}

	// After verb + entity, detect what clause context we're in
	if len(tokens) >= 2 && first.Type.IsVerb() {
		entityName := strings.ToLower(tokens[1].Literal)
		es := e.registry.Entity(entityName)

		// create <entity> → suggest "set"
		if first.Type == pql.TokenCreate && len(tokens) == 2 {
			return filterItems([]string{"set"}, partial, "keyword")
		}

		// update <entity> "id" → suggest "set"
		if first.Type == pql.TokenUpdate && len(tokens) == 3 {
			return filterItems([]string{"set"}, partial, "keyword")
		}

		// After "set" in create/update, suggest field names
		if (first.Type == pql.TokenCreate || first.Type == pql.TokenUpdate) && len(tokens) >= 3 {
			lastTok := tokens[len(tokens)-1]
			if lastTok.Type == pql.TokenSet || lastTok.Type == pql.TokenComma {
				if es != nil {
					return e.completeFields(es, partial)
				}
			}
			// After field = operator, suggest enum values
			if lastTok.Type == pql.TokenEQ && es != nil {
				fieldName := findFieldBeforeOp(tokens)
				if fieldName != "" {
					fm := es.Fields[fieldName]
					if fm != nil && fm.Type == schema.FieldEnum {
						return e.completeEnumValues(fm, partial)
					}
				}
			}
		}

		return e.completeInClauseContext(tokens[2:], es, partial, suppressAfterString)
	}

	return nil
}

func (e *Engine) completeInClauseContext(tokens []pql.Token, es *schema.EntitySchema, partial string, suppressAfterString bool) []CompletionItem {
	if len(tokens) == 0 {
		// After entity name, suggest clauses
		return e.completeClauses(partial)
	}

	last := tokens[len(tokens)-1]

	switch last.Type {
	case pql.TokenWhere, pql.TokenAnd, pql.TokenOr:
		// After WHERE/AND/OR, suggest field names
		if es != nil {
			return e.completeFields(es, partial)
		}

	case pql.TokenSelect:
		if es != nil {
			return e.completeFields(es, partial)
		}

	case pql.TokenInclude:
		if es != nil {
			return e.completeEdges(es, partial)
		}

	case pql.TokenIdent:
		// Could be after a field name — suggest operators
		prev := findPrevKeyword(tokens)
		if prev == pql.TokenWhere || prev == pql.TokenAnd || prev == pql.TokenOr {
			return e.completeOperators(partial)
		}
		return e.completeClauses(partial)

	case pql.TokenEQ, pql.TokenNEQ, pql.TokenGT, pql.TokenLT, pql.TokenGTE, pql.TokenLTE:
		// After a comparison operator, suggest enum values if applicable
		if es != nil {
			fieldName := findFieldBeforeOp(tokens)
			if fieldName != "" {
				fm := es.Fields[fieldName]
				if fm != nil && fm.Type == schema.FieldEnum {
					return e.completeEnumValues(fm, partial)
				}
			}
		}

	case pql.TokenComma:
		ctx := findClauseContext(tokens)
		if es != nil {
			switch ctx {
			case pql.TokenSelect:
				return e.completeFields(es, partial)
			case pql.TokenInclude:
				return e.completeEdges(es, partial)
			case pql.TokenOrder:
				return e.completeFields(es, partial)
			}
		}

	case pql.TokenString:
		// If the string is right after a comparison operator, treat it as a
		// partial value being typed and suggest enum values when applicable.
		if len(tokens) >= 2 && isComparisonOp(tokens[len(tokens)-2].Type) && es != nil {
			fieldName := findFieldBeforeOp(tokens)
			if fieldName != "" {
				fm := es.Fields[fieldName]
				if fm != nil && fm.Type == schema.FieldEnum {
					return e.completeEnumValues(fm, strings.ToLower(last.Literal))
				}
			}
		}
		// If the string is unterminated (user is still typing inside quotes),
		// don't suggest after-value keywords.
		if suppressAfterString {
			return nil
		}
		return e.completeAfterValue(partial)

	case pql.TokenInt, pql.TokenFloat, pql.TokenBool, pql.TokenNull, pql.TokenRBrack:
		// After a literal value (or end of IN list), suggest connectors and clauses
		return e.completeAfterValue(partial)

	case pql.TokenBy:
		if es != nil {
			return e.completeFields(es, partial)
		}

	default:
		// Don't suggest anything for unrecognized contexts
		return nil
	}

	return nil
}

// ── Completion providers ────────────────────────────────────────────────────

func (e *Engine) completeVerbs(partial string) []CompletionItem {
	return filterItems(verbs, partial, "verb")
}

func (e *Engine) completeMetaCmds(partial string) []CompletionItem {
	return filterItems(metaCommands, partial, "command")
}

func (e *Engine) completeEntities(partial string) []CompletionItem {
	return filterItems(e.registry.EntityNames(), partial, "entity")
}

func (e *Engine) completeClauses(partial string) []CompletionItem {
	return filterItems(clauses, partial, "keyword")
}

// afterValueKeywords are suggested after a complete comparison value.
var afterValueKeywords = []string{"and", "or", "select", "include", "order", "limit", "offset"}

func (e *Engine) completeAfterValue(partial string) []CompletionItem {
	return filterItems(afterValueKeywords, partial, "keyword")
}

func (e *Engine) completeOperators(partial string) []CompletionItem {
	return filterItems(operators, partial, "operator")
}

func (e *Engine) completeFields(es *schema.EntitySchema, partial string) []CompletionItem {
	var items []CompletionItem
	for _, name := range es.FieldOrder {
		if partial == "" || strings.HasPrefix(strings.ToLower(name), partial) {
			fm := es.Fields[name]
			items = append(items, CompletionItem{
				Label:  name,
				Kind:   "field",
				Detail: fm.Type.String(),
			})
		}
	}
	return items
}

func (e *Engine) completeEdges(es *schema.EntitySchema, partial string) []CompletionItem {
	var items []CompletionItem
	for _, name := range es.EdgeOrder {
		if partial == "" || strings.HasPrefix(strings.ToLower(name), partial) {
			em := es.Edges[name]
			items = append(items, CompletionItem{
				Label:  name,
				Kind:   "edge",
				Detail: em.Target + " (" + em.Cardinality + ")",
			})
		}
	}
	return items
}

func (e *Engine) completeEnumValues(fm *schema.FieldMeta, partial string) []CompletionItem {
	var items []CompletionItem
	for _, v := range fm.EnumValues {
		if partial == "" || strings.HasPrefix(strings.ToLower(v), partial) {
			items = append(items, CompletionItem{
				Label:      v,
				Kind:       "value",
				InsertText: "\"" + v + "\"",
			})
		}
	}
	return items
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func filterItems(candidates []string, partial, kind string) []CompletionItem {
	var items []CompletionItem
	for _, c := range candidates {
		if partial == "" || strings.HasPrefix(strings.ToLower(c), partial) {
			items = append(items, CompletionItem{
				Label: c,
				Kind:  kind,
			})
		}
	}
	return items
}

func findPrevKeyword(tokens []pql.Token) pql.TokenType {
	for i := len(tokens) - 1; i >= 0; i-- {
		t := tokens[i].Type
		if t == pql.TokenWhere || t == pql.TokenAnd || t == pql.TokenOr ||
			t == pql.TokenSelect || t == pql.TokenInclude || t == pql.TokenOrder {
			return t
		}
	}
	return pql.TokenEOF
}

func findFieldBeforeOp(tokens []pql.Token) string {
	for i := len(tokens) - 2; i >= 0; i-- {
		if tokens[i].Type == pql.TokenIdent {
			return strings.ToLower(tokens[i].Literal)
		}
	}
	return ""
}

// isUnterminatedString checks whether a TokenString in the source text is missing
// its closing quote (i.e., the user is still typing inside the string).
func isUnterminatedString(source string, tok pql.Token) bool {
	// The token Pos points to the opening quote character.
	if tok.Pos >= len(source) {
		return true
	}
	quote := source[tok.Pos]
	// Look for the closing quote after the opening one.
	// The content between quotes is tok.Literal (without escapes re-encoded),
	// so just check if the source ends with a matching closing quote.
	rest := source[tok.Pos+1:]
	// Find last occurrence of the quote character in the rest
	lastQuote := strings.LastIndexByte(rest, quote)
	return lastQuote < 0
}

func isComparisonOp(t pql.TokenType) bool {
	return t == pql.TokenEQ || t == pql.TokenNEQ ||
		t == pql.TokenGT || t == pql.TokenLT ||
		t == pql.TokenGTE || t == pql.TokenLTE
}

func findClauseContext(tokens []pql.Token) pql.TokenType {
	for i := len(tokens) - 1; i >= 0; i-- {
		t := tokens[i].Type
		if t.IsClause() {
			return t
		}
	}
	return pql.TokenEOF
}

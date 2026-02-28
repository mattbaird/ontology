package pql

import "fmt"

// ParseError is a structured error from the PQL parser with position
// information and optional suggestions.
type ParseError struct {
	Message    string
	Line       int
	Col        int
	Pos        int
	Suggestion string // "Did you mean 'lease'?" or ""
}

func (e *ParseError) Error() string {
	msg := fmt.Sprintf("line %d col %d: %s", e.Line, e.Col, e.Message)
	if e.Suggestion != "" {
		msg += " (" + e.Suggestion + ")"
	}
	return msg
}

// newParseError creates a ParseError from a token and message.
func newParseError(tok Token, msg string) *ParseError {
	return &ParseError{
		Message: msg,
		Line:    tok.Line,
		Col:     tok.Col,
		Pos:     tok.Pos,
	}
}

// newParseErrorf creates a formatted ParseError from a token.
func newParseErrorf(tok Token, format string, args ...any) *ParseError {
	return &ParseError{
		Message: fmt.Sprintf(format, args...),
		Line:    tok.Line,
		Col:     tok.Col,
		Pos:     tok.Pos,
	}
}

// Levenshtein computes the edit distance between two strings.
func Levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use single-row DP
	prev := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			curr[j] = min(ins, min(del, sub))
		}
		prev = curr
	}
	return prev[lb]
}

// SuggestFrom finds the closest match from candidates within a maximum
// edit distance. Returns "" if no good match is found.
func SuggestFrom(input string, candidates []string, maxDist int) string {
	best := ""
	bestDist := maxDist + 1
	for _, c := range candidates {
		d := Levenshtein(input, c)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	if bestDist <= maxDist {
		return fmt.Sprintf("did you mean '%s'?", best)
	}
	return ""
}

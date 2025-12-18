package modifier

import (
	"strings"
	"unicode"
)

type clauseScanner struct {
	q  string
	up string
}

func newClauseScanner(q string) *clauseScanner {
	return &clauseScanner{q: q, up: strings.ToUpper(q)}
}

func (s *clauseScanner) findMainStart() int {
	i := 0
	i = s.skipSpaceAndComments(i)

	// If not starting with WITH, main starts at i.
	if !s.matchKeywordAt(i, "WITH") {
		return i
	}

	// Consume WITH
	i += len("WITH")
	i = s.skipSpaceAndComments(i)

	// Optional RECURSIVE
	if s.matchKeywordAt(i, "RECURSIVE") {
		i += len("RECURSIVE")
		i = s.skipSpaceAndComments(i)
	}

	// Parse one or more CTE definitions:
	// cte_name [ (col_list) ] AS ( cte_query ) [, ...]
	for i < len(s.q) {
		i = s.skipSpaceAndComments(i)
		if i >= len(s.q) {
			return i
		}

		// Read CTE name (identifier)
		name, _, next := s.readWord(i)
		if name == "" {
			// malformed: fallback to current position
			return i
		}
		i = next
		i = s.skipSpaceAndComments(i)

		// Optional column list after name: (a,b,c)
		if i < len(s.q) && s.q[i] == '(' {
			i = s.skipParenBlock(i) // skips balanced (...) with strings/comments inside
			i = s.skipSpaceAndComments(i)
		}

		// Expect AS
		if !s.matchKeywordAt(i, "AS") {
			// malformed: fallback
			return i
		}
		i += len("AS")
		i = s.skipSpaceAndComments(i)

		// Expect '(' that starts CTE subquery
		if i >= len(s.q) || s.q[i] != '(' {
			// malformed
			return i
		}

		// Skip the entire CTE query parentheses: ( ... )
		i = s.skipParenBlock(i)
		i = s.skipSpaceAndComments(i)

		// If next token is comma, more CTEs
		if i < len(s.q) && s.q[i] == ',' {
			i++
			continue
		}

		// Otherwise: main statement starts here
		return i
	}

	return i
}

// matchClauseAt tries to match multi-token clause at position i in UPPERCASE domain.
// Returns the starting position if matched, else -1.
func (s *clauseScanner) matchClauseAt(i int, tokens []string) int {
	if len(tokens) == 0 {
		return -1
	}
	orig := i

	// First token must match as a keyword with boundaries.
	w, wstart, wend := s.readWord(i)
	if w == "" || strings.ToUpper(w) != tokens[0] {
		return -1
	}
	if wstart != orig {
		return -1
	}
	i = wend

	// Remaining tokens must match in order, separated by spaces/comments (no need parentheses)
	for t := 1; t < len(tokens); t++ {
		i = s.skipSpaceAndComments(i)

		w2, _, w2end := s.readWord(i)
		if w2 == "" || strings.ToUpper(w2) != tokens[t] {
			return -1
		}
		i = w2end
	}

	return orig
}

func splitClauseTokens(clauseUpper string) []string {
	parts := strings.Fields(clauseUpper)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// skipSpaceAndComments skips whitespace and SQL comments (-- ... \n) and (/* ... */)
func (s *clauseScanner) skipSpaceAndComments(i int) int {
	for i < len(s.q) {
		// whitespace
		if isSpace(s.q[i]) {
			i++
			continue
		}
		// line comment --
		if i+1 < len(s.q) && s.q[i] == '-' && s.q[i+1] == '-' {
			i += 2
			for i < len(s.q) && s.q[i] != '\n' {
				i++
			}
			continue
		}
		// block comment /* */
		if i+1 < len(s.q) && s.q[i] == '/' && s.q[i+1] == '*' {
			i += 2
			for i+1 < len(s.q) && !(s.q[i] == '*' && s.q[i+1] == '/') {
				i++
			}
			if i+1 < len(s.q) {
				i += 2
			}
			continue
		}
		break
	}
	return i
}

func (s *clauseScanner) skipQuoted(i int, quote byte) int {
	// i points to opening quote
	i++
	for i < len(s.q) {
		ch := s.q[i]
		if ch == quote {
			// handle doubled quote for SQL strings: '' or "" (not for backticks typically)
			if quote != '`' && i+1 < len(s.q) && s.q[i+1] == quote {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return i
}

// skipParenBlock skips a balanced parenthesis block starting at '(' including nested blocks,
// while ignoring parentheses inside strings/comments.
func (s *clauseScanner) skipParenBlock(i int) int {
	if i >= len(s.q) || s.q[i] != '(' {
		return i
	}
	depth := 0
	for i < len(s.q) {
		i = s.skipSpaceAndComments(i)
		if i >= len(s.q) {
			return i
		}

		ch := s.q[i]
		if ch == '\'' || ch == '"' || ch == '`' {
			i = s.skipQuoted(i, ch)
			continue
		}
		if ch == '(' {
			depth++
			i++
			continue
		}
		if ch == ')' {
			depth--
			i++
			if depth == 0 {
				return i
			}
			continue
		}
		i++
	}
	return i
}

func (s *clauseScanner) readWord(i int) (word string, start int, end int) {
	i = s.skipSpaceAndComments(i)
	if i >= len(s.q) {
		return "", -1, -1
	}
	if !isWordStart(s.q, i) {
		return "", -1, -1
	}
	start = i
	for i < len(s.q) && isWordChar(s.q[i]) {
		i++
	}
	end = i
	return s.q[start:end], start, end
}

func (s *clauseScanner) matchKeywordAt(i int, kwUpper string) bool {
	// kwUpper must be uppercase
	i = s.skipSpaceAndComments(i)
	if i < 0 || i+len(kwUpper) > len(s.up) {
		return false
	}
	// boundary check: before and after not word char
	if i > 0 && isWordChar(s.up[i-1]) {
		return false
	}
	if s.up[i:i+len(kwUpper)] != kwUpper {
		return false
	}
	if i+len(kwUpper) < len(s.up) && isWordChar(s.up[i+len(kwUpper)]) {
		return false
	}
	return true
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}

func isWordStart(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return false
	}
	r := rune(s[i])
	return unicode.IsLetter(r) || r == '_'
}

func isWordChar(b byte) bool {
	r := rune(b)
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

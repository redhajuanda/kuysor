package modifier

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLModifier handles parsing and modifying SQL queries
type SQLModifier struct {
	query     string
	cteTarget string // when set, modifications target this named CTE's body
}

// NewSQLModifier creates a new SQLModifier instance
func NewSQLModifier(query string) *SQLModifier {
	return &SQLModifier{
		query: strings.TrimSpace(query),
	}
}

// SetCTETarget configures the modifier to apply WHERE / ORDER BY / LIMIT
// modifications inside the named CTE's body instead of the main query.
func (m *SQLModifier) SetCTETarget(name string) {
	m.cteTarget = name
}

// findCTEBodyBounds returns the start and end byte positions of the content
// inside the named CTE's outer parentheses.
// e.g. for "WITH foo AS ( SELECT id FROM t WHERE x=1 )", it returns the
// positions of " SELECT id FROM t WHERE x=1 " (exclusive of the parens).
// Returns (-1, -1, err) when the CTE is not found or parens are unmatched.
func (m *SQLModifier) findCTEBodyBounds(cteName string) (start, end int, err error) {
	queryUpper := strings.ToUpper(m.query)
	cteNameUpper := regexp.QuoteMeta(strings.ToUpper(cteName))

	// Match: <cteName> followed by optional whitespace, AS, optional whitespace, then (
	re := regexp.MustCompile(`\b` + cteNameUpper + `\s+AS\s*\(`)
	loc := re.FindStringIndex(queryUpper)
	if loc == nil {
		return -1, -1, fmt.Errorf("CTE %q not found in query", cteName)
	}

	// loc[1] points just past the '(' in "AS ("
	openParenPos := loc[1] - 1 // position of the '('
	start = openParenPos + 1   // content begins after '('

	// Walk forward tracking depth to find the matching closing ')'
	depth := 1
	i := start
	for i < len(m.query) {
		switch m.query[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
				return start, end, nil
			}
		}
		i++
	}

	return -1, -1, fmt.Errorf("unmatched parentheses for CTE %q", cteName)
}

// applyToCTEBody extracts the named CTE's body, applies fn to a sub-modifier
// of that body, then splices the modified body back into the full query.
func (m *SQLModifier) applyToCTEBody(fn func(sub *SQLModifier)) error {
	start, end, err := m.findCTEBodyBounds(m.cteTarget)
	if err != nil {
		return err
	}

	sub := &SQLModifier{query: strings.TrimSpace(m.query[start:end])}
	fn(sub)

	// splice the modified body back (preserve surrounding whitespace layout)
	m.query = m.query[:start] + sub.query + m.query[end:]
	return nil
}

// findMainClausePosition finds the position of a main clause (not in subqueries/CTEs)
// Returns the position of the clause keyword, or -1 if not found
func (m *SQLModifier) findMainClausePosition(clauseKeyword string) int {
	queryUpper := strings.ToUpper(m.query)
	clauseKeywordUpper := strings.ToUpper(clauseKeyword)

	// Create a regex pattern for the clause keyword with word boundaries
	re := regexp.MustCompile(`\b` + clauseKeywordUpper + `\b`)
	matches := re.FindAllStringIndex(queryUpper, -1)

	// For queries with WITH (CTE), find the main SELECT position first
	mainSelectPos := -1
	if strings.Contains(queryUpper, "WITH") {
		mainSelectPos = m.findMainSelectPosition()
	}

	for _, match := range matches {
		pos := match[0]

		// If we have a CTE and this clause is before the main SELECT, skip it
		if mainSelectPos != -1 && pos < mainSelectPos {
			continue
		}

		// Check if this position is inside parentheses
		// Count open and close parentheses before this position
		queryBefore := m.query[:pos]
		openCount := strings.Count(queryBefore, "(")
		closeCount := strings.Count(queryBefore, ")")

		// If open and close counts match, it's not in parentheses
		if openCount == closeCount {
			return pos
		}
	}

	return -1
}

// findMainSelectPosition finds the position of the main SELECT clause (not in subqueries/CTEs)
func (m *SQLModifier) findMainSelectPosition() int {
	queryUpper := strings.ToUpper(m.query)

	// Find all SELECT positions
	re := regexp.MustCompile(`\bSELECT\b`)
	matches := re.FindAllStringIndex(queryUpper, -1)

	for _, match := range matches {
		pos := match[0]

		// Check if this position is inside parentheses
		queryBefore := m.query[:pos]
		openCount := strings.Count(queryBefore, "(")
		closeCount := strings.Count(queryBefore, ")")

		// If open and close counts match, it's not in parentheses (subquery)
		if openCount == closeCount {
			// Check if it's after WITH clause (main query after CTE)
			withPos := strings.LastIndex(strings.ToUpper(queryBefore), "WITH")
			if withPos != -1 {
				// Check if there's a closing parenthesis after WITH but before this SELECT
				// This would indicate the end of CTE definitions
				afterWith := queryBefore[withPos:]
				if strings.Contains(afterWith, ")") {
					return pos
				}
			} else {
				// No WITH clause, this is the main SELECT
				return pos
			}
		}
	}

	return -1
}

// ConvertToCount converts the SELECT query to a COUNT query
// Replaces the SELECT columns with COUNT(*) as total_data while preserving all other clauses
func (m *SQLModifier) ConvertToCount() error {
	queryUpper := strings.ToUpper(m.query)

	// Check if it's a SELECT query
	if !strings.HasPrefix(queryUpper, "SELECT") {
		return fmt.Errorf("query must start with SELECT")
	}

	// Find the main SELECT position (not in subqueries)
	selectPos := m.findMainSelectPosition()
	if selectPos == -1 {
		return fmt.Errorf("could not find main SELECT clause")
	}

	// Find the FROM clause position
	fromPos := m.findMainClausePosition("FROM")
	if fromPos == -1 {
		return fmt.Errorf("query must contain a FROM clause")
	}

	// Handle WITH clause (CTE) if present
	withPos := m.findMainClausePosition("WITH")
	var withClause string
	if withPos != -1 && withPos < selectPos {
		// Extract the WITH clause up to the main SELECT
		withClause = strings.TrimSpace(m.query[withPos:selectPos])
		if !strings.HasSuffix(withClause, " ") {
			withClause += " "
		}
	}

	// Extract everything from FROM onwards (including JOINs, WHERE, etc.)
	fromClause := strings.TrimSpace(m.query[fromPos:])

	// Build the new query
	var newQuery string
	if withClause != "" {
		newQuery = fmt.Sprintf("%sSELECT COUNT(*) AS total_data %s", withClause, fromClause)
	} else {
		newQuery = fmt.Sprintf("SELECT COUNT(*) AS total_data %s", fromClause)
	}

	m.query = newQuery
	return nil
}

// AppendWhere appends a condition to the WHERE clause.
// When cteTarget is set it targets the CTE body; otherwise it targets the main query.
// Returns an error only when cteTarget is set and the CTE cannot be found.
func (m *SQLModifier) AppendWhere(condition string) error {
	if m.cteTarget != "" {
		return m.applyToCTEBody(func(sub *SQLModifier) {
			sub.appendWhereInternal(condition)
		})
	}
	m.appendWhereInternal(condition)
	return nil
}

// appendWhereInternal performs the WHERE append on m.query without any CTE targeting.
func (m *SQLModifier) appendWhereInternal(condition string) {
	wherePos := m.findMainClausePosition("WHERE")

	if wherePos == -1 {
		// No WHERE clause found, add one before GROUP BY, HAVING, ORDER BY, LIMIT, etc.
		clauses := []string{"GROUP BY", "HAVING", "ORDER BY", "LIMIT", "OFFSET", "FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		minPos := -1

		for _, clause := range clauses {
			clausePos := m.findMainClausePosition(clause)
			if clausePos != -1 && (minPos == -1 || clausePos < minPos) {
				minPos = clausePos
			}
		}

		if minPos != -1 {
			m.query = strings.TrimSpace(m.query[:minPos]) + fmt.Sprintf(" WHERE %s ", condition) + m.query[minPos:]
			return
		}

		// No other clauses found
		m.query = m.query + fmt.Sprintf(" WHERE %s", condition)
		return

	} else {
		// Find the end of the WHERE clause (next clause or end of query)
		clauses := []string{"GROUP BY", "HAVING", "ORDER BY", "LIMIT", "OFFSET", "FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		nextClausePos := -1

		for _, clause := range clauses {
			pos := m.findMainClausePosition(clause)
			if pos != -1 && (nextClausePos == -1 || pos < nextClausePos) {
				nextClausePos = pos
			}
		}

		// Extract the existing WHERE condition
		var existingCondition string
		if nextClausePos == -1 {
			existingCondition = strings.TrimSpace(m.query[wherePos+5:]) // +5 to skip "WHERE"
		} else {
			existingCondition = strings.TrimSpace(m.query[wherePos+5 : nextClausePos])
		}

		// Check if the existing condition has multiple conditions (contains AND or OR operators)
		reMultipleConditions := regexp.MustCompile(`\b(AND|OR)\b`)
		needsParentheses := reMultipleConditions.MatchString(existingCondition)

		var newWhere string
		if needsParentheses {
			newWhere = fmt.Sprintf("WHERE (%s) AND %s", existingCondition, condition)
		} else {
			newWhere = fmt.Sprintf("WHERE %s AND %s", existingCondition, condition)
		}

		if nextClausePos == -1 {
			m.query = m.query[:wherePos] + newWhere
			return
		} else {
			m.query = m.query[:wherePos] + newWhere + " " + m.query[nextClausePos:]
			return
		}
	}
}

// SetOrderBy sets the ORDER BY clause.
// When cteTarget is set it targets the CTE body; otherwise it targets the main query.
// Returns an error only when cteTarget is set and the CTE cannot be found.
func (m *SQLModifier) SetOrderBy(orderBy ...string) error {
	if m.cteTarget != "" {
		return m.applyToCTEBody(func(sub *SQLModifier) {
			sub.setOrderByInternal(orderBy...)
		})
	}
	m.setOrderByInternal(orderBy...)
	return nil
}

// SetMainOrderBy always sets the ORDER BY on the main query, regardless of cteTarget.
// Used when WithCTETarget is active to mirror the effective sort order on the outer
// SELECT so the joined result set is returned in the correct order.
func (m *SQLModifier) SetMainOrderBy(orderBy ...string) error {
	m.setOrderByInternal(orderBy...)
	return nil
}

// setOrderByInternal performs the ORDER BY set on m.query without any CTE targeting.
func (m *SQLModifier) setOrderByInternal(orderBy ...string) {
	newOrderBy := strings.Join(orderBy, ", ")

	orderByPos := m.findMainClausePosition("ORDER BY")

	if orderByPos == -1 {
		clauses := []string{"LIMIT", "OFFSET", "FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		minPos := -1

		for _, clause := range clauses {
			clausePos := m.findMainClausePosition(clause)
			if clausePos != -1 && (minPos == -1 || clausePos < minPos) {
				minPos = clausePos
			}
		}

		if minPos != -1 {
			m.query = strings.TrimSpace(m.query[:minPos]) + fmt.Sprintf(" ORDER BY %s ", newOrderBy) + m.query[minPos:]
			return
		}

		m.query = m.query + fmt.Sprintf(" ORDER BY %s", newOrderBy)
		return
	} else {
		clauses := []string{"LIMIT", "OFFSET", "FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		nextClausePos := -1

		for _, clause := range clauses {
			pos := m.findMainClausePosition(clause)
			if pos != -1 && (nextClausePos == -1 || pos < nextClausePos) {
				nextClausePos = pos
			}
		}

		if nextClausePos == -1 {
			m.query = m.query[:orderByPos] + fmt.Sprintf("ORDER BY %s", newOrderBy)
			return
		} else {
			m.query = m.query[:orderByPos] + fmt.Sprintf("ORDER BY %s ", newOrderBy) + m.query[nextClausePos:]
			return
		}
	}
}

// SetLimit sets the LIMIT clause.
// When cteTarget is set it targets the CTE body; otherwise it targets the main query.
// Returns an error only when cteTarget is set and the CTE cannot be found.
func (m *SQLModifier) SetLimit(newLimit string) error {
	if m.cteTarget != "" {
		return m.applyToCTEBody(func(sub *SQLModifier) {
			sub.setLimitInternal(newLimit)
		})
	}
	m.setLimitInternal(newLimit)
	return nil
}

// setLimitInternal performs the LIMIT set on m.query without any CTE targeting.
func (m *SQLModifier) setLimitInternal(newLimit string) {
	limitPos := m.findMainClausePosition("LIMIT")

	if limitPos == -1 {
		clauses := []string{"OFFSET", "FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		minPos := -1

		for _, clause := range clauses {
			clausePos := m.findMainClausePosition(clause)
			if clausePos != -1 && (minPos == -1 || clausePos < minPos) {
				minPos = clausePos
			}
		}

		if minPos != -1 {
			m.query = strings.TrimSpace(m.query[:minPos]) + fmt.Sprintf(" LIMIT %s ", newLimit) + m.query[minPos:]
			return
		}

		m.query = m.query + fmt.Sprintf(" LIMIT %s", newLimit)
		return
	} else {
		clauses := []string{"OFFSET", "FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		nextClausePos := -1

		for _, clause := range clauses {
			pos := m.findMainClausePosition(clause)
			if pos != -1 && pos > limitPos && (nextClausePos == -1 || pos < nextClausePos) {
				nextClausePos = pos
			}
		}

		if nextClausePos == -1 {
			queryAfterLimit := m.query[limitPos+5:] // +5 to skip "LIMIT"
			re := regexp.MustCompile(`^\s*\d+(?:\s*,\s*\d+)?`)
			match := re.FindStringIndex(queryAfterLimit)

			if match != nil {
				limitEndPos := limitPos + 5 + match[1]
				m.query = m.query[:limitPos] + fmt.Sprintf("LIMIT %s", newLimit) + m.query[limitEndPos:]
				return
			} else {
				m.query = m.query[:limitPos] + fmt.Sprintf("LIMIT %s", newLimit)
				return
			}
		} else {
			m.query = m.query[:limitPos] + fmt.Sprintf("LIMIT %s ", newLimit) + m.query[nextClausePos:]
			return
		}
	}
}

// SetOffset sets the OFFSET clause.
// When cteTarget is set it targets the CTE body; otherwise it targets the main query.
// Returns an error only when cteTarget is set and the CTE cannot be found.
func (m *SQLModifier) SetOffset(newOffset string) error {
	if m.cteTarget != "" {
		return m.applyToCTEBody(func(sub *SQLModifier) {
			sub.setOffsetInternal(newOffset)
		})
	}
	m.setOffsetInternal(newOffset)
	return nil
}

// setOffsetInternal performs the OFFSET set on m.query without any CTE targeting.
func (m *SQLModifier) setOffsetInternal(newOffset string) {
	offsetPos := m.findMainClausePosition("OFFSET")

	if offsetPos == -1 {
		clauses := []string{"FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		minPos := -1

		for _, clause := range clauses {
			clausePos := m.findMainClausePosition(clause)
			if clausePos != -1 && (minPos == -1 || clausePos < minPos) {
				minPos = clausePos
			}
		}

		if minPos != -1 {
			m.query = strings.TrimSpace(m.query[:minPos]) + fmt.Sprintf(" OFFSET %s ", newOffset) + m.query[minPos:]
			return
		}

		m.query = m.query + fmt.Sprintf(" OFFSET %s", newOffset)
		return
	} else {
		clauses := []string{"FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		nextClausePos := -1

		for _, clause := range clauses {
			pos := m.findMainClausePosition(clause)
			if pos != -1 && pos > offsetPos && (nextClausePos == -1 || pos < nextClausePos) {
				nextClausePos = pos
			}
		}

		if nextClausePos == -1 {
			m.query = m.query[:offsetPos] + fmt.Sprintf("OFFSET %s", newOffset)
			return
		} else {
			m.query = m.query[:offsetPos] + fmt.Sprintf("OFFSET %s ", newOffset) + m.query[nextClausePos:]
			return
		}
	}
}

// Build returns the normalized SQL query
func (m *SQLModifier) Build() (string, error) {

	q, err := normalizeSQL(m.query)
	if err != nil {
		return "", err
	}

	return q, nil
}

func normalizeSQL(query string) (string, error) {
	// Define regular expressions to match quoted strings
	quotedPatterns := []string{
		`'([^']*)'`, // single quotes
		`"([^"]*)"`, // double quotes
		"`([^`]*)`", // backticks
	}

	// Temporary map to store quoted sections and their positions
	quotedSections := make([]string, 0)

	// Replace the quoted sections with placeholders
	for _, pattern := range quotedPatterns {
		re := regexp.MustCompile(pattern)
		query = re.ReplaceAllStringFunc(query, func(matched string) string {
			quotedSections = append(quotedSections, matched)
			return fmt.Sprintf("<QUOTE%d>", len(quotedSections)-1)
		})
	}

	// Now remove extra spaces, tabs, and newlines (outside the quoted sections)
	re := regexp.MustCompile(`\s+`)
	normalizedQuery := re.ReplaceAllString(query, " ")

	// Restore the quoted sections back to their original places
	for i, quoted := range quotedSections {
		placeholder := fmt.Sprintf("<QUOTE%d>", i)
		normalizedQuery = strings.ReplaceAll(normalizedQuery, placeholder, quoted)
	}

	// Trim leading and trailing spaces
	normalizedQuery = strings.TrimSpace(normalizedQuery)

	return normalizedQuery, nil
}

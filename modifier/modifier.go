package modifier

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLModifier handles parsing and modifying SQL queries
type SQLModifier struct {
	query string
}

// NewSQLModifier creates a new SQLModifier instance
func NewSQLModifier(query string) *SQLModifier {
	return &SQLModifier{
		query: strings.TrimSpace(query),
	}
}

// findMainClausePosition finds the position of a main clause (not in subqueries/CTEs)
// Returns the position of the clause keyword, or -1 if not found
func (m *SQLModifier) findMainClausePosition(clauseKeyword string) int {
	queryUpper := strings.ToUpper(m.query)
	clauseKeywordUpper := strings.ToUpper(clauseKeyword)

	// Create a regex pattern for the clause keyword with word boundaries
	re := regexp.MustCompile(`\b` + clauseKeywordUpper + `\b`)
	matches := re.FindAllStringIndex(queryUpper, -1)

	for _, match := range matches {
		pos := match[0]

		// Check if this position is inside parentheses
		// Count open and close parentheses before this position
		queryBefore := m.query[:pos]
		openCount := strings.Count(queryBefore, "(")
		closeCount := strings.Count(queryBefore, ")")

		// If open and close counts match, it's not in parentheses
		if openCount == closeCount {
			// Additional check to avoid matching in WITH clauses or subqueries
			queryBeforeClause := queryUpper[:pos]

			// Skip if it's inside a CTE definition
			if strings.Contains(queryBeforeClause, "WITH") {
				cteAsCount := strings.Count(queryBeforeClause, " AS ")
				if cteAsCount > 0 && openCount < cteAsCount {
					continue
				}
			}

			return pos
		}
	}

	return -1
}

// AppendWhere appends a condition to the main WHERE clause
// or adds a WHERE clause if none exists
func (m *SQLModifier) AppendWhere(condition string) {
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
		// We use regular expressions to avoid matching these words inside string literals or identifiers
		reMultipleConditions := regexp.MustCompile(`\b(AND|OR)\b`)
		needsParentheses := reMultipleConditions.MatchString(existingCondition)

		var newWhere string
		if needsParentheses {
			newWhere = fmt.Sprintf("WHERE (%s) AND %s", existingCondition, condition)
		} else {
			newWhere = fmt.Sprintf("WHERE %s AND %s", existingCondition, condition)
		}

		if nextClausePos == -1 {
			// No next clause, replace to the end
			m.query = m.query[:wherePos] + newWhere
			return
		} else {
			// Replace until the next clause
			m.query = m.query[:wherePos] + newWhere + " " + m.query[nextClausePos:]
			return
		}
	}
}

// SetOrderBy sets the ORDER BY clause or adds one if none exists
func (m *SQLModifier) SetOrderBy(orderBy ...string) {
	// Join multiple order by clauses with commas
	newOrderBy := strings.Join(orderBy, ", ")

	orderByPos := m.findMainClausePosition("ORDER BY")

	if orderByPos == -1 {
		// No ORDER BY clause found, add one before LIMIT, OFFSET, FETCH, FOR UPDATE, etc.
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

		// No other clauses found
		m.query = m.query + fmt.Sprintf(" ORDER BY %s", newOrderBy)
		return
	} else {
		// Find the end of the ORDER BY clause (next clause or end of query)
		clauses := []string{"LIMIT", "OFFSET", "FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		nextClausePos := -1

		for _, clause := range clauses {
			pos := m.findMainClausePosition(clause)
			if pos != -1 && (nextClausePos == -1 || pos < nextClausePos) {
				nextClausePos = pos
			}
		}

		if nextClausePos == -1 {
			// No next clause, replace to the end
			m.query = m.query[:orderByPos] + fmt.Sprintf("ORDER BY %s", newOrderBy)
			return
		} else {
			// Replace until the next clause
			m.query = m.query[:orderByPos] + fmt.Sprintf("ORDER BY %s ", newOrderBy) + m.query[nextClausePos:]
			return
		}
	}
}

// SetLimit sets the LIMIT clause or adds one if none exists
func (m *SQLModifier) SetLimit(newLimit string) {
	limitPos := m.findMainClausePosition("LIMIT")

	if limitPos == -1 {
		// No LIMIT clause found, add before OFFSET, FETCH, FOR UPDATE, etc.
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

		// No other clauses found, add to the end
		m.query = m.query + fmt.Sprintf(" LIMIT %s", newLimit)
		return
	} else {
		// Replace the existing LIMIT clause
		// First, find where the LIMIT clause ends (typically the end of the query or another clause)
		clauses := []string{"OFFSET", "FETCH", "FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "INTO"}
		nextClausePos := -1

		for _, clause := range clauses {
			pos := m.findMainClausePosition(clause)
			if pos != -1 && pos > limitPos && (nextClausePos == -1 || pos < nextClausePos) {
				nextClausePos = pos
			}
		}

		if nextClausePos == -1 {
			// No next clause, extract the current limit value using regex
			queryAfterLimit := m.query[limitPos+5:] // +5 to skip "LIMIT"
			re := regexp.MustCompile(`^\s*\d+(?:\s*,\s*\d+)?`)
			match := re.FindStringIndex(queryAfterLimit)

			if match != nil {
				limitEndPos := limitPos + 5 + match[1]
				m.query = m.query[:limitPos] + fmt.Sprintf("LIMIT %s", newLimit) + m.query[limitEndPos:]
				return
			} else {
				// Couldn't parse the current LIMIT value, just replace everything after LIMIT
				m.query = m.query[:limitPos] + fmt.Sprintf("LIMIT %s", newLimit)
				return
			}
		} else {
			// Replace until the next clause
			m.query = m.query[:limitPos] + fmt.Sprintf("LIMIT %s ", newLimit) + m.query[nextClausePos:]
			return
		}
	}
}

// Build returns the modified SQL query
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

package modifier

import (
	"fmt"
	"regexp"
	"sort"
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

// ConvertToCount converts the main query's SELECT to COUNT(*).
// Preserves CTEs, subqueries, JOINs, WHERE, and all other clauses.
func (m *SQLModifier) ConvertToCount() error {
	return m.ConvertToCountExpr("*")
}

// ConvertToCountExpr converts the main query's SELECT to a COUNT query.
// Only replaces the main query's SELECT (not in subqueries or CTEs).
// expr: "*" for count(*), "1" for count(1), or a column name like "id" or "t.id" for count(id).
//
// Queries with GROUP BY, DISTINCT, or UNION at the main level are wrapped in a subquery
// to produce a correct scalar count. ORDER BY, LIMIT, and OFFSET are always stripped
// since they are meaningless for a count.
func (m *SQLModifier) ConvertToCountExpr(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		expr = "*"
	}

	// Must have a main SELECT (either at start or after WITH clause)
	selectPos := m.findMainSelectPosition()
	if selectPos == -1 {
		return fmt.Errorf("could not find main SELECT clause")
	}

	// Must have a main FROM clause
	fromPos := m.findMainClausePosition("FROM")
	if fromPos == -1 {
		return fmt.Errorf("query must contain a FROM clause")
	}

	// Build the count expression
	var countExpr string
	switch strings.ToUpper(expr) {
	case "*":
		countExpr = "COUNT(*)"
	case "1":
		countExpr = "COUNT(1)"
	default:
		countExpr = "COUNT(" + expr + ")"
	}

	// Queries with GROUP BY, DISTINCT, or UNION must be wrapped in a subquery;
	// otherwise COUNT would return multiple rows or lose distinctness.
	if m.hasMainGroupBy() || m.hasMainDistinct() || m.hasMainUnion() {
		// Strip ORDER BY / LIMIT / OFFSET — meaningless inside a counting subquery.
		m.stripMainOrderByAndLimit()

		// Re-find selectPos after stripping (positions before the cut are unchanged,
		// but re-finding is safer in case future stripping changes that).
		selectPos = m.findMainSelectPosition()

		// Extract WITH clause if present so it stays at the statement level
		// (CTEs must be accessible to the inner subquery).
		var withClause string
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(m.query)), "WITH") {
			withPos := strings.Index(strings.ToUpper(m.query), "WITH")
			if withPos != -1 && withPos < selectPos {
				withClause = strings.TrimSpace(m.query[withPos:selectPos])
			}
		}

		innerQuery := strings.TrimSpace(m.query[selectPos:])
		if withClause != "" {
			m.query = fmt.Sprintf("%s SELECT %s FROM (%s) kuysor_count", withClause, countExpr, innerQuery)
		} else {
			m.query = fmt.Sprintf("SELECT %s FROM (%s) kuysor_count", countExpr, innerQuery)
		}
		return nil
	}

	// Simple case: replace SELECT columns with COUNT expression.
	var withClause string
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(m.query)), "WITH") {
		withPos := strings.Index(strings.ToUpper(m.query), "WITH")
		if withPos != -1 && withPos < selectPos {
			withClause = strings.TrimSpace(m.query[withPos:selectPos])
			if !strings.HasSuffix(withClause, " ") {
				withClause += " "
			}
		}
	}

	fromClause := strings.TrimSpace(m.query[fromPos:])
	if withClause != "" {
		m.query = fmt.Sprintf("%sSELECT %s %s", withClause, countExpr, fromClause)
	} else {
		m.query = fmt.Sprintf("SELECT %s %s", countExpr, fromClause)
	}

	// Strip ORDER BY / LIMIT / OFFSET — not needed for a count query.
	m.stripMainOrderByAndLimit()

	return nil
}

// hasMainDistinct returns true if the main SELECT uses the DISTINCT keyword.
func (m *SQLModifier) hasMainDistinct() bool {
	selectPos := m.findMainSelectPosition()
	if selectPos == -1 {
		return false
	}
	afterSelect := strings.TrimSpace(m.query[selectPos+6:]) // skip "SELECT"
	return strings.HasPrefix(strings.ToUpper(afterSelect), "DISTINCT")
}

// hasMainGroupBy returns true if the main query has a GROUP BY clause at the top level.
func (m *SQLModifier) hasMainGroupBy() bool {
	return m.findMainClausePosition("GROUP BY") != -1
}

// hasMainUnion returns true if the query has a UNION or UNION ALL clause at the top
// level (not inside subqueries or CTEs).
func (m *SQLModifier) hasMainUnion() bool {
	queryUpper := strings.ToUpper(m.query)
	re := regexp.MustCompile(`\bUNION\b`)
	matches := re.FindAllStringIndex(queryUpper, -1)
	for _, match := range matches {
		pos := match[0]
		queryBefore := m.query[:pos]
		if strings.Count(queryBefore, "(") == strings.Count(queryBefore, ")") {
			return true
		}
	}
	return false
}

// stripMainOrderByAndLimit removes ORDER BY, LIMIT, and OFFSET clauses from the main
// query. These clauses are meaningless in a count query and must be stripped to avoid
// wrong results (e.g. LIMIT caps the count to 1 row, ORDER BY wastes resources).
func (m *SQLModifier) stripMainOrderByAndLimit() {
	cutPos := -1
	for _, clause := range []string{"ORDER BY", "LIMIT", "OFFSET"} {
		pos := m.findMainClausePosition(clause)
		if pos != -1 && (cutPos == -1 || pos < cutPos) {
			cutPos = pos
		}
	}
	if cutPos != -1 {
		m.query = strings.TrimSpace(m.query[:cutPos])
	}
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

// AppendWhereMain appends a WHERE condition to the main query, regardless of cteTarget.
func (m *SQLModifier) AppendWhereMain(condition string) error {
	m.appendWhereInternal(condition)
	return nil
}

// SetLimitMain sets the LIMIT clause on the main query, regardless of cteTarget.
func (m *SQLModifier) SetLimitMain(newLimit string) error {
	m.setLimitInternal(newLimit)
	return nil
}

// SetOffsetMain sets the OFFSET clause on the main query, regardless of cteTarget.
func (m *SQLModifier) SetOffsetMain(newOffset string) error {
	m.setOffsetInternal(newOffset)
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

// StripUnusedLeftJoins removes main-level LEFT JOIN clauses whose table/alias
// is not referenced in the WHERE, GROUP BY, or HAVING clauses. LEFT JOINs that
// are transitively needed (referenced in ON clauses of other needed LEFT JOINs)
// are kept. SELECT columns referencing removed aliases are also cleaned up.
func (m *SQLModifier) StripUnusedLeftJoins() {
	// Collect clause texts that determine whether a LEFT JOIN alias is "needed".
	// An alias is needed if it appears in WHERE, GROUP BY, or HAVING.
	var clauseTextsUpper []string
	for _, clause := range []struct {
		keyword    string
		terminators []string
	}{
		{"WHERE", []string{"GROUP BY", "HAVING", "ORDER BY", "LIMIT", "OFFSET"}},
		{"GROUP BY", []string{"HAVING", "ORDER BY", "LIMIT", "OFFSET"}},
		{"HAVING", []string{"ORDER BY", "LIMIT", "OFFSET"}},
	} {
		pos := m.findMainClausePosition(clause.keyword)
		if pos == -1 {
			continue
		}
		end := len(m.query)
		for _, kw := range clause.terminators {
			if p := m.findMainClausePosition(kw); p != -1 && p > pos && p < end {
				end = p
			}
		}
		clauseTextsUpper = append(clauseTextsUpper, strings.ToUpper(m.query[pos:end]))
	}

	// Find all main-level LEFT [OUTER] JOIN positions.
	ljRe := regexp.MustCompile(`(?i)\bLEFT\s+(?:OUTER\s+)?JOIN\b`)
	allMatches := ljRe.FindAllStringIndex(m.query, -1)
	mainSelectPos := m.findMainSelectPosition()

	type ljEntry struct {
		start     int      // position of "LEFT" keyword
		kwEnd     int      // end of "LEFT [OUTER] JOIN" keyword
		end       int      // end of the entire LEFT JOIN clause (ON condition inclusive)
		tableName string   // table name (e.g. "ticket" in "LEFT JOIN ticket t")
		alias     string   // alias if present, otherwise same as tableName
		idents    []string // all identifiers to check (uppercase): [alias] or [tableName, alias]
		onUpper   string   // ON clause text (uppercase) for dependency analysis
	}

	var entries []ljEntry
	for _, match := range allMatches {
		pos := match[0]
		if mainSelectPos != -1 && pos < mainSelectPos {
			continue
		}
		before := m.query[:pos]
		if strings.Count(before, "(") != strings.Count(before, ")") {
			continue
		}
		entries = append(entries, ljEntry{start: pos, kwEnd: match[1]})
	}

	if len(entries) == 0 {
		return
	}

	// Find all main-level clause boundary positions to determine ON clause extents.
	joinRe := regexp.MustCompile(`(?i)\b(?:(?:LEFT|RIGHT|FULL)\s+(?:OUTER\s+)?|INNER\s+|CROSS\s+)?JOIN\b`)
	clauseRe := regexp.MustCompile(`(?i)\b(?:WHERE|GROUP\s+BY|HAVING|ORDER\s+BY|LIMIT|OFFSET|UNION(?:\s+ALL)?)\b`)
	var boundaries []int
	for _, re := range []*regexp.Regexp{joinRe, clauseRe} {
		for _, match := range re.FindAllStringIndex(m.query, -1) {
			pos := match[0]
			if mainSelectPos != -1 && pos < mainSelectPos {
				continue
			}
			before := m.query[:pos]
			if strings.Count(before, "(") == strings.Count(before, ")") {
				boundaries = append(boundaries, pos)
			}
		}
	}
	sort.Ints(boundaries)

	// For each LEFT JOIN, find its alias and ON clause extent.
	onRe := regexp.MustCompile(`(?i)\bON\b`)
	validCount := 0
	for i := range entries {
		// Find the ON keyword after this LEFT JOIN keyword.
		onMatches := onRe.FindAllStringIndex(m.query[entries[i].kwEnd:], -1)
		onAbsPos := -1
		for _, om := range onMatches {
			candidate := entries[i].kwEnd + om[0]
			before := m.query[:candidate]
			if strings.Count(before, "(") == strings.Count(before, ")") {
				onAbsPos = candidate
				break
			}
		}
		if onAbsPos == -1 {
			continue
		}

		// Extract table name and alias between LEFT JOIN keyword end and ON.
		// Patterns: "table alias", "table AS alias", "table" (no alias),
		//           "(subquery) alias", "(subquery) AS alias"
		between := strings.TrimSpace(m.query[entries[i].kwEnd:onAbsPos])
		words := strings.Fields(between)
		if len(words) == 0 {
			continue
		}

		alias := strings.Trim(words[len(words)-1], "`\"[]")
		tableName := strings.Trim(words[0], "`\"[]")

		// For subqueries, the first word starts with "(" — skip table name tracking.
		if strings.HasPrefix(words[0], "(") {
			tableName = ""
		}

		// Build list of identifiers to match against clauses.
		var idents []string
		aliasUpper := strings.ToUpper(alias)
		tableUpper := strings.ToUpper(tableName)
		idents = append(idents, aliasUpper)
		if tableUpper != "" && tableUpper != aliasUpper {
			idents = append(idents, tableUpper)
		}

		// Find end of this LEFT JOIN clause: next boundary after onAbsPos
		// that is not this LEFT JOIN's own start position.
		clauseEnd := len(m.query)
		for _, bp := range boundaries {
			if bp > onAbsPos && bp != entries[i].start {
				clauseEnd = bp
				break
			}
		}

		entries[i].end = clauseEnd
		entries[i].tableName = tableName
		entries[i].alias = alias
		entries[i].idents = idents
		entries[i].onUpper = strings.ToUpper(m.query[onAbsPos:clauseEnd])
		validCount++
	}

	// Filter out entries that couldn't be parsed.
	valid := make([]ljEntry, 0, validCount)
	for _, e := range entries {
		if e.alias != "" {
			valid = append(valid, e)
		}
	}

	if len(valid) == 0 {
		return
	}

	// Mark entries as needed if any of their identifiers appear in WHERE, GROUP BY, or HAVING.
	neededEntry := make([]bool, len(valid))
	for i, e := range valid {
		for _, id := range e.idents {
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(id) + `\b`)
			for _, text := range clauseTextsUpper {
				if re.MatchString(text) {
					neededEntry[i] = true
					break
				}
			}
			if neededEntry[i] {
				break
			}
		}
	}

	// Propagate: if a needed LEFT JOIN's ON clause references another
	// LEFT JOIN's identifier, that entry is transitively needed.
	for changed := true; changed; {
		changed = false
		for i, isNeeded := range neededEntry {
			if !isNeeded {
				continue
			}
			for j, e := range valid {
				if neededEntry[j] {
					continue
				}
				for _, id := range e.idents {
					re := regexp.MustCompile(`\b` + regexp.QuoteMeta(id) + `\b`)
					if re.MatchString(valid[i].onUpper) {
						neededEntry[j] = true
						changed = true
						break
					}
				}
			}
		}
	}

	// Collect removed identifiers (both table name and alias).
	removedAliases := make(map[string]bool)
	for i, e := range valid {
		if !neededEntry[i] {
			for _, id := range e.idents {
				removedAliases[id] = true
			}
		}
	}

	if len(removedAliases) == 0 {
		return
	}

	// Remove unneeded LEFT JOINs from end to start to preserve positions.
	// Do not trim whitespace here — TrimRight/TrimLeft would shift character
	// positions and corrupt subsequent removals that rely on pre-computed offsets.
	// Extra whitespace is cleaned up by normalizeSQL in Build().
	for i := len(valid) - 1; i >= 0; i-- {
		if !neededEntry[i] {
			m.query = m.query[:valid[i].start] + m.query[valid[i].end:]
		}
	}
	m.query = strings.TrimSpace(m.query)

	// Clean up SELECT columns that reference removed aliases.
	m.cleanSelectForRemovedAliases(removedAliases)
}

// cleanSelectForRemovedAliases removes SELECT column expressions that reference
// any of the given aliases. If all columns are removed, replaces with "1".
func (m *SQLModifier) cleanSelectForRemovedAliases(removedAliases map[string]bool) {
	selectPos := m.findMainSelectPosition()
	fromPos := m.findMainClausePosition("FROM")
	if selectPos == -1 || fromPos == -1 {
		return
	}

	selectKeywordEnd := selectPos + 6 // len("SELECT")
	afterSelect := strings.TrimSpace(m.query[selectKeywordEnd:fromPos])

	// Preserve DISTINCT keyword if present.
	prefix := "SELECT "
	if strings.HasPrefix(strings.ToUpper(afterSelect), "DISTINCT") {
		prefix = "SELECT DISTINCT "
		afterSelect = strings.TrimSpace(afterSelect[8:])
	}

	// Split column expressions on commas, respecting parentheses.
	cols := splitOnTopLevelComma(afterSelect)

	// Filter out columns that reference removed aliases.
	var kept []string
	for _, col := range cols {
		colUpper := strings.ToUpper(col)
		referencesRemoved := false
		for alias := range removedAliases {
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(alias) + `\b`)
			if re.MatchString(colUpper) {
				referencesRemoved = true
				break
			}
		}
		if !referencesRemoved {
			kept = append(kept, col)
		}
	}

	if len(kept) == 0 {
		kept = []string{"1"}
	}

	// Reconstruct the query with cleaned SELECT.
	newSelect := prefix + strings.Join(kept, ", ") + " "
	m.query = m.query[:selectPos] + newSelect + m.query[fromPos:]
}

// splitOnTopLevelComma splits a string on commas that are not inside parentheses.
func splitOnTopLevelComma(s string) []string {
	var result []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				result = append(result, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if trimmed := strings.TrimSpace(s[start:]); trimmed != "" {
		result = append(result, trimmed)
	}
	return result
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

package kuysor

import (
	"fmt"
	"strings"

	"github.com/redhajuanda/kuysor/modifier"
)

// CountExpr constants for UseColumn.
// Use "*" for count(*), "1" for count(1), or a column name like "id" or "t.id" for count(id).
const (
	CountStar = "*"
	CountOne  = "1"
)

// Count converts a SELECT query into a COUNT query.
// It replaces only the main query's SELECT clause (not in subqueries or CTEs)
// with count(*), count(1), or count(column).
type Count struct {
	query             string
	expr              string
	removeUnusedJoins bool
}

// NewCount creates a new Count instance for converting a query to a count query.
// By default uses count(*). Use UseColumn to customize.
func NewCount(query string) *Count {
	return &Count{
		query: strings.TrimSpace(query),
		expr:  CountStar,
	}
}

// UseColumn sets the expression to use inside count().
// Use "*" for count(*), "1" for count(1), or a column name like "id" or "t.id" for count(id).
func (c *Count) UseColumn(expr string) *Count {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		expr = CountStar
	}
	c.expr = expr
	return c
}

// RemoveUnusedLeftJoins enables removal of LEFT JOIN clauses whose table/alias
// is not referenced in the WHERE clause. This optimizes count queries by
// eliminating unnecessary joins that don't affect filtering.
func (c *Count) RemoveUnusedLeftJoins() *Count {
	c.removeUnusedJoins = true
	return c
}

// Build converts the query to a count query and returns the result.
func (c *Count) Build() (string, error) {
	m := modifier.NewSQLModifier(c.query)
	if c.removeUnusedJoins {
		m.StripUnusedLeftJoins()
	}
	if err := m.ConvertToCountExpr(c.expr); err != nil {
		return "", fmt.Errorf("failed to convert to count query: %w", err)
	}
	return m.Build()
}

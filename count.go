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
// Unused LEFT JOINs (not referenced in WHERE, GROUP BY, or HAVING) are removed automatically.
type Count struct {
	query string
	expr  string
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

// Build converts the query to a count query and returns the result.
// Unused LEFT JOINs are automatically removed.
func (c *Count) Build() (string, error) {
	m := modifier.NewSQLModifier(c.query)
	m.StripUnusedLeftJoins()
	if err := m.ConvertToCountExpr(c.expr); err != nil {
		return "", fmt.Errorf("failed to convert to count query: %w", err)
	}
	return m.Build()
}

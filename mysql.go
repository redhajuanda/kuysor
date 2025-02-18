package kuysor

import (
	"fmt"
	"slices"
	"strings"

	"errors"

	"github.com/redhajuanda/sqlparser"
)

type mySQL struct {
	ks   *Kuysor
	stmt *sqlparser.Select
}

// newMySQL creates a new MySQL parser
func newMySQL(ks *Kuysor) *mySQL {
	return &mySQL{ks, nil}
}

// build builds MySQL query
func (m *mySQL) build() (string, error) {

	// create new parser
	ps, err := sqlparser.New(sqlparser.Options{})
	if err != nil {
		return "", fmt.Errorf("failed to create parser: %v", err)
	}

	// parse sql query to sqlparser statement
	stmt, err := ps.Parse(m.ks.sql)
	if err != nil {
		return "", fmt.Errorf("failed to parse sql: %v", err)
	}

	// handle the statement based on the type
	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		m.stmt = stmt
		return m.handleStmt()
	default:
		return "", errors.New("unsupported statement type")
	}
}

// handleSelectStmt handles the select statement.
func (m *mySQL) handleStmt() (string, error) {

	var (
		uPaging *uPaging = m.ks.uTabling.uPaging
		vSorts  *vSorts  = m.ks.vTabling.vSorts
	)

	if uPaging != nil {
		err := m.handlePagination()
		if err != nil {
			return "", err
		}
	} else if vSorts != nil {
		err := m.applySorts(vSorts)
		if err != nil {
			return "", err
		}
	}

	return m.sanitizeQuery(m.formatStatement()), nil

}

// formatStatement formats the statement into query string.
func (m *mySQL) formatStatement() string {

	buf := sqlparser.NewTrackedBuffer(nil)
	m.stmt.Format(buf)
	return buf.String()

}

// sanitizeQuery sanitizes the query.
func (m *mySQL) sanitizeQuery(query string) string {

	ord := findParamOrder(query, ":v0")
	for i, o := range ord {
		m.ks.uArgs = slices.Insert[[]any](m.ks.uArgs, o, m.ks.vArgs[i])
	}
	return replaceBindVariables(query)

}

// handlePagination handles the pagination.
func (m *mySQL) handlePagination() (err error) {

	// if cursor is not empty, it means it is not the first page
	// so we need to apply where clause
	if m.ks.vTabling.vCursor != nil {
		err = m.applyWhere()
		if err != nil {
			return err
		}
	}

	// apply limit and sorts
	return m.applyLimitAndSorts()

}

// getCursorValue gets the cursor value.
func (m *mySQL) getCursorValue(vSort *vSort) (col *sqlparser.Literal, err error) {

	var (
		vCursor = m.ks.vTabling.vCursor
	)

	if vCursor.Cols[vSort.column] == nil {
		col = nil
	} else {
		col = sqlparser.NewBitLiteral(":v0")
	}

	return
}

// applyWhere applies the where clause to the sql query.
func (m *mySQL) applyWhere() (err error) {

	exprs, err := m.constructExprs()
	if err != nil {
		return err
	}

	orExprs := m.createMultipleOrExpr(exprs...)
	m.stmt.AddWhere(orExprs)
	return

}

// constructExprs constructs the expressions.
func (m *mySQL) constructExprs() (expr []sqlparser.Expr, err error) {

	var (
		vSorts  = m.ks.vTabling.vSorts
		vCursor = m.ks.vTabling.vCursor
		exprs   = make([]sqlparser.Expr, 0)
	)

	for i, vSort := range *vSorts {

		var (
			expr     = make([]sqlparser.Expr, 0)
			operator = m.getOperator(vCursor.Prefix, &vSort)
		)

		// get cursor value
		col, err := m.getCursorValue(&vSort)
		if err != nil {
			return nil, err
		}

		if col != nil && vCursor.Prefix.isNext() && vSort.isNullable() && vSort.isAsc() ||
			col != nil && vCursor.Prefix.isPrev() && vSort.isNullable() && vSort.isDesc() {
			// construct IS NULL expression
			e, err := m.constructIsExpr(&vSort, sqlparser.IsNullOp)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, e)
		}

		if col == nil && vCursor.Prefix.isPrev() && vSort.nullable && vSort.isAsc() ||
			col == nil && vCursor.Prefix.isNext() && vSort.nullable && vSort.isDesc() {
			// construct IS NOT NULL expression
			e, err := m.constructIsExpr(&vSort, sqlparser.IsNotNullOp)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, e)
		}

		for j, vSort2 := range *vSorts {

			if j > i {
				continue
			}

			// skip single comparator
			if j == 0 && vSort.isNullable() && col == nil {
				continue
			}

			if j < i {

				col2, err := m.getCursorValue(&vSort2)
				if err != nil {
					return nil, err
				}

				if col2 == nil {
					e, err := m.constructIsExpr(&vSort2, sqlparser.IsNullOp)
					if err != nil {
						return nil, err
					}
					expr = append(expr, e)
				} else {
					e, err := m.constructCompExpr(&vSort2, sqlparser.EqualOp)
					if err != nil {
						return nil, err
					}
					expr = append(expr, e)
				}

			} else {
				e, err := m.constructCompExpr(&vSort2, operator)
				if err != nil {
					return nil, err
				}
				expr = append(expr, e)
			}
		}

		if len(expr) > 0 {
			andExprs := sqlparser.AndExpressions(expr...)
			exprs = append(exprs, andExprs)
		}
	}

	return exprs, nil
}

// constructExpr constructs the comparison expression.
func (m *mySQL) constructCompExpr(vSort *vSort, operator sqlparser.ComparisonExprOperator) (expr sqlparser.Expr, err error) {

	var (
		vCursor = m.ks.vTabling.vCursor
	)

	// extract column
	qualifier, column, err := vSort.extractColumn()
	if err != nil {
		return nil, err
	}

	// get cursor value
	col, err := m.getCursorValue(vSort)
	if err != nil {
		return nil, err
	}

	// create comparison expression
	expr = &sqlparser.ComparisonExpr{
		Operator: operator,
		Left:     sqlparser.NewColNameWithQualifier(column, sqlparser.NewTableName(qualifier)),
		Right:    col,
	}

	m.ks.vArgs = append(m.ks.vArgs, vCursor.Cols[vSort.column])

	return expr, nil

}

// constructIsExpr constructs the IS expression.
func (m *mySQL) constructIsExpr(vSort *vSort, op sqlparser.IsExprOperator) (expr sqlparser.Expr, err error) {

	// extract column
	qualifier, column, err := vSort.extractColumn()
	if err != nil {
		return nil, err
	}

	// create IS expression
	expr = &sqlparser.IsExpr{
		Left:  sqlparser.NewColNameWithQualifier(column, sqlparser.NewTableName(qualifier)),
		Right: op,
	}

	return
}

// createMultipleOrExpr creates multiple OR expressions.
func (m *mySQL) createMultipleOrExpr(exprs ...sqlparser.Expr) sqlparser.Expr {
	if len(exprs) == 0 {
		return nil
	}
	if len(exprs) == 1 {
		return exprs[0]
	}

	// Start with the first expression
	result := exprs[0]

	// Combine with remaining expressions
	for i := 1; i < len(exprs); i++ {
		result = &sqlparser.OrExpr{
			Left:  result,
			Right: exprs[i],
		}
	}

	return result
}

// getOperator gets the operator for the cursor pagination.
func (m *mySQL) getOperator(prefix cursorPrefix, vSort *vSort) sqlparser.ComparisonExprOperator {

	var (
		next     = prefix == cursorPrefixNext
		prev     = !next
		operator sqlparser.ComparisonExprOperator
	)

	if next && vSort.isDesc() || prev && vSort.isAsc() {
		operator = sqlparser.LessThanOp
	} else if next && vSort.isAsc() || prev && vSort.isDesc() {
		operator = sqlparser.GreaterThanOp
	}
	return operator

}

// applyLimitAndSorts applies the limit and sorts to the sql query.
func (m *mySQL) applyLimitAndSorts() error {

	var (
		limit  = m.ks.uTabling.uPaging.Limit
		vSorts = *m.ks.vTabling.vSorts
	)

	// reverse the sorting if the cursor is previous
	if m.ks.vTabling.vCursor != nil {
		if m.ks.vTabling.vCursor.Prefix.isPrev() {
			vSorts = m.ks.vTabling.vSorts.reverseDirection()
		}
	}

	// set limit to limit + 1
	m.stmt.SetLimit(&sqlparser.Limit{
		Rowcount: sqlparser.NewBitLiteral(":v0"),
	})
	m.ks.vArgs = append(m.ks.vArgs, limit+1)

	// set sorting for column
	return m.applySorts(&vSorts)

}

// applySorts applies the sorting to the sql query.
func (m *mySQL) applySorts(vSorts *vSorts) error {

	for _, vSort := range *vSorts {

		var (
			sortsBy   = strings.Split(vSort.column, ".")
			qualifier string
			column    string
		)

		if len(sortsBy) == 2 {
			qualifier = sortsBy[0]
			column = sortsBy[1]
		} else if len(sortsBy) == 1 {
			column = sortsBy[0]
		} else {
			return fmt.Errorf("invalid column name: %s", vSort.column)
		}

		if vSort.isNullable() {
			m.stmt.AddOrder(&sqlparser.Order{
				Expr: &sqlparser.IsExpr{
					Left: &sqlparser.ColName{
						Name:      sqlparser.NewIdentifierCI(column),
						Qualifier: sqlparser.NewTableName(qualifier),
					},
					Right: sqlparser.IsNullOp,
				},
				Direction: sqlparser.OrderDirection(vSort.direction),
			})
		}

		m.stmt.AddOrder(&sqlparser.Order{
			Expr: &sqlparser.ColName{
				Name:      sqlparser.NewIdentifierCI(column),
				Qualifier: sqlparser.NewTableName(qualifier),
			},
			Direction: sqlparser.OrderDirection(vSort.direction),
		})
	}

	return nil
}

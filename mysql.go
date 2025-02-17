package kuysor

import (
	"fmt"
	"reflect"
	"strings"

	"errors"

	"github.com/redhajuanda/sqlparser"
)

type mySQLParser struct {
	*Kuysor
	stmt *sqlparser.Select
}

// newMySQLParser creates a new MySQL parser
func newMySQLParser(p *Kuysor) *mySQLParser {
	return &mySQLParser{p, nil}
}

// Build builds the MySQL query
func (m *mySQLParser) Build() (string, error) {
	// create parser
	ps, err := sqlparser.New(sqlparser.Options{})
	if err != nil {
		return "", fmt.Errorf("failed to create parser: %v", err)
	}

	// parse sql query to sqlparser statement
	stmt, err := ps.Parse(m.sql)
	if err != nil {
		return "", fmt.Errorf("failed to parse sql: %v", err)
	}

	// handle the statement based on the type
	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		m.stmt = stmt
		return m.handleSelectStmt(stmt)
	default:
		return "", errors.New("unsupported statement type")
	}
}

// handleSelectStmt handles the select statement.
func (m *mySQLParser) handleSelectStmt(stmt *sqlparser.Select) (string, error) {

	var (
		result  string
		uPaging *uPaging = m.uTabling.uPaging
		vSorts  *vSorts  = m.vTabling.vSorts
	)

	if uPaging != nil {
		err := m.handlePagination()
		if err != nil {
			return result, err
		}
	} else if vSorts != nil {
		err := m.applySorts(vSorts)
		if err != nil {
			return result, err
		}
	}

	// format the statement
	buf := sqlparser.NewTrackedBuffer(nil)
	stmt.Format(buf)
	result = buf.String()

	// replace bind variables
	return replaceBindVariables(result), nil
}

// handlePagination handles the pagination.
func (m *mySQLParser) handlePagination() (err error) {

	// if cursor is not empty, it means it is not the first page
	// so we need to set where clause
	if m.vTabling.vCursor != nil {
		err = m.applyWhere()
		if err != nil {
			return err
		}
	}

	// apply limit and sorts
	return m.applyLimitAndSorts()

}

// sanitize sanitizes the column name of the order by.
func (m *mySQLParser) sanitize(vSort *vSort) (qualifier string, column string, err error) {

	columns := strings.Split(vSort.column, ".")
	if len(columns) == 2 {
		qualifier = columns[0]
		column = columns[1]
	} else if len(columns) == 1 {
		column = columns[0]
	} else {
		return "", "", fmt.Errorf("invalid column name: %s", vSort.column)
	}

	return
}

// getCursorValue gets the cursor value.
func (m *mySQLParser) getCursorValue(vSort *vSort) (col *sqlparser.Literal, err error) {

	var (
		vCursor = m.vTabling.vCursor
	)

	if vCursor.Cols[vSort.column] == nil {
		col = nil
	} else {
		to := reflect.TypeOf(vCursor.Cols[vSort.column]).Kind()
		switch to {
		case reflect.String:
			col = sqlparser.NewStrLiteral(vCursor.Cols[vSort.column].(string))
		case reflect.Int:
			col = sqlparser.NewIntLiteral(fmt.Sprintf("%v", vCursor.Cols[vSort.column]))
		case reflect.Float64:
			col = sqlparser.NewFloatLiteral(fmt.Sprintf("%v", vCursor.Cols[vSort.column]))
		case reflect.Bool:
			col = sqlparser.NewIntLiteral(fmt.Sprintf("%v", vCursor.Cols[vSort.column]))
		default:
			return nil, fmt.Errorf("invalid column type: %s", to)
		}
	}

	return
}

// applyWhere applies the where clause to the sql query.
func (m *mySQLParser) applyWhere() (err error) {

	e, err := m.constructExprs()
	if err != nil {
		return err
	}

	m.stmt.AddWhere(e)

	return
}

// constructExprs constructs the expression for the where clause.
func (m *mySQLParser) constructExprs() (expr sqlparser.Expr, err error) {

	exprs, err := m.constructExprs2()
	if err != nil {
		return nil, err
	}

	return m.createMultipleOrExpr(exprs...), nil

}

func (m *mySQLParser) constructExprs2() (expr []sqlparser.Expr, err error) {

	var (
		vSorts  = m.vTabling.vSorts
		vCursor = m.vTabling.vCursor
	)

	exprs := make([]sqlparser.Expr, 0)

	for i, vSort := range *vSorts {

		operator := m.getOperator(string(m.vTabling.vCursor.Prefix), vSort.prefix)

		col, err := m.getCursorValue(&vSort)
		if err != nil {
			return nil, err
		}

		expr := make([]sqlparser.Expr, 0)

		if col != nil && vCursor.Prefix.isNext() && vSort.nullable && vSort.prefix == "+" ||
			col != nil && vCursor.Prefix.isPrev() && vSort.nullable && vSort.prefix == "-" {
			e, err := m.constructIsExpr(&vSort, sqlparser.IsNullOp)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, e)
		}

		if col == nil && vCursor.Prefix.isPrev() && vSort.nullable && vSort.prefix == "+" ||
			col == nil && vCursor.Prefix.isNext() && vSort.nullable && vSort.prefix == "-" {
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
			if j == 0 && vSort.nullable && col == nil {
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
func (m *mySQLParser) constructCompExpr(vSort *vSort, operator sqlparser.ComparisonExprOperator) (expr sqlparser.Expr, err error) {
	qualifier, column, err := m.sanitize(vSort)
	if err != nil {
		return nil, err
	}

	col, err := m.getCursorValue(vSort)
	if err != nil {
		return nil, err
	}

	expr = &sqlparser.ComparisonExpr{
		Operator: operator,
		Left:     sqlparser.NewColNameWithQualifier(column, sqlparser.NewTableName(qualifier)),
		Right:    col,
	}
	return expr, nil

}

// constructIsExpr constructs the IS expression.
func (m *mySQLParser) constructIsExpr(vSort *vSort, op sqlparser.IsExprOperator) (expr sqlparser.Expr, err error) {
	qualifier, column, err := m.sanitize(vSort)
	if err != nil {
		return nil, err
	}

	expr = &sqlparser.IsExpr{
		Left:  sqlparser.NewColNameWithQualifier(column, sqlparser.NewTableName(qualifier)),
		Right: op,
	}

	return
}

// createMultipleOrExpr creates multiple OR expressions.
func (m *mySQLParser) createMultipleOrExpr(exprs ...sqlparser.Expr) sqlparser.Expr {
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
func (m *mySQLParser) getOperator(prefix string, orderType string) sqlparser.ComparisonExprOperator {

	var (
		next     = prefix == "next"
		prev     = !next
		operator sqlparser.ComparisonExprOperator
	)

	if next && orderType == "-" {
		operator = sqlparser.LessThanOp
	} else if next && orderType == "+" {
		operator = sqlparser.GreaterThanOp
	} else if prev && orderType == "-" {
		operator = sqlparser.GreaterThanOp
	} else if prev && orderType == "+" {
		operator = sqlparser.LessThanOp
	}
	return operator
}

// applyLimitAndSorts applies the limit and sorts to the sql query.
func (m *mySQLParser) applyLimitAndSorts() error {

	var (
		limit  = m.uTabling.uPaging.Limit
		vSorts = *m.vTabling.vSorts
	)

	// reverse the sorting if the cursor is previous
	if m.vTabling.vCursor != nil {
		if m.vTabling.vCursor.Prefix.isPrev() {
			vSorts = m.vTabling.vSorts.reverseDirection()
		}
	}

	// set limit to limit + 1
	m.stmt.SetLimit(&sqlparser.Limit{
		Rowcount: sqlparser.NewIntLiteral(fmt.Sprintf("%d", limit+1)),
	})

	// set sorting for column
	return m.applySorts(&vSorts)

}

// applySorts applies the sorting to the sql query.
func (m *mySQLParser) applySorts(vSorts *vSorts) error {

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
				Direction: vSort.direction,
			})
		}

		m.stmt.AddOrder(&sqlparser.Order{
			Expr: &sqlparser.ColName{
				Name:      sqlparser.NewIdentifierCI(column),
				Qualifier: sqlparser.NewTableName(qualifier),
			},
			Direction: vSort.direction,
		})
	}

	return nil
}

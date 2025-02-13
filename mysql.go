package kuysor

import (
	"fmt"
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
		err := m.handlePaging()
		if err != nil {
			return result, err
		}
	} else if vSorts != nil {
		err := m.setSorts()
		if err != nil {
			return result, err
		}
	}

	buf := sqlparser.NewTrackedBuffer(nil)
	stmt.Format(buf)
	result = buf.String()

	// replace bind variables
	return replaceBindVariables(result), nil
}

// handlePaging handles the pagination.
func (m *mySQLParser) handlePaging() (err error) {

	// if cursor is not empty, it means it is not the first page
	// so we need to set where clause
	if m.vTabling.vCursor != nil {
		err = m.cursorSetWhere()
		if err != nil {
			return err
		}
	}

	// set limit and sort
	return m.cursorSetLimitAndSort()

}

func (m *mySQLParser) cursorSetWhere() (err error) {
	var (
		vSorts  = m.vTabling.vSorts
		vCursor = m.vTabling.vCursor
	)

	exprs := make([]sqlparser.Expr, 0)
	gr := make([]sqlparser.Expr, 0)

	for _, vSort := range *vSorts {

		fmt.Println("==> order:", vSort.column, len(exprs))

		exs := make([]sqlparser.Expr, 0)
		cutOff := 0

		for _, cp := range exprs {
			e := cp
			if v, ok := e.(*sqlparser.ComparisonExpr); ok {
				vv := *v
				vv.Operator = sqlparser.EqualOp
				exs = append(exs, &vv)
			} else if v, ok := e.(*sqlparser.OrExpr); ok {
				lft := *(v.Left.(*sqlparser.IsExpr))
				rgt := *(v.Right.(*sqlparser.ComparisonExpr))
				rgt.Operator = sqlparser.EqualOp
				exs = append(exs, &sqlparser.OrExpr{
					Left:  &lft,
					Right: &rgt,
				})
			}
		}

		cutOff = len(exs)

		comparisonOp := getOperator(string(vCursor.Prefix), vSort.prefix)
		columns := strings.Split(vSort.column, ".")
		var columnQualifier string
		var columnName string
		if len(columns) == 2 {
			columnQualifier = columns[0]
			columnName = columns[1]
		} else if len(columns) == 1 {
			columnName = columns[0]
		} else {
			return fmt.Errorf("invalid column name: %s", vSort.column)
		}

		if vSort.nullable {

			expr := &sqlparser.OrExpr{
				Left: &sqlparser.IsExpr{
					Left:  sqlparser.NewColNameWithQualifier(columnName, sqlparser.NewTableName(columnQualifier)),
					Right: sqlparser.IsNullOp,
				},
				Right: &sqlparser.ComparisonExpr{
					Left:     sqlparser.NewColNameWithQualifier(columnName, sqlparser.NewTableName(columnQualifier)),
					Right:    sqlparser.NewStrLiteral(vCursor.Cols[vSort.column]),
					Operator: comparisonOp,
				},
			}

			exs = append(exs, expr)
			fmt.Println("append 3")

		} else {
			expr := &sqlparser.ComparisonExpr{
				Operator: comparisonOp,
			}
			expr.Left = sqlparser.NewColNameWithQualifier(columnName, sqlparser.NewTableName(columnQualifier))
			expr.Right = sqlparser.NewStrLiteral(vCursor.Cols[vSort.column])
			exs = append(exs, expr)
			fmt.Println("append 4")
		}

		exprs = append(exprs, exs[cutOff:]...)

		grExpr := sqlparser.AndExpressions(exs...)
		gr = append(gr, grExpr)

	}

	orExpr := createMultipleOrExpr(gr...)

	m.stmt.AddWhere(orExpr)

	return nil
}

func createMultipleOrExpr(exprs ...sqlparser.Expr) sqlparser.Expr {
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
func getOperator(prefix string, orderType string) sqlparser.ComparisonExprOperator {

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

// getSlectColumns returns the select columns from the statement.
// func (m *mySQLParser) getSelectColumns() (columnOriginal map[string]string, columnSanitized map[string]string, columnNullables map[string]bool, err error) {

// 	var (
// 		columns = m.stmt.SelectExprs
// 	)

// 	columnOriginal = make(map[string]string)
// 	columnSanitized = make(map[string]string)
// 	columnNullables = make(map[string]bool)

// 	for _, col := range columns {
// 		switch expr := col.(type) {
// 		case *sqlparser.AliasedExpr:

// 			var alias = expr.As.String()
// 			var colName string
// 			if alias == "" {
// 				alias = expr.Expr.(*sqlparser.ColName).Name.String()
// 			}

// 			if col, ok := expr.Expr.(*sqlparser.ColName); ok {
// 				if col.Qualifier.Name.String() != "" {
// 					colName = fmt.Sprintf("%s.%s", col.Qualifier.Name.String(), col.Name.String())
// 				} else {
// 					colName = col.Name.String()
// 				}
// 			}
// 			columnOriginal[alias] = colName

// 		case *sqlparser.StarExpr:
// 			err = fmt.Errorf("star expression is not supported")
// 			return
// 		}
// 	}

// 	for i := range columnOriginal {
// 		val, err := url.Parse(i)
// 		if err != nil {
// 			return nil, nil, nil, errors.Wrap(err, "failed to parse query")
// 		}

// 		columnSanitized[val.Path] = columnOriginal[i]
// 		columnNullables[val.Path] = val.Query().Get("nullable") == "true"
// 	}

// 	return

// }

// cursorSetLimitAndSort sets the limit and sort for the cursor pagination.
func (m *mySQLParser) cursorSetLimitAndSort() error {

	var (
		limit  = m.uTabling.uPaging.Limit
		vSorts = *m.vTabling.vSorts
	)

	if m.vTabling.vCursor != nil {
		if m.vTabling.vCursor.Prefix.isPrev() {
			vSorts = m.vTabling.vSorts.reverseDirection()
		}
	}

	// vSorts = append(vSorts, vSort{column: m.uTabling.uPaging.ColumnID, direction: sqlparser.AscOrder})

	// set limit to limit + 1
	m.stmt.SetLimit(&sqlparser.Limit{
		Rowcount: sqlparser.NewIntLiteral(fmt.Sprintf("%d", limit+1)),
	})

	// set sorting for column
	return m.applySorts(&vSorts)

}

// addSorting adds the sorting for the sql query.
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

// setSorting sets the sorting for the sql query.
func (m *mySQLParser) setSorts() error {

	var (
		vSorts = m.vTabling.vSorts
	)

	// return if sorting is nil
	if vSorts == nil {
		return nil
	}

	return m.applySorts(vSorts)

}

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
		// tableName string   = strings.ReplaceAll(strings.Split(sqlparser.String(stmt.GetFrom()[0]), " ")[0], "`", "") // get main table name
	)

	if uPaging != nil {
		err := m.handlePaging()
		if err != nil {
			return result, err
		}
	} else if vSorts != nil {
		m.setSorts()

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
	m.cursorSetLimitAndSort()
	return

}

func (m *mySQLParser) cursorSetWhere() (err error) {
	var (
		vSorts  = m.vTabling.vSorts
		vCursor = m.vTabling.vCursor
	)

	exprs := make([]*sqlparser.ComparisonExpr, 0)
	gr := make([]sqlparser.Expr, 0)

	// var conditions []sqlparser.OrExpr

	for i, vSort := range *vSorts {

		comparisonOp, _ := getOperator(string(vCursor.Prefix), vSort.prefix)

		expr := &sqlparser.ComparisonExpr{
			Operator: comparisonOp,
		}
		// sqlparser.ParenTableExpr

		if vSort.nullable {
			expr.Left = &sqlparser.FuncExpr{
				Name: sqlparser.NewIdentifierCI("COALESCE"),
				Exprs: sqlparser.Exprs{
					&sqlparser.ColName{
						Name: sqlparser.NewIdentifierCI(vSort.column),
					},
					sqlparser.NewStrLiteral(""),
				},
			}

			expr.Right = &sqlparser.FuncExpr{
				Name: sqlparser.NewIdentifierCI("COALESCE"),
				Exprs: sqlparser.Exprs{
					sqlparser.NewStrLiteral(vCursor.Cols[vSort.column]),
					sqlparser.NewStrLiteral(""),
				},
			}
		} else {
			expr.Left = sqlparser.NewColName(vSort.column)
			expr.Right = sqlparser.NewStrLiteral(vCursor.Cols[vSort.column])
		}

		exprs = append(exprs, expr)

		exprSlice := make([]sqlparser.Expr, len(exprs))
		for j, expr := range exprs {
			cpExpr := *expr
			if i != 0 && j != len(exprs)-1 {
				cpExpr.Operator = sqlparser.EqualOp
			}
			exprSlice[j] = &cpExpr // This works because ComparisonExpr implements Expr
		}

		grExpr := sqlparser.AndExpressions(exprSlice...)
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
func getOperator(prefix string, orderType string) (sqlparser.ComparisonExprOperator, sqlparser.OrderDirection) {

	var (
		next           = prefix == "next"
		prev           = !next
		operator       sqlparser.ComparisonExprOperator
		orderDirection sqlparser.OrderDirection
		// changeDirection bool
	)

	if next && orderType == "-" {
		operator = sqlparser.LessThanOp
	} else if next && orderType == "+" {
		operator = sqlparser.GreaterThanOp
	} else if prev && orderType == "-" {
		operator = sqlparser.GreaterThanOp
		orderDirection = sqlparser.AscOrder
		// changeDirection = true
	} else if prev && orderType == "+" {
		operator = sqlparser.LessThanOp
		orderDirection = sqlparser.DescOrder
		// changeDirection = true
	}
	return operator, orderDirection
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
func (m *mySQLParser) cursorSetLimitAndSort() {

	var (
		limit  = m.uTabling.uPaging.Limit
		vSorts = *m.vTabling.vSorts
	)

	if m.vTabling.vCursor != nil {
		if m.vTabling.vCursor.isPrev() {
			// changeOrderDirection = true
			vSorts = m.vTabling.vSorts.reverseDirection()
		}
	}

	vSorts = append(vSorts, vSort{column: m.uTabling.uPaging.ColumnID, direction: sqlparser.AscOrder})

	// vSorts := parseSort(sorts)
	// if changeOrderDirection {
	// 	vSorts = vSorts.reverseDirection()
	// }

	// set limit to limit + 1
	m.stmt.SetLimit(&sqlparser.Limit{
		Rowcount: sqlparser.NewIntLiteral(fmt.Sprintf("%d", limit+1)),
	})

	// set sorting for column
	m.applySorts(&vSorts)

}

// addSorting adds the sorting for the sql query.
func (m *mySQLParser) applySorts(vSorts *vSorts) {

	// // parse sort string to get the sort by and sort type
	// vSorts := parseSort(sorting.Sort)

	for _, vSort := range *vSorts {
		// sort by column contains table name
		sortsBy := strings.Split(vSort.column, ".")
		if len(sortsBy) == 2 {
			// add order to the statement
			m.stmt.AddOrder(&sqlparser.Order{
				Expr: &sqlparser.ColName{
					Name:      sqlparser.NewIdentifierCI(sortsBy[1]),
					Qualifier: sqlparser.NewTableName(sortsBy[0]),
				},
				Direction: vSort.direction,
			})

		} else {
			// add order to the statement
			m.stmt.AddOrder(&sqlparser.Order{
				Expr: &sqlparser.ColName{
					Name: sqlparser.NewIdentifierCI(vSort.column),
				},
				Direction: vSort.direction,
			})
		}
	}
}

// setSorting sets the sorting for the sql query.
func (m *mySQLParser) setSorts() {

	var (
		vSorts = m.vTabling.vSorts
	)

	// return if sorting is nil
	if vSorts == nil {
		return
	}

	m.applySorts(vSorts)

	// parse sort string to get the sort by and sort type
	// vSorts := parseSort(m.tabling.Sorting.Sort)

	// for _, vSort := range *vSorts {
	// 	// sort by column contains table name
	// 	sortBySplit := strings.Split(vSort.column, ".")
	// 	if len(sortBySplit) == 2 {
	// 		// add order to the statement
	// 		m.stmt.AddOrder(&sqlparser.Order{
	// 			Expr: &sqlparser.ColName{
	// 				Name:      sqlparser.NewIdentifierCI(sortBySplit[1]),
	// 				Qualifier: sqlparser.NewTableName(sortBySplit[0]),
	// 			},
	// 			Direction: vSort.direction,
	// 		})

	// 	} else {
	// 		// add order to the statement
	// 		m.stmt.AddOrder(&sqlparser.Order{
	// 			Expr:      &sqlparser.ColName{Name: sqlparser.NewIdentifierCI(vSort.column)},
	// 			Direction: vSort.direction,
	// 		})
	// 	}
	// }
}

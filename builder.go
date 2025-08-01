package kuysor

import (
	"fmt"
	"slices"

	"github.com/redhajuanda/kuysor/modifier"
)

type builder struct {
	ks     *Kuysor
	sqlMod *modifier.SQLModifier
}

func newBuilder(ks *Kuysor) *builder {
	return &builder{ks, &modifier.SQLModifier{}}
}

func (b *builder) build() (string, error) {

	var (
		// uPaging *uPaging = b.ks.uTabling.uPaging
		vCursor *vCursor = b.ks.vTabling.vCursor
		vOffset *vOffset = b.ks.vTabling.vOffset
		vSorts  *vSorts  = b.ks.vTabling.vSorts
	)

	b.sqlMod = modifier.NewSQLModifier(b.ks.sql)

	// if uPaging != nil {
	// 	err := b.handlePagination()
	// 	if err != nil {
	// 		return "", err
	// 	}
	// } else if vSorts != nil {
	// 	err := b.applySorts(vSorts)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }

	if vCursor != nil {
		err := b.handlePaginationCursor()
		if err != nil {
			return "", err
		}
	} else if vOffset != nil {
		err := b.handlePaginationOffset()
		if err != nil {
			return "", err
		}
	} else if vSorts != nil {
		err := b.applySorts(vSorts)
		if err != nil {
			return "", err
		}
	}

	res, err := b.sqlMod.Build()
	if err != nil {
		return "", err
	}

	return b.sanitizeQuery(res), nil
}

// sanitizeQuery sanitizes the query.
func (b *builder) sanitizeQuery(query string) string {

	ord := findOrderOfInternalPlaceholders(query)
	for i, o := range ord {
		b.ks.uArgs = slices.Insert[[]any](b.ks.uArgs, o, b.ks.vArgs[i])
	}
	return replacePlaceholders(query, b.ks.options.PlaceHolderType)

}

// handlePagination handles the pagination.
func (b *builder) handlePagination() (err error) {

	if b.ks.uTabling.uPaging.PaginationType == Cursor {
		return b.handlePaginationCursor()
	} else if b.ks.uTabling.uPaging.PaginationType == Offset {
		return b.handlePaginationOffset()
	}

	return fmt.Errorf("unsupported pagination type: %s", b.ks.uTabling.uPaging.PaginationType)

}

func (b *builder) handlePaginationCursor() (err error) {

	var (
		vCursor = b.ks.vTabling.vCursor
	)

	// if cursor is not empty, it means it is not the first page
	// so we need to apply where clause
	if vCursor != nil && vCursor.cursor != "" {
		err = b.applyWhere()
		if err != nil {
			return err
		}
	}

	// apply limit and sorts
	return b.applyLimitAndSorts()

}

func (b *builder) handlePaginationOffset() (err error) {

	var (
		vOffset = b.ks.vTabling.vOffset
		vSorts  = b.ks.vTabling.vSorts
	)

	err = b.applyLimit()
	if err != nil {
		return err
	}

	// if offset is not set, it means it is not the first page
	// so we need to apply where clause
	if vOffset != nil {

		err = b.applyOffset()
		if err != nil {
			return err
		}
	}

	if vSorts != nil {
		err = b.applySorts(vSorts)
		if err != nil {
			return err
		}
	}

	return nil

}

// applyWhere applies the where clause to the sql query.
func (b *builder) applyWhere() (err error) {

	exprs, err := b.constructExprs()
	if err != nil {
		return err
	}

	if len(exprs) == 1 {
		b.sqlMod.AppendWhere(exprs[0].Expression)
	} else {
		orExprs := modifier.NewNestedCondition("OR", exprs...)
		b.sqlMod.AppendWhere(orExprs.Expression)
	}

	return

}

// constructExprs constructs the expressions.
func (b *builder) constructExprs() (expr []modifier.SQLCondition, err error) {

	var (
		vSorts  = b.ks.vTabling.vSorts
		vCursor = b.ks.vTabling.vCursor
		exprs   = make([]modifier.SQLCondition, 0)
	)

	for i, vSort := range *vSorts {

		var (
			expr     = make([]modifier.SQLCondition, 0)
			operator = b.getOperator(vCursor.Prefix, &vSort)
		)

		// get cursor value
		col, err := b.getCursorValue(&vSort)
		if err != nil {
			return nil, err
		}

		if col != nil && vCursor.Prefix.isNext() && vSort.isNullable() && vSort.isAsc() ||
			col != nil && vCursor.Prefix.isPrev() && vSort.isNullable() && vSort.isDesc() {
			// construct IS NULL expression
			e, err := b.constructIsExpr(&vSort, "NULL")
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, e)
		}

		if col == nil && vCursor.Prefix.isPrev() && vSort.nullable && vSort.isAsc() ||
			col == nil && vCursor.Prefix.isNext() && vSort.nullable && vSort.isDesc() {
			// construct IS NOT NULL expression
			e, err := b.constructIsExpr(&vSort, "NOT NULL")
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

				col2, err := b.getCursorValue(&vSort2)
				if err != nil {
					return nil, err
				}

				if col2 == nil {
					e, err := b.constructIsExpr(&vSort2, "NULL")
					if err != nil {
						return nil, err
					}
					expr = append(expr, e)
				} else {
					e, err := b.constructCompExpr(&vSort2, "=")
					if err != nil {
						return nil, err
					}
					expr = append(expr, e)
				}

			} else {
				e, err := b.constructCompExpr(&vSort2, operator)
				if err != nil {
					return nil, err
				}
				expr = append(expr, e)
			}
		}

		if len(expr) > 0 {
			andExprs := modifier.NewNestedCondition("AND", expr...)
			exprs = append(exprs, andExprs)
		}
	}

	return exprs, nil
}

func (b *builder) getOperator(prefix cursorPrefix, vSort *vSort) string {

	var (
		next     = prefix == cursorPrefixNext
		prev     = !next
		operator string
	)

	if next && vSort.isDesc() || prev && vSort.isAsc() {
		operator = "<"
	} else if next && vSort.isAsc() || prev && vSort.isDesc() {
		operator = ">"
	}
	return operator

}

// getCursorValue gets the cursor value.
func (b *builder) getCursorValue(vSort *vSort) (col *string, err error) {

	var (
		vCursor        = b.ks.vTabling.vCursor
		t       string = defaultInternalPlaceHolder
	)

	_, column, err := vSort.extractColumn()
	if err != nil {
		return nil, err
	}

	if vCursor.Cols[column] == nil {
		col = nil
	} else {
		col = &t
	}

	return
}

// constructExpr constructs the comparison expression.
func (b *builder) constructCompExpr(vSort *vSort, operator string) (cnd modifier.SQLCondition, err error) {

	var (
		vCursor = b.ks.vTabling.vCursor
	)

	// get cursor value
	col, err := b.getCursorValue(vSort)
	if err != nil {
		return modifier.SQLCondition{}, err
	}

	cnd = modifier.NewCondition(fmt.Sprintf("%s %s %s", vSort.column, operator, *col))

	_, column, err := vSort.extractColumn()
	if err != nil {
		return modifier.SQLCondition{}, err
	}

	b.ks.vArgs = append(b.ks.vArgs, vCursor.Cols[column])

	return cnd, nil

}

// constructIsExpr constructs the IS expression.
func (b *builder) constructIsExpr(vSort *vSort, condition string) (expr modifier.SQLCondition, err error) {

	expr = modifier.NewCondition(fmt.Sprintf("%s IS %s", vSort.column, condition))
	return expr, nil

}

func (b *builder) applyOffset() error {

	var (
		offset = b.ks.uTabling.uPaging.Offset
	)

	b.sqlMod.SetOffset(defaultInternalPlaceHolder)

	b.ks.vArgs = append(b.ks.vArgs, offset)

	return nil

}

func (b *builder) applyLimit() error {

	var (
		limit = b.ks.uTabling.uPaging.Limit
	)

	b.sqlMod.SetLimit(defaultInternalPlaceHolder)

	b.ks.vArgs = append(b.ks.vArgs, limit)

	return nil

}

// applyLimitAndSorts applies the limit and sorts to the sql query.
func (b *builder) applyLimitAndSorts() error {

	var (
		limit  = b.ks.uTabling.uPaging.Limit
		vSorts = *b.ks.vTabling.vSorts
	)

	// reverse the sorting if the cursor is previous
	if b.ks.vTabling.vCursor != nil {
		if b.ks.vTabling.vCursor.Prefix.isPrev() {
			vSorts = b.ks.vTabling.vSorts.reverseDirection()
		}
	}

	b.sqlMod.SetLimit(defaultInternalPlaceHolder)

	b.ks.vArgs = append(b.ks.vArgs, limit+1)

	// set sorting for column
	return b.applySorts(&vSorts)

}

// applySorts applies the sorting to the sql query.
func (b *builder) applySorts(vSorts *vSorts) error {

	var clauses []string

	for _, vSort := range *vSorts {

		var direction string
		if vSort.isAsc() {
			direction = "ASC"
		} else {
			direction = "DESC"
		}

		if vSort.isNullable() && vSort.nullSortMethod == CaseWhen {
			clauses = append(clauses, fmt.Sprintf("CASE WHEN %s IS NULL THEN 1 ELSE 0 END %s", vSort.column, direction))

		}

		if vSort.isNullable() && vSort.nullSortMethod == FirstLast {
			var lf string
			if vSort.isAsc() {
				lf = "LAST"
			} else {
				lf = "FIRST"
			}
			clauses = append(clauses, fmt.Sprintf("%s %s NULLS %s", vSort.column, direction, lf))

			continue
		}

		if vSort.isNullable() && vSort.nullSortMethod == BoolSort {
			clauses = append(clauses, fmt.Sprintf("%s IS NULL %s", vSort.column, direction))
		}

		clauses = append(clauses, fmt.Sprintf("%s %s", vSort.column, direction))
	}

	b.sqlMod.SetOrderBy(clauses...)
	return nil

}

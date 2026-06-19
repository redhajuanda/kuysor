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
		vCursor *vCursor = b.ks.vTabling.vCursor
		vOffset *vOffset = b.ks.vTabling.vOffset
		vSorts  *vSorts  = b.ks.vTabling.vSorts
	)

	b.sqlMod = modifier.NewSQLModifier(b.ks.sql)

	// when the user has specified a CTE to target, tell the modifier so that
	// all subsequent WHERE / ORDER BY / LIMIT calls operate on that CTE body
	if b.ks.uTabling != nil && b.ks.uTabling.uPaging != nil && b.ks.uTabling.uPaging.CTETarget != "" {
		b.sqlMod.SetCTETarget(b.ks.uTabling.uPaging.CTETarget)
	}

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
		vCursor    = b.ks.vTabling.vCursor
		cursorArgs []any
	)

	// Inject secondary CTE bodies first. They are defined before the primary CTE
	// in the WITH clause, so their cursor WHERE / ORDER BY / LIMIT placeholders
	// appear earliest in the SQL string and their args must be appended first.
	// ORDER BY uses the same (possibly reversed) sort direction and limit+1 as the
	// primary; the cursor WHERE is only applied beyond the first page.
	if b.hasSecondaryCTEs() && b.ks.vTabling.vSorts != nil {
		secSorts := *b.ks.vTabling.vSorts
		if vCursor != nil && vCursor.Prefix.isPrev() {
			secSorts = b.ks.vTabling.vSorts.reverseDirection()
		}
		if err = b.applySecondaryCTEs(secSorts, b.ks.uTabling.uPaging.Limit+1, vCursor != nil && vCursor.cursor != ""); err != nil {
			return err
		}
	}

	// if cursor is not empty, it means it is not the first page
	// so we need to apply where clause
	if vCursor != nil && vCursor.cursor != "" {
		// snapshot vArgs length before WHERE so we can identify the cursor args
		vArgsBefore := len(b.ks.vArgs)
		err = b.applyWhere()
		if err != nil {
			return err
		}
		// capture the args added by applyWhere (cursor values)
		if len(b.ks.vArgs) > vArgsBefore {
			cursorArgs = make([]any, len(b.ks.vArgs)-vArgsBefore)
			copy(cursorArgs, b.ks.vArgs[vArgsBefore:])
		}
	}

	// apply limit and sorts
	if err = b.applyLimitAndSorts(); err != nil {
		return err
	}

	// When WHERE mode is CTETargetModeBoth, the cursor WHERE condition is placed
	// in both the CTE body (early in the string) and the main query (late in the
	// string, after LIMIT). The internal placeholder for the main WHERE therefore
	// appears AFTER the LIMIT placeholder in the final SQL string, so we append
	// the cursor arg(s) now — after limit+1 was added — to keep vArgs aligned
	// with placeholder string order: [CTE WHERE, CTE LIMIT, main WHERE].
	if len(cursorArgs) > 0 && b.ks.uTabling != nil && b.ks.uTabling.uPaging != nil && b.ks.uTabling.uPaging.CTETarget != "" {
		var opts *CTEOptions
		if b.ks.uTabling.uPaging.CTEOptions != nil {
			opts = b.ks.uTabling.uPaging.CTEOptions
		}
		if effectiveWhereMode(opts) == CTETargetModeBoth {
			b.ks.vArgs = append(b.ks.vArgs, cursorArgs...)
		}
	}

	return nil

}

func (b *builder) handlePaginationOffset() (err error) {

	var (
		vOffset = b.ks.vTabling.vOffset
		vSorts  = b.ks.vTabling.vSorts
	)

	// Inject secondary CTE bodies first (defined before the primary CTE). For
	// offset pagination secondaries receive ORDER BY + LIMIT only (no OFFSET, no
	// WHERE) as a coarse early cap; the primary CTE/main query still applies the
	// exact offset window. Requires an ORDER BY (the column the LIMIT is meaningful on).
	if b.hasSecondaryCTEs() && vSorts != nil {
		if err = b.applySecondaryCTEs(*vSorts, b.ks.uTabling.uPaging.Limit, false); err != nil {
			return err
		}
	}

	// When LIMIT/OFFSET mode is CTETargetModeBoth we must interleave args in
	// string-position order: CTE_LIMIT, CTE_OFFSET, main_LIMIT, main_OFFSET.
	// Delegating to applyLimit/applyOffset would produce the wrong order
	// ([limit,limit,offset,offset] instead of [limit,offset,limit,offset]),
	// so handle this case explicitly at this level.
	if b.ks.uTabling.uPaging.CTETarget != "" {
		var opts *CTEOptions
		if b.ks.uTabling.uPaging.CTEOptions != nil {
			opts = b.ks.uTabling.uPaging.CTEOptions
		}
		if effectiveLimitOffsetMode(opts) == CTETargetModeBoth {
			return b.handlePaginationOffsetBoth(vOffset, vSorts)
		}
	}

	err = b.applyLimit()
	if err != nil {
		return err
	}

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

// handlePaginationOffsetBoth handles the CTETargetModeBoth case for offset pagination.
// It writes CTE modifications first, then main-query modifications, so that vArgs
// stays aligned with the internal-placeholder string-position order:
// [CTE LIMIT, CTE OFFSET, main LIMIT, main OFFSET].
func (b *builder) handlePaginationOffsetBoth(vOffset *vOffset, vSorts *vSorts) error {

	limit := b.ks.uTabling.uPaging.Limit

	// ── CTE phase ──────────────────────────────────────────────────────────────
	if err := b.sqlMod.SetLimit(defaultInternalPlaceHolder); err != nil {
		return err
	}
	b.ks.vArgs = append(b.ks.vArgs, limit)

	if vOffset != nil {
		if err := b.sqlMod.SetOffset(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, vOffset.Offset)
	}

	if vSorts != nil {
		if err := b.applySorts(vSorts); err != nil {
			return err
		}
	}

	// ── Main phase ─────────────────────────────────────────────────────────────
	if err := b.sqlMod.SetLimitMain(defaultInternalPlaceHolder); err != nil {
		return err
	}
	b.ks.vArgs = append(b.ks.vArgs, limit)

	if vOffset != nil {
		if err := b.sqlMod.SetOffsetMain(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, vOffset.Offset)
	}

	return nil

}

// applyWhere applies the where clause to the sql query.
func (b *builder) applyWhere() (err error) {

	// When no CTE target is set, always route to main query.
	if b.ks.uTabling.uPaging.CTETarget == "" {
		condition, err := b.buildCondition(nil, true)
		if err != nil {
			return err
		}
		return b.sqlMod.AppendWhere(condition)
	}

	var opts *CTEOptions
	if b.ks.uTabling.uPaging.CTEOptions != nil {
		opts = b.ks.uTabling.uPaging.CTEOptions
	}
	colMap := cteColumnMap(opts)

	switch effectiveWhereMode(opts) {
	case CTETargetModeCTE:
		// CTE body uses the remapped column; this is the single arg-producing build.
		condition, err := b.buildCondition(colMap, true)
		if err != nil {
			return err
		}
		return b.sqlMod.AppendWhere(condition)
	case CTETargetModeMain:
		condition, err := b.buildCondition(nil, true)
		if err != nil {
			return err
		}
		return b.sqlMod.AppendWhereMain(condition)
	case CTETargetModeBoth:
		// CTE placement uses the remapped column and produces the cursor args.
		// Main placement keeps the original column and must NOT append args again —
		// its args are re-appended by handlePaginationCursor AFTER applyLimitAndSorts
		// runs, to keep vArgs aligned with placeholder string order. With a nil map
		// both conditions are identical, preserving previous behavior exactly.
		cteCond, err := b.buildCondition(colMap, true)
		if err != nil {
			return err
		}
		mainCond, err := b.buildCondition(nil, false)
		if err != nil {
			return err
		}
		if err := b.sqlMod.AppendWhere(cteCond); err != nil {
			return err
		}
		return b.sqlMod.AppendWhereMain(mainCond)
	}
	return nil

}

// buildCondition constructs the cursor WHERE condition string. colMap (when
// non-nil) remaps columns to their CTE-body equivalents; appendArgs controls
// whether the cursor values are appended to vArgs (set false for a secondary
// rendering of the same condition to avoid duplicating args).
func (b *builder) buildCondition(colMap map[string]string, appendArgs bool) (string, error) {

	exprs, err := b.constructExprs(colMap, appendArgs)
	if err != nil {
		return "", err
	}

	if len(exprs) == 1 {
		return exprs[0].Expression, nil
	}
	return modifier.NewNestedCondition("OR", exprs...).Expression, nil
}

// constructExprs constructs the expressions.
func (b *builder) constructExprs(colMap map[string]string, appendArgs bool) (expr []modifier.SQLCondition, err error) {

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
			e, err := b.constructIsExpr(&vSort, "NULL", colMap)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, e)
		}

		if col == nil && vCursor.Prefix.isPrev() && vSort.nullable && vSort.isAsc() ||
			col == nil && vCursor.Prefix.isNext() && vSort.nullable && vSort.isDesc() {
			// construct IS NOT NULL expression
			e, err := b.constructIsExpr(&vSort, "NOT NULL", colMap)
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
					e, err := b.constructIsExpr(&vSort2, "NULL", colMap)
					if err != nil {
						return nil, err
					}
					expr = append(expr, e)
				} else {
					e, err := b.constructCompExpr(&vSort2, "=", colMap, appendArgs)
					if err != nil {
						return nil, err
					}
					expr = append(expr, e)
				}

			} else {
				e, err := b.constructCompExpr(&vSort2, operator, colMap, appendArgs)
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

// constructCompExpr constructs the comparison expression.
// colMap (when non-nil) remaps the rendered column for the CTE body; the cursor
// value lookup still uses the original column. appendArgs controls whether the
// cursor value is appended to vArgs (false when re-rendering the same condition).
func (b *builder) constructCompExpr(vSort *vSort, operator string, colMap map[string]string, appendArgs bool) (cnd modifier.SQLCondition, err error) {

	var (
		vCursor = b.ks.vTabling.vCursor
	)

	// get cursor value
	col, err := b.getCursorValue(vSort)
	if err != nil {
		return modifier.SQLCondition{}, err
	}

	cnd = modifier.NewCondition(fmt.Sprintf("%s %s %s", renderColumn(vSort.column, colMap), operator, *col))

	_, column, err := vSort.extractColumn()
	if err != nil {
		return modifier.SQLCondition{}, err
	}

	if appendArgs {
		b.ks.vArgs = append(b.ks.vArgs, vCursor.Cols[column])
	}

	return cnd, nil

}

// constructIsExpr constructs the IS expression.
// colMap (when non-nil) remaps the rendered column for the CTE body.
func (b *builder) constructIsExpr(vSort *vSort, condition string, colMap map[string]string) (expr modifier.SQLCondition, err error) {

	expr = modifier.NewCondition(fmt.Sprintf("%s IS %s", renderColumn(vSort.column, colMap), condition))
	return expr, nil

}

func (b *builder) applyOffset() error {

	var (
		offset = b.ks.uTabling.uPaging.Offset
	)

	// When no CTE target is set, always route to main query.
	if b.ks.uTabling.uPaging.CTETarget == "" {
		if err := b.sqlMod.SetOffset(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, offset)
		return nil
	}

	var opts *CTEOptions
	if b.ks.uTabling.uPaging.CTEOptions != nil {
		opts = b.ks.uTabling.uPaging.CTEOptions
	}
	// CTETargetModeBoth is handled at the handlePaginationOffset level.
	switch effectiveLimitOffsetMode(opts) {
	case CTETargetModeCTE, CTETargetModeBoth:
		if err := b.sqlMod.SetOffset(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, offset)
	case CTETargetModeMain:
		if err := b.sqlMod.SetOffsetMain(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, offset)
	}
	return nil

}

func (b *builder) applyLimit() error {

	var (
		limit = b.ks.uTabling.uPaging.Limit
	)

	// When no CTE target is set, always route to main query.
	if b.ks.uTabling.uPaging.CTETarget == "" {
		if err := b.sqlMod.SetLimit(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit)
		return nil
	}

	var opts *CTEOptions
	if b.ks.uTabling.uPaging.CTEOptions != nil {
		opts = b.ks.uTabling.uPaging.CTEOptions
	}
	// CTETargetModeBoth is handled at the handlePaginationOffset level.
	switch effectiveLimitOffsetMode(opts) {
	case CTETargetModeCTE, CTETargetModeBoth:
		if err := b.sqlMod.SetLimit(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit)
	case CTETargetModeMain:
		if err := b.sqlMod.SetLimitMain(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit)
	}
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

	// When no CTE target is set, always route to main query.
	if b.ks.uTabling.uPaging.CTETarget == "" {
		if err := b.sqlMod.SetLimit(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit+1)
		return b.applySorts(&vSorts)
	}

	var opts *CTEOptions
	if b.ks.uTabling.uPaging.CTEOptions != nil {
		opts = b.ks.uTabling.uPaging.CTEOptions
	}
	switch effectiveLimitOffsetMode(opts) {
	case CTETargetModeCTE:
		if err := b.sqlMod.SetLimit(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit+1)
		return b.applySorts(&vSorts)
	case CTETargetModeMain:
		if err := b.sqlMod.SetLimitMain(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit+1)
		return b.applySorts(&vSorts)
	case CTETargetModeBoth:
		// String-position order for cursor+both:
		// CTE LIMIT → CTE ORDER BY → main LIMIT → main ORDER BY
		// So vArgs must be [limit+1, limit+1] with ORDER BY between the two LIMITs in SQL.
		// CTE phase:
		if err := b.sqlMod.SetLimit(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit+1)
		if err := b.applySorts(&vSorts); err != nil {
			return err
		}
		// Main phase (appended after CTE ORDER BY is already in the string):
		if err := b.sqlMod.SetLimitMain(defaultInternalPlaceHolder); err != nil {
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit+1)
		return nil
	}

	return nil

}

// hasSecondaryCTEs reports whether any secondary CTE targets are registered.
func (b *builder) hasSecondaryCTEs() bool {
	return b.ks.uTabling != nil && b.ks.uTabling.uPaging != nil && len(b.ks.uTabling.uPaging.SecondaryCTEs) > 0
}

// applySecondaryCTEs injects the cursor WHERE (when withCursor), LIMIT, and
// ORDER BY into each registered secondary CTE body, in registration order. The
// secondaries must be defined before the primary CTE in the WITH clause so that
// these placeholders — appended to vArgs here, before the primary is processed —
// stay aligned with their query-string positions. Each secondary's clauses are
// confined to its CTE body (never mirrored on the main query); a UNION body is
// auto-wrapped by the modifier, and ColumnMap remaps the injected column.
func (b *builder) applySecondaryCTEs(vSorts vSorts, limit int, withCursor bool) error {

	up := b.ks.uTabling.uPaging
	if up == nil || len(up.SecondaryCTEs) == 0 {
		return nil
	}

	// Secondaries are layered on top of a primary CTE target; without one there is
	// no defined query-string ordering for the injected placeholders.
	if up.CTETarget == "" {
		return fmt.Errorf("WithCTESecondaryTarget requires a primary WithCTETarget")
	}

	// Restore the primary CTE as the modifier target on the way out so the primary
	// pagination flow operates on the correct CTE.
	restore := func() { b.sqlMod.SetCTETarget(up.CTETarget) }

	for _, sec := range up.SecondaryCTEs {
		colMap := cteColumnMap(sec.options)
		b.sqlMod.SetCTETarget(sec.name)

		// cursor WHERE (uses the original sort directions, like the primary)
		if withCursor {
			cond, err := b.buildCondition(colMap, true)
			if err != nil {
				restore()
				return err
			}
			if err := b.sqlMod.AppendWhere(cond); err != nil {
				restore()
				return err
			}
		}

		// LIMIT
		if err := b.sqlMod.SetLimit(defaultInternalPlaceHolder); err != nil {
			restore()
			return err
		}
		b.ks.vArgs = append(b.ks.vArgs, limit)

		// ORDER BY (CTE body only — not mirrored on main)
		if err := b.sqlMod.SetOrderBy(orderClauses(&vSorts, colMap)...); err != nil {
			restore()
			return err
		}
	}

	restore()
	return nil
}

// applySorts applies the sorting to the sql query.
// Routing is controlled by CTEOptions.OrderBy when a CTE target is active.
// Default (no options): ORDER BY goes into CTE body AND is mirrored on main query.
func (b *builder) applySorts(vSorts *vSorts) error {

	// When no CTE target is set, always route ORDER BY to the main query.
	if b.ks.uTabling == nil || b.ks.uTabling.uPaging == nil || b.ks.uTabling.uPaging.CTETarget == "" {
		return b.sqlMod.SetOrderBy(orderClauses(vSorts, nil)...)
	}

	var opts *CTEOptions
	if b.ks.uTabling.uPaging.CTEOptions != nil {
		opts = b.ks.uTabling.uPaging.CTEOptions
	}

	// The CTE body uses the remapped column (when ColumnMap is set); the main
	// query always keeps the original column. With a nil map both are identical,
	// preserving previous behavior exactly.
	cteClauses := orderClauses(vSorts, cteColumnMap(opts))
	mainClauses := orderClauses(vSorts, nil)

	switch effectiveOrderByMode(opts) {
	case CTETargetModeCTE:
		// ORDER BY goes into CTE body only — do NOT mirror on main query.
		return b.sqlMod.SetOrderBy(cteClauses...)
	case CTETargetModeMain:
		// ORDER BY goes onto the main query only — skip the CTE body.
		return b.sqlMod.SetMainOrderBy(mainClauses...)
	case CTETargetModeBoth:
		// ORDER BY goes into CTE body AND is mirrored on the main query (default).
		if err := b.sqlMod.SetOrderBy(cteClauses...); err != nil {
			return err
		}
		return b.sqlMod.SetMainOrderBy(mainClauses...)
	}
	return nil

}

// orderClauses renders the ORDER BY clause fragments for the given sorts.
// colMap (when non-nil) remaps each column to its CTE-body equivalent; the
// sort direction and null handling are unchanged.
func orderClauses(vSorts *vSorts, colMap map[string]string) []string {

	var clauses []string

	for _, vSort := range *vSorts {

		column := renderColumn(vSort.column, colMap)

		var direction string
		if vSort.isAsc() {
			direction = "ASC"
		} else {
			direction = "DESC"
		}

		if vSort.isNullable() && vSort.nullSortMethod == CaseWhen {
			clauses = append(clauses, fmt.Sprintf("CASE WHEN %s IS NULL THEN 1 ELSE 0 END %s", column, direction))
		}

		if vSort.isNullable() && vSort.nullSortMethod == FirstLast {
			var lf string
			if vSort.isAsc() {
				lf = "LAST"
			} else {
				lf = "FIRST"
			}
			clauses = append(clauses, fmt.Sprintf("%s %s NULLS %s", column, direction, lf))
			continue
		}

		if vSort.isNullable() && vSort.nullSortMethod == BoolSort {
			clauses = append(clauses, fmt.Sprintf("%s IS NULL %s", column, direction))
		}

		clauses = append(clauses, fmt.Sprintf("%s %s", column, direction))
	}

	return clauses
}

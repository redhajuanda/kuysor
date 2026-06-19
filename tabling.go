package kuysor

type uTabling struct {
	uPaging *uPaging
	uSort   *uSort
}

type vTabling struct {
	vCursor *vCursor
	vOffset *vOffset
	vSorts  *vSorts
}

// CTETargetMode controls where a specific SQL clause is injected when
// WithCTETarget is active.
type CTETargetMode int

const (
	// CTETargetModeDefault uses the natural default for each clause:
	// ORDER BY defaults to Both, LIMIT/OFFSET and WHERE default to CTE.
	// This is the zero value so an empty CTEOptions uses all defaults.
	CTETargetModeDefault CTETargetMode = iota
	// CTETargetModeCTE injects the clause inside the named CTE body only.
	CTETargetModeCTE
	// CTETargetModeMain injects the clause on the outer/main query only.
	CTETargetModeMain
	// CTETargetModeBoth injects the clause in both the CTE body and the main query.
	CTETargetModeBoth
)

// CTEOptions provides per-clause control over where each SQL modification is
// applied when WithCTETarget is active.
// Zero value (CTETargetModeDefault) uses natural defaults:
//   - OrderBy     → CTETargetModeBoth  (CTE body + mirrored on main)
//   - LimitOffset → CTETargetModeCTE   (CTE body only)
//   - Where       → CTETargetModeCTE   (CTE body only)
type CTEOptions struct {
	// OrderBy controls where ORDER BY is injected.
	OrderBy CTETargetMode
	// LimitOffset controls where LIMIT and OFFSET are injected.
	LimitOffset CTETargetMode
	// Where controls where the cursor WHERE clause is injected.
	Where CTETargetMode
	// ColumnMap remaps the order-by column used inside the CTE body only, for
	// both the ORDER BY and the cursor WHERE clause. It is keyed by the original
	// order-by column (as passed to WithOrderBy, without the +/- prefix) and maps
	// to the column/expression to emit inside the CTE. The main query always keeps
	// the original column. Sort direction, LIMIT/OFFSET, and cursor values are
	// unaffected — only the rendered column text changes inside the CTE.
	//
	// Example: WithOrderBy("-t.id") + ColumnMap{"t.id": "id"} renders
	// "ORDER BY id DESC" / "id < ?" inside the CTE while the main query keeps
	// "t.id". A nil map leaves all behavior unchanged.
	ColumnMap map[string]string
}

// cteColumnMap returns the per-CTE column remap, or nil when unset.
func cteColumnMap(opts *CTEOptions) map[string]string {
	if opts == nil {
		return nil
	}
	return opts.ColumnMap
}

// renderColumn returns the column to emit, applying the CTE column remap when
// present. An unmapped column (or nil map) is returned unchanged.
func renderColumn(column string, m map[string]string) string {
	if m != nil {
		if mapped, ok := m[column]; ok {
			return mapped
		}
	}
	return column
}

// effectiveOrderByMode returns the resolved ORDER BY routing mode.
// Default (when nil or Default): CTETargetModeBoth.
func effectiveOrderByMode(opts *CTEOptions) CTETargetMode {
	if opts == nil || opts.OrderBy == CTETargetModeDefault {
		return CTETargetModeBoth
	}
	return opts.OrderBy
}

// effectiveLimitOffsetMode returns the resolved LIMIT/OFFSET routing mode.
// Default (when nil or Default): CTETargetModeCTE.
func effectiveLimitOffsetMode(opts *CTEOptions) CTETargetMode {
	if opts == nil || opts.LimitOffset == CTETargetModeDefault {
		return CTETargetModeCTE
	}
	return opts.LimitOffset
}

// effectiveWhereMode returns the resolved WHERE routing mode.
// Default (when nil or Default): CTETargetModeCTE.
func effectiveWhereMode(opts *CTEOptions) CTETargetMode {
	if opts == nil || opts.Where == CTETargetModeDefault {
		return CTETargetModeCTE
	}
	return opts.Where
}

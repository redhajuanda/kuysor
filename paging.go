package kuysor

type uPaging struct {
	PaginationType PaginationType
	Limit          int
	Offset         int         // only used for offset pagination
	Cursor         string      // only used for cursor pagination
	ColumnID       string      // only used for cursor pagination
	CTETarget      string      // optional: name of the primary CTE whose body should be paginated
	CTEOptions     *CTEOptions // optional: per-clause routing when CTETarget is set
	// SecondaryCTEs are ADDITIONAL CTE bodies that also receive the cursor WHERE,
	// ORDER BY, and LIMIT (in addition to the primary CTETarget). They are injected
	// before the primary so placeholder args stay in query-string order; each
	// secondary CTE must therefore be defined BEFORE the primary CTE in the WITH
	// clause. Only the ColumnMap of their options is honored — the clauses are
	// always injected into the CTE body, never mirrored on the main query.
	SecondaryCTEs []secondaryCTE
}

// secondaryCTE names an additional CTE body to receive the pagination clauses.
type secondaryCTE struct {
	name    string
	options *CTEOptions
}

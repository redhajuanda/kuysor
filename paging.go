package kuysor

type uPaging struct {
	PaginationType PaginationType
	Limit          int
	Offset         int    // only used for offset pagination
	Cursor         string // only used for cursor pagination
	ColumnID       string // only used for cursor pagination
}

package kuysor

import (
	"fmt"

	"errors"
)

type Kuysor struct {
	sql      string
	uTabling *uTabling
	vTabling *vTabling
	options  *Options
	uArgs    []any
	vArgs    []any
}

// NewQuery creates a new Kuysor instance.
// It accepts the SQL query.
func NewQuery(query string) *Kuysor {

	p := &Kuysor{
		sql: query,
	}

	p.options = getGlobalOptions()

	return p

}

// WithOrderBy sets the sorting / order for the query.
// Prefix the column name with "-" to sort in descending order, and "+" to sort in ascending order.
// If no prefix is provided, it will default to ascending order.
// The order of the strings in the slice determines the order of sorting.
// For example, if you want to sort by "name" in ascending order and "age" in descending order,
// you can call WithOrderBy("name", "-age").
func (p *Kuysor) WithOrderBy(orderBy ...string) *Kuysor {

	if p.uTabling == nil {
		p.uTabling = &uTabling{}
	}

	p.uTabling.uSort = &uSort{
		Sorts: orderBy,
	}

	return p

}

// WithLimit sets the limit for the query.
func (p *Kuysor) WithLimit(limit int) *Kuysor {

	if p.uTabling == nil {
		p.uTabling = &uTabling{}
	}

	if p.uTabling.uPaging == nil {
		p.uTabling.uPaging = &uPaging{}
	}

	p.uTabling.uPaging.Limit = limit

	return p

}

// WithArgs sets the arguments for the query.
func (p *Kuysor) WithArgs(args ...any) *Kuysor {

	p.uArgs = args
	return p

}

// WithCursor sets the cursor for the query.
func (p *Kuysor) WithCursor(cursor string) *Kuysor {

	if p.uTabling == nil {
		p.uTabling = &uTabling{}
	}

	if p.uTabling.uPaging == nil {
		p.uTabling.uPaging = &uPaging{}
	}

	p.uTabling.uPaging.Cursor = cursor

	return p

}

// WithPlaceHolderType sets the placeholder type for the query.
// It is useful when you want to override the instance options or the global options.
func (p *Kuysor) WithPlaceHolderType(placeHolderType PlaceHolderType) *Kuysor {

	p.options.PlaceHolderType = placeHolderType
	return p

}

// WithNullSortMethod sets the null sort method for the query.
// It is useful when you want to override the instance options or the global options.
func (p *Kuysor) WithNullSortMethod(method NullSortMethod) *Kuysor {

	p.options.NullSortMethod = method
	return p

}

// Build builds the paginated / sorted query.
func (p *Kuysor) Build() (*Result, error) {

	return p.build()

}

// build builds the paginated / sorted query.
func (p *Kuysor) build() (*Result, error) {

	var (
		sql    string
		result *Result
	)

	// validate user input
	if p.uTabling == nil {
		return result, errors.New("nothing to build")
	}
	if p.uTabling.uPaging != nil && p.uTabling.uSort == nil {
		return result, errors.New("sort is required for cursor pagination")
	}
	if p.uArgs == nil {
		p.uArgs = make([]any, 0)
	}

	// prepare vTabling
	err := p.prepareVTabling()
	if err != nil {
		return result, fmt.Errorf("failed to prepare vTabling: %v", err)
	}

	// build the query
	sql, err = newBuilder(p).build()
	if err != nil {
		return result, fmt.Errorf("failed to build query: %v", err)
	}

	return &Result{
		Query: sql,
		Args:  p.uArgs,
		ks:    p,
	}, nil
}

// prepareVTabling prepares the vTabling data.
// vTabling is the parsed version of uTabling, it is used internally to build the query.
func (p *Kuysor) prepareVTabling() (err error) {

	var (
		cursorBase64    = cursorBase64(p.uTabling.uPaging.Cursor)
		counterNullable = 0
	)

	p.vTabling = &vTabling{}

	// parse cursor
	if cursorBase64 != "" {
		p.vTabling.vCursor, err = cursorBase64.parse()
		if err != nil {
			return
		}
	}

	// parse sort
	p.vTabling.vSorts = parseSort(p.uTabling.uSort.Sorts, p.options.NullSortMethod)

	for _, vSort := range *p.vTabling.vSorts {
		if vSort.isNullable() {
			counterNullable++
		}
	}
	if counterNullable > 1 {
		return errors.New("only one nullable sort is allowed")
	}
	return

}

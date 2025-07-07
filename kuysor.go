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

type PaginationType string

const (
	Cursor PaginationType = "cursor"
	Offset PaginationType = "offset"
)

// NewQuery creates a new Kuysor instance.
// It accepts the SQL query.
func NewQuery(query string, paginationType PaginationType) *Kuysor {

	fmt.Println("==> creating new query...")

	p := &Kuysor{
		sql: query,
	}

	p.options = getGlobalOptions()

	if paginationType != "" {
		p.uTabling = &uTabling{
			uPaging: &uPaging{
				PaginationType: paginationType,
			},
		}
	}

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

// WithOffset sets the offset for the query.
func (p *Kuysor) WithOffset(offset int) *Kuysor {

	if p.uTabling == nil {
		p.uTabling = &uTabling{}
	}

	if p.uTabling.uPaging == nil {
		p.uTabling.uPaging = &uPaging{}
	}

	p.uTabling.uPaging.Offset = offset

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
		sql      string
		result   *Result
		uTabling = p.uTabling
	)

	// validate user input
	if uTabling == nil {
		return result, errors.New("nothing to build")
	}
	if uTabling.uPaging != nil && uTabling.uPaging.PaginationType == Cursor && uTabling.uSort == nil {
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

	p.vTabling = &vTabling{}

	if p.uTabling.uPaging != nil {
		if p.uTabling.uPaging.PaginationType == Cursor {
			err = p.prepareVTablingCursor()
			if err != nil {
				return fmt.Errorf("failed to prepare vTabling cursor: %v", err)
			}
		} else if p.uTabling.uPaging.PaginationType == Offset {
			err = p.prepareVTablingOffset()
			if err != nil {
				return fmt.Errorf("failed to prepare vTabling offset: %v", err)
			}
		}
	}
	if p.uTabling.uSort != nil {
		err = p.prepareVTablingSort()
		if err != nil {
			return fmt.Errorf("failed to prepare vTabling sort: %v", err)
		}
	}
	return nil

}

func (p *Kuysor) prepareVTablingOffset() (err error) {

	if p.uTabling.uPaging.Offset < 0 {
		return errors.New("offset cannot be negative")
	}

	p.vTabling.vOffset = &vOffset{
		Offset: p.uTabling.uPaging.Offset,
	}

	return nil

}

func (p *Kuysor) prepareVTablingCursor() (err error) {

	var (
		cursorBase64 = cursorBase64(p.uTabling.uPaging.Cursor)
	)

	// parse cursor
	if cursorBase64 != "" {
		p.vTabling.vCursor, err = cursorBase64.parse()
		if err != nil {
			return fmt.Errorf("failed to parse cursor: %v", err)
		}
	} else {
		p.vTabling.vCursor = &vCursor{}
	}

	return nil
}

func (p *Kuysor) prepareVTablingSort() (err error) {

	var (
		counterNullable = 0
	)

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
	return nil
}

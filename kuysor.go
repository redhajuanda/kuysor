package kuysor

import (
	"fmt"
	"reflect"

	"errors"
)

type Kuysor struct {
	sql      string
	uTabling *uTabling
	vTabling *vTabling
	options  Options
	uArgs    []any
	vArgs    []any
}

// New creates a new Kuysor instance.
// It accepts the sql query and options.
// Options is optional, if not provided, it will use the default options.
func New(sql string, opt ...Options) *Kuysor {

	p := &Kuysor{
		sql: sql,
	}

	if len(opt) > 0 { // override the options
		p.options = opt[0]
	} else if options != nil { // set global options
		p.options = *options
	} else { // set default options
		p.options = Options{
			Dialect:      MySQL,
			DefaultLimit: defaulLimit,
			StructTag:    defaultStructTag,
		}
	}

	return p

}

// WithArgs sets the arguments for the query.
func (p *Kuysor) WithArgs(args ...any) *Kuysor {

	p.uArgs = args
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

// WithSort sets the sorting / order for the query.
func (p *Kuysor) WithSort(sorts ...string) *Kuysor {

	if p.uTabling == nil {
		p.uTabling = &uTabling{}
	}

	p.uTabling.uSort = &uSort{
		Sorts: sorts,
	}

	return p

}

// Build builds the paginated / sorted query.
func (p *Kuysor) Build() (string, []any, error) {

	return p.build()

}

// build builds the paginated / sorted query.
func (p *Kuysor) build() (string, []any, error) {

	var (
		result string
	)

	// validate user input
	if p.uTabling == nil {
		return result, p.uArgs, errors.New("nothing to build")
	}
	if p.uTabling.uPaging != nil && p.uTabling.uSort == nil {
		return result, nil, errors.New("sort is required for cursor pagination")
	}
	if p.uArgs == nil {
		p.uArgs = make([]any, 0)
	}

	// prepare vTabling
	err := p.prepareVTabling()
	if err != nil {
		return result, nil, fmt.Errorf("failed to prepare vTabling: %v", err)
	}

	// build query based on the dialect
	switch p.options.Dialect {
	case MySQL:
		result, err = newMySQL(p).build()
	}

	return result, p.uArgs, err
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
	p.vTabling.vSorts = parseSort(p.uTabling.uSort.Sorts)

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

// SanitizeMap handles the map data for the cursor pagination.
// It returns the next and previous cursor.
func (p *Kuysor) SanitizeMap(data *[]map[string]interface{}) (next string, prev string, err error) {

	if p.uTabling == nil {
		return next, prev, errors.New("uTabling is nil")
	}

	if p.uTabling.uPaging == nil {
		return next, prev, errors.New("uPaging is nil")
	}

	var (
		totalData        = len(*data)
		totalDataUpdated = totalData
		vSorts           = p.vTabling.vSorts
		limit            = p.uTabling.uPaging.Limit
		vcursor          = p.vTabling.vCursor
		isFirstPage      = vcursor == nil
		cursorPrev       = vCursor{
			Prefix: cursorPrefixPrev,
			Cols:   make(map[string]interface{}),
		}
		cursorNext = vCursor{
			Prefix: cursorPrefixNext,
			Cols:   make(map[string]interface{}),
		}
	)

	// return if there is no data
	if totalData == 0 {
		return next, prev, nil
	}

	// set cursor to next if it is the first page
	if isFirstPage {
		vcursor = &vCursor{
			Prefix: cursorPrefixNext,
		}
	}

	// reverse the data if it is previous page
	if vcursor.Prefix.isPrev() {
		reverse(data)
	}

	if totalData > limit {
		// remove extra element
		if vcursor.Prefix.isNext() {
			deleteElement(data, totalData-1)
		} else {
			deleteElement(data, 0)
		}

		// update total data after delete
		totalDataUpdated = len(*data)
	}

	for _, vSort := range *vSorts {

		_, column, err := vSort.extractColumn()
		if err != nil {
			return next, prev, err
		}

		cursorPrev.Cols[column] = (*data)[0][column]
		cursorNext.Cols[column] = (*data)[totalDataUpdated-1][column]

	}

	// generate cursor
	return p.generateCursor(totalData, limit, isFirstPage, vcursor, &cursorPrev, &cursorNext)

}

// SanitizeStruct handles struct data for the cursor pagination.
// It returns the next and previous cursor.
func (p *Kuysor) SanitizeStruct(data interface{}) (next string, prev string, err error) {

	if p.uTabling == nil {
		return next, prev, errors.New("uTabling is nil")
	}

	if p.uTabling.uPaging == nil {
		return next, prev, errors.New("uPaging is nil")
	}

	// Get the reflect.Value of the data
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Slice || v.Elem().Type().Elem().Kind() != reflect.Struct {
		return "", "", fmt.Errorf("data must be a pointer to slice of struct")
	}

	var (
		sliceVal         = v.Elem()
		totalData        = sliceVal.Len()
		totalDataUpdated = totalData
		vSorts           = p.vTabling.vSorts
		limit            = p.uTabling.uPaging.Limit
		vcursor          = p.vTabling.vCursor
		isFirstPage      = vcursor == nil
		cursorPrev       = vCursor{
			Prefix: cursorPrefixPrev,
			Cols:   make(map[string]interface{}),
		}
		cursorNext = vCursor{
			Prefix: cursorPrefixNext,
			Cols:   make(map[string]interface{}),
		}
	)

	if totalData == 0 {
		return next, prev, nil
	}

	// Set cursor to next if it is the first page
	if isFirstPage {
		vcursor = &vCursor{
			Prefix: cursorPrefixNext,
		}
	}

	// Reverse the data if it is previous page
	if vcursor.Prefix.isPrev() {
		reverseSlice(sliceVal)
	}

	if totalData > limit {
		// Remove extra element
		if vcursor.Prefix.isNext() {
			sliceVal.Set(sliceVal.Slice(0, totalData-1))
		} else {
			sliceVal.Set(sliceVal.Slice(1, totalData))
		}

		// Update total data after deletion
		totalDataUpdated = sliceVal.Len()
	}

	// Get field values for cursors using the tag key
	for _, vSort := range *vSorts {
		firstItem := sliceVal.Index(0)
		lastItem := sliceVal.Index(totalDataUpdated - 1)

		firstVal := getFieldValueByTag(firstItem, vSort.column, p.options.StructTag)
		lastVal := getFieldValueByTag(lastItem, vSort.column, p.options.StructTag)

		cursorPrev.Cols[vSort.column] = firstVal
		cursorNext.Cols[vSort.column] = lastVal
	}

	// generate cursor
	return p.generateCursor(totalData, limit, isFirstPage, vcursor, &cursorPrev, &cursorNext)

}

// generateCursor generates the next and previous cursor for the pagination.
func (p *Kuysor) generateCursor(totalData int, limit int, isFirstPage bool, vcursor *vCursor, cursorPrev *vCursor, cursorNext *vCursor) (string, string, error) {

	var (
		next, prev string
	)

	if (totalData > limit) || (vcursor.Prefix.isPrev() && totalData <= limit) {
		nextB64, err := cursorNext.generateCursorBase64()
		if err != nil {
			return next, prev, err
		}
		next = string(nextB64)
	}

	if (totalData > limit && !isFirstPage) || (totalData <= limit && vcursor.Prefix.isNext() && !isFirstPage) {
		prevB64, err := cursorPrev.generateCursorBase64()
		if err != nil {
			return next, prev, err
		}
		prev = string(prevB64)
	}

	return next, prev, nil
}

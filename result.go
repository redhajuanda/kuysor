package kuysor

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/redhajuanda/kuysor/modifier"
)

// Result represents the result of a query.
type Result struct {
	Query string
	Args  []any
	ks    *Kuysor
}

// BuildCountQuery builds a count query from the original query.
func BuildCountQuery(query string) (string, error) {

	s := modifier.NewSQLModifier(query)

	err := s.ConvertToCount()
	if err != nil {
		return "", fmt.Errorf("failed to convert to count query: %v", err)
	}

	count, err := s.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build count query: %v", err)
	}

	return count, nil

}

// SanitizeMap handles the map data for the cursor pagination.
// It returns the next and previous cursor.
func (r *Result) SanitizeMap(data *[]map[string]any) (next string, prev string, err error) {

	if r.ks.uTabling == nil {
		return next, prev, errors.New("uTabling is nil")
	}

	if r.ks.uTabling.uPaging == nil {
		return next, prev, errors.New("uPaging is nil")
	}

	var (
		totalData        = len(*data)
		totalDataUpdated = totalData
		vSorts           = r.ks.vTabling.vSorts
		limit            = r.ks.uTabling.uPaging.Limit
		vcursor          = r.ks.vTabling.vCursor
		isFirstPage      = vcursor == nil
		cursorPrev       = vCursor{
			Prefix: cursorPrefixPrev,
			Cols:   make(map[string]any),
		}
		cursorNext = vCursor{
			Prefix: cursorPrefixNext,
			Cols:   make(map[string]any),
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
			if err := deleteElement(data, totalData-1); err != nil {
				return next, prev, fmt.Errorf("failed to delete element: %v", err)
			}
		} else {
			if err := deleteElement(data, 0); err != nil {
				return next, prev, fmt.Errorf("failed to delete element: %v", err)
			}
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
	return r.generateCursor(totalData, limit, isFirstPage, vcursor, &cursorPrev, &cursorNext)

}

// SanitizeStruct handles struct data for the cursor pagination.
// It returns the next and previous cursor.
func (r *Result) SanitizeStruct(data any) (next string, prev string, err error) {

	if r.ks.uTabling == nil {
		return next, prev, errors.New("uTabling is nil")
	}

	if r.ks.uTabling.uPaging == nil {
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
		vSorts           = r.ks.vTabling.vSorts
		limit            = r.ks.uTabling.uPaging.Limit
		vcursor          = r.ks.vTabling.vCursor
		isFirstPage      = vcursor.cursor == ""
		cursorPrev       = vCursor{
			Prefix: cursorPrefixPrev,
			Cols:   make(map[string]any),
		}
		cursorNext = vCursor{
			Prefix: cursorPrefixNext,
			Cols:   make(map[string]any),
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

		firstVal := getFieldValueByTag(firstItem, vSort.column, r.ks.options.StructTag)
		lastVal := getFieldValueByTag(lastItem, vSort.column, r.ks.options.StructTag)

		cursorPrev.Cols[vSort.column] = firstVal
		cursorNext.Cols[vSort.column] = lastVal
	}

	// generate cursor
	return r.generateCursor(totalData, limit, isFirstPage, vcursor, &cursorPrev, &cursorNext)

}

// generateCursor generates the next and previous cursor for the pagination.
func (r *Result) generateCursor(totalData int, limit int, isFirstPage bool, vcursor *vCursor, cursorPrev *vCursor, cursorNext *vCursor) (string, string, error) {

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

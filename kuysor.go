package kuysor

import (
	"fmt"

	"errors"
)

type Kuysor struct {
	sql      string
	uTabling *uTabling
	vTabling *vTabling
	options  Options
}

// NewTabling creates a new tabling instance.
// It accepts the sql query and options.
// Options is optional, if not provided, it will use the default options.
func New(sql string, opt ...Options) *Kuysor {

	p := &Kuysor{sql: sql}

	if len(opt) > 0 { // override the options
		p.options = opt[0]
	} else if options != nil { // set global options
		p.options = *options
	} else { // set default options
		p.options = Options{
			Dialect: MySQL,
		}
	}

	return p

}

// Paging sets the pagination configuration.
func (p *Kuysor) Paging(limit int, cursor string, columnID string) *Kuysor {

	if p.uTabling == nil {
		p.uTabling = &uTabling{}
	}

	p.uTabling.uPaging = &uPaging{
		Limit:    limit,
		Cursor:   cursor,
		ColumnID: columnID,
	}

	return p

}

// Sort sets the sorting / order for the query.
func (p *Kuysor) Sort(sorts ...string) *Kuysor {

	if p.uTabling == nil {
		p.uTabling = &uTabling{}
	}

	p.uTabling.uSort = &uSort{
		Sorts: sorts,
	}

	return p

}

// Build builds the paginated / sorted query.
func (p *Kuysor) Build() (string, error) {

	return p.build()

}

// build builds the paginated / sorted query.
func (p *Kuysor) build() (string, error) {

	var (
		result string
	)

	// validate user input
	if p.uTabling == nil {
		return p.sql, errors.New("nothing to build")
	}
	if p.uTabling.uPaging != nil && p.uTabling.uSort == nil {
		return "", errors.New("sort is required for cursor pagination")
	}

	// prepare vTabling
	err := p.prepareVTabling()
	if err != nil {
		return "", fmt.Errorf("failed to prepare vTabling: %v", err)
	}

	switch p.options.Dialect {
	case MySQL:
		result, err = newMySQLParser(p).Build()
	}

	return result, err
}

// prepareVTabling prepares the vTabling data.
// vTabling is the parsed version of uTabling, it is used internally to build the query.
func (p *Kuysor) prepareVTabling() (err error) {

	p.vTabling = &vTabling{}

	if p.uTabling.uPaging != nil {

		var cursor = p.uTabling.uPaging.Cursor

		if cursor != "" {
			p.vTabling.vCursor, err = parseCursor(cursor)
			if err != nil {
				return
			}
		}

	}

	if p.uTabling.uSort != nil {
		p.vTabling.vSorts = parseSort(p.uTabling.uSort.Sorts)
	}

	return

}

// handleDataCursor handles the data for the cursor pagination.
func (p *Kuysor) HandleDataCursorMap(data *[]map[string]interface{}) (string, string, error) {

	if p.uTabling == nil {
		return "", "", nil
	}

	if p.uTabling.uPaging == nil {
		return "", "", nil
	}

	var (
		totalData = len(*data)

		vSorts = p.vTabling.vSorts
		// sortBy string
		// sortType string

		limit   = p.uTabling.uPaging.Limit
		vcursor = p.vTabling.vCursor
		// cursor           = p
		isFirstPage = vcursor == nil
		// isNext      bool = true

		next string
		prev string
	)

	if totalData == 0 {
		return next, prev, nil
	}

	// parse sort string to get the sort by and sort type
	// sortBy, _ = p.parseSort(sort)

	// if vCursor != nil {
	// 	decodedCursor, err := decodeCursor(vCursor.cursor)
	// 	if err != nil {
	// 		return "", "", err
	// 	}

	// 	if strings.HasPrefix(decodedCursor, "prev") {
	// 		isNext = false
	// 	}
	// }

	// reverse the data if it is previous page
	if vcursor == nil {
		vcursor = &vCursor{
			Prefix: CursorPrefixNext,
		}
	}
	if vcursor.isPrev() {
		reverse(data)
	}

	if totalData > limit {

		if vcursor.isNext() {
			deleteElement(data, totalData-1)
		} else {
			deleteElement(data, 0)
		}

		// update total data after delete
		totalData = len(*data)

		cursorPrev := vCursor{
			Prefix: CursorPrefixPrev,
		}
		cursorNext := vCursor{
			Prefix: CursorPrefixNext,
		}

		firstID, ok := (*data)[0][p.uTabling.uPaging.ColumnID].(string)
		if !ok {
			return "", "", nil
		}

		prevID := map[string]string{
			p.uTabling.uPaging.ColumnID: firstID,
		}

		lastID, ok := (*data)[totalData-1][p.uTabling.uPaging.ColumnID].(string)
		if !ok {
			return "", "", nil
		}

		nextID := map[string]string{
			p.uTabling.uPaging.ColumnID: lastID,
		}

		var (
			prevCols = make(map[string]string)
			nextCols = make(map[string]string)
		)

		for _, vSort := range *vSorts {

			var (
				firstSort = fmt.Sprintf("%v", (*data)[0][vSort.column])
				lastSort  = fmt.Sprintf("%v", (*data)[totalData-1][vSort.column])
			)

			prevCols[vSort.column] = firstSort
			nextCols[vSort.column] = lastSort

		}
		// var (
		// 	firstSort = fmt.Sprintf("%v", (*data)[0][sortBy])
		// 	lastSort  = fmt.Sprintf("%v", (*data)[totalData-1][sortBy])
		// )

		cursorPrev.Id = prevID
		cursorNext.Id = nextID
		cursorPrev.Cols = prevCols
		cursorNext.Cols = nextCols

		var err error
		next, err = generateCursor(cursorNext)
		if err != nil {
			return "", "", err
		}

		// next = t.generateCursor("next", lastID, lastSort)
		if !isFirstPage {
			// prev = t.generateCursor("prev", firstID, firstSort)
			prev, err = generateCursor(cursorPrev)
			if err != nil {
				return "", "", err
			}
		}
	} else {
		// firstID, ok := (*data)[0][t.tabling.Paging.ColumnID].(string)
		// if !ok {
		// 	return "", "", nil
		// }

		// lastID, ok := (*data)[totalData-1][t.tabling.Paging.ColumnID].(string)
		// if !ok {
		// 	return "", "", nil
		// }

		// var (
		// 	firstSort = fmt.Sprintf("%v", (*data)[0][sortBy])
		// 	lastSort  = fmt.Sprintf("%v", (*data)[totalData-1][sortBy])
		// )

		// if isNext && !isFirstPage {
		// 	prev = t.generateCursor("prev", firstID, firstSort)
		// } else if !isNext {
		// 	next = t.generateCursor("next", lastID, lastSort)
		// }

		var (
			cursorPrev = vCursor{
				Prefix: CursorPrefixPrev,
				Id:     make(map[string]string),
				Cols:   make(map[string]string),
			}
			cursorNext = vCursor{
				Prefix: CursorPrefixNext,
				Id:     make(map[string]string),
				Cols:   make(map[string]string),
			}
		)

		firstID, ok := (*data)[0][p.uTabling.uPaging.ColumnID].(string)
		if !ok {
			return "", "", nil
		}

		prevID := map[string]string{
			p.uTabling.uPaging.ColumnID: firstID,
		}

		lastID, ok := (*data)[totalData-1][p.uTabling.uPaging.ColumnID].(string)
		if !ok {
			return "", "", nil
		}

		nextID := map[string]string{
			p.uTabling.uPaging.ColumnID: lastID,
		}

		var (
			prevCols = make(map[string]string)
			nextCols = make(map[string]string)
		)

		for _, vSort := range *vSorts {

			var (
				firstSort = fmt.Sprintf("%v", (*data)[0][vSort.column])
				lastSort  = fmt.Sprintf("%v", (*data)[totalData-1][vSort.column])
			)

			prevCols[vSort.column] = firstSort
			nextCols[vSort.column] = lastSort

		}

		cursorPrev.Id = prevID
		cursorNext.Id = nextID
		cursorPrev.Cols = prevCols
		cursorNext.Cols = nextCols

		var err error

		if vcursor.isNext() && !isFirstPage {
			prev, err = generateCursor(cursorPrev)
			if err != nil {
				return "", "", err
			}

		} else if vcursor.isPrev() {
			next, err = generateCursor(cursorNext)
			if err != nil {
				return "", "", err
			}
		}

	}

	return next, prev, nil
}

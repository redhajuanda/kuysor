package kuysor

import (
	"fmt"
	"strings"
)

type orderDirection int8

const (
	ascOrder orderDirection = iota
	descOrder
)

type uSort struct {
	Sorts []string
}

type NullSortMethod uint8

const (
	BoolSort  NullSortMethod = iota
	FirstLast                = iota
	CaseWhen                 = iota
)

type vSort struct {
	prefix         string
	column         string
	nullable       bool
	nullSortMethod NullSortMethod
	direction      orderDirection
}

// isNullable returns true if the sort is nullable.
func (s *vSort) isNullable() bool {
	return s.nullable
}

// isAsc returns true if the sort is ascending.
func (s *vSort) isAsc() bool {
	return s.direction == ascOrder
}

// isDesc returns true if the sort is descending.
func (s *vSort) isDesc() bool {
	return s.direction == descOrder
}

// extractColumn extracts the qualifier and column from the column name.
func (s *vSort) extractColumn() (qualifier string, column string, err error) {

	columns := strings.Split(s.column, ".")
	if len(columns) == 2 {
		qualifier = columns[0]
		column = columns[1]
	} else if len(columns) == 1 {
		column = columns[0]
	} else {
		return "", "", fmt.Errorf("invalid column name: %s", s.column)
	}

	return
}

type vSorts []vSort

// reverseDirection reverses the direction of the vSorts.
func (s vSorts) reverseDirection() vSorts {

	vSorts := make(vSorts, len(s))
	copy(vSorts, s)

	for i := range s {
		if vSorts[i].direction == ascOrder {
			vSorts[i].direction = descOrder
		} else {
			vSorts[i].direction = ascOrder
		}
	}

	return vSorts
}

// parseSort parses the sort string and returns the vSorts.
func parseSort(sorts []string, nullSortMethod NullSortMethod) *vSorts {

	var sSorts = make(vSorts, 0)

	for _, s := range sorts {
		nullable := false

		// split the sort string into parts
		s = strings.TrimSpace(s)

		t := strings.Split(s, " ")
		if len(t) > 1 {
			if t[1] != "null" {
				panic(fmt.Sprintf("unexpected syntax: %s", t[1]))
			}

			nullable = true
			s = t[0]

		} else if len(t) > 2 {
			panic(fmt.Sprintf("unexpected syntax: %s", t[2]))
		}

		if strings.HasPrefix(s, "-") {
			sSorts = append(sSorts, vSort{column: strings.TrimPrefix(s, "-"), prefix: "-", direction: descOrder, nullable: nullable, nullSortMethod: nullSortMethod})
		} else {
			sSorts = append(sSorts, vSort{column: strings.TrimPrefix(s, "+"), prefix: "+", direction: ascOrder, nullable: nullable, nullSortMethod: nullSortMethod})
		}
	}

	return &sSorts

}

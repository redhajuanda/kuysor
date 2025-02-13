package kuysor

import (
	"strings"

	"github.com/redhajuanda/sqlparser"
)

type uSort struct {
	Sorts []string
}

type vSort struct {
	prefix    string
	column    string
	nullable  bool
	direction sqlparser.OrderDirection
}

func (s *vSort) isNullable() bool {
	return s.nullable
}

// func (s *vSort) desc() bool {
// 	return s.prefix == "-"
// }

// func (s *vSort) asc() bool {
// 	return s.prefix == "+"
// }

type vSorts []vSort

func (s vSorts) reverseDirection() vSorts {

	vSorts := make(vSorts, len(s))
	copy(vSorts, s)

	for i := range s {
		if vSorts[i].direction == sqlparser.AscOrder {
			vSorts[i].direction = sqlparser.DescOrder
		} else {
			vSorts[i].direction = sqlparser.AscOrder
		}
	}

	return vSorts
}

// parseSort parses the sort string and returns the vSorts.
func parseSort(sorts []string) *vSorts {

	var sSorts = make(vSorts, 0)

	for _, s := range sorts {
		nullable := false

		if strings.HasSuffix(strings.ToLower(s), "nullable") {
			nullable = true
			s = strings.TrimSuffix(s, " nullable")
		}
		if strings.HasPrefix(s, "-") {
			sSorts = append(sSorts, vSort{column: strings.TrimPrefix(s, "-"), prefix: "-", direction: sqlparser.DescOrder, nullable: nullable})
		} else {
			sSorts = append(sSorts, vSort{column: strings.TrimPrefix(s, "+"), prefix: "+", direction: sqlparser.AscOrder, nullable: nullable})
		}
	}

	return &sSorts

}

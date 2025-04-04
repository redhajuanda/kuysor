package kuysor

import (
	"strings"
	"testing"
)

func TestFirstPage(t *testing.T) {
	var testCases = []struct {
		in        string
		out       string
		orderBy   []string
		paramsIn  []any
		paramsOut []any
	}{
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"-id"},
			out:       "SELECT * FROM `table` ORDER BY id DESC LIMIT ?",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"-code", "-id"},
			out:       "SELECT * FROM `table` ORDER BY code DESC, id DESC LIMIT ?",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"code", "+id"},
			out:       "SELECT * FROM `table` ORDER BY code ASC, id ASC LIMIT ?",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"code", "-id"},
			out:       "SELECT * FROM `table` ORDER BY code ASC, id DESC LIMIT ?",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table` as t",
			orderBy:   []string{"t.code", "-t.id"},
			out:       "SELECT * FROM `table` as t ORDER BY t.code ASC, t.id DESC LIMIT ?",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table` as t",
			orderBy:   []string{"t.code", "-t.id", "t.name"},
			out:       "SELECT * FROM `table` as t ORDER BY t.code ASC, t.id DESC, t.name ASC LIMIT ?",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table` as t WHERE t.id = ?",
			orderBy:   []string{"t.code", "-t.id", "t.name"},
			out:       "SELECT * FROM `table` as t WHERE t.id = ? ORDER BY t.code ASC, t.id DESC, t.name ASC LIMIT ?",
			paramsIn:  []any{1},
			paramsOut: []any{1, 11},
		},
		{
			in:        "SELECT * FROM `table` as t GROUP BY t.id",
			orderBy:   []string{"t.code", "-t.id", "t.name"},
			out:       "SELECT * FROM `table` as t GROUP BY t.id ORDER BY t.code ASC, t.id DESC, t.name ASC LIMIT ?",
			paramsOut: []any{11},
		},
	}

	for _, tc := range testCases {
		p := NewQuery(tc.in)
		p.WithOrderBy(tc.orderBy...).WithLimit(10)
		if len(tc.paramsIn) > 0 {
			p.WithArgs(tc.paramsIn...)
		}
		res, err := p.Build()
		if err != nil {
			t.Error(err)
		}

		expected := strings.ToLower(tc.out)
		got := strings.ToLower(res.Query)
		if got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}

		if len(res.Args) != len(tc.paramsOut) {
			t.Errorf("Expected %d params, got %d", len(tc.paramsOut), len(res.Args))
		}
		for i, v := range res.Args {
			if v != tc.paramsOut[i] {
				t.Errorf("Expected %v, got %v", tc.paramsOut[i], v)
			}
		}
	}
}

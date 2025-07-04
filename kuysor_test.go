package kuysor

import (
	"strings"
	"testing"
)

func TestCursorFirstPage(t *testing.T) {
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
		p := NewQuery(tc.in, Cursor)
		p.WithOrderBy(tc.orderBy...).WithLimit(10).WithCursor("")
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

func TestOrderByOnly(t *testing.T) {
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
			out:       "SELECT * FROM `table` ORDER BY id DESC",
			paramsOut: []any{},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"-code", "-id"},
			out:       "SELECT * FROM `table` ORDER BY code DESC, id DESC",
			paramsOut: []any{},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"code", "+id"},
			out:       "SELECT * FROM `table` ORDER BY code ASC, id ASC",
			paramsOut: []any{},
		},
	}

	for _, tc := range testCases {
		p := NewQuery(tc.in, "")
		p.WithOrderBy(tc.orderBy...)
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

func TestOffset(t *testing.T) {
	var testCases = []struct {
		in        string
		out       string
		limit     int
		offset    int
		paramsIn  []any
		paramsOut []any
	}{
		{
			in:        "SELECT * FROM `table`",
			limit:     10,
			out:       "SELECT * FROM `table` LIMIT ? OFFSET ?",
			paramsOut: []any{10, 0},
		},
		{
			in:        "SELECT * FROM `table`",
			limit:     10,
			offset:    5,
			out:       "SELECT * FROM `table` LIMIT ? OFFSET ?",
			paramsOut: []any{10, 5},
		},
		{
			in:        "SELECT * FROM `table`",
			limit:     10,
			offset:    0,
			out:       "SELECT * FROM `table` LIMIT ? OFFSET ?",
			paramsOut: []any{10, 0},
		},
		{
			in:        "SELECT * FROM `table`",
			limit:     10,
			offset:    10,
			out:       "SELECT * FROM `table` LIMIT ? OFFSET ?",
			paramsOut: []any{10, 10},
		},
	}

	for _, tc := range testCases {
		p := NewQuery(tc.in, Offset)
		p.WithLimit(tc.limit).WithOffset(tc.offset)
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

func TestOffsetWithOrder(t *testing.T) {
	var testCases = []struct {
		in        string
		out       string
		orderBy   []string
		limit     int
		offset    int
		paramsIn  []any
		paramsOut []any
	}{
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"-id"},
			limit:     10,
			out:       "SELECT * FROM `table` ORDER BY id DESC LIMIT ? OFFSET ?",
			paramsOut: []any{10, 0},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"-code", "-id"},
			limit:     10,
			offset:    5,
			out:       "SELECT * FROM `table` ORDER BY code DESC, id DESC LIMIT ? OFFSET ?",
			paramsOut: []any{10, 5},
		},
		{
			in:        "SELECT * FROM `table` WHERE id = ?",
			orderBy:   []string{"-code", "-id"},
			limit:     10,
			offset:    10,
			paramsIn:  []any{1},
			out:       "SELECT * FROM `table` WHERE id = ? ORDER BY code DESC, id DESC LIMIT ? OFFSET ?",
			paramsOut: []any{1, 10, 10},
		},
	}

	for _, tc := range testCases {
		p := NewQuery(tc.in, Offset)
		p.WithOrderBy(tc.orderBy...).WithLimit(tc.limit).WithOffset(tc.offset)
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

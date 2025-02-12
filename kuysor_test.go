package kuysor

import (
	"strings"
	"testing"
)

func TestFirstPage(t *testing.T) {
	var testCases = []struct {
		in      string
		out     string
		orderBy []string
	}{
		{
			in:      "SELECT * FROM `table`",
			orderBy: []string{"-id"},
			out:     "SELECT * FROM `table` ORDER BY id DESC LIMIT 11",
		},
		{
			in:      "SELECT * FROM `table`",
			orderBy: []string{"-code", "-id"},
			out:     "SELECT * FROM `table` ORDER BY `code` DESC, id DESC LIMIT 11",
		},
		{
			in:      "SELECT * FROM `table`",
			orderBy: []string{"code", "+id"},
			out:     "SELECT * FROM `table` ORDER BY `code` ASC, id ASC LIMIT 11",
		},
		{
			in:      "SELECT * FROM `table`",
			orderBy: []string{"code", "-id"},
			out:     "SELECT * FROM `table` ORDER BY `code` ASC, id DESC LIMIT 11",
		},
		{
			in:      "SELECT * FROM `table` as t",
			orderBy: []string{"t.code", "-t.id"},
			out:     "SELECT * FROM `table` as t ORDER BY t.`code` ASC, t.id DESC LIMIT 11",
		},
		{
			in:      "SELECT * FROM `table` as t",
			orderBy: []string{"t.code", "-t.id", "t.name"},
			out:     "SELECT * FROM `table` as t ORDER BY t.`code` ASC, t.id DESC, t.`name` ASC LIMIT 11",
		},
		{
			in:      "SELECT * FROM `table` as t WHERE t.id = 1",
			orderBy: []string{"t.code", "-t.id", "t.name"},
			out:     "SELECT * FROM `table` as t WHERE t.id = 1 ORDER BY t.`code` ASC, t.id DESC, t.`name` ASC LIMIT 11",
		},
		{
			in:      "SELECT * FROM `table` as t GROUP BY t.id",
			orderBy: []string{"t.code", "-t.id", "t.name"},
			out:     "SELECT * FROM `table` as t GROUP BY t.id ORDER BY t.`code` ASC, t.id DESC, t.`name` ASC LIMIT 11",
		},
	}

	for _, tc := range testCases {
		p := New(tc.in)
		p.WithSort(tc.orderBy...).WithLimit(10)
		sql, err := p.Build()
		if err != nil {
			t.Error(err)
		}

		expected := strings.ToLower(tc.out)
		got := strings.ToLower(sql)
		if got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	}
}

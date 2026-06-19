package kuysor

import (
	"strings"
	"testing"
)

func TestCursorFirstPageQuestion(t *testing.T) {
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

func TestCursorFirstPageColon(t *testing.T) {
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
			out:       "SELECT * FROM `table` ORDER BY id DESC LIMIT :1",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"-code", "-id"},
			out:       "SELECT * FROM `table` ORDER BY code DESC, id DESC LIMIT :1",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"code", "+id"},
			out:       "SELECT * FROM `table` ORDER BY code ASC, id ASC LIMIT :1",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table`",
			orderBy:   []string{"code", "-id"},
			out:       "SELECT * FROM `table` ORDER BY code ASC, id DESC LIMIT :1",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table` as t",
			orderBy:   []string{"t.code", "-t.id"},
			out:       "SELECT * FROM `table` as t ORDER BY t.code ASC, t.id DESC LIMIT :1",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table` as t",
			orderBy:   []string{"t.code", "-t.id", "t.name"},
			out:       "SELECT * FROM `table` as t ORDER BY t.code ASC, t.id DESC, t.name ASC LIMIT :1",
			paramsOut: []any{11},
		},
		{
			in:        "SELECT * FROM `table` as t WHERE t.id = :1",
			orderBy:   []string{"t.code", "-t.id", "t.name"},
			out:       "SELECT * FROM `table` as t WHERE t.id = :1 ORDER BY t.code ASC, t.id DESC, t.name ASC LIMIT :2",
			paramsIn:  []any{1},
			paramsOut: []any{1, 11},
		},
		{
			in:        "SELECT * FROM `table` as t GROUP BY t.id",
			orderBy:   []string{"t.code", "-t.id", "t.name"},
			out:       "SELECT * FROM `table` as t GROUP BY t.id ORDER BY t.code ASC, t.id DESC, t.name ASC LIMIT :1",
			paramsOut: []any{11},
		},
	}

	for _, tc := range testCases {

		i := NewInstance(Options{
			PlaceHolderType: Colon,
		})
		p := i.NewQuery(tc.in, Cursor)
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

func TestCursorSecondPage(t *testing.T) {
	var testCases = []struct {
		in        string
		out       string
		orderBy   []string
		cursor    string
		paramsIn  []any
		paramsOut []any
	}{
		{
			in:        `WITH last_activity_log AS ( SELECT object_instance, MAX(created_at) AS created_at FROM activity_log WHERE kind = 'internal-ticket-activity-log' AND JSON_TYPE(JSON_EXTRACT(attribute, '$.log')) = 'OBJECT' AND JSON_LENGTH(JSON_EXTRACT(attribute, '$.log')) > 0 GROUP BY object_instance ) select it.id, it.code, it.name, it.stage, lal.created_at as stage_changed_at, it.assigned_to_id, it.total_awb, it.complaining_hub_id, it.team_id, team.name as team_name, team.code as team_code, it.assigned_to_timestamp, it.attribute, it.created_at, it.updated_at, branch.indexed_property_1 as complaining_hub__id, branch.code as complaining_hub__code, branch.name as complaining_hub__name from internal_ticket it left join account team on team.id = it.team_id LEFT JOIN account branch ON branch.indexed_property_1 = it.complaining_hub_id left join account assigned_to on assigned_to.id = it.assigned_to_id left join last_activity_log lal ON lal.object_instance = it.id where it.deleted_at = 0 and it.assigned_to_id in (?)`,
			orderBy:   []string{"-it.id"},
			paramsIn:  []any{1},
			paramsOut: []any{1, "01KCR6ET11CM8M45GNQQHJS7K0", 11},
			cursor:    "eyJwcmVmaXgiOiJuZXh0IiwiY29scyI6eyJpZCI6IjAxS0NSNkVUMTFDTThNNDVHTlFRSEpTN0swIn19",
			out:       `WITH last_activity_log AS ( SELECT object_instance, MAX(created_at) AS created_at FROM activity_log WHERE kind = 'internal-ticket-activity-log' AND JSON_TYPE(JSON_EXTRACT(attribute, '$.log')) = 'OBJECT' AND JSON_LENGTH(JSON_EXTRACT(attribute, '$.log')) > 0 GROUP BY object_instance ) select it.id, it.code, it.name, it.stage, lal.created_at as stage_changed_at, it.assigned_to_id, it.total_awb, it.complaining_hub_id, it.team_id, team.name as team_name, team.code as team_code, it.assigned_to_timestamp, it.attribute, it.created_at, it.updated_at, branch.indexed_property_1 as complaining_hub__id, branch.code as complaining_hub__code, branch.name as complaining_hub__name from internal_ticket it left join account team on team.id = it.team_id LEFT JOIN account branch ON branch.indexed_property_1 = it.complaining_hub_id left join account assigned_to on assigned_to.id = it.assigned_to_id left join last_activity_log lal ON lal.object_instance = it.id where it.deleted_at = 0 and it.assigned_to_id in (?) and (it.id < ?) ORDER BY it.id DESC LIMIT ?`,
		},
	}

	for _, tc := range testCases {

		i := NewInstance(Options{
			PlaceHolderType: Question,
		})
		p := i.NewQuery(tc.in, Cursor)
		p.WithOrderBy(tc.orderBy...).WithLimit(10).WithCursor(tc.cursor)
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

// TestCTETargetFirstPage verifies that WithCTETarget routes ORDER BY and LIMIT
// into the named CTE body and leaves the main query untouched.
func TestCTETargetFirstPage(t *testing.T) {
	query := `
		WITH filtered_ticket AS (
			SELECT t.id
			FROM ticket t
			WHERE t.deleted_at = 0
			AND t.status = ?
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		GROUP BY t.id
		ORDER BY t.id
	`

	res, err := NewQuery(query, Cursor).
		WithCTETarget("filtered_ticket").
		WithOrderBy("t.id").
		WithLimit(10).
		WithArgs("active").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := strings.ToLower(res.Query)

	// ORDER BY and LIMIT must appear inside the CTE (before the closing ')' of the CTE)
	// and also the main ORDER BY must remain
	cteClose := strings.Index(q, ") select")
	if cteClose == -1 {
		t.Fatalf("could not find CTE closing in query: %s", q)
	}

	cteBody := q[:cteClose]
	mainBody := q[cteClose:]

	if !strings.Contains(cteBody, "order by t.id asc") {
		t.Errorf("expected ORDER BY inside CTE body, got CTE body: %s", cteBody)
	}
	if !strings.Contains(cteBody, "limit ?") {
		t.Errorf("expected LIMIT inside CTE body, got CTE body: %s", cteBody)
	}
	// main query ORDER BY must not be changed
	if !strings.Contains(mainBody, "order by t.id") {
		t.Errorf("expected original ORDER BY to remain in main query, got: %s", mainBody)
	}

	// args: user arg "active" first, then limit+1=11
	expected := []any{"active", 11}
	if len(res.Args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(res.Args), res.Args)
	}
	for i, v := range expected {
		if v != res.Args[i] {
			t.Errorf("arg[%d]: expected %v, got %v", i, v, res.Args[i])
		}
	}
}

// TestCTETargetSecondPageNext verifies cursor WHERE is appended inside the CTE
// and ORDER BY / LIMIT are also placed inside the CTE on the next page.
func TestCTETargetSecondPageNext(t *testing.T) {
	// cursor: {"prefix":"next","cols":{"id":"100"}}
	cursor := base64Encode(`{"prefix":"next","cols":{"id":"100"}}`)

	query := `
		WITH filtered_ticket AS (
			SELECT t.id FROM ticket t
			WHERE t.deleted_at = 0 AND t.status = ?
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		GROUP BY t.id
		ORDER BY t.id
	`

	res, err := NewQuery(query, Cursor).
		WithCTETarget("filtered_ticket").
		WithOrderBy("t.id").
		WithLimit(10).
		WithCursor(cursor).
		WithArgs("active").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := strings.ToLower(res.Query)

	cteClose := strings.Index(q, ") select")
	if cteClose == -1 {
		t.Fatalf("could not find CTE closing in query: %s", q)
	}
	cteBody := q[:cteClose]
	mainBody := q[cteClose:]

	// cursor WHERE must be inside the CTE
	if !strings.Contains(cteBody, "t.id > ?") {
		t.Errorf("expected cursor WHERE inside CTE body, got: %s", cteBody)
	}
	if !strings.Contains(cteBody, "order by t.id asc") {
		t.Errorf("expected ORDER BY inside CTE body, got: %s", cteBody)
	}
	if !strings.Contains(cteBody, "limit ?") {
		t.Errorf("expected LIMIT inside CTE body, got: %s", cteBody)
	}

	// main query ORDER BY must mirror the CTE sort direction (ASC)
	if !strings.Contains(mainBody, "order by t.id asc") {
		t.Errorf("expected ORDER BY t.id ASC in main query, got: %s", mainBody)
	}

	// args: "active" (user WHERE), then cursor value "100", then limit+1=11
	expected := []any{"active", "100", 11}
	if len(res.Args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(res.Args), res.Args)
	}
	for i, v := range expected {
		if v != res.Args[i] {
			t.Errorf("arg[%d]: expected %v, got %v", i, v, res.Args[i])
		}
	}
}

// TestCTETargetSecondPagePrev verifies that the previous-page cursor reverses
// ORDER BY inside the CTE body.
func TestCTETargetSecondPagePrev(t *testing.T) {
	// cursor: {"prefix":"prev","cols":{"id":"100"}}
	cursor := base64Encode(`{"prefix":"prev","cols":{"id":"100"}}`)

	query := `
		WITH filtered_ticket AS (
			SELECT t.id FROM ticket t
			WHERE t.deleted_at = 0
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		ORDER BY t.id
	`

	res, err := NewQuery(query, Cursor).
		WithCTETarget("filtered_ticket").
		WithOrderBy("t.id").
		WithLimit(10).
		WithCursor(cursor).
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := strings.ToLower(res.Query)

	cteClose := strings.Index(q, ") select")
	if cteClose == -1 {
		t.Fatalf("could not find CTE closing in query: %s", q)
	}
	cteBody := q[:cteClose]
	mainBody := q[cteClose:]

	// prev page: ORDER BY must be reversed (DESC) inside CTE
	if !strings.Contains(cteBody, "order by t.id desc") {
		t.Errorf("expected reversed ORDER BY DESC inside CTE body, got: %s", cteBody)
	}
	// cursor WHERE uses < for ascending column going backwards
	if !strings.Contains(cteBody, "t.id < ?") {
		t.Errorf("expected cursor WHERE (t.id < ?) inside CTE body, got: %s", cteBody)
	}

	// main query ORDER BY must mirror the CTE reversed direction (DESC) so the
	// final joined result set is returned in the same order as the CTE selected.
	if !strings.Contains(mainBody, "order by t.id desc") {
		t.Errorf("expected ORDER BY t.id DESC in main query to mirror CTE, got: %s", mainBody)
	}
}

// TestCTETargetValidationNoCTE verifies WithCTETarget returns an error
// when the query does not contain a WITH clause.
func TestCTETargetValidationNoCTE(t *testing.T) {
	_, err := NewQuery("SELECT id FROM ticket WHERE deleted_at = 0", Cursor).
		WithCTETarget("filtered_ticket").
		WithOrderBy("id").
		WithLimit(10).
		Build()
	if err == nil {
		t.Error("expected error for CTETarget on non-CTE query, got nil")
	}
}

// TestCTETargetValidationCTENotFound verifies that an error is returned when
// the specified CTE name does not exist in the query.
func TestCTETargetValidationCTENotFound(t *testing.T) {
	query := `WITH ft AS (SELECT id FROM ticket t) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id`
	_, err := NewQuery(query, Cursor).
		WithCTETarget("nonexistent").
		WithOrderBy("t.id").
		WithLimit(10).
		Build()
	if err == nil {
		t.Error("expected error for non-existent CTE name, got nil")
	}
}

// TestCTETargetOptions verifies per-clause routing via CTEOptions.
func TestCTETargetOptions(t *testing.T) {
	baseQuery := `
		WITH cte AS (
			SELECT t.id FROM ticket t
			WHERE t.deleted_at = 0 AND t.status = ?
		)
		SELECT t.id, t.code
		FROM cte
		JOIN ticket t ON t.id = cte.id
		ORDER BY t.id
	`

	findBoundary := func(t *testing.T, q string) (cteBody, mainBody string) {
		t.Helper()
		q = strings.ToLower(q)
		idx := strings.Index(q, ") select")
		if idx == -1 {
			t.Fatalf("could not find CTE boundary in: %s", q)
		}
		return q[:idx], q[idx:]
	}

	tests := []struct {
		name           string
		opts           CTEOptions
		paginationType PaginationType
		cursor         string
		wantCTEHas     []string
		wantCTENotHas  []string
		wantMainHas    []string
		wantMainNotHas []string
		wantArgs       []any
	}{
		{
			name:           "default (no options) — ORDER BY both, LIMIT/cursor WHERE in CTE",
			opts:           CTEOptions{},
			paginationType: Cursor,
			wantCTEHas:     []string{"order by t.id asc", "limit ?"},
			wantMainHas:    []string{"order by t.id asc"},
			wantArgs:       []any{"active", 11},
		},
		{
			name: "ORDER BY CTE only — no mirror on main",
			opts: CTEOptions{
				OrderBy: CTETargetModeCTE,
			},
			paginationType: Cursor,
			wantCTEHas:     []string{"order by t.id asc", "limit ?"},
			wantMainNotHas: []string{"order by t.id asc"},
			wantArgs:       []any{"active", 11},
		},
		{
			name: "ORDER BY main only — no ORDER BY in CTE",
			opts: CTEOptions{
				OrderBy: CTETargetModeMain,
			},
			paginationType: Cursor,
			wantCTENotHas:  []string{"order by"},
			wantCTEHas:     []string{"limit ?"},
			wantMainHas:    []string{"order by t.id asc"},
			wantArgs:       []any{"active", 11},
		},
		{
			name: "ORDER BY both (explicit)",
			opts: CTEOptions{
				OrderBy: CTETargetModeBoth,
			},
			paginationType: Cursor,
			wantCTEHas:     []string{"order by t.id asc", "limit ?"},
			wantMainHas:    []string{"order by t.id asc"},
			wantArgs:       []any{"active", 11},
		},
		{
			name: "LIMIT main only — no LIMIT in CTE",
			opts: CTEOptions{
				LimitOffset: CTETargetModeMain,
			},
			paginationType: Cursor,
			wantCTENotHas:  []string{"limit ?"},
			wantMainHas:    []string{"limit ?", "order by t.id asc"},
			wantArgs:       []any{"active", 11},
		},
		{
			name: "LIMIT both — LIMIT in CTE and main",
			opts: CTEOptions{
				LimitOffset: CTETargetModeBoth,
			},
			paginationType: Cursor,
			wantCTEHas:     []string{"limit ?"},
			wantMainHas:    []string{"limit ?"},
			wantArgs:       []any{"active", 11, 11},
		},
		{
			name: "cursor WHERE main only — WHERE only on main, not in CTE",
			opts: CTEOptions{
				Where: CTETargetModeMain,
			},
			paginationType: Cursor,
			cursor:         base64Encode(`{"prefix":"next","cols":{"id":"100"}}`),
			wantCTENotHas:  []string{"t.id >"},
			wantMainHas:    []string{"t.id >"},
			wantArgs:       []any{"active", "100", 11},
		},
		{
			name: "cursor WHERE both — WHERE in CTE and main",
			opts: CTEOptions{
				Where: CTETargetModeBoth,
			},
			paginationType: Cursor,
			cursor:         base64Encode(`{"prefix":"next","cols":{"id":"100"}}`),
			wantCTEHas:     []string{"t.id >"},
			wantMainHas:    []string{"t.id >"},
			// placeholder string order: CTE WHERE → CTE LIMIT → main WHERE
			wantArgs: []any{"active", "100", 11, "100"},
		},
		{
			name: "offset pagination LIMIT main only",
			opts: CTEOptions{
				LimitOffset: CTETargetModeMain,
			},
			paginationType: Offset,
			wantCTENotHas:  []string{"limit ?", "offset ?"},
			wantMainHas:    []string{"limit ?", "offset ?"},
			wantArgs:       []any{"active", 10, 0},
		},
		{
			name: "offset pagination LIMIT both",
			opts: CTEOptions{
				LimitOffset: CTETargetModeBoth,
			},
			paginationType: Offset,
			wantCTEHas:     []string{"limit ?", "offset ?"},
			wantMainHas:    []string{"limit ?", "offset ?"},
			// placeholder string order: CTE LIMIT → CTE OFFSET → main LIMIT → main OFFSET
			wantArgs: []any{"active", 10, 0, 10, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := NewQuery(baseQuery, tt.paginationType).
				WithCTETarget("cte", tt.opts).
				WithOrderBy("t.id").
				WithLimit(10).
				WithArgs("active")

			if tt.paginationType == Offset {
				ks = ks.WithOffset(0)
			} else {
				ks = ks.WithCursor(tt.cursor)
			}

			res, err := ks.Build()
			if err != nil {
				t.Fatalf("unexpected build error: %v", err)
			}

			cteBody, mainBody := findBoundary(t, res.Query)

			for _, want := range tt.wantCTEHas {
				if !strings.Contains(cteBody, want) {
					t.Errorf("CTE body should contain %q\n  got: %s", want, cteBody)
				}
			}
			for _, notWant := range tt.wantCTENotHas {
				if strings.Contains(cteBody, notWant) {
					t.Errorf("CTE body should NOT contain %q\n  got: %s", notWant, cteBody)
				}
			}
			for _, want := range tt.wantMainHas {
				if !strings.Contains(mainBody, want) {
					t.Errorf("main body should contain %q\n  got: %s", want, mainBody)
				}
			}
			for _, notWant := range tt.wantMainNotHas {
				if strings.Contains(mainBody, notWant) {
					t.Errorf("main body should NOT contain %q\n  got: %s", notWant, mainBody)
				}
			}

			if len(tt.wantArgs) > 0 {
				if len(res.Args) != len(tt.wantArgs) {
					t.Fatalf("args: expected %v, got %v", tt.wantArgs, res.Args)
				}
				for i, v := range tt.wantArgs {
					if v != res.Args[i] {
						t.Errorf("arg[%d]: expected %v, got %v", i, v, res.Args[i])
					}
				}
			}
		})
	}
}

// TestCTEColumnMapCursorNext verifies that CTEOptions.ColumnMap remaps the
// order-by column inside the CTE body (ORDER BY and cursor WHERE) while the
// main query keeps the original column. Direction, limit, and cursor value
// are unchanged.
func TestCTEColumnMapCursorNext(t *testing.T) {
	// cursor: {"prefix":"next","cols":{"id":"100"}}
	cursor := base64Encode(`{"prefix":"next","cols":{"id":"100"}}`)

	query := `
		WITH filtered_ticket AS (
			SELECT t.id FROM ticket t
			WHERE t.deleted_at = 0 AND t.status = ?
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		ORDER BY t.id
	`

	res, err := NewQuery(query, Cursor).
		WithCTETarget("filtered_ticket", CTEOptions{
			ColumnMap: map[string]string{"t.id": "id"},
		}).
		WithOrderBy("-t.id").
		WithLimit(10).
		WithCursor(cursor).
		WithArgs("active").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := strings.ToLower(res.Query)

	cteClose := strings.Index(q, ") select")
	if cteClose == -1 {
		t.Fatalf("could not find CTE closing in query: %s", q)
	}
	cteBody := q[:cteClose]
	mainBody := q[cteClose:]

	// CTE body must use the mapped column "id" for both WHERE and ORDER BY...
	if !strings.Contains(cteBody, "id < ?") {
		t.Errorf("expected mapped cursor WHERE (id < ?) inside CTE body, got: %s", cteBody)
	}
	if !strings.Contains(cteBody, "order by id desc") {
		t.Errorf("expected mapped ORDER BY (id DESC) inside CTE body, got: %s", cteBody)
	}
	// ...and the injected clauses must NOT use the qualified main column.
	// (The CTE's own "SELECT t.id FROM ticket t" comes from the template and is
	// expected to remain; we only assert the injected WHERE/ORDER BY are remapped.)
	if strings.Contains(cteBody, "t.id <") || strings.Contains(cteBody, "order by t.id") {
		t.Errorf("did not expect injected clauses to use t.id inside CTE body, got: %s", cteBody)
	}

	// Main query must keep the original qualified column.
	if !strings.Contains(mainBody, "order by t.id desc") {
		t.Errorf("expected original ORDER BY (t.id DESC) in main query, got: %s", mainBody)
	}

	// args unchanged by the remap: "active" (user WHERE), cursor "100", limit+1=11
	expected := []any{"active", "100", 11}
	if len(res.Args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(res.Args), res.Args)
	}
	for i, v := range expected {
		if v != res.Args[i] {
			t.Errorf("arg[%d]: expected %v, got %v", i, v, res.Args[i])
		}
	}
}

// TestCTEColumnMapOffset verifies the column remap also applies to the CTE-body
// ORDER BY under offset pagination, leaving the main query column intact.
func TestCTEColumnMapOffset(t *testing.T) {
	query := `
		WITH filtered_ticket AS (
			SELECT t.id FROM ticket t
			WHERE t.deleted_at = 0
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		ORDER BY t.id
	`

	res, err := NewQuery(query, Offset).
		WithCTETarget("filtered_ticket", CTEOptions{
			ColumnMap: map[string]string{"t.id": "id"},
		}).
		WithOrderBy("-t.id").
		WithLimit(10).
		WithOffset(20).
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := strings.ToLower(res.Query)

	cteClose := strings.Index(q, ") select")
	if cteClose == -1 {
		t.Fatalf("could not find CTE closing in query: %s", q)
	}
	cteBody := q[:cteClose]
	mainBody := q[cteClose:]

	if !strings.Contains(cteBody, "order by id desc") {
		t.Errorf("expected mapped ORDER BY (id DESC) inside CTE body, got: %s", cteBody)
	}
	if strings.Contains(cteBody, "order by t.id") {
		t.Errorf("did not expect injected ORDER BY to use t.id inside CTE body, got: %s", cteBody)
	}
	if !strings.Contains(mainBody, "order by t.id desc") {
		t.Errorf("expected original ORDER BY (t.id DESC) in main query, got: %s", mainBody)
	}
}

// TestCTEColumnMapNilUnchanged verifies a nil ColumnMap leaves output identical
// to the equivalent call without any CTEOptions.
func TestCTEColumnMapNilUnchanged(t *testing.T) {
	cursor := base64Encode(`{"prefix":"next","cols":{"id":"100"}}`)
	query := `
		WITH filtered_ticket AS (
			SELECT t.id FROM ticket t
			WHERE t.deleted_at = 0 AND t.status = ?
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		ORDER BY t.id
	`

	build := func(opts ...CTEOptions) *Result {
		res, err := NewQuery(query, Cursor).
			WithCTETarget("filtered_ticket", opts...).
			WithOrderBy("-t.id").
			WithLimit(10).
			WithCursor(cursor).
			WithArgs("active").
			Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return res
	}

	bare := build()
	nilMap := build(CTEOptions{ColumnMap: nil})

	if bare.Query != nilMap.Query {
		t.Errorf("nil ColumnMap changed query:\n bare:   %s\n nilMap: %s", bare.Query, nilMap.Query)
	}
}

// TestCTEUnionDerivedTableCursor proves a single filtered_ticket CTE whose body
// is "SELECT id FROM ( <UNION> ) x" gets the cursor WHERE, ORDER BY, and LIMIT
// injected onto the OUTER select (column "id" via ColumnMap), while the union's
// inner per-branch WHEREs are left untouched and the main query keeps "t.id".
func TestCTEUnionDerivedTableCursor(t *testing.T) {
	cursor := base64Encode(`{"prefix":"next","cols":{"id":"100"}}`)

	query := `
		WITH filtered_ticket AS (
			SELECT id FROM (
				SELECT t.id FROM ticket t WHERE t.deleted_at = 0 AND t.created_by = ?
				UNION DISTINCT
				SELECT t.id FROM ticket t WHERE t.deleted_at = 0 AND t.tracer_id = ?
				UNION DISTINCT
				SELECT t.id FROM ticket_assignee ta JOIN ticket t ON t.id = ta.ticket_id AND t.deleted_at = 0 WHERE ta.deleted_at = 0 AND ta.account_id = ?
			) x
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		ORDER BY t.id
	`

	res, err := NewQuery(query, Cursor).
		WithCTETarget("filtered_ticket", CTEOptions{
			ColumnMap: map[string]string{"t.id": "id"},
		}).
		WithOrderBy("-t.id").
		WithLimit(10).
		WithCursor(cursor).
		WithArgs("ACC1", "ACC1", "ACC1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := strings.ToLower(res.Query)
	t.Logf("built query: %s", q)

	cteClose := strings.LastIndex(q, ") select")
	if cteClose == -1 {
		t.Fatalf("could not find CTE closing in query: %s", q)
	}
	cteBody := q[:cteClose]
	mainBody := q[cteClose:]

	// Outer select of the CTE must carry the injected clauses on column "id".
	if !strings.Contains(cteBody, "id < ?") {
		t.Errorf("expected injected cursor WHERE (id < ?) on the CTE outer select, got: %s", cteBody)
	}
	if !strings.Contains(cteBody, "order by id desc") {
		t.Errorf("expected injected ORDER BY (id DESC) on the CTE outer select, got: %s", cteBody)
	}
	if !strings.Contains(cteBody, "limit ?") {
		t.Errorf("expected injected LIMIT on the CTE outer select, got: %s", cteBody)
	}
	// The union's inner branch WHEREs must be untouched (all three still present).
	for _, frag := range []string{"t.created_by = ?", "t.tracer_id = ?", "ta.account_id = ?"} {
		if !strings.Contains(cteBody, frag) {
			t.Errorf("expected union branch %q to be preserved, got: %s", frag, cteBody)
		}
	}
	// Injected clauses must NOT use the qualified t.id (only the union branches may mention t.*).
	if strings.Contains(cteBody, "t.id < ?") || strings.Contains(cteBody, "order by t.id") {
		t.Errorf("injected clauses must use bare id, not t.id, got: %s", cteBody)
	}
	// Main query keeps the original qualified column.
	if !strings.Contains(mainBody, "order by t.id desc") {
		t.Errorf("expected main ORDER BY t.id DESC, got: %s", mainBody)
	}

	// args: 3 union account args (earliest in string), cursor "100", limit+1=11
	expected := []any{"ACC1", "ACC1", "ACC1", "100", 11}
	if len(res.Args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(res.Args), res.Args)
	}
	for i, v := range expected {
		if v != res.Args[i] {
			t.Errorf("arg[%d]: expected %v, got %v", i, v, res.Args[i])
		}
	}
}

// TestCTETargetUnionOwned verifies that targeting a CTE whose body is a raw
// top-level UNION (the "owned" CTE) auto-wraps the union in a derived table and
// injects the cursor WHERE, ORDER BY, and LIMIT onto the wrapper using the
// mapped column ("id"), leaving every union branch intact.
func TestCTETargetUnionOwned(t *testing.T) {
	cursor := base64Encode(`{"prefix":"next","cols":{"id":"100"}}`)

	query := `
		WITH owned AS (
			SELECT t.id AS id FROM ticket t WHERE t.deleted_at = 0 AND t.created_by = ?
			UNION DISTINCT
			SELECT t.id FROM ticket t WHERE t.deleted_at = 0 AND t.tracer_id = ?
			UNION DISTINCT
			SELECT ta.ticket_id FROM ticket_assignee ta WHERE ta.deleted_at = 0 AND ta.account_id = ?
		),
		filtered_ticket AS (
			SELECT t.id FROM ticket t INNER JOIN owned o ON t.id = o.id WHERE t.deleted_at = 0
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		ORDER BY t.id
	`

	res, err := NewQuery(query, Cursor).
		WithCTETarget("owned", CTEOptions{
			ColumnMap: map[string]string{"t.id": "id"},
		}).
		WithOrderBy("-t.id").
		WithLimit(10).
		WithCursor(cursor).
		WithArgs("ACC1", "ACC1", "ACC1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := strings.ToLower(res.Query)
	t.Logf("built query: %s", q)

	// The union must have been wrapped in a derived table.
	if !strings.Contains(q, "kuysor_cte_union") {
		t.Errorf("expected union CTE body to be wrapped in a derived table, got: %s", q)
	}

	// Isolate the owned CTE body (from "owned as (" up to ", filtered_ticket as").
	ownedStart := strings.Index(q, "owned as (")
	ftStart := strings.Index(q, "filtered_ticket as (")
	if ownedStart == -1 || ftStart == -1 || ftStart <= ownedStart {
		t.Fatalf("could not isolate owned CTE body: %s", q)
	}
	owned := q[ownedStart:ftStart]

	// Injected clauses on the wrapper, using the mapped column "id".
	if !strings.Contains(owned, "id < ?") {
		t.Errorf("expected cursor WHERE (id < ?) on the union wrapper, got: %s", owned)
	}
	if !strings.Contains(owned, "order by id desc") {
		t.Errorf("expected ORDER BY (id DESC) on the union wrapper, got: %s", owned)
	}
	if !strings.Contains(owned, "limit ?") {
		t.Errorf("expected LIMIT on the union wrapper, got: %s", owned)
	}
	// All three union branches preserved.
	for _, frag := range []string{"t.created_by = ?", "t.tracer_id = ?", "ta.account_id = ?"} {
		if !strings.Contains(owned, frag) {
			t.Errorf("expected union branch %q preserved, got: %s", frag, owned)
		}
	}
	// Injected clauses must not use the qualified t.id.
	if strings.Contains(owned, "t.id < ?") || strings.Contains(owned, "order by t.id") {
		t.Errorf("injected clauses must use bare id, not t.id, got: %s", owned)
	}

	// args: 3 union account args, cursor "100", limit+1=11
	expected := []any{"ACC1", "ACC1", "ACC1", "100", 11}
	if len(res.Args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(res.Args), res.Args)
	}
	for i, v := range expected {
		if v != res.Args[i] {
			t.Errorf("arg[%d]: expected %v, got %v", i, v, res.Args[i])
		}
	}
}

// TestCTEDualTargetOwnedAndFiltered verifies that a secondary CTE target (owned,
// a UNION) is capped IN ADDITION TO the primary CTE target (filtered_ticket):
// both bodies receive the cursor WHERE + ORDER BY + LIMIT, owned using the mapped
// "id" column on its wrapped union and filtered_ticket using t.id as today, with
// args ordered by query-string position (union args, owned cursor+limit, then
// filtered cursor+limit).
func TestCTEDualTargetOwnedAndFiltered(t *testing.T) {
	cursor := base64Encode(`{"prefix":"next","cols":{"id":"100"}}`)

	query := `
		WITH owned AS (
			SELECT t.id AS id FROM ticket t WHERE t.deleted_at = 0 AND t.created_by = ?
			UNION DISTINCT
			SELECT t.id FROM ticket t WHERE t.deleted_at = 0 AND t.tracer_id = ?
			UNION DISTINCT
			SELECT ta.ticket_id FROM ticket_assignee ta WHERE ta.deleted_at = 0 AND ta.account_id = ?
		),
		filtered_ticket AS (
			SELECT t.id FROM ticket t INNER JOIN owned o ON t.id = o.id WHERE t.deleted_at = 0
		)
		SELECT t.id, t.code
		FROM filtered_ticket ft
		JOIN ticket t ON t.id = ft.id
		ORDER BY t.id
	`

	res, err := NewQuery(query, Cursor).
		WithCTETarget("filtered_ticket").
		WithCTESecondaryTarget("owned", CTEOptions{ColumnMap: map[string]string{"t.id": "id"}}).
		WithOrderBy("-t.id").
		WithLimit(10).
		WithCursor(cursor).
		WithArgs("ACC1", "ACC1", "ACC1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := strings.ToLower(res.Query)
	t.Logf("built query: %s", q)

	ownedStart := strings.Index(q, "owned as (")
	ftStart := strings.Index(q, "filtered_ticket as (")
	mainStart := strings.LastIndex(q, ") select")
	if ownedStart == -1 || ftStart == -1 || mainStart == -1 || !(ownedStart < ftStart && ftStart < mainStart) {
		t.Fatalf("could not isolate CTE bodies: %s", q)
	}
	owned := q[ownedStart:ftStart]
	filtered := q[ftStart:mainStart]
	main := q[mainStart:]

	// owned: union wrapped, injected clauses on mapped column "id"
	if !strings.Contains(owned, "kuysor_cte_union") {
		t.Errorf("expected owned union to be wrapped, got: %s", owned)
	}
	if !strings.Contains(owned, "id < ?") || !strings.Contains(owned, "order by id desc") || !strings.Contains(owned, "limit ?") {
		t.Errorf("expected owned to get id-based cursor WHERE + ORDER BY + LIMIT, got: %s", owned)
	}
	if strings.Contains(owned, "t.id < ?") || strings.Contains(owned, "order by t.id") {
		t.Errorf("owned injected clauses must use bare id, got: %s", owned)
	}

	// filtered_ticket: unchanged behavior, on t.id
	if !strings.Contains(filtered, "t.id < ?") || !strings.Contains(filtered, "order by t.id desc") || !strings.Contains(filtered, "limit ?") {
		t.Errorf("expected filtered_ticket to keep t.id cursor WHERE + ORDER BY + LIMIT, got: %s", filtered)
	}

	// main keeps mirrored t.id order
	if !strings.Contains(main, "order by t.id desc") {
		t.Errorf("expected main ORDER BY t.id DESC, got: %s", main)
	}

	// args: union(ACC×3), owned cursor 100 + limit 11, filtered cursor 100 + limit 11
	expected := []any{"ACC1", "ACC1", "ACC1", "100", 11, "100", 11}
	if len(res.Args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(res.Args), res.Args)
	}
	for i, v := range expected {
		if v != res.Args[i] {
			t.Errorf("arg[%d]: expected %v, got %v", i, v, res.Args[i])
		}
	}
}

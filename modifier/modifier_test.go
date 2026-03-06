package modifier

import (
	"strings"
	"testing"
)

func TestSQLModifier(t *testing.T) {

	var testCases = []struct {
		in      string
		out     string
		where   SQLCondition
		limit   string
		orderBy []string
	}{
		// SIMPLE QUERY
		{
			in:      "SELECT * FROM table",
			where:   NewCondition("id = 1"),
			orderBy: []string{"id ASC"},
			limit:   "10",
			out:     "SELECT * FROM table WHERE id = 1 ORDER BY id ASC LIMIT 10",
		},

		// SIMPLE QUERY WITH EXISTING WHERE CLAUSE
		{
			in:      "SELECT * FROM table WHERE id = 1",
			where:   NewCondition("name = 'John'"),
			orderBy: []string{"id DESC"},
			limit:   "?",
			out:     "SELECT * FROM table WHERE id = 1 AND name = 'John' ORDER BY id DESC LIMIT ?",
		},
		{
			in:      "SELECT * FROM table WHERE id = 1",
			where:   NewNestedCondition("AND", NewCondition("name = 'John'"), NewCondition("age = 20")),
			orderBy: []string{"id DESC", "name ASC", "age DESC"},
			limit:   "$0",
			out:     "SELECT * FROM table WHERE id = 1 AND (name = 'John' AND age = 20) ORDER BY id DESC, name ASC, age DESC LIMIT $0",
		},
		{
			in:    "SELECT * FROM table WHERE id = 1",
			where: NewNestedCondition("OR", NewNestedCondition("AND", NewCondition("name = 'John'"), NewCondition("age = 20")), NewCondition("status = 'active'")),
			out:   "SELECT * FROM table WHERE id = 1 AND ((name = 'John' AND age = 20) OR status = 'active')",
		},
		{
			in:    "SELECT * FROM table WHERE id = 1 OR name = 'John'",
			where: NewNestedCondition("AND", NewCondition("age = 20"), NewCondition("status = 'active'")),
			out:   "SELECT * FROM table WHERE (id = 1 OR name = 'John') AND (age = 20 AND status = 'active')",
		},
		// WITH ALIAS TABLE
		{
			in:      "SELECT * FROM table t WHERE t.id = 1",
			where:   NewCondition("t.name = 'John'"),
			orderBy: []string{"t.id DESC"},
			limit:   "10",
			out:     "SELECT * FROM table t WHERE t.id = 1 AND t.name = 'John' ORDER BY t.id DESC LIMIT 10",
		},
		{
			in:      "SELECT * FROM table as t WHERE t.id = 1",
			where:   NewCondition("t.name = 'John'"),
			orderBy: []string{"t.id DESC"},
			limit:   "10",
			out:     "SELECT * FROM table as t WHERE t.id = 1 AND t.name = 'John' ORDER BY t.id DESC LIMIT 10",
		},
		{
			in:    "SELECT * FROM (SELECT * FROM table WHERE id = 1) as t",
			where: NewCondition("t.name = 'John'"),
			out:   "SELECT * FROM (SELECT * FROM table WHERE id = 1) as t WHERE t.name = 'John'",
		},
		// JOIN QUERY
		{
			in:      "SELECT u.id, u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id",
			where:   NewCondition("u.id = 1"),
			orderBy: []string{"u.id DESC, o.total ASC"},
			limit:   "?",
			out:     "SELECT u.id, u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id WHERE u.id = 1 ORDER BY u.id DESC, o.total ASC LIMIT ?",
		},
		{
			in:    "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id",
			where: NewCondition("a.id = 1"),
			out:   "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1",
		},
		{
			in:    "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1",
			where: NewCondition("b.id = 2"),
			out:   "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 AND b.id = 2",
		},
		{
			in:      "SELECT a.* FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1",
			where:   NewCondition("b.id = 2"),
			orderBy: []string{"a.id DESC"},
			limit:   "10",
			out:     "SELECT a.* FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 AND b.id = 2 ORDER BY a.id DESC LIMIT 10",
		},
		{
			in:      "SELECT u.id, u.name, o.total, p.name AS product_name FROM users u JOIN orders o ON u.id = o.user_id JOIN products p ON o.product_id = p.id",
			where:   NewCondition("u.id = 1"),
			orderBy: []string{"u.id DESC, o.total ASC"},
			limit:   "?",
			out:     "SELECT u.id, u.name, o.total, p.name AS product_name FROM users u JOIN orders o ON u.id = o.user_id JOIN products p ON o.product_id = p.id WHERE u.id = 1 ORDER BY u.id DESC, o.total ASC LIMIT ?",
		},
		// FULL OUTER JOIN
		{
			in:      "SELECT * FROM employees e FULL OUTER JOIN contractors c ON e.id = c.employee_id",
			where:   NewCondition("e.id = 1"),
			orderBy: []string{"e.id DESC"},
			limit:   "10",
			out:     "SELECT * FROM employees e FULL OUTER JOIN contractors c ON e.id = c.employee_id WHERE e.id = 1 ORDER BY e.id DESC LIMIT 10",
		},
		// LATERAL JOIN
		{
			in: `
			SELECT c.id, c.name, o.total_orders
			FROM customers c
			CROSS JOIN LATERAL (
				SELECT COUNT(*) AS total_orders FROM orders o WHERE o.customer_id = c.id
			) o
			`,
			where:   NewCondition("c.id = 1"),
			orderBy: []string{"c.id DESC", "o.total_orders ASC"},
			limit:   "10",
			out:     `SELECT c.id, c.name, o.total_orders FROM customers c CROSS JOIN LATERAL ( SELECT COUNT(*) AS total_orders FROM orders o WHERE o.customer_id = c.id ) o WHERE c.id = 1 ORDER BY c.id DESC, o.total_orders ASC LIMIT 10`,
		},
		// GROUP BY
		{
			in:      "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 GROUP BY a.id",
			where:   NewCondition("b.id = 2"),
			orderBy: []string{"a.id IS NULL"},
			limit:   "10",
			out:     "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 AND b.id = 2 GROUP BY a.id ORDER BY a.id IS NULL LIMIT 10",
		},
		// HAVING
		{
			in:      "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 GROUP BY a.id HAVING COUNT(b.id) > 1",
			where:   NewCondition("b.id = 2"),
			orderBy: []string{"a.id DESC", "b.id ASC"},
			limit:   "10",
			out:     "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 AND b.id = 2 GROUP BY a.id HAVING COUNT(b.id) > 1 ORDER BY a.id DESC, b.id ASC LIMIT 10",
		},
		// SUBQUERY WHERE
		{
			in:      "SELECT * FROM customers WHERE id IN (SELECT customer_id FROM orders)",
			where:   NewCondition("name = 'John'"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     "SELECT * FROM customers WHERE id IN (SELECT customer_id FROM orders) AND name = 'John' ORDER BY id DESC LIMIT 10",
		},
		{
			in:      "SELECT * FROM customers WHERE id IN (SELECT customer_id FROM orders WHERE status = 'active')",
			where:   NewCondition("name = 'John'"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     "SELECT * FROM customers WHERE id IN (SELECT customer_id FROM orders WHERE status = 'active') AND name = 'John' ORDER BY id DESC LIMIT 10",
		},
		{
			in:      "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id IN (SELECT id FROM table3 WHERE status = ?)",
			where:   NewCondition("b.id = ?"),
			orderBy: []string{"a.id DESC", "b.id ASC"},
			limit:   "10",
			out:     "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id IN (SELECT id FROM table3 WHERE status = ?) AND b.id = ? ORDER BY a.id DESC, b.id ASC LIMIT 10",
		},
		// SUBQUERY JOIN
		{
			in:      "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN (SELECT * FROM table2 WHERE id = 1) b ON a.id = b.id WHERE a.id = 1",
			where:   NewCondition("b.id = 2"),
			orderBy: []string{"a.id DESC", "b.id ASC"},
			limit:   "10",
			out:     "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN (SELECT * FROM table2 WHERE id = 1) b ON a.id = b.id WHERE a.id = 1 AND b.id = 2 ORDER BY a.id DESC, b.id ASC LIMIT 10",
		},
		{
			in:      "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN (SELECT * FROM table2 WHERE id = 1) b ON a.id = b.id",
			where:   NewCondition("a.id = 1"),
			orderBy: []string{"a.id DESC", "b.id ASC"},
			limit:   "10",
			out:     "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN (SELECT * FROM table2 WHERE id = 1) b ON a.id = b.id WHERE a.id = 1 ORDER BY a.id DESC, b.id ASC LIMIT 10",
		},
		// NESTED SUBQUERY
		{
			in:      "SELECT * FROM users WHERE id IN (SELECT user_id FROM (SELECT user_id FROM orders WHERE amount > 100) AS subquery)",
			where:   NewCondition("name = $1"),
			orderBy: []string{"id DESC"},
			limit:   "$2",
			out:     "SELECT * FROM users WHERE id IN (SELECT user_id FROM (SELECT user_id FROM orders WHERE amount > 100) AS subquery) AND name = $1 ORDER BY id DESC LIMIT $2",
		},
		// CASE WHEN
		{
			in:    "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE CASE WHEN a.id = 1 THEN 1 ELSE 0 END = 1",
			where: NewCondition("b.id = 2"),
			out:   "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE CASE WHEN a.id = 1 THEN 1 ELSE 0 END = 1 AND b.id = 2",
		},
		{
			in:    `SELECT id, name, CASE WHEN salary > 5000 THEN 'High' ELSE 'Low' END AS salary_level FROM employees`,
			where: NewCondition("id = 1"),
			out:   `SELECT id, name, CASE WHEN salary > 5000 THEN 'High' ELSE 'Low' END AS salary_level FROM employees WHERE id = 1`,
		},
		// FOR UPDATE
		{
			in:      "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 FOR UPDATE",
			where:   NewCondition("b.id = 2"),
			orderBy: []string{"a.id DESC", "b.id ASC"},
			limit:   "10",
			out:     "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 AND b.id = 2 ORDER BY a.id DESC, b.id ASC LIMIT 10 FOR UPDATE",
		},
		// FOR SHARE
		{
			in:      "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 FOR SHARE",
			where:   NewCondition("b.id = 2"),
			orderBy: []string{"a.id DESC", "b.id ASC"},
			limit:   "10",
			out:     "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 AND b.id = 2 ORDER BY a.id DESC, b.id ASC LIMIT 10 FOR SHARE",
		},
		// LOCK IN SHARE MODE
		{
			in:      "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 LOCK IN SHARE MODE",
			where:   NewCondition("b.id = 2"),
			orderBy: []string{"a.id DESC", "b.id ASC"},
			limit:   "10",
			out:     "SELECT a.id, a.name, b.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 AND b.id = 2 ORDER BY a.id DESC, b.id ASC LIMIT 10 LOCK IN SHARE MODE",
		},
		// SUBQUERY IN SELECT
		{
			in:    "SELECT a.id, a.name, b.id, b.name, (SELECT COUNT(*) FROM table3 WHERE id = 1) as count FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1",
			where: NewCondition("b.id = 2"),
			out:   "SELECT a.id, a.name, b.id, b.name, (SELECT COUNT(*) FROM table3 WHERE id = 1) as count FROM table1 a JOIN table2 b ON a.id = b.id WHERE a.id = 1 AND b.id = 2",
		},
		// CTE
		{
			in:      "WITH recent_orders AS (SELECT * FROM orders WHERE created_at > NOW() - INTERVAL '30 days') SELECT * FROM recent_orders",
			where:   NewCondition("id = 1"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     "WITH recent_orders AS (SELECT * FROM orders WHERE created_at > NOW() - INTERVAL '30 days') SELECT * FROM recent_orders WHERE id = 1 ORDER BY id DESC LIMIT 10",
		},
		{
			in: `
			WITH RECURSIVE employee_tree AS (
				SELECT id, name, manager_id FROM employees WHERE manager_id IS NULL
				UNION ALL
				SELECT e.id, e.name, e.manager_id FROM employees e
				INNER JOIN employee_tree et ON e.manager_id = et.id
			)
			SELECT * FROM employee_tree
			`,
			where:   NewCondition("id = 1"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     `WITH RECURSIVE employee_tree AS ( SELECT id, name, manager_id FROM employees WHERE manager_id IS NULL UNION ALL SELECT e.id, e.name, e.manager_id FROM employees e INNER JOIN employee_tree et ON e.manager_id = et.id ) SELECT * FROM employee_tree WHERE id = 1 ORDER BY id DESC LIMIT 10`,
		},
		{
			in: `
			WITH recent_sales AS (
				SELECT user_id, SUM(amount) AS total FROM sales WHERE sale_date > NOW() - INTERVAL '30 days' GROUP BY user_id
			),
			high_spenders AS (
				SELECT user_id FROM recent_sales WHERE total > 1000
			)
			SELECT u.id, u.name, hs.user_id FROM users u LEFT JOIN high_spenders hs ON u.id = hs.user_id
			`,
			where:   NewCondition("u.id = 1"),
			orderBy: []string{"u.id DESC"},
			limit:   "10",
			out:     `WITH recent_sales AS ( SELECT user_id, SUM(amount) AS total FROM sales WHERE sale_date > NOW() - INTERVAL '30 days' GROUP BY user_id ), high_spenders AS ( SELECT user_id FROM recent_sales WHERE total > 1000 ) SELECT u.id, u.name, hs.user_id FROM users u LEFT JOIN high_spenders hs ON u.id = hs.user_id WHERE u.id = 1 ORDER BY u.id DESC LIMIT 10`,
		},
		// EXIST
		{
			in:      "SELECT * FROM customers WHERE EXISTS (SELECT 1 FROM orders WHERE orders.customer_id = customers.id)",
			where:   NewCondition("name = 'John'"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     "SELECT * FROM customers WHERE EXISTS (SELECT 1 FROM orders WHERE orders.customer_id = customers.id) AND name = 'John' ORDER BY id DESC LIMIT 10",
		},
		// JSON FUNCTIONS
		{
			in:      "SELECT * FROM settings WHERE data->>'theme' = 'dark'",
			where:   NewCondition("name = 'John'"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     "SELECT * FROM settings WHERE data->>'theme' = 'dark' AND name = 'John' ORDER BY id DESC LIMIT 10",
		},
		// DISTINCT
		{
			in:      "SELECT DISTINCT city FROM customers",
			where:   NewCondition("name = 'John'"),
			orderBy: []string{"city ASC"},
			limit:   "10",
			out:     "SELECT DISTINCT city FROM customers WHERE name = 'John' ORDER BY city ASC LIMIT 10",
		},
		// WINDOW FUNCTION
		{
			in:      "SELECT id, name, RANK() OVER(PARTITION BY department ORDER BY salary DESC) AS rank FROM employees",
			where:   NewCondition("id = 1"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     "SELECT id, name, RANK() OVER(PARTITION BY department ORDER BY salary DESC) AS rank FROM employees WHERE id = 1 ORDER BY id DESC LIMIT 10",
		},
		// UNFORMATTED QUERY
		{
			in: `
				SELECT 
				id,
				name
				FROM employees
			`,
			where:   NewCondition("id = 1"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     `SELECT id, name FROM employees WHERE id = 1 ORDER BY id DESC LIMIT 10`,
		},
		{
			in: `
				       SELECT
				  id,
				       name
				  FROM     employees
				WHERE      id =  ?
			`,
			where:   NewCondition("name = ?"),
			orderBy: []string{"id DESC"},
			limit:   "10",
			out:     `SELECT id, name FROM employees WHERE id = ? AND name = ? ORDER BY id DESC LIMIT 10`,
		},
		{
			in:      "SELECT e.`id` FROM employees e",
			where:   NewCondition("`name` = 'John'"),
			orderBy: []string{"e.`id` DESC"},
			limit:   "10",
			out:     "SELECT e.`id` FROM employees e WHERE `name` = 'John' ORDER BY e.`id` DESC LIMIT 10",
		},
	}

	for _, tc := range testCases {

		p := NewSQLModifier(tc.in)

		// append where
		if err := p.AppendWhere(tc.where.Expression); err != nil {
			t.Error(err)
		}

		// set order by
		if len(tc.orderBy) > 0 {
			if err := p.SetOrderBy(tc.orderBy...); err != nil {
				t.Error(err)
			}
		}

		// set limit
		if tc.limit != "" {
			if err := p.SetLimit(tc.limit); err != nil {
				t.Error(err)
			}
		}

		// build the query
		s, err := p.Build()
		if err != nil {
			t.Error(err)
		}

		if !strings.EqualFold(s, tc.out) {
			t.Errorf("expected %s, got %s", strings.ToLower(tc.out), strings.ToLower(s))
		}
	}

}

func TestFindCTEBodyBounds(t *testing.T) {
	testCases := []struct {
		name      string
		query     string
		cteName   string
		wantBody  string // expected content between the CTE parens
		wantError bool
	}{
		{
			name:     "single CTE",
			query:    "WITH foo AS (SELECT id FROM t WHERE x = 1) SELECT * FROM foo",
			cteName:  "foo",
			wantBody: "SELECT id FROM t WHERE x = 1",
		},
		{
			name:     "single CTE case-insensitive",
			query:    "WITH Foo AS (SELECT id FROM t) SELECT * FROM Foo",
			cteName:  "foo",
			wantBody: "SELECT id FROM t",
		},
		{
			name: "multiple CTEs - target first",
			query: `WITH cte1 AS (SELECT id FROM t1),
					cte2 AS (SELECT id FROM t2)
					SELECT * FROM cte1 JOIN cte2 ON cte1.id = cte2.id`,
			cteName:  "cte1",
			wantBody: "SELECT id FROM t1",
		},
		{
			name: "multiple CTEs - target second",
			query: `WITH cte1 AS (SELECT id FROM t1),
					cte2 AS (SELECT id FROM t2)
					SELECT * FROM cte1 JOIN cte2 ON cte1.id = cte2.id`,
			cteName:  "cte2",
			wantBody: "SELECT id FROM t2",
		},
		{
			name: "nested subquery inside CTE body",
			query: `WITH filtered AS (
						SELECT id FROM t
						JOIN (SELECT max_id FROM meta) m ON m.max_id = t.id
						WHERE t.status = 'active'
					)
					SELECT * FROM filtered`,
			cteName: "filtered",
			// wantBody is compared after normalizing whitespace
			wantBody: "SELECT id FROM t JOIN (SELECT max_id FROM meta) m ON m.max_id = t.id WHERE t.status = 'active'",
		},
		{
			name:      "CTE not found",
			query:     "WITH foo AS (SELECT id FROM t) SELECT * FROM foo",
			cteName:   "bar",
			wantError: true,
		},
		{
			name:      "no CTE at all",
			query:     "SELECT * FROM t WHERE id = 1",
			cteName:   "foo",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewSQLModifier(tc.query)
			start, end, err := m.findCTEBodyBounds(tc.cteName)
			if tc.wantError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// normalize whitespace for comparison
			normalizeWS := func(s string) string {
				return strings.Join(strings.Fields(s), " ")
			}
			got := normalizeWS(tc.query[start:end])
			want := normalizeWS(tc.wantBody)
			if !strings.EqualFold(got, want) {
				t.Errorf("body mismatch\nwant: %q\n got: %q", want, got)
			}
		})
	}
}

func TestCTETargetModifier(t *testing.T) {
	testCases := []struct {
		name    string
		query   string
		cte     string
		where   string
		orderBy []string
		limit   string
		out     string
	}{
		{
			name:    "append where into CTE body, no existing where in CTE",
			query:   "WITH ft AS (SELECT id FROM ticket t) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id",
			cte:     "ft",
			where:   "t.id > $0",
			out:     "WITH ft AS (SELECT id FROM ticket t WHERE t.id > $0) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id",
		},
		{
			name:    "append where into CTE body with existing where",
			query:   "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0) SELECT * FROM ft JOIN ticket t ON t.id = ft.id",
			cte:     "ft",
			where:   "(t.id > $0)",
			out:     "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0 AND (t.id > $0)) SELECT * FROM ft JOIN ticket t ON t.id = ft.id",
		},
		{
			name:    "set order by inside CTE body, main order by untouched",
			query:   "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id",
			cte:     "ft",
			orderBy: []string{"t.id ASC"},
			out:     "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0 ORDER BY t.id ASC) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id",
		},
		{
			name:    "set limit inside CTE body, no limit in main query",
			query:   "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id",
			cte:     "ft",
			limit:   "$0",
			out:     "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0 LIMIT $0) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id",
		},
		{
			name:    "set limit replaces existing limit inside CTE",
			query:   "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0 ORDER BY t.id LIMIT 10) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id",
			cte:     "ft",
			limit:   "$0",
			out:     "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0 ORDER BY t.id LIMIT $0) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id",
		},
		{
			name:    "all three: where + order by + limit inside CTE, main query untouched",
			query:   "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0) SELECT * FROM ft JOIN ticket t ON t.id = ft.id GROUP BY t.id ORDER BY t.id",
			cte:     "ft",
			where:   "(t.id > $0)",
			orderBy: []string{"t.id ASC"},
			limit:   "$0",
			out:     "WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0 AND (t.id > $0) ORDER BY t.id ASC LIMIT $0) SELECT * FROM ft JOIN ticket t ON t.id = ft.id GROUP BY t.id ORDER BY t.id",
		},
		{
			name: "multi-CTE: target only the specified CTE",
			query: `WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0),
					other AS (SELECT id FROM other_table)
					SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id`,
			cte:     "ft",
			where:   "(t.id > $0)",
			orderBy: []string{"t.id ASC"},
			limit:   "$0",
			// Build() normalizes whitespace across the whole query
			out: `WITH ft AS (SELECT id FROM ticket t WHERE t.deleted_at = 0 AND (t.id > $0) ORDER BY t.id ASC LIMIT $0), other AS (SELECT id FROM other_table) SELECT * FROM ft JOIN ticket t ON t.id = ft.id ORDER BY t.id`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewSQLModifier(tc.query)
			m.SetCTETarget(tc.cte)

			if tc.where != "" {
				if err := m.AppendWhere(tc.where); err != nil {
					t.Fatalf("AppendWhere error: %v", err)
				}
			}
			if len(tc.orderBy) > 0 {
				if err := m.SetOrderBy(tc.orderBy...); err != nil {
					t.Fatalf("SetOrderBy error: %v", err)
				}
			}
			if tc.limit != "" {
				if err := m.SetLimit(tc.limit); err != nil {
					t.Fatalf("SetLimit error: %v", err)
				}
			}

			got, err := m.Build()
			if err != nil {
				t.Fatalf("Build error: %v", err)
			}

			if !strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(tc.out)) {
				t.Errorf("output mismatch\nwant: %s\n got: %s", tc.out, got)
			}
		})
	}
}

func TestCTETargetNotFound(t *testing.T) {
	m := NewSQLModifier("WITH foo AS (SELECT id FROM t) SELECT * FROM foo")
	m.SetCTETarget("nonexistent")

	err := m.AppendWhere("id = 1")
	if err == nil {
		t.Error("expected error when CTE not found, got nil")
	}
}

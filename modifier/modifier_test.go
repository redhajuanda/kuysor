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
		p.AppendWhere(tc.where.Expression)

		// set order by
		if len(tc.orderBy) > 0 {
			p.SetOrderBy(tc.orderBy...)
		}

		// set limit
		if tc.limit != "" {
			p.SetLimit(tc.limit)
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

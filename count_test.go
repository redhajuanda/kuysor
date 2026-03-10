package kuysor

import (
	"strings"
	"testing"
)

func TestNewCount(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		useColumn string
		expected  string
		expectErr bool
	}{
		// Simple select
		{
			name:     "simple select with count star",
			query:    "SELECT id, code, name FROM table",
			expected: "select count(*) from table",
		},
		{
			name:      "simple select with count one",
			query:     "SELECT id, code, name FROM table",
			useColumn: "1",
			expected:  "select count(1) from table",
		},
		{
			name:      "simple select with count id",
			query:     "SELECT id, code, name FROM table",
			useColumn: "id",
			expected:  "select count(id) from table",
		},
		{
			name:      "simple select with count qualified column",
			query:     "SELECT t.id, t.code FROM table t",
			useColumn: "t.id",
			expected:  "select count(t.id) from table t",
		},
		{
			name:     "select star with default count",
			query:    "SELECT * FROM users",
			expected: "select count(*) from users",
		},
		// WHERE
		{
			name:     "where clause preserved",
			query:    "SELECT id, name FROM users WHERE status = ?",
			expected: "select count(*) from users where status = ?",
		},
		{
			name:     "complex where clause preserved",
			query:    "SELECT id, name FROM users WHERE status = ? AND created_at > ? AND deleted_at IS NULL",
			expected: "select count(*) from users where status = ? and created_at > ? and deleted_at is null",
		},
		// JOIN
		{
			name:     "inner join preserved",
			query:    "SELECT u.id, u.name, p.title FROM users u INNER JOIN profiles p ON u.id = p.user_id",
			expected: "select count(*) from users u inner join profiles p on u.id = p.user_id",
		},
		{
			name:     "left join preserved",
			query:    "SELECT u.id, u.name FROM users u LEFT JOIN profiles p ON u.id = p.user_id WHERE u.status = 'active'",
			expected: "select count(*) from users u left join profiles p on u.id = p.user_id where u.status = 'active'",
		},
		{
			name:     "multiple joins preserved",
			query:    "SELECT u.id, u.name, p.title, c.name FROM users u LEFT JOIN profiles p ON u.id = p.user_id LEFT JOIN companies c ON u.company_id = c.id",
			expected: "select count(*) from users u left join profiles p on u.id = p.user_id left join companies c on u.company_id = c.id",
		},
		// CTE
		{
			name:     "CTE preserved main select replaced",
			query:    "WITH filtered AS ( SELECT id FROM users WHERE status = ? ) SELECT f.id, f.name FROM filtered f JOIN users u ON u.id = f.id",
			expected: "with filtered as ( select id from users where status = ? ) select count(*) from filtered f join users u on u.id = f.id",
		},
		{
			name:     "multiple CTEs preserved",
			query:    "WITH cte1 AS (SELECT id FROM t1), cte2 AS (SELECT id FROM t2) SELECT c1.id, c2.id FROM cte1 c1 JOIN cte2 c2 ON c1.id = c2.id",
			expected: "with cte1 as (select id from t1), cte2 as (select id from t2) select count(*) from cte1 c1 join cte2 c2 on c1.id = c2.id",
		},
		// Subquery
		{
			name:     "subquery in FROM preserved",
			query:    "SELECT id, name FROM (SELECT id, name FROM users WHERE active = 1) AS sub",
			expected: "select count(*) from (select id, name from users where active = 1) as sub",
		},
		{
			name:     "subquery in WHERE preserved",
			query:    "SELECT id, name FROM users WHERE id IN (SELECT user_id FROM orders WHERE total > 100)",
			expected: "select count(*) from users where id in (select user_id from orders where total > 100)",
		},
		{
			name:     "scalar subquery in select list - main select replaced",
			query:    "SELECT id, (SELECT COUNT(*) FROM orders WHERE user_id = users.id) AS order_count FROM users",
			expected: "select count(*) from users",
		},
		// GROUP BY
		{
			name:     "group by preserved",
			query:    "SELECT category, COUNT(*) FROM products GROUP BY category",
			expected: "select count(*) from products group by category",
		},
		{
			name:     "group by and having preserved",
			query:    "SELECT category FROM products GROUP BY category HAVING COUNT(*) > 5",
			expected: "select count(*) from products group by category having count(*) > 5",
		},
		// ORDER BY and LIMIT
		{
			name:     "order by and limit preserved",
			query:    "SELECT id, name FROM users WHERE status = ? ORDER BY name LIMIT 10",
			expected: "select count(*) from users where status = ? order by name limit 10",
		},
		{
			name:     "order by and limit offset preserved",
			query:    "SELECT id FROM users ORDER BY id LIMIT 10 OFFSET 20",
			expected: "select count(*) from users order by id limit 10 offset 20",
		},
		// DISTINCT
		{
			name:     "distinct replaced with count",
			query:    "SELECT DISTINCT id, name FROM users",
			expected: "select count(*) from users",
		},
		// Placeholders
		{
			name:     "question placeholder",
			query:    "SELECT id FROM users WHERE status = ? AND id > ?",
			expected: "select count(*) from users where status = ? and id > ?",
		},
		{
			name:     "named placeholder",
			query:    "SELECT id FROM users WHERE status = :status AND id > :id",
			expected: "select count(*) from users where status = :status and id > :id",
		},
		// Quoted identifiers
		{
			name:     "backtick identifiers",
			query:    "SELECT `id`, `name` FROM `users` WHERE `status` = ?",
			expected: "select count(*) from `users` where `status` = ?",
		},
		// MariaDB/MySQL partition
		{
			name:     "partition - single partition",
			query:    "SELECT id, name FROM orders PARTITION (p3) WHERE user_id = ?",
			expected: "select count(*) from orders partition (p3) where user_id = ?",
		},
		{
			name:     "partition - multiple partitions",
			query:    "SELECT id FROM logs PARTITION (p1, p2, p3) WHERE created_at > ?",
			expected: "select count(*) from logs partition (p1, p2, p3) where created_at > ?",
		},
		// Query already has COUNT in SELECT
		{
			name:     "already count star - replaced with count star",
			query:    "SELECT COUNT(*) FROM users",
			expected: "select count(*) from users",
		},
		{
			name:     "already count star with alias - alias removed",
			query:    "SELECT COUNT(*) AS total FROM users",
			expected: "select count(*) from users",
		},
		{
			name:     "already count id - replaced with count star",
			query:    "SELECT COUNT(id) FROM users",
			expected: "select count(*) from users",
		},
		{
			name:      "already count star - UseColumn id replaces with count id",
			query:     "SELECT COUNT(*) FROM users",
			useColumn: "id",
			expected:  "select count(id) from users",
		},
		{
			name:     "already count with other columns - replaced with count only",
			query:    "SELECT COUNT(*), category FROM products GROUP BY category",
			expected: "select count(*) from products group by category",
		},
		// Complex query
		{
			name:     "full featured query",
			query:    "WITH last_activity AS ( SELECT user_id, MAX(created_at) AS created_at FROM activity_log GROUP BY user_id ) SELECT u.id, u.name, la.created_at FROM users u LEFT JOIN last_activity la ON la.user_id = u.id WHERE u.status = ? AND u.deleted_at IS NULL GROUP BY u.id, u.name, la.created_at HAVING COUNT(*) > 0 ORDER BY u.name LIMIT 10",
			expected: "with last_activity as ( select user_id, max(created_at) as created_at from activity_log group by user_id ) select count(*) from users u left join last_activity la on la.user_id = u.id where u.status = ? and u.deleted_at is null group by u.id, u.name, la.created_at having count(*) > 0 order by u.name limit 10",
		},
		// Errors
		{
			name:      "empty query",
			query:     "",
			expectErr: true,
		},
		{
			name:      "no from clause",
			query:     "SELECT 1",
			expectErr: true,
		},
		{
			name:      "not select - insert",
			query:     "INSERT INTO users (id) VALUES (1)",
			expectErr: true,
		},
		{
			name:      "not select - update",
			query:     "UPDATE users SET name = ? WHERE id = ?",
			expectErr: true,
		},
		{
			name:      "not select - delete",
			query:     "DELETE FROM users WHERE id = ?",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCount(tt.query)
			if tt.useColumn != "" {
				c = c.UseColumn(tt.useColumn)
			}
			got, err := c.Build()

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotLower := strings.ToLower(got)
			if gotLower != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, gotLower)
			}
		})
	}
}

func TestNewCountUseColumnChaining(t *testing.T) {
	// Last UseColumn wins
	query := "SELECT id FROM users"
	c := NewCount(query).UseColumn("1").UseColumn("id")
	got, err := c.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "select count(id) from users"
	if strings.ToLower(got) != expected {
		t.Errorf("expected %q, got %q", expected, strings.ToLower(got))
	}
}


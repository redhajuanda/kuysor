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
		// GROUP BY — must wrap in subquery to return a scalar count of groups
		{
			name:     "group by wraps in subquery",
			query:    "SELECT category, COUNT(*) FROM products GROUP BY category",
			expected: "select count(*) from (select category, count(*) from products group by category) kuysor_count",
		},
		{
			name:     "group by and having wraps in subquery",
			query:    "SELECT category FROM products GROUP BY category HAVING COUNT(*) > 5",
			expected: "select count(*) from (select category from products group by category having count(*) > 5) kuysor_count",
		},
		{
			name:     "group by with order by and limit - stripped before wrapping",
			query:    "SELECT department, COUNT(*) FROM employees GROUP BY department ORDER BY department LIMIT 10",
			expected: "select count(*) from (select department, count(*) from employees group by department) kuysor_count",
		},
		{
			name:     "CTE with group by in main query wraps inner select only",
			query:    "WITH dept AS (SELECT id, name FROM departments) SELECT d.name, COUNT(*) FROM employees e JOIN dept d ON e.dept_id = d.id GROUP BY d.name",
			expected: "with dept as (select id, name from departments) select count(*) from (select d.name, count(*) from employees e join dept d on e.dept_id = d.id group by d.name) kuysor_count",
		},
		// DISTINCT — must wrap to preserve distinctness; naive COUNT(*) overcounts
		{
			name:     "distinct wraps in subquery",
			query:    "SELECT DISTINCT department FROM employees",
			expected: "select count(*) from (select distinct department from employees) kuysor_count",
		},
		{
			name:     "distinct multiple columns wraps in subquery",
			query:    "SELECT DISTINCT id, name FROM users",
			expected: "select count(*) from (select distinct id, name from users) kuysor_count",
		},
		{
			name:     "distinct with order by - stripped before wrapping",
			query:    "SELECT DISTINCT department FROM employees ORDER BY department",
			expected: "select count(*) from (select distinct department from employees) kuysor_count",
		},
		{
			name:     "distinct with where preserved inside subquery",
			query:    "SELECT DISTINCT department FROM employees WHERE active = 1",
			expected: "select count(*) from (select distinct department from employees where active = 1) kuysor_count",
		},
		// UNION / UNION ALL — must wrap; naive SELECT COUNT(*) FROM t1 UNION ... is broken
		{
			name:     "union wraps in subquery",
			query:    "SELECT id FROM employees UNION SELECT id FROM contractors",
			expected: "select count(*) from (select id from employees union select id from contractors) kuysor_count",
		},
		{
			name:     "union all wraps in subquery",
			query:    "SELECT id FROM employees UNION ALL SELECT id FROM contractors",
			expected: "select count(*) from (select id from employees union all select id from contractors) kuysor_count",
		},
		{
			name:     "union with order by - stripped before wrapping",
			query:    "SELECT id, name FROM employees UNION ALL SELECT id, name FROM contractors ORDER BY name LIMIT 20",
			expected: "select count(*) from (select id, name from employees union all select id, name from contractors) kuysor_count",
		},
		{
			name:     "union inside subquery is not top-level - no wrapping",
			query:    "SELECT * FROM (SELECT id FROM t1 UNION SELECT id FROM t2) AS sub",
			expected: "select count(*) from (select id from t1 union select id from t2) as sub",
		},
		{
			name:     "CTE with union in main query wraps inner select only",
			query:    "WITH cte AS (SELECT id FROM t) SELECT id FROM cte UNION ALL SELECT id FROM other",
			expected: "with cte as (select id from t) select count(*) from (select id from cte union all select id from other) kuysor_count",
		},
		// ORDER BY and LIMIT stripped for simple queries (no GROUP BY / DISTINCT / UNION)
		{
			name:     "order by stripped",
			query:    "SELECT id, name FROM users WHERE status = ? ORDER BY name",
			expected: "select count(*) from users where status = ?",
		},
		{
			name:     "order by and limit stripped",
			query:    "SELECT id, name FROM users WHERE status = ? ORDER BY name LIMIT 10",
			expected: "select count(*) from users where status = ?",
		},
		{
			name:     "order by and limit and offset stripped",
			query:    "SELECT id FROM users ORDER BY id LIMIT 10 OFFSET 20",
			expected: "select count(*) from users",
		},
		{
			name:     "offset only stripped",
			query:    "SELECT id FROM users OFFSET 20",
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
			name:     "already count with other columns and group by - wraps in subquery",
			query:    "SELECT COUNT(*), category FROM products GROUP BY category",
			expected: "select count(*) from (select count(*), category from products group by category) kuysor_count",
		},
		// Complex query
		{
			name:  "full featured query - group by wraps, order by and limit stripped",
			query: "WITH last_activity AS ( SELECT user_id, MAX(created_at) AS created_at FROM activity_log GROUP BY user_id ) SELECT u.id, u.name, la.created_at FROM users u LEFT JOIN last_activity la ON la.user_id = u.id WHERE u.status = ? AND u.deleted_at IS NULL GROUP BY u.id, u.name, la.created_at HAVING COUNT(*) > 0 ORDER BY u.name LIMIT 10",
			expected: "with last_activity as ( select user_id, max(created_at) as created_at from activity_log group by user_id ) select count(*) from (select u.id, u.name, la.created_at from users u left join last_activity la on la.user_id = u.id where u.status = ? and u.deleted_at is null group by u.id, u.name, la.created_at having count(*) > 0) kuysor_count",
		},
		// CTE with group by only in CTE body (not in main query) — no wrapping
		{
			name:     "group by only inside CTE body - main query has no group by",
			query:    "WITH agg AS (SELECT dept, COUNT(*) AS cnt FROM employees GROUP BY dept) SELECT dept, cnt FROM agg",
			expected: "with agg as (select dept, count(*) as cnt from employees group by dept) select count(*) from agg",
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

func TestNewCountRemoveUnusedLeftJoins(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "single left join not in where - removed",
			query:    "SELECT u.id, u.name, p.title FROM users u LEFT JOIN profiles p ON u.id = p.user_id",
			expected: "select count(*) from users u",
		},
		{
			name:     "single left join used in where - kept",
			query:    "SELECT u.id, u.name FROM users u LEFT JOIN profiles p ON u.id = p.user_id WHERE p.verified = 1",
			expected: "select count(*) from users u left join profiles p on u.id = p.user_id where p.verified = 1",
		},
		{
			name:     "multiple left joins none in where - all removed",
			query:    "SELECT u.id, p.name, a.url FROM users u LEFT JOIN profiles p ON u.id = p.user_id LEFT JOIN avatars a ON u.id = a.user_id",
			expected: "select count(*) from users u",
		},
		{
			name:     "multiple left joins one in where - only used one kept",
			query:    "SELECT u.id FROM users u LEFT JOIN profiles p ON u.id = p.user_id LEFT JOIN avatars a ON u.id = a.user_id WHERE a.url IS NOT NULL",
			expected: "select count(*) from users u left join avatars a on u.id = a.user_id where a.url is not null",
		},
		{
			name:     "transitive dependency - both kept",
			query:    "SELECT u.id FROM users u LEFT JOIN profiles p ON u.id = p.user_id LEFT JOIN avatars a ON p.id = a.profile_id WHERE a.url IS NOT NULL",
			expected: "select count(*) from users u left join profiles p on u.id = p.user_id left join avatars a on p.id = a.profile_id where a.url is not null",
		},
		{
			name:     "inner join preserved left join removed",
			query:    "SELECT u.id FROM users u INNER JOIN orders o ON u.id = o.user_id LEFT JOIN profiles p ON u.id = p.user_id WHERE o.total > 100",
			expected: "select count(*) from users u inner join orders o on u.id = o.user_id where o.total > 100",
		},
		{
			name:     "left join between other joins - removed cleanly",
			query:    "SELECT u.id FROM users u LEFT JOIN profiles p ON u.id = p.user_id INNER JOIN orders o ON u.id = o.user_id WHERE o.total > 100",
			expected: "select count(*) from users u inner join orders o on u.id = o.user_id where o.total > 100",
		},
		{
			name:     "no where clause - all left joins removed",
			query:    "SELECT u.id, p.name FROM users u LEFT JOIN profiles p ON u.id = p.user_id",
			expected: "select count(*) from users u",
		},
		{
			name:     "left outer join not in where - removed",
			query:    "SELECT u.id FROM users u LEFT OUTER JOIN profiles p ON u.id = p.user_id",
			expected: "select count(*) from users u",
		},
		{
			name:     "left join with order by and limit - all stripped",
			query:    "SELECT u.id, p.name FROM users u LEFT JOIN profiles p ON u.id = p.user_id ORDER BY u.id LIMIT 10",
			expected: "select count(*) from users u",
		},
		{
			name:     "no left joins - query unchanged",
			query:    "SELECT u.id FROM users u INNER JOIN orders o ON u.id = o.user_id WHERE o.total > 100",
			expected: "select count(*) from users u inner join orders o on u.id = o.user_id where o.total > 100",
		},
		{
			name:     "left join in subquery not affected",
			query:    "SELECT id FROM (SELECT u.id, p.name FROM users u LEFT JOIN profiles p ON u.id = p.user_id) AS sub",
			expected: "select count(*) from (select u.id, p.name from users u left join profiles p on u.id = p.user_id) as sub",
		},
		{
			name:     "CTE with left join in main query - removed",
			query:    "WITH active AS (SELECT id FROM users WHERE status = 1) SELECT a.id, p.name FROM active a LEFT JOIN profiles p ON a.id = p.user_id",
			expected: "with active as (select id from users where status = 1) select count(*) from active a",
		},
		{
			name:     "real world CTE with many left joins none in where - all removed",
			query:    "WITH filtered_ticket AS ( SELECT t.id FROM ticket t WHERE t.deleted_at = 0 ) select t.id as id, t.code as ticket_code, t.name as ticket_name, t.customer_name as customer_name, t.issue_tag_id as issue_tag_id, team.name as team_name, assign.name as assign_name, t.is_fcr as is_fcr, t.hcp_id, t.ticket_category as ticket_category, t.ticket_type as ticket_type, t.created_at as created_at, t.closed_timestamp as closed_timestamp, t.assigned_to_timestamp as assign_to_branch_timestamp, t.stage as stage, csat.rating as csat_rating, t.number_ticket_vendor as number_ticket_vendor, tx_claim_status.indexed_property_4 as claim_payment_method, tx_claim_form.indexed_property_3 as jenis_pembayaran, CONCAT_WS(' - ', actor.code, actor.name) as created_by, JSON_EXTRACT(t.attribute, '$.aging_time') as aging_time from filtered_ticket ft left join ticket t on (t.id = ft.id) left join account team on (team.id = t.team_id) left join account assign on (assign.id = t.assigned_to_id) left join transaction tx_claim_form on ( tx_claim_form.number = t.id and tx_claim_form.transaction_type_id = '01JQ6FE2A5DRZCXNGJAAYY3XC0' ) left join transaction tx_claim_status on ( tx_claim_status.number = t.id and tx_claim_status.transaction_type_id = '01K82Z22M402866QBT5E8SQCCC' ) left join account actor on (actor.id = t.created_by) left join csat on csat.unique_id = t.number_ticket_vendor and csat.deleted_at = 0 where 1 = 1",
			expected: "with filtered_ticket as ( select t.id from ticket t where t.deleted_at = 0 ) select count(*) from filtered_ticket ft where 1 = 1",
		},
		{
			name:     "real world CTE with many left joins and group by - only needed joins kept",
			query:    "WITH filtered_ticket AS ( SELECT t.id FROM ticket t WHERE t.deleted_at = 0 ) select t.id as id, t.code as ticket_code, t.name as ticket_name, t.customer_name as customer_name, t.issue_tag_id as issue_tag_id, team.name as team_name, assign.name as assign_name, t.is_fcr as is_fcr, t.hcp_id, t.ticket_category as ticket_category, t.ticket_type as ticket_type, t.created_at as created_at, t.closed_timestamp as closed_timestamp, t.assigned_to_timestamp as assign_to_branch_timestamp, t.stage as stage, csat.rating as csat_rating, t.number_ticket_vendor as number_ticket_vendor, tx_claim_status.indexed_property_4 as claim_payment_method, tx_claim_form.indexed_property_3 as jenis_pembayaran, CONCAT_WS(' - ', actor.code, actor.name) as created_by, JSON_EXTRACT(t.attribute, '$.aging_time') as aging_time from filtered_ticket ft left join ticket t on (t.id = ft.id) left join account team on (team.id = t.team_id) left join account assign on (assign.id = t.assigned_to_id) left join transaction tx_claim_form on ( tx_claim_form.number = t.id and tx_claim_form.transaction_type_id = '01JQ6FE2A5DRZCXNGJAAYY3XC0' ) left join transaction tx_claim_status on ( tx_claim_status.number = t.id and tx_claim_status.transaction_type_id = '01K82Z22M402866QBT5E8SQCCC' ) left join account actor on (actor.id = t.created_by) left join csat on csat.unique_id = t.number_ticket_vendor and csat.deleted_at = 0 where 1 = 1 GROUP BY t.id",
			expected: "with filtered_ticket as ( select t.id from ticket t where t.deleted_at = 0 ) select count(*) from (select t.id as id, t.code as ticket_code, t.name as ticket_name, t.customer_name as customer_name, t.issue_tag_id as issue_tag_id, t.is_fcr as is_fcr, t.hcp_id, t.ticket_category as ticket_category, t.ticket_type as ticket_type, t.created_at as created_at, t.closed_timestamp as closed_timestamp, t.assigned_to_timestamp as assign_to_branch_timestamp, t.stage as stage, t.number_ticket_vendor as number_ticket_vendor, json_extract(t.attribute, '$.aging_time') as aging_time from filtered_ticket ft left join ticket t on (t.id = ft.id) where 1 = 1 group by t.id) kuysor_count",
		},
		{
			name:     "group by with left join alias in group by - kept",
			query:    "SELECT p.category, COUNT(*) FROM products pr LEFT JOIN categories p ON pr.cat_id = p.id GROUP BY p.category",
			expected: "select count(*) from (select p.category, count(*) from products pr left join categories p on pr.cat_id = p.id group by p.category) kuysor_count",
		},
		{
			name:     "group by with left join alias in having - kept",
			query:    "SELECT pr.id FROM products pr LEFT JOIN categories p ON pr.cat_id = p.id GROUP BY pr.id HAVING COUNT(p.id) > 0",
			expected: "select count(*) from (select pr.id from products pr left join categories p on pr.cat_id = p.id group by pr.id having count(p.id) > 0) kuysor_count",
		},
		{
			name:     "group by with unused left join - removed",
			query:    "SELECT u.dept, COUNT(*) FROM users u LEFT JOIN profiles p ON u.id = p.user_id GROUP BY u.dept",
			expected: "select count(*) from (select u.dept, count(*) from users u group by u.dept) kuysor_count",
		},
		{
			name:     "table name used directly without alias - not in where removed",
			query:    "SELECT users.id, profiles.name FROM users LEFT JOIN profiles ON users.id = profiles.user_id",
			expected: "select count(*) from users",
		},
		{
			name:     "table name used directly - in where kept",
			query:    "SELECT users.id FROM users LEFT JOIN profiles ON users.id = profiles.user_id WHERE profiles.verified = 1",
			expected: "select count(*) from users left join profiles on users.id = profiles.user_id where profiles.verified = 1",
		},
		// Alias with AS keyword
		{
			name:     "alias with AS - not in where removed",
			query:    "SELECT u.id, p.name FROM users u LEFT JOIN profiles AS p ON u.id = p.user_id",
			expected: "select count(*) from users u",
		},
		{
			name:     "alias with AS - used in where kept",
			query:    "SELECT u.id FROM users u LEFT JOIN profiles AS p ON u.id = p.user_id WHERE p.verified = 1",
			expected: "select count(*) from users u left join profiles as p on u.id = p.user_id where p.verified = 1",
		},
		// Table name referenced in WHERE instead of alias
		{
			name:     "where uses table name instead of alias - kept",
			query:    "SELECT u.id FROM users u LEFT JOIN profiles p ON u.id = p.user_id WHERE profiles.verified = 1",
			expected: "select count(*) from users u left join profiles p on u.id = p.user_id where profiles.verified = 1",
		},
		{
			name:     "where uses table name instead of AS alias - kept",
			query:    "SELECT u.id FROM users u LEFT JOIN profiles AS p ON u.id = p.user_id WHERE profiles.verified = 1",
			expected: "select count(*) from users u left join profiles as p on u.id = p.user_id where profiles.verified = 1",
		},
		// Group by uses table name instead of alias
		{
			name:     "group by uses table name instead of alias - kept",
			query:    "SELECT p.category FROM products LEFT JOIN categories AS p ON products.cat_id = p.id GROUP BY categories.category",
			expected: "select count(*) from (select p.category from products left join categories as p on products.cat_id = p.id group by categories.category) kuysor_count",
		},
		// Multiple left joins with mixed alias styles
		{
			name:     "mixed alias styles - AS and without AS",
			query:    "SELECT u.id FROM users u LEFT JOIN profiles AS p ON u.id = p.user_id LEFT JOIN avatars a ON u.id = a.user_id LEFT JOIN settings AS s ON u.id = s.user_id WHERE p.verified = 1",
			expected: "select count(*) from users u left join profiles as p on u.id = p.user_id where p.verified = 1",
		},
		// Transitive dependency with table name in where
		{
			name:     "transitive dependency via table name in where",
			query:    "SELECT u.id FROM users u LEFT JOIN profiles p ON u.id = p.user_id LEFT JOIN avatars a ON p.id = a.profile_id WHERE avatars.url IS NOT NULL",
			expected: "select count(*) from users u left join profiles p on u.id = p.user_id left join avatars a on p.id = a.profile_id where avatars.url is not null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCount(tt.query).RemoveUnusedLeftJoins().Build()
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

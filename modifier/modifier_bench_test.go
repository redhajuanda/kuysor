package modifier

import (
	"testing"
)

// BenchmarkAppendWhere benchmarks appending WHERE conditions
func BenchmarkAppendWhere(b *testing.B) {
	baseQuery := "SELECT * FROM users WHERE status = 'active'"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := NewSQLModifier(baseQuery)
		m.AppendWhere("id > 100")
		_, err := m.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAppendWhereComplex benchmarks appending complex WHERE conditions
func BenchmarkAppendWhereComplex(b *testing.B) {
	baseQuery := "SELECT u.id, u.name, p.title FROM users u LEFT JOIN profiles p ON u.id = p.user_id WHERE u.status = 'active'"
	condition := "(u.created_at > '2023-01-01' OR u.updated_at > '2023-06-01') AND u.email IS NOT NULL"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := NewSQLModifier(baseQuery)
		m.AppendWhere(condition)
		_, err := m.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSetOrderBy benchmarks setting ORDER BY clauses
func BenchmarkSetOrderBy(b *testing.B) {
	baseQuery := "SELECT * FROM users WHERE status = 'active'"

	testCases := []struct {
		name    string
		orderBy []string
	}{
		{"single_column", []string{"id ASC"}},
		{"two_columns", []string{"name ASC", "id DESC"}},
		{"four_columns", []string{"name ASC", "id DESC", "created_at ASC", "email DESC"}},
		{"eight_columns", []string{"name ASC", "id DESC", "created_at ASC", "email DESC", "status ASC", "updated_at DESC", "company_id ASC", "last_login DESC"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m := NewSQLModifier(baseQuery)
				m.SetOrderBy(tc.orderBy...)
				_, err := m.Build()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSetLimit benchmarks setting LIMIT clauses
func BenchmarkSetLimit(b *testing.B) {
	baseQuery := "SELECT * FROM users WHERE status = 'active' ORDER BY id"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := NewSQLModifier(baseQuery)
		m.SetLimit("10")
		_, err := m.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSetOffset benchmarks setting OFFSET clauses
func BenchmarkSetOffset(b *testing.B) {
	baseQuery := "SELECT * FROM users WHERE status = 'active' ORDER BY id LIMIT 10"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := NewSQLModifier(baseQuery)
		m.SetOffset("100")
		_, err := m.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConvertToCount benchmarks converting SELECT to COUNT queries
func BenchmarkConvertToCount(b *testing.B) {
	queries := []string{
		"SELECT id, name FROM users WHERE status = 'active'",
		"SELECT u.id, u.name, p.title FROM users u LEFT JOIN profiles p ON u.id = p.user_id WHERE u.status = 'active'",
		"SELECT u.id, u.name, p.title, c.name FROM users u LEFT JOIN profiles p ON u.id = p.user_id LEFT JOIN companies c ON u.company_id = c.id WHERE u.status = 'active' AND u.created_at > '2023-01-01'",
	}

	for i, query := range queries {
		b.Run(getQueryName(i), func(b *testing.B) {
			b.ResetTimer()
			for j := 0; j < b.N; j++ {
				m := NewSQLModifier(query)
				err := m.ConvertToCount()
				if err != nil {
					b.Fatal(err)
				}
				_, err = m.Build()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFindMainClausePosition benchmarks finding main clause positions
func BenchmarkFindMainClausePosition(b *testing.B) {
	query := `
		SELECT u.id, u.name, 
		       (SELECT COUNT(*) FROM orders WHERE user_id = u.id) as order_count
		FROM users u 
		LEFT JOIN profiles p ON u.id = p.user_id 
		WHERE u.status = 'active' 
		  AND u.created_at > '2023-01-01'
		  AND EXISTS (SELECT 1 FROM companies c WHERE c.id = u.company_id AND c.active = true)
		ORDER BY u.name, u.id
		LIMIT 50
	`

	clauses := []string{"WHERE", "ORDER BY", "LIMIT", "GROUP BY", "HAVING"}

	for _, clause := range clauses {
		b.Run(clause, func(b *testing.B) {
			m := NewSQLModifier(query)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m.findMainClausePosition(clause)
			}
		})
	}
}

// BenchmarkNormalizeSQL benchmarks SQL normalization
func BenchmarkNormalizeSQL(b *testing.B) {
	queries := []string{
		"SELECT    id,    name   FROM   users   WHERE   status   =   'active'",
		`SELECT 
			u.id, 
			u.name, 
			u.email
		FROM 
			users u 
		WHERE 
			u.status = 'active' 
			AND u.created_at > '2023-01-01'`,
		"SELECT * FROM users WHERE name = 'John O''Connor' AND description = \"Test 'description'\" AND data = `some data`",
	}

	for i, query := range queries {
		b.Run(getQueryName(i), func(b *testing.B) {
			b.ResetTimer()
			for j := 0; j < b.N; j++ {
				_, err := normalizeSQL(query)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCompleteModification benchmarks a complete query modification
func BenchmarkCompleteModification(b *testing.B) {
	baseQuery := "SELECT u.id, u.name, u.email FROM users u WHERE u.status = ?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := NewSQLModifier(baseQuery)

		// Add WHERE condition
		m.AppendWhere("(u.created_at > ? OR u.updated_at > ?) AND u.email IS NOT NULL")

		// Set ORDER BY
		m.SetOrderBy("u.name ASC", "u.id DESC")

		// Set LIMIT
		m.SetLimit("25")

		// Build final query
		_, err := m.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryAllocations benchmarks memory usage during SQL modification
func BenchmarkMemoryAllocations(b *testing.B) {
	baseQuery := "SELECT u.id, u.name, u.email FROM users u WHERE u.status = 'active'"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m := NewSQLModifier(baseQuery)
		m.AppendWhere("u.created_at > '2023-01-01'")
		m.SetOrderBy("u.name ASC", "u.id DESC")
		m.SetLimit("10")

		result, err := m.Build()
		if err != nil {
			b.Fatal(err)
		}

		// Prevent optimization from removing the result
		if result == "" {
			b.Fatal("empty result")
		}
	}
}

// Helper function to generate query names for benchmarks
func getQueryName(index int) string {
	names := []string{"simple", "join", "complex", "cte"}
	if index < len(names) {
		return names[index]
	}
	return "unknown"
}

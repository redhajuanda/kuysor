package kuysor

import (
	"fmt"
	"testing"
)

// BenchmarkQueryBuild benchmarks the basic query building process
func BenchmarkQueryBuild(b *testing.B) {
	query := "SELECT id, name, email FROM users WHERE status = ?"
	args := []interface{}{"active"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewQuery(query, Cursor).
			WithOrderBy("id", "-name").
			WithLimit(10).
			WithArgs(args...).
			Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQueryBuildWithCursor benchmarks query building with cursor
func BenchmarkQueryBuildWithCursor(b *testing.B) {
	query := "SELECT id, name, email FROM users WHERE status = ?"
	args := []interface{}{"active"}
	// Create a valid cursor with proper format: next_|id|2||name|John
	cursorData := `{"prefix":"next_","cols":{"id":2,"name":"John"}}`
	cursor := base64Encode(cursorData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewQuery(query, Cursor).
			WithOrderBy("id", "-name").
			WithLimit(10).
			WithArgs(args...).
			WithCursor(cursor).
			Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComplexQuery benchmarks building complex queries with multiple columns
func BenchmarkComplexQuery(b *testing.B) {
	query := `
		SELECT u.id, u.name, u.email, p.title, c.name as company 
		FROM users u 
		LEFT JOIN profiles p ON u.id = p.user_id 
		LEFT JOIN companies c ON u.company_id = c.id 
		WHERE u.status = ? AND u.created_at > ?
	`
	args := []interface{}{"active", "2023-01-01"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewQuery(query, Cursor).
			WithOrderBy("u.created_at", "-u.id", "p.title", "c.name").
			WithLimit(25).
			WithArgs(args...).
			Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCursorParsing benchmarks cursor parsing performance
func BenchmarkCursorParsing(b *testing.B) {
	// Create a valid cursor with proper JSON format
	cursorData := `{"prefix":"next_","cols":{"id":2,"name":"John"}}`
	cursor := base64Encode(cursorData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cursorBase64 := cursorBase64(cursor)
		_, err := cursorBase64.parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSortParsing benchmarks sort parsing for different numbers of columns
func BenchmarkSortParsing(b *testing.B) {
	testCases := []struct {
		name  string
		sorts []string
	}{
		{"single_column", []string{"id"}},
		{"two_columns", []string{"id", "-name"}},
		{"four_columns", []string{"id", "-name", "email", "-created_at"}},
		{"eight_columns", []string{"id", "-name", "email", "-created_at", "status", "-updated_at", "company_id", "-last_login"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				parseSort(tc.sorts, BoolSort)
			}
		})
	}
}

// BenchmarkOffsetQuery benchmarks offset-based pagination
func BenchmarkOffsetQuery(b *testing.B) {
	query := "SELECT id, name, email FROM users WHERE status = ?"
	args := []interface{}{"active"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewQuery(query, Offset).
			WithOrderBy("id", "-name").
			WithLimit(10).
			WithOffset(100).
			WithArgs(args...).
			Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNullableSort benchmarks nullable column sorting
func BenchmarkNullableSort(b *testing.B) {
	query := "SELECT id, name, optional_field FROM users WHERE status = ?"
	args := []interface{}{"active"}

	testCases := []struct {
		name   string
		method NullSortMethod
	}{
		{"BoolSort", BoolSort},
		{"FirstLast", FirstLast},
		{"CaseWhen", CaseWhen},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := NewQuery(query, Cursor).
					WithOrderBy("optional_field null", "id").
					WithNullSortMethod(tc.method).
					WithLimit(10).
					WithArgs(args...).
					Build()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSanitizeMap benchmarks result sanitization
func BenchmarkSanitizeMap(b *testing.B) {
	// Create sample data
	data := make([]map[string]any, 11) // limit + 1
	for i := 0; i < 11; i++ {
		data[i] = map[string]any{
			"id":   i + 1,
			"name": fmt.Sprintf("User%d", i+1),
		}
	}

	// Build query result
	query := "SELECT id, name FROM users"
	result, err := NewQuery(query, Cursor).
		WithOrderBy("id").
		WithLimit(10).
		Build()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create fresh copy for each iteration
		dataCopy := make([]map[string]any, len(data))
		copy(dataCopy, data)

		_, _, err := result.SanitizeMap(&dataCopy)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPlaceholderReplacement benchmarks placeholder replacement
func BenchmarkPlaceholderReplacement(b *testing.B) {
	query := "SELECT * FROM users WHERE id = $0 AND name = $0 AND email = $0 AND status = $0"

	testCases := []struct {
		name        string
		placeholder PlaceHolderType
	}{
		{"Question", Question},
		{"Dollar", Dollar},
		{"At", At},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				replacePlaceholders(query, tc.placeholder)
			}
		})
	}
}

// BenchmarkBuildCountQuery benchmarks count query building
func BenchmarkBuildCountQuery(b *testing.B) {
	queries := []string{
		"SELECT id, name FROM users WHERE status = 'active'",
		"SELECT u.id, u.name, p.title FROM users u LEFT JOIN profiles p ON u.id = p.user_id WHERE u.status = 'active'",
		"SELECT u.id, u.name, p.title, c.name FROM users u LEFT JOIN profiles p ON u.id = p.user_id LEFT JOIN companies c ON u.company_id = c.id WHERE u.status = 'active' AND u.created_at > '2023-01-01'",
	}

	for i, query := range queries {
		b.Run(fmt.Sprintf("query_%d", i+1), func(b *testing.B) {
			b.ResetTimer()
			for j := 0; j < b.N; j++ {
				_, err := BuildCountQuery(query)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMemoryAllocations benchmarks memory usage during query building
func BenchmarkMemoryAllocations(b *testing.B) {
	query := "SELECT id, name, email FROM users WHERE status = ?"
	args := []interface{}{"active"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := NewQuery(query, Cursor).
			WithOrderBy("id", "-name").
			WithLimit(10).
			WithArgs(args...).
			Build()
		if err != nil {
			b.Fatal(err)
		}

		// Prevent optimization from removing the result
		if result.Query == "" {
			b.Fatal("empty query")
		}
	}
}

// BenchmarkConcurrentAccess benchmarks concurrent access patterns
func BenchmarkConcurrentAccess(b *testing.B) {
	query := "SELECT id, name FROM users WHERE status = ?"
	args := []interface{}{"active"}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := NewQuery(query, Cursor).
				WithOrderBy("id").
				WithLimit(10).
				WithArgs(args...).
				Build()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

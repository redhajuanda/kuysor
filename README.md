# Kuysor

Tired of wrestling with complex SQL for cursor pagination? Kuysor is here to help. This lightweight Go SDK makes cursor-based pagination straightforward, letting you focus on your application logic instead of database plumbing.


## What is Cursor Pagination?
Cursor-based pagination (aka Keyset Pagination) is a more efficient and scalable alternative to offset-based pagination, particularly for large datasets. Instead of specifying an offset, it uses a cursor—a unique identifier from the last retrieved record—to determine the starting point for the next / previous set of results.

## Why choose kuysor?
- Designed for simplicity and ease of use.
- The only Go SDK built for this purpose.
- Compatible with most if not all SQL-based databases
- Supports multiple sorting columns and ordering directions.
- Handles nullable columns in sorting (maximum one nullable column).
- Zero dependencies.

## How kuysor works?
1. Kuysor modifies your SQL query to support cursor-based pagination, appending the necessary `ORDER BY`, `LIMIT`, and `WHERE` clauses.
2. You execute the modified query to fetch paginated data from your database.
3. Kuysor then sanitizes the data and generates the next and previous cursors.
4. When fetching the next or previous page, you pass the generated cursor back to Kuysor, which modifies the query accordingly, repeating the process from Step 1.

## Installation

```bash
go get github.com/redhajuanda/kuysor
```

## Quick Start

### Retrieving the First Page

Suppose you have a query like this:
```sql
SELECT a.id, a.code, a.name 
FROM account a 
WHERE a.status = ? 
ORDER BY a.code ASC, a.id DESC
LIMIT ?
```

To implement Kuysor, first you need to remove the `ORDER BY` and `LIMIT` clauses from your original query. And set the `ORDER BY` and `LIMIT` clauses using Kuysor's methods `WithOrderBy()` and `WithLimit()`.

So the query will look like this:
```sql
SELECT a.id, a.code, a.name
FROM account a 
WHERE a.status = ?
```

Then, you can use Kuysor to modify the query and add the `ORDER BY` and `LIMIT` clauses. Here's how to do it:

```go
package main
import (
	"fmt"
	"github.com/redhajuanda/kuysor"
)

func main() {
	query := `
		SELECT a.id, a.code, a.name 
		FROM account a 
		WHERE a.status = ?
	`
	args := []interface{}{"active"}

	ks, err := kuysor.
		NewQuery(query, kuysor.Cursor).
		WithOrderBy("a.code", "-a.id"). // Required. Defines the order by. Prefix columns with `-` for descending order, `+` for ascending order. Default is ascending.
		WithLimit(10). // Override the default limit.
		WithArgs(args...). // Since Kuysor modifies the query by appending additional conditions and limit, it also adjusts the argument list accordingly, and ensuring the generated arguments are placed in the correct order.
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println(ks.Query) // Prints the modified query
	fmt.Println(ks.Args)  // Prints the modified arguments
}
```
Return query:
```sql
SELECT a.id, a.code, a.name FROM account a WHERE a.status = ? ORDER BY a.code ASC, a.id DESC LIMIT ? -- ORDER BY and LIMIT are automatically appended based on the options
```
Return args:
```go
["active", 11] // 11 is automatically appended to the arguments based on the limit + 1, additional 1 is used to check if there are more data to fetch for the next page
```

### Fetching The Data
Use the modified query and arguments from the previous step to fetch the data from your database like usual. 

For example, using the `database/sql` package:
```go
type Account struct {
	ID     int    	`kuysor:"a.id"`
	Code   string 	`kuysor:"a.code"`
	Name   string 	`kuysor:"a.name"`
	Status *string 	`kuysor:"a.status"`
}

// fetching the data
rows, err := db.Query(res.Query, res.Args...)
if err != nil {
	return Result{}, err
}
defer rows.Close()
var result = make([]Account, 0)
for rows.Next() {
	var row Account
	err = rows.Scan(&row.ID, &row.Code, &row.Name, &row.Status)
	if err != nil {
		return Result{}, err
	}
	result = append(result, row)
}
```

### Sanitizing the Result
Cursor pagination can sometimes return extra data or even reverse the order of results. This is because the query may return more than the specified limit, and the order of the results may not match the expected order due to the cursor.
To handle the extra data and correct the order, use the `SanitizeStruct()` or `SanitizeMap()` function. These functions will trim the extra data, correct the order when using a previous cursor, and generate the next and previous cursors properly.
```go
// sanitize the data
next, prev, err := res.SanitizeStruct(&result)
if err != nil {
	return Result{}, err
}
```

> Note: struct tags are required to match the column names in the query. Kuysor uses these tags to map the struct fields to the query columns, so it can generate the next and previous cursors correctly.


### Retrieving The Next/Previous Page
To fetch the next or previous page, simply include the cursor from the previous query result.

```go
package main
import (
	"fmt"
	"github.com/redhajuanda/kuysor"
)
func main() {
	query := `SELECT a.id, a.code, a.name FROM account a WHERE a.status = ?`
	args := []interface{}{"active"}

	ks, err := kuysor.
		NewQuery(query, kuysor.Cursor).
		WithLimit(10). 
		WithOrderBy("a.code", "-a.id"). 
		WithArgs(args...).
		WithCursor("xxx"). // the query will start from the cursor
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println(ks.Query) // Prints the modified query
	fmt.Println(ks.Args)  // Prints the modified arguments

	// >
	// execute the query and get the result
	// ...

	// >
	// sanitize the result and get the next and previous cursor
	// ...
}
```
Return query:
```sql
SELECT a.id, a.`code`, a.`name` FROM account as a 
WHERE a.`status` = ? AND 
(a.`code` > ? OR (a.`code` = ? AND a.id < ?))
ORDER BY a.`code` ASC, a.id DESC
LIMIT ?
```
Return args:
```go
["active", "C", "C", 3, 11]
```

### Converting to Count Query

Use `NewCount` to convert a SELECT query into a COUNT query for pagination metadata (total row count). Pass the same query you use for data fetching — Kuysor handles all the structural transformations needed to produce a correct scalar count.

#### Basic usage

```go
query := "SELECT id, code, name FROM users WHERE status = ?"

// Default: COUNT(*)
countQuery, err := kuysor.NewCount(query).Build()
// → SELECT COUNT(*) FROM users WHERE status = ?

// COUNT(1)
countQuery, err := kuysor.NewCount(query).UseColumn("1").Build()
// → SELECT COUNT(1) FROM users WHERE status = ?

// COUNT(id) or COUNT(t.id)
countQuery, err := kuysor.NewCount(query).UseColumn("id").Build()
// → SELECT COUNT(id) FROM users WHERE status = ?
```

WHERE, JOIN, and CTE clauses are preserved. Only the main SELECT list is replaced.

#### GROUP BY and HAVING — subquery wrapping

A query with `GROUP BY` returns one row per group, so a naive `SELECT COUNT(*) … GROUP BY …` returns multiple rows instead of a single total. Kuysor detects this and wraps the original query in a subquery:

```go
query := "SELECT department, COUNT(employee_id) FROM employees GROUP BY department"

countQuery, err := kuysor.NewCount(query).Build()
// → SELECT COUNT(*) FROM (
//       SELECT department, COUNT(employee_id) FROM employees GROUP BY department
//   ) kuysor_count
```

`HAVING` is preserved inside the subquery so group filters remain effective:

```go
query := "SELECT department FROM employees GROUP BY department HAVING COUNT(*) > 5"

countQuery, err := kuysor.NewCount(query).Build()
// → SELECT COUNT(*) FROM (
//       SELECT department FROM employees GROUP BY department HAVING COUNT(*) > 5
//   ) kuysor_count
```

#### DISTINCT — subquery wrapping

`SELECT DISTINCT` must also be wrapped. Without wrapping, `COUNT(*)` counts all rows and ignores the `DISTINCT`, producing an incorrect (inflated) total:

```go
query := "SELECT DISTINCT department FROM employees"

countQuery, err := kuysor.NewCount(query).Build()
// → SELECT COUNT(*) FROM (SELECT DISTINCT department FROM employees) kuysor_count
//
// Without wrapping this would be: SELECT COUNT(*) FROM employees
// which counts all rows, not distinct departments.
```

#### UNION / UNION ALL — subquery wrapping

A top-level `UNION` must be wrapped too. Without wrapping, `SELECT COUNT(*) FROM t1 UNION SELECT … FROM t2` returns multiple rows (one per union branch) instead of a single total:

```go
query := "SELECT id FROM employees UNION ALL SELECT id FROM contractors"

countQuery, err := kuysor.NewCount(query).Build()
// → SELECT COUNT(*) FROM (
//       SELECT id FROM employees UNION ALL SELECT id FROM contractors
//   ) kuysor_count
```

#### ORDER BY, LIMIT, and OFFSET — always stripped

These clauses are meaningless for a count and are always removed, whether or not wrapping is needed:

```go
query := "SELECT id, name FROM users WHERE status = ? ORDER BY name LIMIT 10 OFFSET 20"

countQuery, err := kuysor.NewCount(query).Build()
// → SELECT COUNT(*) FROM users WHERE status = ?
//   (ORDER BY, LIMIT, OFFSET removed)
```

When wrapping is needed (GROUP BY / DISTINCT / UNION), `ORDER BY` and `LIMIT` are stripped from the **inner** query before wrapping, so they don't accidentally limit the counted rows:

```go
query := "SELECT department, COUNT(*) FROM employees GROUP BY department ORDER BY department LIMIT 10"

countQuery, err := kuysor.NewCount(query).Build()
// → SELECT COUNT(*) FROM (
//       SELECT department, COUNT(*) FROM employees GROUP BY department
//       -- ORDER BY and LIMIT stripped from inner query
//   ) kuysor_count
```

#### CTE queries

CTEs are always preserved at the statement level so the inner subquery can reference them:

```go
query := `
    WITH dept AS (SELECT id, name FROM departments)
    SELECT d.name, COUNT(*) FROM employees e
    JOIN dept d ON e.dept_id = d.id
    GROUP BY d.name
`
countQuery, err := kuysor.NewCount(query).Build()
// → WITH dept AS (SELECT id, name FROM departments)
//   SELECT COUNT(*) FROM (
//       SELECT d.name, COUNT(*) FROM employees e
//       JOIN dept d ON e.dept_id = d.id GROUP BY d.name
//   ) kuysor_count
```

#### Database compatibility

The subquery alias `kuysor_count` is written **without** the `AS` keyword (`FROM (...) kuysor_count`), which is compatible with all major databases including Oracle (which does not support `AS` for table aliases in `FROM`).

| Database | Supported |
|---|---|
| MySQL | ✓ |
| PostgreSQL | ✓ |
| SQL Server | ✓ |
| Snowflake | ✓ |
| Oracle | ✓ |

### Handling Nullable Columns
If sorting involves nullable columns, specify them explicitly by adding `null` after the column name. This is mandatory to handle null values correctly, as they can affect the order of results.
To indicate a nullable column, append `null` after the column name, like so:

```go
WithOrderBy("a.status null", "a.id")
```
This generates the following SQL query:
```sql
ORDER BY a.status IS NULL ASC, a.status ASC,  a.id ASC
```

You can also specify the sorting method for the nullable sorting column. Kuysor provides 3 methods to handle sorting of nullable columns:
- `BoolSort`: `ORDER BY a.status IS NULL ASC, a.status ASC` // the default, supported by most sql databases
- `FirstLast`: `ORDER BY a.status ASC NULLS LAST` // Only supported by few databases
- `CaseWhen`: `ORDER BY CASE WHEN a.status IS NULL THEN 1 ELSE 0 END ASC, a.status ASC` // work around for databases that doesn't support both direct boolean in order by and aslo `NULLS FIRST`/`NULLS LAST` (like MySQL before 8.0)

To specify the method to use, you can set it in the global `SetGlobalOptions`, in the instance level, or in the query level. 

Here's an example of how to use `FirstLast` at the query level:
```go
ks, err := kuysor.
	NewQuery(query, kuysor.Cursor).
	WithOrderBy("a.status null", "a.id").
	WithNullSortMethod(kuysor.FirstLast). // set the null sort method to FirstLast
	WithLimit(10).
	WithArgs(args...).
	Build()
```

> Note: You can only use one nullable column in the sort, due to complexity of the query, it will beat the purpose of using cursor pagination in the first place.

### Ensuring Unique Ordering

Cursor pagination requires that the ordering is based on at least one unique column or a combination of columns that are unique. Kuysor does not validate the uniqueness of the order columns - it is the user's responsibility to ensure that the ordering criteria are unique.

To avoid issues, always include the primary key as the last ordering column when defining your pagination rules. This ensures that even if your main sorting column contains duplicate values (including NULL), pagination remains stable.


### Paginating Inside a CTE (`WithCTETarget`)

Some queries use a CTE (Common Table Expression) to pre-filter rows, and the pagination clauses — cursor `WHERE` condition, `ORDER BY`, and `LIMIT` — must go **inside the CTE body** rather than the outer `SELECT`. This is common when:

- The CTE performs expensive filtering and you want keyset pagination to operate on the pre-filtered set.
- The main query joins the CTE to other tables, adds aggregations, or computes columns that should not interfere with the keyset `WHERE` condition.

#### The Problem

Without `WithCTETarget`, Kuysor places all modifications on the outermost `SELECT`. In a CTE-driven query this is wrong: the cursor `WHERE` clause refers to columns that only exist inside the CTE, and the `LIMIT` would apply after all joins/aggregations rather than before.

```sql
-- ❌ Without WithCTETarget: WHERE / ORDER BY / LIMIT land on the outer SELECT
WITH filtered_ticket AS (
    SELECT t.id
    FROM ticket t
    WHERE t.status = ?
    -- ORDER BY and LIMIT should go here, inside the CTE
)
SELECT t.id, t.code, ...
FROM filtered_ticket ft
JOIN ticket t ON t.id = ft.id
GROUP BY t.id
```

#### Basic Usage

Use `WithCTETarget("cte_name")` to tell Kuysor which CTE body to modify.  
Kuysor locates the named CTE (case-insensitive), routes the cursor `WHERE`, `ORDER BY`, and `LIMIT` inside it, and mirrors the same `ORDER BY` on the main query so the final result set is returned in a consistent order.

```go
query := `
    WITH filtered_ticket AS (
        SELECT t.id
        FROM ticket t
        WHERE t.status = ?
    )
    SELECT t.id, t.code
    FROM filtered_ticket ft
    JOIN ticket t ON t.id = ft.id
    GROUP BY t.id
`

// First page
res, err := kuysor.
    NewQuery(query, kuysor.Cursor).
    WithCTETarget("filtered_ticket"). // ← route modifications into this CTE
    WithOrderBy("-t.id").             // descending by t.id
    WithLimit(10).
    WithArgs("active").               // argument for WHERE t.status = ?
    Build()
```

**Generated SQL — first page:**
```sql
WITH filtered_ticket AS (
    SELECT t.id FROM ticket t
    WHERE t.status = ?
    ORDER BY t.id DESC   -- ← appended inside CTE body
    LIMIT ?              -- ← appended inside CTE body (limit+1 = 11)
)
SELECT t.id, t.code
FROM filtered_ticket ft JOIN ticket t ON t.id = ft.id
GROUP BY t.id
ORDER BY t.id DESC       -- ← mirrored on main query (default behaviour)
```
```go
Args: ["active", 11]
```

**Generated SQL — next page** (cursor passed via `WithCursor`):
```sql
WITH filtered_ticket AS (
    SELECT t.id FROM ticket t
    WHERE (t.status = ?) AND (t.id < ?)  -- ← cursor condition appended inside CTE
    ORDER BY t.id DESC
    LIMIT ?
)
SELECT t.id, t.code ...
GROUP BY t.id
ORDER BY t.id DESC
```
```go
Args: ["active", "<last_id>", 11]
```

**Generated SQL — previous page** (prev cursor passed via `WithCursor`):
```sql
WITH filtered_ticket AS (
    SELECT t.id FROM ticket t
    WHERE (t.status = ?) AND (t.id > ?)  -- ← cursor condition appended inside CTE
    ORDER BY t.id ASC   -- ← direction reversed for prev page, inside CTE
    LIMIT ?
)
SELECT t.id, t.code ...
GROUP BY t.id
ORDER BY t.id ASC       -- ← reversed direction mirrored on main query
```
```go
Args: ["active", "<first_id>", 11]
```

> **Note:** For previous-page queries the result set is automatically re-reversed by `SanitizeStruct` / `SanitizeMap`, so the caller always receives rows in the original sort order.

#### Per-Clause Routing with `CTEOptions`

By default Kuysor uses sensible routing for each clause:

| Clause | Default routing |
|---|---|
| `ORDER BY` | Both CTE body **and** main query (`CTETargetModeBoth`) |
| `LIMIT` / `OFFSET` | CTE body only (`CTETargetModeCTE`) |
| Cursor `WHERE` | CTE body only (`CTETargetModeCTE`) |

You can override any of these by passing a `CTEOptions` value as the second argument to `WithCTETarget`:

```go
res, err := kuysor.
    NewQuery(query, kuysor.Cursor).
    WithCTETarget("filtered_ticket", kuysor.CTEOptions{
        OrderBy:     kuysor.CTETargetModeBoth, // ORDER BY in CTE + mirrored on main (default)
        LimitOffset: kuysor.CTETargetModeCTE,  // LIMIT only inside CTE (default)
        Where:       kuysor.CTETargetModeCTE,  // cursor WHERE only inside CTE (default)
    }).
    WithOrderBy("-t.id").
    WithLimit(10).
    WithArgs("active").
    Build()
```

The three routing modes available for each clause are:

| Mode | Constant | Description |
|---|---|---|
| Default | `CTETargetModeDefault` (zero value) | Uses the natural default for that clause (see table above). |
| CTE only | `CTETargetModeCTE` | Clause is injected inside the named CTE body only. |
| Main only | `CTETargetModeMain` | Clause is injected on the outer SELECT only; the CTE is not modified. |
| Both | `CTETargetModeBoth` | Clause is injected in both places. Arguments are duplicated when `Both` is used for `LIMIT`/`OFFSET` or `WHERE`. |

**Example — LIMIT on main query only** (useful when the CTE already has its own LIMIT):
```go
WithCTETarget("filtered_ticket", kuysor.CTEOptions{
    LimitOffset: kuysor.CTETargetModeMain,
})
```

**Example — ORDER BY inside CTE only, no mirroring on main query:**
```go
WithCTETarget("filtered_ticket", kuysor.CTEOptions{
    OrderBy: kuysor.CTETargetModeCTE,
})
```

**Example — cursor WHERE on both CTE and main query** (e.g. to apply double filtering):
```go
WithCTETarget("filtered_ticket", kuysor.CTEOptions{
    Where: kuysor.CTETargetModeBoth,
})
// Note: cursor arg is duplicated → Args: ["active", "<cursor_id>", "<cursor_id>", 11]
```

#### Multiple CTEs

If your query has several CTEs, `WithCTETarget` modifies **only the named one**; all other CTEs are left unchanged:

```go
query := `
    WITH stats AS (
        SELECT user_id, COUNT(*) AS total FROM orders GROUP BY user_id
    ),
    active_users AS (
        SELECT u.id FROM users u WHERE u.status = ?
    )
    SELECT u.id, u.name, s.total
    FROM active_users au
    JOIN users u ON u.id = au.id
    LEFT JOIN stats s ON s.user_id = u.id
`

res, err := kuysor.
    NewQuery(query, kuysor.Cursor).
    WithCTETarget("active_users"). // only active_users is modified; stats is untouched
    WithOrderBy("u.id").
    WithLimit(20).
    WithArgs("active").
    Build()
```

#### Constraints

| Constraint | Behaviour |
|---|---|
| Query must contain a `WITH` clause | `Build()` returns an error if no CTE is present. |
| CTE name must exist in the query | `Build()` returns an error if the name is not found (case-insensitive match). |
| Sort columns must appear in the CTE's `SELECT` | Required for cursor generation — the same rule as standard cursor pagination. |
| One nullable sort column maximum | Same limitation as standard cursor pagination. |


### Configuring Options 

Kuysor provides flexible configuration / options to customize default behaviors, such as query placeholder types, default limits, struct tags, and null sorting methods. These settings can be applied globally, per instance, or at the query level to accommodate various needs.

#### Configuring Global Options

You can set global options that apply to all Kuysor instances within your application. This is useful for standardizing behavior across queries.
```go
package main

import (
    "fmt"
    "github.com/redhajuanda/kuysor"
)

func main() {
    // (Optional) Configure Kuysor with custom settings, only call once at the beginning of your program
    kuysor.SetGlobalOptions(kuysor.Options{
        PlaceHolderType: kuysor.Question,  // Default: `Question`. Options: `Question`, `Dollar`, `At`.
        DefaultLimit:    10,               // Default: 10. Specifies the default query limit.
        StructTag:       "kuysor",         // Default: `kuysor`. Defines the struct tag used for field mapping.
        NullSortMethod:  kuysor.BoolSort,  // Default: `BoolSort`. Options: `BoolSort`, `FirstLast`, `CaseWhen`.
    })
}
```
Note: Global settings affect all queries unless overridden at the instance or query level.

#### Creating Instances with Custom Options

In applications that interact with multiple databases, you may need different Kuysor configurations for each database connection. Instead of modifying global settings, you can create custom instances with `NewInstance()`.

Example: Different Placeholder Formats for PostgreSQL & MySQL

```go
package main

import (
 "github.com/redhajuanda/kuysor"
)

func main() {
 // PostgreSQL instance using `$` placeholders
 ksPostgres := kuysor.NewInstance(kuysor.Options{
  PlaceHolderType: kuysor.Dollar, // Use `$` for parameter substitution
  DefaultLimit:    5,
 })

 // MySQL instance using `?` placeholders
 ksMysql := kuysor.NewInstance(kuysor.Options{
  PlaceHolderType: kuysor.Question, // Use `?` for parameter substitution
  DefaultLimit:    10,
 })
}
```

#### Overriding Options at the Query Level

Some options can be specified directly when building a query. These query-level settings take precedence over both global and instance options. 

These options are:
- PlaceHolderType: Use Method `WithPlaceHolderType` to set the placeholder type for the query.
- Limit: Use Method `WithLimit` to set the limit for the query.
- NullSortMethod: Use Method `WithNullSortMethod` to set the null sort method for the query.

Example:
```go
package main

import (
 "github.com/redhajuanda/kuysor"
)

func main() {
  ks, err := kuysor.
    NewQuery("SELECT * FROM account", kuysor.Cursor).
    WithPlaceHolderType(kuysor.Dollar). // use $ placeholder
    WithLimit(5). // set the limit to 5
    WithNullSortMethod(kuysor.FirstLast) // set the null sort method to FirstLast
}
```

### Processing the Query Result With `SanitizeMap()`

```go
package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redhajuanda/kuysor"
)

func main() {

	// connect mysql
	db, err := sql.Open("mysql", "xxx:xxx@tcp(localhost:3300)/xxx")
	if err != nil {
		panic(err)
	}

	query := `SELECT a.id as id, a.code as code, a.name as name FROM account a WHERE a.status = ?`
	args := []any{"active"}

	ks, err := kuysor.
		NewQuery(query, kuysor.Cursor).
		WithLimit(10).
		WithOrderBy("code", "-id").
		WithArgs(args...).
		Build()
	if err != nil {
		panic(err)
	}

	rows, err := db.Query(ks.Query, ks.Args...)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var result []map[string]interface{}

	for rows.Next() {
		var (
			id   int
			code string
			name *string
			row  = make(map[string]interface{})
		)

		err = rows.Scan(&id, &code, &name)
		if err != nil {
			panic(err)
		}

		row["id"] = id
		row["code"] = code
		row["name"] = name

		result = append(result, row)
	}

	nextCursor, prevCursor, err := ks.SanitizeMap(&result)
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
	fmt.Println(nextCursor)
	fmt.Println(prevCursor)
}
```

## Advance Usage

I wrote a brief article about how to use Kuysor in more detail, you can read it [here](https://medium.com/@redhajuanda/golang-kuysor-making-cursor-pagination-suck-less-7d71dead0c99).

## Performance

Kuysor is designed for performance and efficiency. Benchmark results on Apple M1 show excellent performance characteristics for production use.

### 🚀 **Core Performance Metrics**

| Operation | Time (μs) | Memory (KB) | Allocations | Performance |
|-----------|-----------|-------------|-------------|-------------|
| **Basic Query Build** | 64.1μs | 46.8 KB | 473 | ⭐⭐⭐⭐⭐ Excellent |
| **Query with Cursor** | 131.3μs | 83.2 KB | 809 | ⭐⭐⭐⭐ Very Good |
| **Complex Query** | 160.2μs | 53.2 KB | 488 | ⭐⭐⭐⭐ Very Good |
| **Cursor Parsing** | 1.65μs | 0.9 KB | 20 | ⭐⭐⭐⭐⭐ Blazing Fast |
| **Result Sanitization** | 1.45μs | 1.4 KB | 22 | ⭐⭐⭐⭐⭐ Blazing Fast |

### 📈 **Scalability**

**Sort Performance by Column Count:**
- 1 column: 113ns (3 allocs)
- 2 columns: 206ns (5 allocs) 
- 4 columns: 352ns (8 allocs)
- 8 columns: 688ns (13 allocs)

✅ **Linear scaling** - Performance degrades gracefully with complexity.

**Nullable Sort Method Comparison:**
- **FirstLast**: 70.3μs (fastest, PostgreSQL/Oracle)
- **CaseWhen**: 72.4μs (MySQL < 8.0)  
- **BoolSort**: 83.4μs (most compatible)

### ⚡ **Throughput Estimates**

| Operation | Ops/sec (single core) | Daily Capacity |
|-----------|----------------------|----------------|
| **Basic Query Build** | ~15,600 | ~1.3 billion |
| **With Cursor** | ~7,600 | ~656 million |
| **Cursor Parsing** | ~606,000 | ~52 billion |

### 🔧 **Performance Tips**

1. **Use Question mark placeholders** (fastest: 909ns vs 1316ns for Dollar)
2. **Keep sort columns under 4** for optimal performance
3. **Use BoolSort** for maximum database compatibility

### Benchmarking

Run performance benchmarks on your system:

```bash
# Run all benchmarks
make bench

# Run specific benchmark categories
go test -bench=BenchmarkQueryBuild -benchmem ./...
go test -bench=BenchmarkCursor -benchmem ./...
go test -bench=BenchmarkSort -benchmem ./...

# Generate performance comparison
go test -bench=. -count=5 -benchmem ./... > baseline.txt
```

**Benchmark Categories:**
- Query building performance across complexity levels
- Cursor parsing and generation efficiency  
- Sort parsing with 1-8 columns
- Memory allocation patterns and optimization
- Concurrent access and thread safety
- SQL modification operations (WHERE, ORDER BY, LIMIT)
- Count query conversion performance

📋 **For detailed performance analysis and optimization guides, see [PERFORMANCE.md](PERFORMANCE.md)**

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- Setting up your development environment
- Code standards and best practices
- Testing guidelines
- How to submit pull requests
- Reporting issues

## Support

- 📖 **Documentation**: Check the examples above and the [contributing guide](CONTRIBUTING.md)
- 🐛 **Bug Reports**: [Open an issue](https://github.com/redhajuanda/kuysor/issues) with a detailed description
- 💡 **Feature Requests**: [Open an issue](https://github.com/redhajuanda/kuysor/issues) with your use case
- 💬 **Questions**: Start a [discussion](https://github.com/redhajuanda/kuysor/discussions) for general questions

## Limitation

- It requires that the ordering is based on at least one unique column or a combination of columns that are unique. 
- Each column in the sort must be included in the SELECT statement, and the column names must match exactly. This is because Kuysor uses the column values to generate the next and previous cursors.
- Only one nullable column is allowed in the sort, due to complexity of the query, it will beat the purpose of using cursor pagination in the first place.
- You need to handle indexing properly to make the query efficient.

## License

This project is licensed under [LICENSE](LICENSE) - see the LICENSE file for details.

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
		NewQuery(query).
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
		NewQuery(query).
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
	NewQuery(query).
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
    NewQuery("SELECT * FROM account").
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
		NewQuery(query).
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

I wrote a brief article about how to use Kuysor in more detail, you can read it [here](https://medium.com/@redhajuanda/kuysor-a-simple-sdk-for-cursor-pagination-in-golang-4f1a0b2c3d7e).

## Limitation
- It requires that the ordering is based on at least one unique column or a combination of columns that are unique. 
- Each column in the sort must be included in the SELECT statement, and the column names must match exactly. This is because Kuysor uses the column values to generate the next and previous cursors.
- Only one nullable column is allowed in the sort, due to complexity of the query, it will beat the purpose of using cursor pagination in the first place.
- You need to handle indexing properly to make the query efficient.
# kuysor

Kuysor is a lightweight Go SDK designed to facilitate cursor-based pagination queries. It supports ordering by multiple columns in both ascending and descending order.

## What is Cursor Pagination?
Cursor-based pagination (aka Keyset Pagination) is a more efficient and scalable alternative to offset-based pagination, particularly for large datasets. Instead of specifying an offset, it uses a cursor—a unique identifier from the last retrieved record—to determine the starting point for the next / previous set of results.

## How kuysor works?
1. Kuysor modifies your SQL query to support cursor-based pagination, allowing you to set limits, sorting, and cursors.
2. You execute the modified query to fetch paginated data from your database.
3. Kuysor then sanitizes the query result and generates the next and previous cursors.

## Installation

```bash
go get github.com/redhajuanda/kuysor
```

## Usage

### Configuring Options (Optional)

You can configure Kuysor with default settings such as database dialect, default limit, and struct tags.

```go
package main

import (
    "fmt"
    "github.com/redhajuanda/kuysor"
)

func main() {
    kuysor.SetOptions(kuysor.Options{
        Dialect: kuysor.MySQL, // default is MySQL (currently only support MySQL)
        DefaultLimit: 10, // default is 10
        StructTag: "kuysor", // default is `kuysor`
    })
}
```

### Retrieving the First Page

For initial queries, you can set the limit and sorting order. 

```go
package main

import (
    "fmt"
    "github.com/redhajuanda/kuysor"
)

func main() {

    query := `SELECT a.id, a.code, a.name FROM account a WHERE a.status = ?`
    args := []interface{}{"active"}

	ks := kuysor.
		New(query).
		WithLimit(10). // limit is optional, if not set, it will check the default limit from `.SetOptions` or if also not set, it will use 10 as default limit
		WithSort("a.code", "-a.id"). // sort is required for cursor pagination
        WithArgs(args...) // args is your query arguments, it is needed because kuysor will also modify the args

	finalQuery, finalArgs, err := ks.Build() // kuysor returns the final query and the args 
	if err != nil {
		panic(err)
	}
}
```

Return query:
```sql
SELECT a.id, a.code, a.name FROM account a WHERE a.status = ? ORDER BY a.code ASC, a.id DESC LIMIT ?
```
Return args:
```go
["active", 11]
```
In the first page, kuysor modified your query to include the limit and sorting order. Kuysor also applies SQL placeholders to the limit to prevent SQL injection.
That's why kuysor requires you to pass the args, so it can modify the args as well.

### Sorting Rules
- Use WithSort to define one or multiple sorting columns.
- Prefix columns with `-` for descending order and `+` for ascending order (default is ascending).
- If sorting involves nullable columns, specify them explicitly by adding `nullable` after the column name.
```go
WithSort("+name nullable", "code", "-id")
```
Return query:
```sql
ORDER BY name IS NULL ASC, name ASC, code ASC, id DESC
```
Note: `null` value will be in the bottom of the order if ascending, and in the top if descending.

### Tie Breaker Column

Cursor pagination requires that the ordering is based on at least one unique column or a combination of columns that are unique.
To make it simple, I recommend to always use the last column as a tie breaker by adding primary key (id) as the last column in the sort even if you can guarantee that the combination of the ordered columns is unique.
Tie breaker column also can be set to use ascendant or descendant order.

```go
WithSort("name", "code", "-id")
```
or
```go
WithOrder("name", "code", "id")
```

### Fetching and Sanitizing Results

Even if you set the limit to 10, Kuysor will automatically set it to 11. This is because it needs to check if more data is available. If the result contains 11 items, there are more pages to fetch; if it's less than 11, there are no more pages to load.

Since cursor pagination can sometimes return extra data or even reverse the order of results, data sanitization is necessary. To handle this, you can use the `SanitizeStruct()` or `SanitizeMap()` function. These functions will trim the extra data, correct the order when using a previous cursor, and generate the next and previous cursors properly.

Here's an example of how to use `SanitizeStruct()` to sanitize the query result:
```go
package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redhajuanda/kuysor"
)

type Account struct {
	ID     int     `kuysor:"a.id"`
	Code   string  `kuysor:"a.code"`
	Name   string  `kuysor:"a.name"`
}

func main() {

	// connect mysql
	db, err := sql.Open("mysql", "mariadb:mariadb@tcp(localhost:3300)/db_app")
	if err != nil {
		panic(err)
	}

	query := `SELECT a.id, a.code, a.name FROM account a WHERE a.status = ?`
	args := []interface{}{"active"}

	ks := kuysor.
		New(query).
		WithLimit(10).
		WithSort("a.code", "-a.id").
		WithArgs(args...)

	finalQuery, finalArgs, err := ks.Build()
	if err != nil {
		panic(err)
	}

	// execute the query and get the result
	rows, err := db.Query(finalQuery, finalArgs...)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var result = make([]Account, 0)

	for rows.Next() {
		var row Account
		err = rows.Scan(&row.ID, &row.Code, &row.Name)
		if err != nil {
			panic(err)
		}
		result = append(result, row)
	}

	nextCursor, prevCursor, err := ks.SanitizeStruct(&result) // pass the pointer of the result, so kuysor can modify it
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
	fmt.Println(nextCursor)
	fmt.Println(prevCursor)
}
```
To generate the next and previous cursor, Kuysor automatically checks your struct result. Since the struct field names may differ from the column names in the query, you need to add a kuysor tag to the struct fields to match the column names.

```go
type Account struct {
    ID   int    `kuysor:"a.id"`
    Code string `kuysor:"a.code"`
    Name string `kuysor:"a.name"`
}
```

`SanitizeStruct()` returns the next and previous cursors. If there is no more data to fetch, the next cursor will be empty. If the cursor is on the first page, the previous cursor will be empty.

### Retrieving the Next Page

To fetch the next page, simply include the cursor from the previous query result.

```go
package main

import (
    "fmt"
    "github.com/redhajuanda/kuysor"
)

func main() {

    query := `SELECT a.id, a.code, a.name FROM account a WHERE a.status = ?`
    args := []interface{}{"active"}
    
    ks := kuysor.
        New(query).
        WithLimit(10). 
        WithSort("a.code", "-a.id"). 
        WithArgs(args...).
        WithCursor("xxx") // the query will start from the cursor

    finalQuery, err := ks.Build()
    if err != nil {
        panic(err)
    }
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

	query := `SELECT a.id as id, a.code as code, a.name as name FROM tester a WHERE a.status = ?`
	args := []any{"active"}

	ks := kuysor.
		New(query).
		WithLimit(10).
		WithSort("code", "-id").
		WithArgs(args...)

	finalQuery, finalArgs, err := ks.Build()
	if err != nil {
		panic(err)
	}

	rows, err := db.Query(finalQuery, finalArgs...)
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

### Limitation
- Kuysor currently only supports MySQL dialect
- It requires that the ordering is based on at least one unique column or a combination of columns that are unique. 
- Each column in the sort must be included in the SELECT statement, and the column names must match exactly. This is because Kuysor uses the column values to generate the next and previous cursors.
- Only one nullable column is allowed in the sort, due to complexity of the query, it will beat the purpose of using cursor pagination in the first place.
- You need to handle indexing properly to make the query efficient.

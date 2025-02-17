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

    query := `SELECT a.id, a.code, a.name FROM account a`
    
	ky := kuysor.
		New(query).
		WithLimit(10). // limit is optional, if not set, it will check the default limit from `.SetOptions` or if also not set, it will use 10 as default limit
		WithSort("a.code", "-a.id") // sort is required for cursor pagination

	finalQuery, err := ky.Build()
	if err != nil {
		panic(err)
	}
}
```

Generated SQL Query:
```sql
SELECT a.id, a.code, a.name FROM account a ORDER BY a.code ASC, a.id DESC LIMIT 11
```

### Sorting Rules
- Use WithSort to define multiple sorting columns.
- Prefix columns with - for descending order and + for ascending order (default is ascending).
- If sorting involves nullable columns, specify them explicitly.
```go
WithSort("+name nullable", "code", "-id")
```
Generated SQL Query:
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

### Retrieving the Next Page

To fetch the next page, include the cursor from the previous query result.

```go
package main

import (
    "fmt"
    "github.com/redhajuanda/kuysor"
)

func main() {

    query := `SELECT a.id, a.code, a.name FROM account a`
    
    ky := kuysor.
        New(query).
        WithLimit(10). // limit is optional, if not set, it will check the default limit from `.SetOptions` or if also not set, it will use 10 as default limit
        WithSort("a.code", "-a.id"). // sort is required for cursor pagination
        WithCursor("xxx") // the query will start from the cursor

    finalQuery, err := ky.Build()
    if err != nil {
        panic(err)
    }
}
```

Generated SQL Query:
```sql
SELECT a.id, a.`code`, a.`name` FROM account as a 
WHERE a.`code` > 'C' 
OR (a.`code` = 'C' AND a.id < 3) 
ORDER BY a.`code` ASC, a.id DESC 
LIMIT 11
```

### Processing the Query Result
Kuysor supports mapping results into either a slice of structs or a slice of map[string]interface{}.


Struct Mapping

```go
package main

type Account struct {
    ID   int    `kuysor:"a.id"`
    Code string `kuysor:"a.code"`
    Name string `kuysor:"a.name"`
}
```
Ensure that struct tags align with the column names used in sorting.

### Full Example with `db/sql` using map
```go
package main

import (
    "database/sql"
    "fmt"
    "github.com/redhajuanda/kuysor"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    db, err := sql.Open("mysql", "...")
    if err != nil {
        panic(err)
    }

    query := `SELECT a.id, a.code, a.name FROM account a`

    ky := kuysor.
        New(query).
        WithLimit(10). 
        WithSort("a.code", "-a.id") 

    finalQuery, err := ky.Build()

    rows, err := db.Query(finalQuery)

    if err != nil {
        panic(err)
    }

    defer rows.Close()

    var result []map[string]interface{}

    for rows.Next() {
        row := make(map[string]interface{})
        err = rows.Scan(&row["id"], &row["code"], &row["name"])
        if err != nil {
            panic(err)
        }
        result = append(result, row)
    }

    nextCursor, prevCursor, err := ky.SanitizeMap(result)
    if err != nil {
        panic(err)
    }

    fmt.Println(result)
    fmt.Println(nextCursor)
    fmt.Println(prevCursor)

}
```

### Full Example with `db/sql` using struct
```go   
package main

import (
    "database/sql"
    "fmt"
    "github.com/redhajuanda/kuysor"
    _ "github.com/go-sql-driver/mysql"
)

type Account struct {
    ID   int    `kuysor:"a.id"`
    Code string `kuysor:"a.code"`
    Name string `kuysor:"a.name"`
}

func main() {
    db, err := sql.Open("mysql", "...")
    if err != nil {
        panic(err)
    }

    query := `SELECT a.id, a.code, a.name FROM account a`

    ky := kuysor.
        New(query).
        WithLimit(10). 
        WithSort("a.code", "-a.id") 

    finalQuery, err := ky.Build()

    rows, err := db.Query(finalQuery)

    if err != nil {
        panic(err)
    }

    defer rows.Close()

    var result []Account

    for rows.Next() {
        var row Account
        err = rows.Scan(&row.ID, &row.Code, &row.Name)
        if err != nil {
            panic(err)
        }
        result = append(result, row)
    }

    nextCursor, prevCursor, err := ky.SanitizeStruct(result)
    if err != nil {
        panic(err)
    }

    fmt.Println(result)
    fmt.Println(nextCursor)
    fmt.Println(prevCursor)

}
```

### Full Example with `jmoiron/sqlx` using struct
```go
package main

import (
    "fmt"
    "github.com/jmoiron/sqlx"
    "github.com/redhajuanda/kuysor"
    _ "github.com/go-sql-driver/mysql"
)

type Account struct {
    ID   int    `kuysor:"a.id"`
    Code string `kuysor:"a.code"`
    Name string `kuysor:"a.name"`
}

func main() {
    db, err := sqlx.Connect("mysql", "...")
    if err != nil {
        panic(err)
    }

    query := `SELECT a.id, a.code, a.name FROM account a`

    ky := kuysor.
        New(query).
        WithLimit(10). 
        WithSort("a.code", "-a.id") 

    finalQuery, err := ky.Build()

    var result []Account

    err = db.Select(&result, finalQuery)
    if err != nil {
        panic(err)
    }

    nextCursor, prevCursor, err := ky.SanitizeStruct(result)
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
- Only one nullable column is allowed in the sort, due to complexity of the query, it will beat the purpose of using cursor pagination in the first place.
- You need to handle indexing properly to make the query efficient.

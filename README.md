# kuysor

kuysor is a relatively small sdk that helps you to build cursor-based pagination query
It also supports ordering by multiple column in either ascending or descending order.

## What is Cursor Pagination?
Cursor-based pagination (aka Keyset Pagination) is a more efficient and scalable alternative to offset-based pagination, particularly for large datasets. Instead of specifying an offset, it uses a cursor—a unique identifier from the last retrieved record—to determine the starting point for the next / previous set of results.

## How kuysor works?
- first, kuysor will help you to automatically modify your query to support cursor-based pagination. You can set the limit, sort/order, and cursor.
- then you can use the query to fetch the data from your database
- finally, you can ask kuysor again to sanitize the result from the database and generate the next and previous cursor.

## Installation

```bash
go get github.com/redhajuanda/kuysor
```

## Usage

### Set Options

Set options is optional, but if you want to set the default limit, struct tag or dialect, you can use this function at the beginning of your application.

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



### First Page

When querying the first page, you can set the limit and sort/order. You don't need to set the cursor for the first page as the cursor is not available yet.

```go
package main

import (
    "fmt"
    "github.com/redhajuanda/kuysor"
)

func main() {

    query := `SELECT a.id, a.code, a.name FROM account a`
    
	pg := kuysor.
		New(query).
		WithLimit(10). // limit is optional, if not set, it will check the default limit from `.SetOptions` or if also not set, it will use 10 as default limit
		WithSort("a.name", "a.code", "-a.id") // sort is required for cursor pagination

	finalQuery, err := pg.Build()
	if err != nil {
		panic(err)
	}
}
```

result:
```sql
SELECT a.id, a.code, a.name FROM account a ORDER BY a.name ASC, a.code ASC, a.id DESC LIMIT 11
```

### Sorting Rules
`WithSort` accepts multiple columns to sort. To set the order to be descending you can add prefix `-` to the column, and `+` for ascending, the default is ascending.
```go
    WithSort("+name", "code", "-id")
```
Result:
```sql
ORDER BY name ASC, code ASC, id DESC 
```
Cursor Pagi


### Tie Breaker Column

With multiple sort columns, cursor pagination requires that the ordering is based on at least one unique column or a combination of columns that are unique.
To make it simple, I recommend to always use the last column as a tie breaker by adding primary key (id) as the last column in the sort even if you can guarantee that the combination of the ordered columns is unique.
Tie breaker column also can be set to use ascendant or descendant order.

```go
    WithSort("name", "code", "-id")
```
or
```go
    WithOrder("name", "code", "id")
```

### Nulls Order



### Next Page

When querying the next page, you can set the cursor from the previous query.

```go
package main

import (
    "fmt"
    "github.com/redhajuanda/kuysor"
)

func main() {

    query := `SELECT a.id, a.code, a.name FROM account a`
    
    pg := kuysor.
        New(query).
        WithLimit(10). // limit is optional, if not set, it will check the default limit from `.SetOptions` or if also not set, it will use 10 as default limit
        WithSort("a.name", "a.code", "-a.id"). // sort is required for cursor pagination
        WithCursor("xxx") // the query will start from the cursor

    finalQuery, err := pg.Build()
    if err != nil {
        panic(err)
    }
}
```

result:
```sql

# kuysor

kuysor is 
It also supports ordering by any column on the relation in either ascending or descending order.

## Installation

```bash
go get github.com/redhajuanda/kuysor
```

## Usage

### Set Options

Set options is optional, but if you want to set the default limit and dialect, you can use this function at the beginning of your application.

```go
package main

import (
    "fmt"
    "github.com/redhajuanda/kuysor"
)

func main() {
    kuysor.SetOptions(kuysor.Options{
        Dialect: kuysor.MySQL, // this is optional, default is MySQL (currently only support MySQL)
        DefaultLimit: 10, 
        StructTag: "kuysor", // this is optional, default is `kuysor`
    })
}
```



### First Page

When querying the first page, you don't need to set the cursor, this is needed to make sure the order and the limit is correct.

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

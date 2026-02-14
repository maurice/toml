# github.com/maurice/toml

> [!WARNING]
>
> This library is in development, is incomplete, almost certainly has bugs, and is subject to change without notice!

A TOML library for Go that parses into a Concrete Syntax Tree (CST), preserving whitespace, comments, and formatting for lossless round-trip editing.

Compliant with TOML v1.1.0. Zero dependencies. Verified against [toml-test](https://github.com/toml-lang/toml-test).

## Features

- **CST-based** -- preserves whitespace, comments, and original formatting
- **Round-trip** -- parse, modify, and serialize without losing formatting
- **TOML v1.1.0** -- supports `\e` and `\xHH` escapes, multi-line inline tables, trailing commas, optional seconds in datetimes
- **Query by path** -- find keys and tables using dotted paths
- **Value extraction** -- get typed Go values from nodes
- **Programmatic construction** -- build documents from scratch with constructors
- **Mutation** -- set, delete, append, and insert nodes
- **Zero dependencies** -- only the Go standard library

## Install

```
go get github.com/maurice/toml
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/maurice/toml"
)

func main() {
    // Parse
    doc, err := toml.Parse([]byte(`
[server]
host = "localhost"
port = 8080
`))
    if err != nil {
        panic(err)
    }

    // Query
    host := doc.Get("server.host").Val.(*toml.StringNode).Value()
    fmt.Println(host) // localhost

    // Modify
    doc.Get("server.port").SetValue(toml.NewInteger(9090))

    // Add a new key
    tbl := doc.Table("server")
    tbl.Append(toml.NewKeyValue("debug", toml.NewBool(true)))

    // Serialize -- original formatting is preserved
    fmt.Print(doc.String())
}
```

## Parsing

`Parse` takes a byte slice and returns a `*Document` and an error:

```go
doc, err := toml.Parse(data)
if err != nil {
    // err may be a *toml.ParseError with line/column info
    var pe *toml.ParseError
    if errors.As(err, &pe) {
        fmt.Printf("line %d, col %d: %s\n", pe.Line, pe.Column, pe.Message)
    }
}
```

Passing `nil` returns `toml.ErrNilInput`. An empty byte slice returns an empty document.

The parser validates:

- UTF-8 encoding
- String escape sequences and control characters
- Number formats (leading zeros, underscores, prefix integers)
- Date/time ranges (month, day, hour, minute, second)
- Semantic rules (duplicate keys, table redefinition, inline table immutability)

## Querying

### Finding values

Use `Document.Get` with a dotted path to find key-value pairs anywhere in the document:

```go
// Top-level key
kv := doc.Get("title")

// Key inside a table
kv := doc.Get("server.host")

// Dotted key
kv := doc.Get("a.b.c")
```

Use `Table.Get` or `ArrayOfTables.Get` to search within a specific section:

```go
tbl := doc.Table("server")
kv := tbl.Get("port")
```

Use `InlineTableNode.Get` for inline tables:

```go
kv := doc.Get("point")
it := kv.Val.(*toml.InlineTableNode)
x := it.Get("x")
```

All `Get` methods return `nil` if no matching key is found.

### Finding tables

```go
// Find first table matching the path
tbl := doc.Table("server")

// Get all tables
tables := doc.Tables()

// Get all arrays-of-tables
aots := doc.ArraysOfTables()
```

### Extracting Go values

Leaf nodes have typed value extraction methods:

```go
// Strings -- unquotes and processes escape sequences
s := kv.Val.(*toml.StringNode).Value() // string

// Integers -- handles decimal, hex (0x), octal (0o), binary (0b), underscores
n, err := kv.Val.(*toml.NumberNode).Int() // int64

// Floats -- handles floats, integers, inf, nan
f, err := kv.Val.(*toml.NumberNode).Float() // float64

// Booleans
b := kv.Val.(*toml.BooleanNode).Value() // bool
```

### Walking the tree

`Document.Walk` traverses the entire CST in pre-order. Return `false` from the visitor to stop traversal:

```go
doc.Walk(func(n toml.Node) bool {
    if n.Type() == toml.NodeComment {
        fmt.Println("Comment:", n.Text())
    }
    return true // continue
})
```

## Modifying Documents

### Updating values

Use `KeyValue.SetValue` to replace a value. This updates both the typed node and the raw text used for serialization:

```go
kv := doc.Get("port")
kv.SetValue(toml.NewInteger(9090))
```

You can change the type of a value:

```go
kv.SetValue(toml.NewString("dynamic"))
```

### Adding content

Construct new nodes with the `New*` functions:

```go
toml.NewString("hello")       // "hello"
toml.NewInteger(42)            // 42
toml.NewFloat(3.14)            // 3.14
toml.NewBool(true)             // true
toml.NewKeyValue("key", val)   // key = val\n
toml.NewTable("section")       // [section]\n
toml.NewTable("a", "b")        // [a.b]\n
```

Keys that aren't valid bare keys must be quoted using TOML syntax:

```go
toml.NewKeyValue(`"has spaces"`, toml.NewString("val"))
// "has spaces" = "val"

toml.NewKeyValue(`site."google.com"`, toml.NewBool(true))
// site."google.com" = true
```

Append nodes to a document or table:

```go
doc.Append(toml.NewKeyValue("name", toml.NewString("Alice")))

tbl := doc.Table("server")
tbl.Append(toml.NewKeyValue("port", toml.NewInteger(8080)))
```

### Inserting at a position

Insert a node at a specific index in a document or table:

```go
doc.InsertAt(0, toml.NewKeyValue("first", toml.NewBool(true)))

tbl.InsertAt(1, toml.NewKeyValue("middle", toml.NewInteger(5)))
```

Out-of-range indices are clamped (negative becomes 0, beyond length appends).

### Deleting content

Remove key-value pairs by path:

```go
doc.Delete("old_key")           // top-level key
doc.Delete("server.deprecated") // key inside a table

tbl.Delete("key")               // from a specific table
aot.Delete("key")               // from a specific array-of-tables
```

Remove an entire table:

```go
doc.DeleteTable("old_section")
```

All delete methods return `true` if something was removed, `false` otherwise.

## Serializing

`Document.String()` renders the document back to TOML text:

```go
output := doc.String()
os.WriteFile("config.toml", []byte(output), 0644)
```

For parsed documents, the original formatting (whitespace, comments, quote style) is preserved exactly. New nodes created with constructors use standard formatting (`key = value\n`).

## CST Node Types

| Type                | Node               | Description                     |
| ------------------- | ------------------ | ------------------------------- |
| `NodeDocument`      | `*Document`        | Root container                  |
| `NodeKeyValue`      | `*KeyValue`        | `key = value` pair              |
| `NodeTable`         | `*TableNode`       | `[table.header]` section        |
| `NodeArrayOfTables` | `*ArrayOfTables`   | `[[array.header]]` section      |
| `NodeArray`         | `*ArrayNode`       | `[val1, val2, ...]` value       |
| `NodeInlineTable`   | `*InlineTableNode` | `{key = val, ...}` value        |
| `NodeString`        | `*StringNode`      | String value (all 4 TOML forms) |
| `NodeNumber`        | `*NumberNode`      | Integer or float value          |
| `NodeBoolean`       | `*BooleanNode`     | `true` or `false`               |
| `NodeDateTime`      | `*DateTimeNode`    | Date, time, or datetime value   |
| `NodeComment`       | `*CommentNode`     | `# comment`                     |
| `NodeWhitespace`    | `*WhitespaceNode`  | Spaces, tabs, newlines          |

Every node implements the `Node` interface:

```go
type Node interface {
    Type() NodeType
    Parent() Node
    Children() []Node
    Text() string
}
```

## TOML 1.1 Support

This library supports the following TOML 1.1 features:

- `\e` escape sequence (ESC, U+001B)
- `\xHH` hex escape sequence
- Multi-line inline tables (newlines allowed inside `{ }`)
- Trailing commas in inline tables and arrays
- Optional seconds in times (`07:32` is valid, equivalent to `07:32:00`)

## License

See [LICENSE](LICENSE) file.

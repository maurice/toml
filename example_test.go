package toml_test

import (
	"fmt"

	"github.com/maurice/toml"
)

func ExampleParse() {
	doc, err := toml.Parse([]byte(`name = "Alice"` + "\n"))
	if err != nil {
		panic(err)
	}
	kv := doc.Nodes[0].(*toml.KeyValue)
	fmt.Println(kv.RawKey)
	fmt.Println(kv.Val.Type() == toml.NodeString)
	// Output:
	// name
	// true
}

func ExampleDocument_String() {
	input := "# Config\ntitle = \"My App\"\n"
	doc, _ := toml.Parse([]byte(input))
	fmt.Print(doc.String())
	// Output:
	// # Config
	// title = "My App"
}

func ExampleDocument_Get() {
	doc, _ := toml.Parse([]byte("[server]\nhost = \"localhost\"\nport = 8080\n"))
	kv := doc.Get("server.host")
	fmt.Println(kv.Val.(*toml.StringNode).Value())
	// Output:
	// localhost
}

func ExampleDocument_Table() {
	doc, _ := toml.Parse([]byte("[database]\nport = 5432\n"))
	tbl := doc.Table("database")
	fmt.Println(tbl.RawHeader)
	// Output:
	// database
}

func ExampleDocument_Walk() {
	doc, _ := toml.Parse([]byte("# comment\nkey = 1\n"))
	comments := 0
	doc.Walk(func(n toml.Node) bool {
		if n.Type() == toml.NodeComment {
			comments++
		}
		return true
	})
	fmt.Println(comments)
	// Output:
	// 1
}

func ExampleDocument_Delete() {
	doc, _ := toml.Parse([]byte("a = 1\nb = 2\nc = 3\n"))
	doc.Delete("b")
	fmt.Print(doc.String())
	// Output:
	// a = 1
	// c = 3
}

func ExampleDocument_DeleteTable() {
	doc, _ := toml.Parse([]byte("[keep]\nx = 1\n[remove]\ny = 2\n"))
	doc.DeleteTable("remove")
	fmt.Print(doc.String())
	// Output:
	// [keep]
	// x = 1
}

func ExampleDocument_Append() {
	doc, _ := toml.Parse([]byte("a = 1\n"))
	doc.Append(toml.NewKeyValue("b", toml.NewInteger(2)))
	fmt.Print(doc.String())
	// Output:
	// a = 1
	// b = 2
}

func ExampleDocument_InsertAt() {
	doc, _ := toml.Parse([]byte("a = 1\nc = 3\n"))
	doc.InsertAt(1, toml.NewKeyValue("b", toml.NewInteger(2)))
	fmt.Print(doc.String())
	// Output:
	// a = 1
	// b = 2
	// c = 3
}

func ExampleTableNode_Get() {
	doc, _ := toml.Parse([]byte("[server]\nhost = \"localhost\"\nport = 8080\n"))
	tbl := doc.Table("server")
	kv := tbl.Get("port")
	fmt.Println(kv.Val.Text())
	// Output:
	// 8080
}

func ExampleTableNode_Append() {
	doc, _ := toml.Parse([]byte("[server]\nhost = \"localhost\"\n"))
	tbl := doc.Table("server")
	tbl.Append(toml.NewKeyValue("port", toml.NewInteger(8080)))
	fmt.Print(doc.String())
	// Output:
	// [server]
	// host = "localhost"
	// port = 8080
}

func ExampleKeyValue_SetValue() {
	doc, _ := toml.Parse([]byte("port = 80\n"))
	kv := doc.Get("port")
	kv.SetValue(toml.NewInteger(8080))
	fmt.Print(doc.String())
	// Output:
	// port = 8080
}

func ExampleStringNode_Value() {
	doc, _ := toml.Parse([]byte(`greeting = "hello\nworld"` + "\n"))
	s := doc.Get("greeting").Val.(*toml.StringNode)
	fmt.Println(s.Value())
	// Output:
	// hello
	// world
}

func ExampleNumberNode_Int() {
	doc, _ := toml.Parse([]byte("count = 1_000\n"))
	n := doc.Get("count").Val.(*toml.NumberNode)
	v, _ := n.Int()
	fmt.Println(v)
	// Output:
	// 1000
}

func ExampleNewKeyValue() {
	kv := toml.NewKeyValue("name", toml.NewString("Alice"))
	doc := &toml.Document{}
	doc.Append(kv)
	fmt.Print(doc.String())
	// Output:
	// name = "Alice"
}

func ExampleNewTable() {
	tbl := toml.NewTable("server")
	tbl.Append(toml.NewKeyValue("host", toml.NewString("localhost")))
	doc := &toml.Document{}
	doc.Append(tbl)
	fmt.Print(doc.String())
	// Output:
	// [server]
	// host = "localhost"
}

func ExampleNewString() {
	s := toml.NewString("hello world")
	fmt.Println(s.Text())
	fmt.Println(s.Value())
	// Output:
	// "hello world"
	// hello world
}

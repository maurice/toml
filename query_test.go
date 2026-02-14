package toml

import (
	"math"
	"reflect"
	"testing"
)

// --- Document.Get tests ---

func TestDocument_Get_TopLevel(t *testing.T) {
	d, err := Parse([]byte("name = \"Alice\"\nage = 30\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("name")
	if kv == nil {
		t.Fatal("expected to find key 'name'")
	}
	if kv.RawKey != "name" {
		t.Fatalf("expected key 'name', got %q", kv.RawKey)
	}
}

func TestDocument_Get_DottedKey(t *testing.T) {
	d, err := Parse([]byte("a.b.c = 42\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("a.b.c")
	if kv == nil {
		t.Fatal("expected to find key 'a.b.c'")
	}
}

func TestDocument_Get_InTable(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\nport = 8080\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("server.host")
	if kv == nil {
		t.Fatal("expected to find key 'server.host'")
	}
	s, ok := kv.Val.(*StringNode)
	if !ok {
		t.Fatalf("expected StringNode, got %T", kv.Val)
	}
	if s.Value() != "localhost" {
		t.Fatalf("expected 'localhost', got %q", s.Value())
	}
}

func TestDocument_Get_Nonexistent(t *testing.T) {
	d, err := Parse([]byte("key = 1\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if d.Get("missing") != nil {
		t.Fatal("expected nil for nonexistent key")
	}
}

func TestDocument_Get_InAOT(t *testing.T) {
	d, err := Parse([]byte("[[items]]\nname = \"widget\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("items.name")
	if kv == nil {
		t.Fatal("expected to find key 'items.name'")
	}
}

// --- Document.Table tests ---

func TestDocument_Table(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\n[database]\nport = 5432\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := d.Table("database")
	if tbl == nil {
		t.Fatal("expected to find table 'database'")
	}
	if tbl.RawHeader != "database" {
		t.Fatalf("expected header 'database', got %q", tbl.RawHeader)
	}
}

func TestDocument_Table_DottedHeader(t *testing.T) {
	d, err := Parse([]byte("[servers.alpha]\nip = \"10.0.0.1\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := d.Table("servers.alpha")
	if tbl == nil {
		t.Fatal("expected to find table 'servers.alpha'")
	}
}

func TestDocument_Table_Nonexistent(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if d.Table("missing") != nil {
		t.Fatal("expected nil for nonexistent table")
	}
}

// --- TableNode.Get tests ---

func TestTableNode_Get(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\nport = 8080\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := d.Table("server")
	if tbl == nil {
		t.Fatal("expected to find table 'server'")
	}
	kv := tbl.Get("port")
	if kv == nil {
		t.Fatal("expected to find key 'port'")
	}
	if kv.Val.Text() != "8080" {
		t.Fatalf("expected '8080', got %q", kv.Val.Text())
	}
}

func TestTableNode_Get_Nonexistent(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if d.Table("server").Get("missing") != nil {
		t.Fatal("expected nil for nonexistent key in table")
	}
}

// --- ArrayOfTables.Get tests ---

func TestArrayOfTables_Get(t *testing.T) {
	d, err := Parse([]byte("[[products]]\nname = \"Widget\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	aots := d.ArraysOfTables()
	if len(aots) == 0 {
		t.Fatal("expected at least one AOT")
	}
	kv := aots[0].Get("name")
	if kv == nil {
		t.Fatal("expected to find key 'name'")
	}
}

// --- InlineTableNode.Get tests ---

func TestInlineTableNode_Get(t *testing.T) {
	d, err := Parse([]byte("point = {x = 1, y = 2}\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("point")
	if kv == nil {
		t.Fatal("expected to find key 'point'")
	}
	it, ok := kv.Val.(*InlineTableNode)
	if !ok {
		t.Fatalf("expected InlineTableNode, got %T", kv.Val)
	}
	xkv := it.Get("x")
	if xkv == nil {
		t.Fatal("expected to find key 'x' in inline table")
	}
	if xkv.Val.Text() != "1" {
		t.Fatalf("expected '1', got %q", xkv.Val.Text())
	}
}

// --- StringNode.Value tests ---

func TestStringNode_Value_Basic(t *testing.T) {
	d, err := Parse([]byte(`s = "hello world"` + "\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	s := d.Get("s").Val.(*StringNode)
	if s.Value() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", s.Value())
	}
}

func TestStringNode_Value_Escapes(t *testing.T) {
	d, err := Parse([]byte(`s = "hello\nworld"` + "\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	s := d.Get("s").Val.(*StringNode)
	if s.Value() != "hello\nworld" {
		t.Fatalf("expected 'hello\\nworld', got %q", s.Value())
	}
}

func TestStringNode_Value_Unicode(t *testing.T) {
	d, err := Parse([]byte(`s = "caf\u00E9"` + "\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	s := d.Get("s").Val.(*StringNode)
	if s.Value() != "caf\u00E9" {
		t.Fatalf("expected 'caf\\u00E9', got %q", s.Value())
	}
}

func TestStringNode_Value_Literal(t *testing.T) {
	d, err := Parse([]byte("s = 'C:\\path\\to\\file'\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	s := d.Get("s").Val.(*StringNode)
	if s.Value() != `C:\path\to\file` {
		t.Fatalf("expected 'C:\\path\\to\\file', got %q", s.Value())
	}
}

func TestStringNode_Value_MultiLineBasic(t *testing.T) {
	d, err := Parse([]byte("s = \"\"\"\nhello\nworld\"\"\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	s := d.Get("s").Val.(*StringNode)
	if s.Value() != "hello\nworld" {
		t.Fatalf("expected 'hello\\nworld', got %q", s.Value())
	}
}

func TestStringNode_Value_MultiLineLiteral(t *testing.T) {
	d, err := Parse([]byte("s = '''\nhello\nworld'''\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	s := d.Get("s").Val.(*StringNode)
	if s.Value() != "hello\nworld" {
		t.Fatalf("expected 'hello\\nworld', got %q", s.Value())
	}
}

func TestStringNode_Value_MultiLineBackslash(t *testing.T) {
	d, err := Parse([]byte("s = \"\"\"\nhello \\\n  world\"\"\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	s := d.Get("s").Val.(*StringNode)
	if s.Value() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", s.Value())
	}
}

func TestStringNode_Value_HexEscape(t *testing.T) {
	d, err := Parse([]byte(`s = "caf\xE9"` + "\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	s := d.Get("s").Val.(*StringNode)
	if s.Value() != "caf\u00E9" {
		t.Fatalf("expected 'caf\\u00E9', got %q", s.Value())
	}
}

// --- NumberNode.Int tests ---

func TestNumberNode_Int_Decimal(t *testing.T) {
	d, err := Parse([]byte("n = 42\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Int()
	if err != nil {
		t.Fatalf("Int() error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestNumberNode_Int_Negative(t *testing.T) {
	d, err := Parse([]byte("n = -17\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Int()
	if err != nil {
		t.Fatalf("Int() error: %v", err)
	}
	if v != -17 {
		t.Fatalf("expected -17, got %d", v)
	}
}

func TestNumberNode_Int_Hex(t *testing.T) {
	d, err := Parse([]byte("n = 0xDEAD\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Int()
	if err != nil {
		t.Fatalf("Int() error: %v", err)
	}
	if v != 0xDEAD {
		t.Fatalf("expected 0xDEAD, got %d", v)
	}
}

func TestNumberNode_Int_Octal(t *testing.T) {
	d, err := Parse([]byte("n = 0o755\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Int()
	if err != nil {
		t.Fatalf("Int() error: %v", err)
	}
	if v != 0o755 {
		t.Fatalf("expected 493, got %d", v)
	}
}

func TestNumberNode_Int_Binary(t *testing.T) {
	d, err := Parse([]byte("n = 0b1101\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Int()
	if err != nil {
		t.Fatalf("Int() error: %v", err)
	}
	if v != 13 {
		t.Fatalf("expected 13, got %d", v)
	}
}

func TestNumberNode_Int_Underscore(t *testing.T) {
	d, err := Parse([]byte("n = 1_000_000\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Int()
	if err != nil {
		t.Fatalf("Int() error: %v", err)
	}
	if v != 1000000 {
		t.Fatalf("expected 1000000, got %d", v)
	}
}

func TestNumberNode_Int_ErrorOnFloat(t *testing.T) {
	d, err := Parse([]byte("n = 3.14\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	_, err = n.Int()
	if err == nil {
		t.Fatal("expected error for Int() on float")
	}
}

// --- NumberNode.Float tests ---

func TestNumberNode_Float_Simple(t *testing.T) {
	d, err := Parse([]byte("n = 3.14\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Float()
	if err != nil {
		t.Fatalf("Float() error: %v", err)
	}
	if v != 3.14 {
		t.Fatalf("expected 3.14, got %f", v)
	}
}

func TestNumberNode_Float_FromInteger(t *testing.T) {
	d, err := Parse([]byte("n = 42\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Float()
	if err != nil {
		t.Fatalf("Float() error: %v", err)
	}
	if v != 42.0 {
		t.Fatalf("expected 42.0, got %f", v)
	}
}

func TestNumberNode_Float_Inf(t *testing.T) {
	d, err := Parse([]byte("n = inf\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Float()
	if err != nil {
		t.Fatalf("Float() error: %v", err)
	}
	if !math.IsInf(v, 1) {
		t.Fatalf("expected +Inf, got %f", v)
	}
}

func TestNumberNode_Float_NegInf(t *testing.T) {
	d, err := Parse([]byte("n = -inf\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Float()
	if err != nil {
		t.Fatalf("Float() error: %v", err)
	}
	if !math.IsInf(v, -1) {
		t.Fatalf("expected -Inf, got %f", v)
	}
}

func TestNumberNode_Float_NaN(t *testing.T) {
	d, err := Parse([]byte("n = nan\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Float()
	if err != nil {
		t.Fatalf("Float() error: %v", err)
	}
	if !math.IsNaN(v) {
		t.Fatalf("expected NaN, got %f", v)
	}
}

func TestNumberNode_Float_Exponent(t *testing.T) {
	d, err := Parse([]byte("n = 5e+22\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	n := d.Get("n").Val.(*NumberNode)
	v, err := n.Float()
	if err != nil {
		t.Fatalf("Float() error: %v", err)
	}
	if v != 5e+22 {
		t.Fatalf("expected 5e+22, got %e", v)
	}
}

// --- BooleanNode.Value tests ---

func TestBooleanNode_Value_True(t *testing.T) {
	d, err := Parse([]byte("b = true\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	b := d.Get("b").Val.(*BooleanNode)
	if !b.Value() {
		t.Fatal("expected true")
	}
}

func TestBooleanNode_Value_False(t *testing.T) {
	d, err := Parse([]byte("b = false\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	b := d.Get("b").Val.(*BooleanNode)
	if b.Value() {
		t.Fatal("expected false")
	}
}

// --- parseDottedPath tests ---

func TestParseDottedPath_Simple(t *testing.T) {
	got := parseDottedPath("foo")
	want := []string{"foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseDottedPath_Dotted(t *testing.T) {
	got := parseDottedPath("foo.bar")
	want := []string{"foo", "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseDottedPath_QuotedBasicWithDot(t *testing.T) {
	got := parseDottedPath(`site."google.com"`)
	want := []string{"site", "google.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseDottedPath_QuotedLiteralWithDot(t *testing.T) {
	got := parseDottedPath(`site.'google.com'`)
	want := []string{"site", "google.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseDottedPath_Mixed(t *testing.T) {
	got := parseDottedPath(`a."b.c".'d.e'.f`)
	want := []string{"a", "b.c", "d.e", "f"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseDottedPath_SpecTaterMan(t *testing.T) {
	got := parseDottedPath(`dog."tater.man"`)
	want := []string{"dog", "tater.man"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseDottedPath_BasicEscape(t *testing.T) {
	got := parseDottedPath(`"key\nname"`)
	want := []string{"key\nname"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseDottedPath_WhitespaceAroundDot(t *testing.T) {
	got := parseDottedPath("a . b")
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// --- Spec example: dotted keys with quoted segments ---

func TestDocument_Get_QuotedDottedKey(t *testing.T) {
	input := "name = \"Orange\"\nphysical.color = \"orange\"\nphysical.shape = \"round\"\nsite.\"google.com\" = true\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	kv := d.Get(`site."google.com"`)
	if kv == nil {
		t.Fatal("expected to find key site.\"google.com\"")
	}
	b, ok := kv.Val.(*BooleanNode)
	if !ok {
		t.Fatalf("expected BooleanNode, got %T", kv.Val)
	}
	if !b.Value() {
		t.Fatal("expected true")
	}

	kv2 := d.Get("physical.color")
	if kv2 == nil {
		t.Fatal("expected to find key physical.color")
	}
	s, ok := kv2.Val.(*StringNode)
	if !ok {
		t.Fatalf("expected StringNode, got %T", kv2.Val)
	}
	if s.Value() != "orange" {
		t.Fatalf("expected 'orange', got %q", s.Value())
	}
}

func TestDocument_Table_QuotedHeader(t *testing.T) {
	input := "[dog.\"tater.man\"]\ntype.name = \"pug\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	tbl := d.Table(`dog."tater.man"`)
	if tbl == nil {
		t.Fatal("expected to find table dog.\"tater.man\"")
	}

	kv := tbl.Get("type.name")
	if kv == nil {
		t.Fatal("expected to find key type.name in table")
	}
	s, ok := kv.Val.(*StringNode)
	if !ok {
		t.Fatalf("expected StringNode, got %T", kv.Val)
	}
	if s.Value() != "pug" {
		t.Fatalf("expected 'pug', got %q", s.Value())
	}
}

func TestDocument_Get_ThroughQuotedTable(t *testing.T) {
	input := "[dog.\"tater.man\"]\ntype.name = \"pug\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	kv := d.Get(`dog."tater.man".type.name`)
	if kv == nil {
		t.Fatal("expected to find key dog.\"tater.man\".type.name")
	}
	s, ok := kv.Val.(*StringNode)
	if !ok {
		t.Fatalf("expected StringNode, got %T", kv.Val)
	}
	if s.Value() != "pug" {
		t.Fatalf("expected 'pug', got %q", s.Value())
	}
}

func TestDocument_Delete_QuotedDottedKey(t *testing.T) {
	input := "name = \"Orange\"\nsite.\"google.com\" = true\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !d.Delete(`site."google.com"`) {
		t.Fatal("expected Delete to return true")
	}
	got := d.String()
	expected := "name = \"Orange\"\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

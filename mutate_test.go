package toml

import (
	"math"
	"testing"
)

// --- Constructor tests ---

func TestNewString(t *testing.T) {
	s := NewString("hello world")
	if s.Text() != `"hello world"` {
		t.Fatalf("expected '\"hello world\"', got %q", s.Text())
	}
	if s.Value() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", s.Value())
	}
}

func TestNewString_Escapes(t *testing.T) {
	s := NewString("line1\nline2")
	if s.Text() != `"line1\nline2"` {
		t.Fatalf("expected '\"line1\\nline2\"', got %q", s.Text())
	}
	if s.Value() != "line1\nline2" {
		t.Fatalf("expected value with newline, got %q", s.Value())
	}
}

func TestNewString_QuotesInValue(t *testing.T) {
	s := NewString(`say "hello"`)
	if s.Text() != `"say \"hello\""` {
		t.Fatalf("unexpected text: %q", s.Text())
	}
	if s.Value() != `say "hello"` {
		t.Fatalf("unexpected value: %q", s.Value())
	}
}

func TestNewInteger(t *testing.T) {
	n := NewInteger(42)
	if n.Text() != "42" {
		t.Fatalf("expected '42', got %q", n.Text())
	}
	v, err := n.Int()
	if err != nil {
		t.Fatalf("Int() error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestNewInteger_Negative(t *testing.T) {
	n := NewInteger(-100)
	if n.Text() != "-100" {
		t.Fatalf("expected '-100', got %q", n.Text())
	}
}

func TestNewFloat(t *testing.T) {
	n := NewFloat(3.14)
	v, err := n.Float()
	if err != nil {
		t.Fatalf("Float() error: %v", err)
	}
	if v != 3.14 {
		t.Fatalf("expected 3.14, got %f", v)
	}
}

func TestNewFloat_Inf(t *testing.T) {
	n := NewFloat(math.Inf(1))
	if n.Text() != "inf" {
		t.Fatalf("expected 'inf', got %q", n.Text())
	}
}

func TestNewBool_True(t *testing.T) {
	b := NewBool(true)
	if b.Text() != "true" {
		t.Fatalf("expected 'true', got %q", b.Text())
	}
	if !b.Value() {
		t.Fatal("expected true")
	}
}

func TestNewBool_False(t *testing.T) {
	b := NewBool(false)
	if b.Text() != "false" {
		t.Fatalf("expected 'false', got %q", b.Text())
	}
	if b.Value() {
		t.Fatal("expected false")
	}
}

func TestNewKeyValue(t *testing.T) {
	kv, err := NewKeyValue("name", NewString("Alice"))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if kv.rawKey != "name" {
		t.Fatalf("expected key 'name', got %q", kv.rawKey)
	}
	if kv.rawVal != `"Alice"` {
		t.Fatalf("expected val '\"Alice\"', got %q", kv.rawVal)
	}
	if kv.PreEq != " " || kv.PostEq != " " {
		t.Fatalf("expected standard spacing around =")
	}
	if kv.Newline != "\n" {
		t.Fatalf("expected newline, got %q", kv.Newline)
	}
}

func TestNewKeyValue_QuotedKey(t *testing.T) {
	kv, err := NewKeyValue(`"key with spaces"`, NewString("val"))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if kv.keyParts[0].Text != `"key with spaces"` {
		t.Fatalf("expected quoted key, got %q", kv.keyParts[0].Text)
	}
	if kv.keyParts[0].Unquoted != "key with spaces" {
		t.Fatalf("expected unquoted 'key with spaces', got %q", kv.keyParts[0].Unquoted)
	}
}

func TestNewKeyValue_DottedKey(t *testing.T) {
	kv, err := NewKeyValue("a.b", NewInteger(1))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if len(kv.keyParts) != 2 {
		t.Fatalf("expected 2 key parts, got %d", len(kv.keyParts))
	}
	if kv.keyParts[0].Unquoted != "a" || kv.keyParts[1].Unquoted != "b" {
		t.Fatalf("unexpected key parts: %v", kv.keyParts)
	}
}

func TestNewTable(t *testing.T) {
	tbl, err := NewTable("server.settings")
	if err != nil {
		t.Fatalf("NewTable: %v", err)
	}
	if tbl.rawHeader != "server.settings" {
		t.Fatalf("expected header 'server.settings', got %q", tbl.rawHeader)
	}
	if len(tbl.headerParts) != 2 {
		t.Fatalf("expected 2 header parts, got %d", len(tbl.headerParts))
	}
	if tbl.Newline != "\n" {
		t.Fatalf("expected newline, got %q", tbl.Newline)
	}
}

func TestNewTable_QuotedSegments(t *testing.T) {
	tbl, err := NewTable(`"has spaces".normal`)
	if err != nil {
		t.Fatalf("NewTable: %v", err)
	}
	if tbl.headerParts[0].Text != `"has spaces"` {
		t.Fatalf("expected quoted segment, got %q", tbl.headerParts[0].Text)
	}
	if tbl.headerParts[0].Unquoted != "has spaces" {
		t.Fatalf("expected unquoted 'has spaces', got %q", tbl.headerParts[0].Unquoted)
	}
}

// --- SetValue tests ---

func TestSetValue(t *testing.T) {
	d, err := Parse([]byte("key = \"old\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("key")
	if err := kv.SetValue(NewString("new")); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if kv.rawVal != `"new"` {
		t.Fatalf("expected '\"new\"', got %q", kv.rawVal)
	}
	got := d.String()
	expected := "key = \"new\"\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestSetValue_ChangeType(t *testing.T) {
	d, err := Parse([]byte("key = \"old\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("key")
	if err := kv.SetValue(NewInteger(42)); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	got := d.String()
	expected := "key = 42\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

// --- Delete tests ---

func TestDocument_Delete_TopLevel(t *testing.T) {
	d, err := Parse([]byte("a = 1\nb = 2\nc = 3\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !d.Delete("b") {
		t.Fatal("expected Delete to return true")
	}
	got := d.String()
	expected := "a = 1\nc = 3\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestDocument_Delete_Nonexistent(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if d.Delete("missing") {
		t.Fatal("expected Delete to return false for nonexistent key")
	}
}

func TestDocument_Delete_InTable(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\nport = 8080\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !d.Delete("server.host") {
		t.Fatal("expected Delete to return true")
	}
	got := d.String()
	expected := "[server]\nport = 8080\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestTableNode_Delete(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\nport = 8080\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := d.Table("server")
	if !tbl.Delete("host") {
		t.Fatal("expected Delete to return true")
	}
	got := d.String()
	expected := "[server]\nport = 8080\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestTableNode_Delete_Nonexistent(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if d.Table("server").Delete("missing") {
		t.Fatal("expected Delete to return false")
	}
}

func TestArrayOfTables_Delete(t *testing.T) {
	d, err := Parse([]byte("[[items]]\nname = \"widget\"\nprice = 10\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	aots := d.ArraysOfTables()
	if !aots[0].Delete("name") {
		t.Fatal("expected Delete to return true")
	}
	got := d.String()
	expected := "[[items]]\nprice = 10\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

// --- DeleteTable tests ---

func TestDocument_DeleteTable(t *testing.T) {
	d, err := Parse([]byte("top = 1\n[server]\nhost = \"localhost\"\n[database]\nport = 5432\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !d.DeleteTable("server") {
		t.Fatal("expected DeleteTable to return true")
	}
	got := d.String()
	expected := "top = 1\n[database]\nport = 5432\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestDocument_DeleteTable_Nonexistent(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if d.DeleteTable("missing") {
		t.Fatal("expected DeleteTable to return false")
	}
}

// --- Append tests ---

func TestDocument_Append(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv, err := NewKeyValue("b", NewInteger(2))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := d.Append(kv); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got := d.String()
	expected := "a = 1\nb = 2\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestDocument_Append_Table(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl, err := NewTable("server")
	if err != nil {
		t.Fatalf("NewTable: %v", err)
	}
	if err := d.Append(tbl); err != nil {
		t.Fatalf("Append table: %v", err)
	}
	kv, err := NewKeyValue("host", NewString("localhost"))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := tbl.Append(kv); err != nil {
		t.Fatalf("Append kv: %v", err)
	}
	got := d.String()
	expected := "a = 1\n[server]\nhost = \"localhost\"\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestTableNode_Append(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := d.Table("server")
	kv, err := NewKeyValue("port", NewInteger(8080))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := tbl.Append(kv); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got := d.String()
	expected := "[server]\nhost = \"localhost\"\nport = 8080\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestArrayOfTables_Append(t *testing.T) {
	d, err := Parse([]byte("[[items]]\nname = \"widget\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	aots := d.ArraysOfTables()
	kv, err := NewKeyValue("price", NewInteger(10))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := aots[0].Append(kv); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got := d.String()
	expected := "[[items]]\nname = \"widget\"\nprice = 10\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

// --- InsertAt tests ---

func TestDocument_InsertAt_Beginning(t *testing.T) {
	d, err := Parse([]byte("b = 2\nc = 3\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv, err := NewKeyValue("a", NewInteger(1))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := d.InsertAt(0, kv); err != nil {
		t.Fatalf("InsertAt: %v", err)
	}
	got := d.String()
	expected := "a = 1\nb = 2\nc = 3\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestDocument_InsertAt_Middle(t *testing.T) {
	d, err := Parse([]byte("a = 1\nc = 3\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv, err := NewKeyValue("b", NewInteger(2))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := d.InsertAt(1, kv); err != nil {
		t.Fatalf("InsertAt: %v", err)
	}
	got := d.String()
	expected := "a = 1\nb = 2\nc = 3\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestDocument_InsertAt_End(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv, err := NewKeyValue("b", NewInteger(2))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := d.InsertAt(999, kv); err != nil {
		t.Fatalf("InsertAt: %v", err)
	}
	got := d.String()
	expected := "a = 1\nb = 2\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestDocument_InsertAt_Negative(t *testing.T) {
	d, err := Parse([]byte("b = 2\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv, err := NewKeyValue("a", NewInteger(1))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := d.InsertAt(-1, kv); err != nil {
		t.Fatalf("InsertAt: %v", err)
	}
	got := d.String()
	expected := "a = 1\nb = 2\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestTableNode_InsertAt(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\nport = 8080\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := d.Table("server")
	kv, err := NewKeyValue("ip", NewString("127.0.0.1"))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if err := tbl.InsertAt(1, kv); err != nil {
		t.Fatalf("InsertAt: %v", err)
	}
	got := d.String()
	expected := "[server]\nhost = \"localhost\"\nip = \"127.0.0.1\"\nport = 8080\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

// --- Quoted key mutation tests ---

func TestDocument_Delete_QuotedKeyInTable(t *testing.T) {
	input := "[dog.\"tater.man\"]\ntype.name = \"pug\"\ncolor = \"brown\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := d.Table(`dog."tater.man"`)
	if tbl == nil {
		t.Fatal("expected to find table")
	}
	if !tbl.Delete("color") {
		t.Fatal("expected Delete to return true")
	}
	got := d.String()
	expected := "[dog.\"tater.man\"]\ntype.name = \"pug\"\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestDocument_DeleteTable_QuotedHeader(t *testing.T) {
	input := "top = 1\n[dog.\"tater.man\"]\ntype = \"pug\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !d.DeleteTable(`dog."tater.man"`) {
		t.Fatal("expected DeleteTable to return true")
	}
	got := d.String()
	expected := "top = 1\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

// --- ArrayNode mutation tests ---

func TestArrayNode_Append(t *testing.T) {
	arr, err := NewArray(NewInteger(1), NewInteger(2))
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	if err := arr.Append(NewInteger(3)); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if arr.Text() != "[1, 2, 3]" {
		t.Fatalf("expected '[1, 2, 3]', got %q", arr.Text())
	}
	elems := arr.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
}

func TestArrayNode_Append_RejectsNil(t *testing.T) {
	arr, err := NewArray(NewInteger(1))
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	if err := arr.Append(nil); err == nil {
		t.Fatal("expected error for nil element")
	}
}

func TestArrayNode_Append_RejectsInvalidType(t *testing.T) {
	arr, err := NewArray(NewInteger(1))
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	if err := arr.Append(&CommentNode{}); err == nil {
		t.Fatal("expected error for invalid value type")
	}
}

func TestArrayNode_Delete(t *testing.T) {
	arr, err := NewArray(NewInteger(1), NewInteger(2), NewInteger(3))
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	if err := arr.Delete(1); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if arr.Text() != "[1, 3]" {
		t.Fatalf("expected '[1, 3]', got %q", arr.Text())
	}
	elems := arr.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
}

func TestArrayNode_Delete_OutOfBounds(t *testing.T) {
	arr, err := NewArray(NewInteger(1))
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	if err := arr.Delete(5); err == nil {
		t.Fatal("expected error for out of bounds index")
	}
	if err := arr.Delete(-1); err == nil {
		t.Fatal("expected error for negative index")
	}
}

func TestArrayNode_Delete_All(t *testing.T) {
	arr, err := NewArray(NewInteger(1))
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	if err := arr.Delete(0); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if arr.Text() != "[]" {
		t.Fatalf("expected '[]', got %q", arr.Text())
	}
}

// --- InlineTableNode mutation tests ---

func TestInlineTableNode_Append(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewInteger(1))
	it, err := NewInlineTable(kv1)
	if err != nil {
		t.Fatalf("NewInlineTable: %v", err)
	}
	kv2, _ := NewKeyValue("b", NewInteger(2))
	if err := it.Append(kv2); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if it.Text() != "{a = 1, b = 2}" {
		t.Fatalf("expected '{a = 1, b = 2}', got %q", it.Text())
	}
}

func TestInlineTableNode_Append_DuplicateKey(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewInteger(1))
	it, err := NewInlineTable(kv1)
	if err != nil {
		t.Fatalf("NewInlineTable: %v", err)
	}
	kv2, _ := NewKeyValue("a", NewInteger(2))
	if err := it.Append(kv2); err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestInlineTableNode_Append_NilEntry(t *testing.T) {
	it, err := NewInlineTable()
	if err != nil {
		t.Fatalf("NewInlineTable: %v", err)
	}
	if err := it.Append(nil); err == nil {
		t.Fatal("expected error for nil entry")
	}
}

func TestInlineTableNode_Delete(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewInteger(1))
	kv2, _ := NewKeyValue("b", NewInteger(2))
	it, err := NewInlineTable(kv1, kv2)
	if err != nil {
		t.Fatalf("NewInlineTable: %v", err)
	}
	if !it.Delete("a") {
		t.Fatal("expected Delete to return true")
	}
	if it.Text() != "{b = 2}" {
		t.Fatalf("expected '{b = 2}', got %q", it.Text())
	}
}

func TestInlineTableNode_Delete_Nonexistent(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewInteger(1))
	it, err := NewInlineTable(kv1)
	if err != nil {
		t.Fatalf("NewInlineTable: %v", err)
	}
	if it.Delete("missing") {
		t.Fatal("expected Delete to return false")
	}
}

func TestInlineTableNode_Delete_All(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewInteger(1))
	it, err := NewInlineTable(kv1)
	if err != nil {
		t.Fatalf("NewInlineTable: %v", err)
	}
	if !it.Delete("a") {
		t.Fatal("expected Delete to return true")
	}
	if it.Text() != "{}" {
		t.Fatalf("expected '{}', got %q", it.Text())
	}
}

// --- SetValue ancestor text regeneration tests ---

func TestSetValue_RegeneratesInlineTableText(t *testing.T) {
	d, err := Parse([]byte("config = {port = 8080, host = \"localhost\"}\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("config")
	it, ok := kv.Val().(*InlineTableNode)
	if !ok {
		t.Fatal("expected InlineTableNode")
	}
	// Change a value inside the inline table.
	for _, entry := range it.Entries() {
		if entry.RawKey() == "port" {
			if err := entry.SetValue(NewInteger(9090)); err != nil {
				t.Fatalf("SetValue: %v", err)
			}
			break
		}
	}
	got := d.String()
	if got != "config = {port = 9090, host = \"localhost\"}\n" {
		t.Fatalf("unexpected result: %q", got)
	}
}

// --- Validation error tests ---

func TestNewKeyValue_RejectsEmptyKey(t *testing.T) {
	_, err := NewKeyValue("", NewString("val"))
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestNewKeyValue_RejectsNilValue(t *testing.T) {
	_, err := NewKeyValue("key", nil)
	if err == nil {
		t.Fatal("expected error for nil value")
	}
}

func TestNewKeyValue_RejectsInvalidValueType(t *testing.T) {
	_, err := NewKeyValue("key", &CommentNode{})
	if err == nil {
		t.Fatal("expected error for invalid value type")
	}
}

func TestNewKeyValue_RejectsInvalidKey(t *testing.T) {
	_, err := NewKeyValue("has spaces", NewString("val"))
	if err == nil {
		t.Fatal("expected error for invalid bare key with spaces")
	}
}

func TestNewTable_RejectsEmptyKey(t *testing.T) {
	_, err := NewTable("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestNewTable_RejectsInvalidKey(t *testing.T) {
	_, err := NewTable("a..b")
	if err == nil {
		t.Fatal("expected error for key with empty segment")
	}
}

func TestNewArrayOfTables_RejectsEmptyKey(t *testing.T) {
	_, err := NewArrayOfTables("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestNewDateTime_RejectsInvalid(t *testing.T) {
	_, err := NewDateTime("not-a-date")
	if err == nil {
		t.Fatal("expected error for invalid datetime")
	}
}

func TestNewDateTime_AcceptsValid(t *testing.T) {
	dt, err := NewDateTime("2024-01-15T10:30:00Z")
	if err != nil {
		t.Fatalf("NewDateTime: %v", err)
	}
	if dt.Text() != "2024-01-15T10:30:00Z" {
		t.Fatalf("unexpected text: %q", dt.Text())
	}
}

func TestNewArray_RejectsNilElement(t *testing.T) {
	_, err := NewArray(NewInteger(1), nil, NewInteger(3))
	if err == nil {
		t.Fatal("expected error for nil element")
	}
}

func TestNewInlineTable_RejectsDuplicateKeys(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewInteger(1))
	kv2, _ := NewKeyValue("a", NewInteger(2))
	_, err := NewInlineTable(kv1, kv2)
	if err == nil {
		t.Fatal("expected error for duplicate keys")
	}
}

func TestNewInlineTable_RejectsNilEntry(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewInteger(1))
	_, err := NewInlineTable(kv1, nil)
	if err == nil {
		t.Fatal("expected error for nil entry")
	}
}

func TestSetValue_RejectsNil(t *testing.T) {
	kv, _ := NewKeyValue("key", NewString("val"))
	if err := kv.SetValue(nil); err == nil {
		t.Fatal("expected error for nil value")
	}
}

func TestSetValue_RejectsInvalidType(t *testing.T) {
	kv, _ := NewKeyValue("key", NewString("val"))
	if err := kv.SetValue(&CommentNode{}); err == nil {
		t.Fatal("expected error for invalid value type")
	}
}

func TestDocument_Append_RejectsDuplicate(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv, _ := NewKeyValue("a", NewInteger(2))
	if err := d.Append(kv); err == nil {
		t.Fatal("expected error for duplicate key")
	}
	// Verify rollback.
	if len(d.Nodes()) != 1 {
		t.Fatalf("expected 1 node after rollback, got %d", len(d.Nodes()))
	}
}

func TestDocument_Append_RejectsDuplicateTable(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl, _ := NewTable("server")
	if err := d.Append(tbl); err == nil {
		t.Fatal("expected error for duplicate table")
	}
}

func TestDocument_Append_RejectsNil(t *testing.T) {
	d, err := Parse([]byte(""))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := d.Append(nil); err == nil {
		t.Fatal("expected error for nil node")
	}
}

func TestDocument_Append_RejectsInvalidType(t *testing.T) {
	d, err := Parse([]byte(""))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := d.Append(&CommentNode{}); err == nil {
		t.Fatal("expected error for invalid node type")
	}
}

func TestTableNode_Append_RejectsDuplicate(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := d.Table("server")
	kv, _ := NewKeyValue("host", NewString("other"))
	if err := tbl.Append(kv); err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestTableNode_Append_RejectsNil(t *testing.T) {
	tbl, _ := NewTable("t")
	if err := tbl.Append(nil); err == nil {
		t.Fatal("expected error for nil key-value")
	}
}

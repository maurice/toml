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
	kv := NewKeyValue("name", NewString("Alice"))
	if kv.RawKey != "name" {
		t.Fatalf("expected key 'name', got %q", kv.RawKey)
	}
	if kv.RawVal != `"Alice"` {
		t.Fatalf("expected val '\"Alice\"', got %q", kv.RawVal)
	}
	if kv.PreEq != " " || kv.PostEq != " " {
		t.Fatalf("expected standard spacing around =")
	}
	if kv.Newline != "\n" {
		t.Fatalf("expected newline, got %q", kv.Newline)
	}
}

func TestNewKeyValue_QuotedKey(t *testing.T) {
	kv := NewKeyValue("key with spaces", NewString("val"))
	if kv.KeyParts[0].Text != `"key with spaces"` {
		t.Fatalf("expected quoted key, got %q", kv.KeyParts[0].Text)
	}
	if kv.KeyParts[0].Unquoted != "key with spaces" {
		t.Fatalf("expected unquoted 'key with spaces', got %q", kv.KeyParts[0].Unquoted)
	}
}

func TestNewKeyValue_DottedKey(t *testing.T) {
	kv := NewKeyValue("a.b", NewInteger(1))
	if len(kv.KeyParts) != 2 {
		t.Fatalf("expected 2 key parts, got %d", len(kv.KeyParts))
	}
	if kv.KeyParts[0].Unquoted != "a" || kv.KeyParts[1].Unquoted != "b" {
		t.Fatalf("unexpected key parts: %v", kv.KeyParts)
	}
}

func TestNewTable(t *testing.T) {
	tbl := NewTable("server", "settings")
	if tbl.RawHeader != "server.settings" {
		t.Fatalf("expected header 'server.settings', got %q", tbl.RawHeader)
	}
	if len(tbl.HeaderParts) != 2 {
		t.Fatalf("expected 2 header parts, got %d", len(tbl.HeaderParts))
	}
	if tbl.Newline != "\n" {
		t.Fatalf("expected newline, got %q", tbl.Newline)
	}
}

func TestNewTable_QuotedSegments(t *testing.T) {
	tbl := NewTable("has spaces", "normal")
	if tbl.HeaderParts[0].Text != `"has spaces"` {
		t.Fatalf("expected quoted segment, got %q", tbl.HeaderParts[0].Text)
	}
	if tbl.HeaderParts[0].Unquoted != "has spaces" {
		t.Fatalf("expected unquoted 'has spaces', got %q", tbl.HeaderParts[0].Unquoted)
	}
}

// --- SetValue tests ---

func TestSetValue(t *testing.T) {
	d, err := Parse([]byte("key = \"old\"\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.Get("key")
	kv.SetValue(NewString("new"))
	if kv.RawVal != `"new"` {
		t.Fatalf("expected '\"new\"', got %q", kv.RawVal)
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
	kv.SetValue(NewInteger(42))
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
	d.Append(NewKeyValue("b", NewInteger(2)))
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
	tbl := NewTable("server")
	d.Append(tbl)
	tbl.Append(NewKeyValue("host", NewString("localhost")))
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
	tbl.Append(NewKeyValue("port", NewInteger(8080)))
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
	aots[0].Append(NewKeyValue("price", NewInteger(10)))
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
	d.InsertAt(0, NewKeyValue("a", NewInteger(1)))
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
	d.InsertAt(1, NewKeyValue("b", NewInteger(2)))
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
	d.InsertAt(999, NewKeyValue("b", NewInteger(2)))
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
	d.InsertAt(-1, NewKeyValue("a", NewInteger(1)))
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
	tbl.InsertAt(1, NewKeyValue("ip", NewString("127.0.0.1")))
	got := d.String()
	expected := "[server]\nhost = \"localhost\"\nip = \"127.0.0.1\"\nport = 8080\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

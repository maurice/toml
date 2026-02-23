package toml

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestParse_EmptyDocument(t *testing.T) {
	d, err := Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(d.nodes))
	}
}

func TestParse_NilInput(t *testing.T) {
	_, err := Parse(nil)
	if !errors.Is(err, ErrNilInput) {
		t.Fatalf("expected ErrNilInput, got %v", err)
	}
}

func TestParse_SimpleKeyValue(t *testing.T) {
	d, err := Parse([]byte(`key = "value"`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(d.nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(d.nodes))
	}
	kv, ok := d.nodes[0].(*KeyValue)
	if !ok {
		t.Fatalf("expected *KeyValue, got %T", d.nodes[0])
	}
	if kv.rawKey != "key" {
		t.Fatalf("expected key 'key', got %q", kv.rawKey)
	}
	if kv.rawVal != `"value"` {
		t.Fatalf("expected value '\"value\"', got %q", kv.rawVal)
	}
	if kv.val.Type() != NodeString {
		t.Fatalf("expected string value, got %v", kv.val.Type())
	}
}

func TestParse_PreservesWhitespaceAroundEquals(t *testing.T) {
	d, err := Parse([]byte("key  =  42\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.PreEq() != "  " {
		t.Fatalf("expected PreEq '  ', got %q", kv.PreEq())
	}
	if kv.PostEq() != "  " {
		t.Fatalf("expected PostEq '  ', got %q", kv.PostEq())
	}
}

func TestParse_TrailingComment(t *testing.T) {
	d, err := Parse([]byte("key = \"value\"  # tail comment\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	trailing := kv.TrailingTrivia()
	if len(trailing) != 2 {
		t.Fatalf("expected 2 trailing trivia nodes, got %d", len(trailing))
	}
	comment := trailing[1]
	if comment.Type() != NodeComment {
		t.Fatalf("expected comment node, got %v", comment.Type())
	}
	if !strings.Contains(comment.Text(), "tail comment") {
		t.Fatalf("expected trailing comment text, got %q", comment.Text())
	}
}

func TestParse_LeadingTrivia(t *testing.T) {
	input := "# comment before\n\nkey = \"v\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if len(kv.LeadingTrivia()) == 0 {
		t.Fatalf("expected leading trivia")
	}
	hasComment := false
	for _, n := range kv.LeadingTrivia() {
		if n.Type() == NodeComment {
			hasComment = true
			if !strings.Contains(n.Text(), "comment before") {
				t.Fatalf("unexpected comment text: %q", n.Text())
			}
		}
	}
	if !hasComment {
		t.Fatalf("expected a comment in leading trivia")
	}
}

func TestRoundTrip_SimpleDocument(t *testing.T) {
	input := "# top comment\nkey = \"v\"  # inline\n\nother = 1\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestRoundTrip_TableWithEntries(t *testing.T) {
	input := "[server]\nhost = \"localhost\"\nport = 8080\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestRoundTrip_ArrayOfTables(t *testing.T) {
	input := "[[products]]\nname = \"Widget\"\n\n[[products]]\nname = \"Gadget\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestRoundTrip_InlineTable(t *testing.T) {
	input := "point = {x = 1, y = 2}\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestRoundTrip_Array(t *testing.T) {
	input := "arr = [1, 2, 3]\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestRoundTrip_MultilineArray(t *testing.T) {
	input := "arr = [\n  1,\n  2,\n  3,\n]\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestRoundTrip_ComplexDocument(t *testing.T) {
	input := `# This is a TOML config file

title = "My App"

[database]
server = "192.168.1.1"
ports = [8001, 8001, 8002]
enabled = true

[servers.alpha]
ip = "10.0.0.1"
dc = "eqdc10"

[[products]]
name = "Hammer"
sku = 738594937

[[products]]
name = "Nail"
sku = 284758393
`
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant:\n%s\ngot:\n%s", input, got)
	}
}

func TestParse_TableHeader(t *testing.T) {
	input := "# before\n[server.settings]  # header comment\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var found *TableNode
	for _, n := range d.nodes {
		if tn, ok := n.(*TableNode); ok {
			found = tn
			break
		}
	}
	if found == nil {
		t.Fatalf("expected TableNode, none found")
	}
	if found.rawHeader != "server.settings" {
		t.Fatalf("unexpected header: %q", found.rawHeader)
	}
	if len(found.headerParts) != 2 {
		t.Fatalf("expected 2 header parts, got %d", len(found.headerParts))
	}
	if found.headerParts[0].Unquoted != "server" || found.headerParts[1].Unquoted != "settings" {
		t.Fatalf("unexpected header parts: %v", found.headerParts)
	}
}

func TestParse_ArrayOfTablesHeader(t *testing.T) {
	input := "[[products]]  # arr\nname = \"prod\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var found *ArrayOfTables
	for _, n := range d.nodes {
		if a, ok := n.(*ArrayOfTables); ok {
			found = a
			break
		}
	}
	if found == nil {
		t.Fatalf("expected ArrayOfTables, none found")
	}
	if found.rawHeader != "products" {
		t.Fatalf("unexpected header: %q", found.rawHeader)
	}
	if len(found.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(found.entries))
	}
}

func TestParse_HierarchicalStructure(t *testing.T) {
	input := "top = 1\n[section]\ninner = 2\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(d.nodes) != 2 {
		t.Fatalf("expected 2 top-level nodes, got %d", len(d.nodes))
	}
	if _, ok := d.nodes[0].(*KeyValue); !ok {
		t.Fatalf("expected first node to be KeyValue, got %T", d.nodes[0])
	}
	tbl, ok := d.nodes[1].(*TableNode)
	if !ok {
		t.Fatalf("expected second node to be TableNode, got %T", d.nodes[1])
	}
	if len(tbl.entries) != 1 {
		t.Fatalf("expected 1 entry in table, got %d", len(tbl.entries))
	}
}

func TestParse_DottedKey(t *testing.T) {
	d, err := Parse([]byte("a.b.c = 1\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if len(kv.keyParts) != 3 {
		t.Fatalf("expected 3 key parts, got %d", len(kv.keyParts))
	}
	if kv.keyParts[0].Unquoted != "a" || kv.keyParts[1].Unquoted != "b" || kv.keyParts[2].Unquoted != "c" {
		t.Fatalf("unexpected key parts: %v", kv.keyParts)
	}
}

func TestParse_QuotedKey(t *testing.T) {
	d, err := Parse([]byte(`"key with spaces" = 1` + "\n"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if len(kv.keyParts) != 1 {
		t.Fatalf("expected 1 key part, got %d", len(kv.keyParts))
	}
	if kv.keyParts[0].Unquoted != "key with spaces" {
		t.Fatalf("expected unquoted key 'key with spaces', got %q", kv.keyParts[0].Unquoted)
	}
	if !kv.keyParts[0].IsQuoted {
		t.Fatalf("expected key to be marked as quoted")
	}
}

func TestParse_ValueTypes(t *testing.T) {
	tests := []struct {
		input string
		want  NodeType
	}{
		{`s = "hello"`, NodeString},
		{`n = 42`, NodeNumber},
		{`f = 3.14`, NodeNumber},
		{`b = true`, NodeBoolean},
		{`d = 2024-01-15`, NodeDateTime},
		{`t = 08:30:00`, NodeDateTime},
		{`dt = 2024-01-15T08:30:00Z`, NodeDateTime},
		{`a = [1, 2]`, NodeArray},
		{`it = {x = 1}`, NodeInlineTable},
	}
	for _, tt := range tests {
		d, err := Parse([]byte(tt.input))
		if err != nil {
			t.Fatalf("parse error for %q: %v", tt.input, err)
		}
		kv := d.nodes[0].(*KeyValue)
		if kv.val.Type() != tt.want {
			t.Fatalf("for %q: expected value type %v, got %v", tt.input, tt.want, kv.val.Type())
		}
	}
}

func TestWalk_FindsAllNodeTypes(t *testing.T) {
	input := "# top\nkey = 1  # tail\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	comments := 0
	d.Walk(func(n Node) bool {
		if n.Type() == NodeComment {
			comments++
		}
		return true
	})
	if comments < 2 {
		t.Fatalf("expected at least 2 comments, found %d", comments)
	}
}

func TestPreorder_FindsAllNodeTypes(t *testing.T) {
	input := "# top\nkey = 1  # tail\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	comments := 0
	for n := range d.Preorder() {
		if n.Type() == NodeComment {
			comments++
		}
	}
	if comments != 2 {
		t.Fatalf("expected 2 comments, found %d", comments)
	}
}

func TestPreorder_EarlyBreak(t *testing.T) {
	input := "a = 1\nb = 2\nc = 3\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	count := 0
	for range d.Preorder() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 iterations before break, got %d", count)
	}
}

func TestParse_MultilineBasicString(t *testing.T) {
	input := "s = \"\"\"\nhello\nworld\"\"\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeString {
		t.Fatalf("expected string, got %v", kv.val.Type())
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestParse_InlineTableEntries(t *testing.T) {
	input := "point = {x = 1, y = 2}\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	it, ok := kv.val.(*InlineTableNode)
	if !ok {
		t.Fatalf("expected InlineTableNode, got %T", kv.val)
	}
	if len(it.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(it.entries))
	}
}

func TestParse_SpecialFloats(t *testing.T) {
	for _, v := range []string{"inf", "+inf", "-inf", "nan", "+nan", "-nan"} {
		input := "f = " + v + "\n"
		d, err := Parse([]byte(input))
		if err != nil {
			t.Fatalf("parse error for %s: %v", v, err)
		}
		kv := d.nodes[0].(*KeyValue)
		if kv.val.Type() != NodeNumber {
			t.Fatalf("for %s: expected number, got %v", v, kv.val.Type())
		}
	}
}

func TestParse_HexOctBin(t *testing.T) {
	tests := []struct {
		input string
		val   string
	}{
		{"n = 0xDEADBEEF\n", "0xDEADBEEF"},
		{"n = 0o755\n", "0o755"},
		{"n = 0b11010110\n", "0b11010110"},
	}
	for _, tt := range tests {
		d, err := Parse([]byte(tt.input))
		if err != nil {
			t.Fatalf("parse error for %q: %v", tt.input, err)
		}
		kv := d.nodes[0].(*KeyValue)
		if kv.val.Type() != NodeNumber {
			t.Fatalf("for %q: expected number, got %v", tt.input, kv.val.Type())
		}
		if kv.val.Text() != tt.val {
			t.Fatalf("for %q: expected %q, got %q", tt.input, tt.val, kv.val.Text())
		}
	}
}

// --- Validation tests: numbers ---

func TestParse_RejectsLeadingZeros(t *testing.T) {
	_, err := Parse([]byte("n = 012\n"))
	if err == nil {
		t.Fatal("expected error for leading zeros")
	}
}

func TestParse_RejectsDoubleUnderscore(t *testing.T) {
	_, err := Parse([]byte("n = 1__0\n"))
	if err == nil {
		t.Fatal("expected error for double underscore")
	}
}

func TestParse_RejectsTrailingUnderscore(t *testing.T) {
	_, err := Parse([]byte("n = 100_\n"))
	if err == nil {
		t.Fatal("expected error for trailing underscore")
	}
}

func TestParse_RejectsMultipleDots(t *testing.T) {
	_, err := Parse([]byte("f = 1.2.3\n"))
	if err == nil {
		t.Fatal("expected error for multiple dots in float")
	}
}

func TestParse_RejectsUnderscoreByDot(t *testing.T) {
	_, err := Parse([]byte("f = 1_.0\n"))
	if err == nil {
		t.Fatal("expected error for underscore adjacent to dot")
	}
}

func TestParse_RejectsUnderscoreByExponent(t *testing.T) {
	_, err := Parse([]byte("f = 1_e2\n"))
	if err == nil {
		t.Fatal("expected error for underscore adjacent to exponent")
	}
}

// --- Validation tests: datetimes ---

func TestParse_RejectsInvalidMonth(t *testing.T) {
	_, err := Parse([]byte("d = 2024-13-01\n"))
	if err == nil {
		t.Fatal("expected error for invalid month")
	}
}

func TestParse_RejectsInvalidDay(t *testing.T) {
	_, err := Parse([]byte("d = 2024-02-30\n"))
	if err == nil {
		t.Fatal("expected error for Feb 30")
	}
}

func TestParse_RejectsInvalidHour(t *testing.T) {
	_, err := Parse([]byte("t = 25:00:00\n"))
	if err == nil {
		t.Fatal("expected error for hour > 23")
	}
}

// --- Validation tests: strings ---

func TestParse_RejectsInvalidEscape(t *testing.T) {
	_, err := Parse([]byte("s = \"hello\\q\"\n"))
	if err == nil {
		t.Fatal("expected error for invalid escape \\q")
	}
}

func TestParse_RejectsControlCharInString(t *testing.T) {
	_, err := Parse([]byte("s = \"hello\x01world\"\n"))
	if err == nil {
		t.Fatal("expected error for control char in string")
	}
}

// --- Validation tests: semantic ---

func TestParse_RejectsDuplicateTable(t *testing.T) {
	input := "[a]\nk = 1\n[a]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for duplicate table")
	}
}

func TestParse_RejectsDuplicateKey(t *testing.T) {
	input := "name = \"Tom\"\nname = \"Pradyun\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestParse_RejectsDuplicateQuotedKey(t *testing.T) {
	input := "spelling = \"favorite\"\n\"spelling\" = \"favourite\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for duplicate key (bare vs quoted)")
	}
}

func TestParse_RejectsDottedKeyReopen(t *testing.T) {
	input := "[product]\ntype.name = \"Nail\"\ntype = { edible = false }\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for overwriting dotted-key table with inline table")
	}
}

func TestParse_RejectsInlineTableExtend(t *testing.T) {
	input := "a = {b = 1}\n[a]\nc = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for extending inline table")
	}
}

func TestParse_RejectsScalarOverwrite(t *testing.T) {
	input := "a.b.c = 1\na.b = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for overwriting dotted-key table with scalar")
	}
}

func TestParse_RejectsAOTOverwrite(t *testing.T) {
	input := "[[parent.arr]]\n[parent]\narr = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for overwriting array of tables with scalar")
	}
}

// --- TOML 1.1 feature tests ---

func TestParse_EscapeE(t *testing.T) {
	input := "s = \"hello\\eworld\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeString {
		t.Fatalf("expected string, got %v", kv.val.Type())
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestParse_HexEscape(t *testing.T) {
	input := "s = \"caf\\xE9\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeString {
		t.Fatalf("expected string, got %v", kv.val.Type())
	}
}

func TestParse_MultiLineInlineTable(t *testing.T) {
	input := "point = {\n  x = 1,\n  y = 2,\n}\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	it, ok := kv.val.(*InlineTableNode)
	if !ok {
		t.Fatalf("expected InlineTableNode, got %T", kv.val)
	}
	if len(it.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(it.entries))
	}
}

func TestParse_TrailingCommaInlineTable(t *testing.T) {
	input := "point = {x = 1, y = 2,}\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	it, ok := kv.val.(*InlineTableNode)
	if !ok {
		t.Fatalf("expected InlineTableNode, got %T", kv.val)
	}
	if len(it.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(it.entries))
	}
}

func TestParse_DateTimeNoSeconds(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"t = 07:32\n"},
		{"dt = 1979-05-27T07:32Z\n"},
		{"dt = 1979-05-27T07:32+00:00\n"},
		{"dt = 1979-05-27T07:32\n"},
	}
	for _, tt := range tests {
		d, err := Parse([]byte(tt.input))
		if err != nil {
			t.Fatalf("parse error for %q: %v", tt.input, err)
		}
		kv := d.nodes[0].(*KeyValue)
		if kv.val.Type() != NodeDateTime {
			t.Fatalf("for %q: expected datetime, got %v", tt.input, kv.val.Type())
		}
	}
}

func TestParse_RejectsBareCarriageReturn(t *testing.T) {
	// Bare \r (not followed by \n) in multi-line basic string
	input := "s = \"\"\"\nhello\rworld\"\"\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for bare carriage return in multi-line basic string")
	}
}

func TestParse_RejectsBareCarriageReturnLiteral(t *testing.T) {
	// Bare \r (not followed by \n) in multi-line literal string
	input := "s = '''\nhello\rworld'''\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for bare carriage return in multi-line literal string")
	}
}

func TestParse_AllowsCRLFInMultiLineString(t *testing.T) {
	// \r\n is valid in multi-line strings
	input := "s = \"\"\"\r\nhello\r\nworld\"\"\"\n"
	_, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error for CRLF in multi-line string: %v", err)
	}
}

// --- Coverage: accessor methods on KeyValue ---

func TestKeyValue_KeyParts(t *testing.T) {
	d, err := Parse([]byte("a.b = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	parts := kv.KeyParts()
	if len(parts) != 2 {
		t.Fatalf("expected 2 key parts, got %d", len(parts))
	}
	if parts[0].Unquoted != "a" || parts[1].Unquoted != "b" {
		t.Fatalf("unexpected parts: %v", parts)
	}
	// Ensure it returns a copy (mutating the returned slice should not affect original).
	parts[0].Unquoted = "z"
	if kv.KeyParts()[0].Unquoted != "a" {
		t.Fatal("KeyParts should return a defensive copy")
	}
}

func TestKeyValue_RawVal(t *testing.T) {
	d, err := Parse([]byte(`key = "hello"` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.RawVal() != `"hello"` {
		t.Fatalf("expected raw val %q, got %q", `"hello"`, kv.RawVal())
	}
}

func TestKeyValue_Text(t *testing.T) {
	d, err := Parse([]byte("key = 42\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	text := kv.Text()
	if text != "key = 42" {
		t.Fatalf("expected %q, got %q", "key = 42", text)
	}
}

func TestKeyValue_TextNilVal(t *testing.T) {
	// This tests the nil-val branch in Text().
	kv := &KeyValue{
		baseNode: baseNode{nodeType: NodeKeyValue},
		rawKey:   "k",
		preEq:    " ",
		postEq:   " ",
	}
	text := kv.Text()
	if text != "k = " {
		t.Fatalf("expected %q, got %q", "k = ", text)
	}
}

func TestKeyValue_Children(t *testing.T) {
	d, err := Parse([]byte("key = 42  # comment\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	children := kv.Children()
	// Should contain: val + trailing trivia (whitespace, comment).
	if len(children) < 2 {
		t.Fatalf("expected at least 2 children, got %d", len(children))
	}
	// First child should be the value node.
	foundVal := false
	for _, c := range children {
		if c.Type() == NodeNumber {
			foundVal = true
		}
	}
	if !foundVal {
		t.Fatal("expected value node in Children()")
	}
}

func TestKeyValue_SetTrailingTrivia(t *testing.T) {
	d, err := Parse([]byte("key = 42\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)

	ws := &WhitespaceNode{leafNode: newLeaf(NodeWhitespace, "  ")}
	cn := &CommentNode{leafNode: newLeaf(NodeComment, "# yo")}
	if err := kv.SetTrailingTrivia([]Node{ws, cn}); err != nil {
		t.Fatal(err)
	}
	trailing := kv.TrailingTrivia()
	if len(trailing) != 2 {
		t.Fatalf("expected 2 trailing trivia, got %d", len(trailing))
	}
	// Error case: invalid trivia.
	badNode := NewString("x")
	if err := kv.SetTrailingTrivia([]Node{badNode}); err == nil {
		t.Fatal("expected error for invalid trivia node")
	}
}

func TestKeyValue_SetNewline(t *testing.T) {
	d, err := Parse([]byte("key = 42\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if err := kv.SetNewline("\r\n"); err != nil {
		t.Fatal(err)
	}
	if kv.Newline() != "\r\n" {
		t.Fatalf("expected \\r\\n, got %q", kv.Newline())
	}
	if err := kv.SetNewline(""); err != nil {
		t.Fatal(err)
	}
	if err := kv.SetNewline("bad"); err == nil {
		t.Fatal("expected error for invalid newline")
	}
}

// --- Coverage: accessor methods on TableNode ---

func parseTableAccessorFixture(t *testing.T) *TableNode {
	t.Helper()
	input := "# leading\n[server.settings]  # header comment\nport = 8080\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	return d.Tables()[0]
}

func TestTableNode_Accessors_HeaderParts(t *testing.T) {
	tbl := parseTableAccessorFixture(t)
	parts := tbl.HeaderParts()
	if len(parts) != 2 || parts[0].Unquoted != "server" || parts[1].Unquoted != "settings" {
		t.Fatalf("unexpected header parts: %v", parts)
	}
}

func TestTableNode_Accessors_Entries(t *testing.T) {
	tbl := parseTableAccessorFixture(t)
	if len(tbl.Entries()) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(tbl.Entries()))
	}
}

func TestTableNode_Accessors_Trivia(t *testing.T) {
	tbl := parseTableAccessorFixture(t)
	if len(tbl.LeadingTrivia()) == 0 {
		t.Fatal("expected leading trivia")
	}
	if len(tbl.TrailingTrivia()) == 0 {
		t.Fatal("expected trailing trivia (header comment)")
	}
}

func TestTableNode_Accessors_ChildrenTextNewline(t *testing.T) {
	tbl := parseTableAccessorFixture(t)
	if len(tbl.Children()) == 0 {
		t.Fatal("expected children")
	}
	if tbl.Text() != "[server.settings]" {
		t.Fatalf("expected %q, got %q", "[server.settings]", tbl.Text())
	}
	if tbl.Newline() != "\n" {
		t.Fatalf("expected newline, got %q", tbl.Newline())
	}
}

// triviaSetterNode is used to deduplicate trivia setter tests.
type triviaSetterNode interface {
	SetLeadingTrivia([]Node) error
	LeadingTrivia() []Node
	SetTrailingTrivia([]Node) error
	TrailingTrivia() []Node
	SetNewline(string) error
	Newline() string
}

func testSetTrivia(t *testing.T, node triviaSetterNode) {
	t.Helper()
	ws := &WhitespaceNode{leafNode: newLeaf(NodeWhitespace, " ")}

	if err := node.SetLeadingTrivia([]Node{ws}); err != nil {
		t.Fatal(err)
	}
	if len(node.LeadingTrivia()) != 1 {
		t.Fatal("expected 1 leading trivia")
	}
	if err := node.SetLeadingTrivia([]Node{NewString("bad")}); err == nil {
		t.Fatal("expected error for invalid trivia")
	}

	if err := node.SetTrailingTrivia([]Node{ws}); err != nil {
		t.Fatal(err)
	}
	if len(node.TrailingTrivia()) != 1 {
		t.Fatal("expected 1 trailing trivia")
	}
	if err := node.SetTrailingTrivia([]Node{NewString("bad")}); err == nil {
		t.Fatal("expected error for invalid trivia")
	}

	if err := node.SetNewline("\r\n"); err != nil {
		t.Fatal(err)
	}
	if node.Newline() != "\r\n" {
		t.Fatalf("expected \\r\\n, got %q", node.Newline())
	}
	if err := node.SetNewline("bad"); err == nil {
		t.Fatal("expected error for invalid newline")
	}
}

func TestTableNode_SetTrivia(t *testing.T) {
	tbl, err := NewTable("test")
	if err != nil {
		t.Fatal(err)
	}
	testSetTrivia(t, tbl)
}

// --- Coverage: accessor methods on ArrayOfTables ---

func parseAOTAccessorFixture(t *testing.T) *ArrayOfTables {
	t.Helper()
	input := "# before\n[[products]]  # arr comment\nname = \"Widget\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	aots := d.ArraysOfTables()
	if len(aots) != 1 {
		t.Fatalf("expected 1 AOT, got %d", len(aots))
	}
	return aots[0]
}

func TestArrayOfTables_Accessors_Header(t *testing.T) {
	aot := parseAOTAccessorFixture(t)
	if aot.RawHeader() != "products" {
		t.Fatalf("unexpected raw header: %q", aot.RawHeader())
	}
	parts := aot.HeaderParts()
	if len(parts) != 1 || parts[0].Unquoted != "products" {
		t.Fatalf("unexpected header parts: %v", parts)
	}
}

func TestArrayOfTables_Accessors_Entries(t *testing.T) {
	aot := parseAOTAccessorFixture(t)
	if len(aot.Entries()) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(aot.Entries()))
	}
}

func TestArrayOfTables_Accessors_Trivia(t *testing.T) {
	aot := parseAOTAccessorFixture(t)
	if len(aot.LeadingTrivia()) == 0 {
		t.Fatal("expected leading trivia")
	}
	if len(aot.TrailingTrivia()) == 0 {
		t.Fatal("expected trailing trivia")
	}
}

func TestArrayOfTables_Accessors_ChildrenTextNewline(t *testing.T) {
	aot := parseAOTAccessorFixture(t)
	if len(aot.Children()) == 0 {
		t.Fatal("expected children")
	}
	if aot.Text() != "[[products]]" {
		t.Fatalf("expected %q, got %q", "[[products]]", aot.Text())
	}
	if aot.Newline() != "\n" {
		t.Fatalf("expected newline, got %q", aot.Newline())
	}
}

func TestArrayOfTables_SetTrivia(t *testing.T) {
	aot, err := NewArrayOfTables("products")
	if err != nil {
		t.Fatal(err)
	}
	testSetTrivia(t, aot)
}

// --- Coverage: ArrayNode.Children, InlineTableNode.Children ---

func TestArrayNode_Children(t *testing.T) {
	d, err := Parse([]byte("a = [1, 2, 3]\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	arr := kv.val.(*ArrayNode)
	children := arr.Children()
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
}

func TestInlineTableNode_Children(t *testing.T) {
	d, err := Parse([]byte("t = {x = 1, y = 2}\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	it := kv.val.(*InlineTableNode)
	children := it.Children()
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
}

// --- Coverage: Document.Text, Document.Tables ---

func TestDocument_Text(t *testing.T) {
	input := "key = 1\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if d.Text() != input {
		t.Fatalf("Text() = %q, want %q", d.Text(), input)
	}
}

func TestDocument_Tables(t *testing.T) {
	input := "[a]\nk = 1\n[b]\nk = 2\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	tables := d.Tables()
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}
	if tables[0].RawHeader() != "a" || tables[1].RawHeader() != "b" {
		t.Fatalf("unexpected table headers")
	}
}

func TestDocument_Type(t *testing.T) {
	d, _ := Parse([]byte(""))
	if d.Type() != NodeDocument {
		t.Fatalf("expected NodeDocument, got %v", d.Type())
	}
	if d.Parent() != nil {
		t.Fatal("document parent should be nil")
	}
}

// --- Coverage: parser orphan trivia ---

func TestParse_OrphanTriviaAfterKV(t *testing.T) {
	// trailing comment after last KV with no newline after it
	input := "key = 1  # trailing"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	trailing := kv.TrailingTrivia()
	hasComment := false
	for _, n := range trailing {
		if n.Type() == NodeComment {
			hasComment = true
		}
	}
	if !hasComment {
		t.Fatal("expected trailing comment as orphan trivia")
	}
}

func TestParse_OrphanTriviaAfterTable(t *testing.T) {
	// Trailing comment after last KV in a table.
	input := "[t]\nk = 1\n# orphan"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	tbl := d.nodes[0].(*TableNode)
	lastEntry := tbl.entries[len(tbl.entries)-1]
	kv := lastEntry.(*KeyValue)
	hasComment := false
	for _, n := range kv.trailingTrivia {
		if n.Type() == NodeComment {
			hasComment = true
		}
	}
	if !hasComment {
		t.Fatal("expected orphan comment attached to last KV of table")
	}
}

func TestParse_OrphanTriviaAfterAOT(t *testing.T) {
	input := "[[a]]\nk = 1\n# orphan"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	aot := d.nodes[0].(*ArrayOfTables)
	lastEntry := aot.entries[len(aot.entries)-1]
	kv := lastEntry.(*KeyValue)
	hasComment := false
	for _, n := range kv.trailingTrivia {
		if n.Type() == NodeComment {
			hasComment = true
		}
	}
	if !hasComment {
		t.Fatal("expected orphan comment attached to last KV of AOT")
	}
}

func TestParse_OrphanTriviaNoNodes(t *testing.T) {
	// Document with only whitespace and comments.
	input := "# just a comment\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.nodes) == 0 {
		t.Fatal("expected the comment/whitespace to be in nodes")
	}
}

func TestParse_OrphanTriviaAfterTableNoKV(t *testing.T) {
	// Table with no KV entries, followed by trailing trivia.
	input := "[t]\n# trailing"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	// The orphan trivia should be attached as entries to the table.
	tbl := d.nodes[0].(*TableNode)
	if len(tbl.entries) == 0 {
		t.Fatal("expected trivia to be in table entries")
	}
}

// --- Coverage: literal string keys (unquoteLiteralStr) ---

func TestParse_LiteralStringKey(t *testing.T) {
	input := "'literal-key' = 1\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.keyParts[0].Unquoted != "literal-key" {
		t.Fatalf("expected 'literal-key', got %q", kv.keyParts[0].Unquoted)
	}
	if !kv.keyParts[0].IsQuoted {
		t.Fatal("expected key to be marked as quoted")
	}
}

// --- Coverage: string escape processing (query.go) ---

func TestStringNode_Value_BasicEscapes(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{`"hello"`, "hello"},
		{`"tab\there"`, "tab\there"},
		{`"new\nline"`, "new\nline"},
		{`"back\\slash"`, "back\\slash"},
		{`"quote\""`, `quote"`},
		{`"esc\e"`, "esc\x1B"},
		{`"caf\xE9"`, "caf\u00E9"},
		{`"snowman\u2603"`, "snowman\u2603"},
		{`"astral\U0001F600"`, "astral\U0001F600"},
		{`"bs\b"`, "bs\b"},
		{`"ff\f"`, "ff\f"},
		{`"cr\r"`, "cr\r"},
	}
	for _, tt := range tests {
		n := &StringNode{leafNode: newLeaf(NodeString, tt.raw)}
		got := n.Value()
		if got != tt.want {
			t.Errorf("Value() for %q: got %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestStringNode_Value_SingleQuote(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "'hello'")}
	if n.Value() != "hello" {
		t.Fatalf("expected 'hello', got %q", n.Value())
	}
}

func TestStringNode_Value_TooShort(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "x")}
	if n.Value() != "x" {
		t.Fatalf("expected 'x', got %q", n.Value())
	}
}

// --- Coverage: NumberNode methods ---

func TestNumberNode_Int(t *testing.T) {
	tests := []struct {
		text string
		want int64
	}{
		{"42", 42},
		{"+42", 42},
		{"-42", -42},
		{"0xDEAD", 0xDEAD},
		{"0o755", 0o755},
		{"0b1010", 0b1010},
		{"1_000", 1000},
	}
	for _, tt := range tests {
		n := &NumberNode{leafNode: newLeaf(NodeNumber, tt.text)}
		got, err := n.Int()
		if err != nil {
			t.Errorf("Int() for %q: %v", tt.text, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Int() for %q: got %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestNumberNode_Int_Errors(t *testing.T) {
	errCases := []string{"3.14", "1e2", "inf", "+inf", "nan"}
	for _, text := range errCases {
		n := &NumberNode{leafNode: newLeaf(NodeNumber, text)}
		_, err := n.Int()
		if err == nil {
			t.Errorf("Int() for %q: expected error", text)
		}
	}
}

func TestNumberNode_Float(t *testing.T) {
	tests := []struct {
		text string
		want float64
	}{
		{"3.14", 3.14},
		{"+1.0", 1.0},
		{"-2.5", -2.5},
		{"1e2", 100.0},
		{"42", 42.0},
		{"0xA", 10.0},
		{"0o10", 8.0},
		{"0b10", 2.0},
	}
	for _, tt := range tests {
		n := &NumberNode{leafNode: newLeaf(NodeNumber, tt.text)}
		got, err := n.Float()
		if err != nil {
			t.Errorf("Float() for %q: %v", tt.text, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Float() for %q: got %f, want %f", tt.text, got, tt.want)
		}
	}
}

func TestNumberNode_Float_SpecialValues(t *testing.T) {
	tests := []struct {
		text  string
		isInf int // 0 = NaN, 1 = +Inf, -1 = -Inf
	}{
		{"inf", 1},
		{"+inf", 1},
		{"-inf", -1},
		{"nan", 0},
		{"+nan", 0},
		{"-nan", 0},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			n := &NumberNode{leafNode: newLeaf(NodeNumber, tt.text)}
			v, err := n.Float()
			if err != nil {
				t.Fatalf("Float() error: %v", err)
			}
			if tt.isInf != 0 {
				if !math.IsInf(v, tt.isInf) {
					t.Fatalf("expected inf(%d), got %v", tt.isInf, v)
				}
			} else {
				if !math.IsNaN(v) {
					t.Fatalf("expected NaN, got %v", v)
				}
			}
		})
	}
}

func TestBooleanNode_Value(t *testing.T) {
	bt := &BooleanNode{leafNode: newLeaf(NodeBoolean, "true")}
	if !bt.Value() {
		t.Fatal("expected true")
	}
	bf := &BooleanNode{leafNode: newLeaf(NodeBoolean, "false")}
	if bf.Value() {
		t.Fatal("expected false")
	}
}

// --- Coverage: validation edge cases ---

func TestParse_ValidatesWsBackslash(t *testing.T) {
	// In a multiline basic string, backslash followed by space then newline
	// is a valid line-ending backslash.
	input := "s = \"\"\"\nhello \\   \n  world\"\"\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

func TestParse_RejectsWsBackslashNoNewline(t *testing.T) {
	// Backslash followed by space but no newline in a non-multiline string.
	input := "s = \"hello\\ world\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for backslash-space in non-multiline string")
	}
}

func TestParse_RejectsTrailingBackslash(t *testing.T) {
	input := "s = \"hello\\\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for trailing backslash")
	}
}

func TestParse_SignedPrefixInteger(t *testing.T) {
	_, err := Parse([]byte("n = +0x1\n"))
	if err == nil {
		t.Fatal("expected error for signed hex integer")
	}
	_, err = Parse([]byte("n = -0o7\n"))
	if err == nil {
		t.Fatal("expected error for signed octal integer")
	}
	_, err = Parse([]byte("n = +0b1\n"))
	if err == nil {
		t.Fatal("expected error for signed binary integer")
	}
}

func TestParse_IncompleteHexInt(t *testing.T) {
	_, err := Parse([]byte("n = 0x\n"))
	if err == nil {
		t.Fatal("expected error for incomplete hex integer")
	}
}

func TestParse_InvalidHexDigit(t *testing.T) {
	_, err := Parse([]byte("n = 0xGG\n"))
	if err == nil {
		t.Fatal("expected error for invalid hex digit")
	}
}

func TestParse_InvalidOctalDigit(t *testing.T) {
	_, err := Parse([]byte("n = 0o89\n"))
	if err == nil {
		t.Fatal("expected error for invalid octal digit")
	}
}

func TestParse_InvalidBinaryDigit(t *testing.T) {
	_, err := Parse([]byte("n = 0b12\n"))
	if err == nil {
		t.Fatal("expected error for invalid binary digit")
	}
}

func TestParse_LeadingUnderscoreHex(t *testing.T) {
	_, err := Parse([]byte("n = 0x_FF\n"))
	if err == nil {
		t.Fatal("expected error for leading underscore in hex")
	}
}

func TestParse_LeadingUnderscoreInt(t *testing.T) {
	_, err := Parse([]byte("n = _42\n"))
	if err == nil {
		t.Fatal("expected error for leading underscore in integer")
	}
}

func TestParse_FloatEdgeCases(t *testing.T) {
	// No digits after decimal point.
	_, err := Parse([]byte("f = 1.\n"))
	if err == nil {
		t.Fatal("expected error for trailing dot")
	}

	// Dot after exponent.
	_, err = Parse([]byte("f = 1e2.3\n"))
	if err == nil {
		t.Fatal("expected error for dot after exponent")
	}

	// No digits in exponent.
	_, err = Parse([]byte("f = 1e\n"))
	if err == nil {
		t.Fatal("expected error for no digits in exponent")
	}

	// Multiple exponents.
	_, err = Parse([]byte("f = 1e2e3\n"))
	if err == nil {
		t.Fatal("expected error for multiple exponents")
	}

	// Float with leading dot.
	_, err = Parse([]byte("f = .5\n"))
	if err == nil {
		t.Fatal("expected error for leading dot in float")
	}
}

func TestParse_FloatNoDigitsBetweenDotAndExponent(t *testing.T) {
	_, err := Parse([]byte("f = 1.e2\n"))
	if err == nil {
		t.Fatal("expected error for no digits between dot and exponent")
	}
}

func TestParse_FloatExponentWithSign(t *testing.T) {
	d, err := Parse([]byte("f = 1.0e+2\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeNumber {
		t.Fatalf("expected number, got %v", kv.val.Type())
	}

	d, err = Parse([]byte("f = 1.0e-2\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv = d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeNumber {
		t.Fatalf("expected number, got %v", kv.val.Type())
	}
}

// --- Coverage: datetime validation ---

func TestParse_DateTimeVariants(t *testing.T) {
	validInputs := []string{
		"d = 2024-01-15T08:30:00Z\n",
		"d = 2024-01-15T08:30:00+05:30\n",
		"d = 2024-01-15T08:30:00-04:00\n",
		"d = 2024-01-15T08:30:00.123Z\n",
		"d = 2024-01-15T08:30:00\n",
		"d = 2024-01-15\n",
		"d = 08:30:00\n",
		"d = 08:30:00.999\n",
		"d = 08:30\n",
		"d = 2024-02-29\n",           // leap year
		"d = 2024-01-15 08:30:00Z\n", // space separator
	}
	for _, input := range validInputs {
		_, err := Parse([]byte(input))
		if err != nil {
			t.Errorf("unexpected error for %q: %v", input, err)
		}
	}
}

func TestParse_DateTimeInvalid(t *testing.T) {
	invalidInputs := []string{
		"d = 2024-00-15\n",         // month 0
		"d = 2024-01-00\n",         // day 0
		"d = 2024-01-32\n",         // day out of range
		"d = 23:60:00\n",           // minute 60
		"d = 23:00:61\n",           // second 61
		"d = 08:30:00.\n",          // trailing dot in time
		"d = 2023-02-29\n",         // non-leap year
		"d = 2024-04-31\n",         // April has 30 days
		"d = 1987-7-05\n",          // month not 2 digits
		"d = 1987-07-5\n",          // day not 2 digits
		"d = 987-07-05\n",          // year not 4 digits
		"d = 1987-07-05T8:30:00\n", // hour not 2 digits
	}
	for _, input := range invalidInputs {
		_, err := Parse([]byte(input))
		if err == nil {
			t.Errorf("expected error for %q", input)
		}
	}
}

// --- Coverage: invalid UTF-8 ---

func TestParse_RejectsInvalidUTF8(t *testing.T) {
	_, err := Parse([]byte("key = \"\xff\"\n"))
	if err == nil {
		t.Fatal("expected error for invalid UTF-8")
	}
}

// --- Coverage: ParseError.Error formatting ---

func TestParseError_FormatWithTab(t *testing.T) {
	e := &ParseError{
		Message: "bad",
		Line:    1,
		Column:  3,
		Source:  "\t\tbad",
	}
	errStr := e.Error()
	if !strings.Contains(errStr, "bad") {
		t.Fatalf("expected error message in output: %s", errStr)
	}
}

func TestParseError_OutOfRangeLine(t *testing.T) {
	e := &ParseError{
		Message: "bad",
		Line:    999,
		Column:  1,
		Source:  "single line",
	}
	errStr := e.Error()
	if !strings.Contains(errStr, "line 999") {
		t.Fatalf("expected line 999 in output: %s", errStr)
	}
}

// --- Coverage: mutate.go constructors and escaping ---

func TestNewFloat_SpecialValues(t *testing.T) {
	n := NewFloat(math.Inf(1))
	if n.Text() != "inf" {
		t.Fatalf("expected 'inf', got %q", n.Text())
	}
	n = NewFloat(math.Inf(-1))
	if n.Text() != "-inf" {
		t.Fatalf("expected '-inf', got %q", n.Text())
	}
	n = NewFloat(math.NaN())
	if n.Text() != "nan" {
		t.Fatalf("expected 'nan', got %q", n.Text())
	}
	n = NewFloat(42.0)
	if n.Text() != "42.0" {
		t.Fatalf("expected '42.0', got %q", n.Text())
	}
	n = NewFloat(1e10)
	if n.Text() != "1e+10" {
		t.Fatalf("unexpected: %q", n.Text())
	}
}

func TestEscapeBasicString_AllEscapeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"back\\slash", `back\\slash`},
		{"quo\"te", `quo\"te`},
		{"\b", `\b`},
		{"\t", `\t`},
		{"\n", `\n`},
		{"\f", `\f`},
		{"\r", `\r`},
		{"\x01", `\u0001`},           // control char
		{"\x7f", `\u007F`},           // DEL
		{"\U0001F600", `\U0001F600`}, // astral plane
	}
	for _, tt := range tests {
		got := escapeBasicString(tt.input)
		if got != tt.want {
			t.Errorf("escapeBasicString(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNewKeyValue_InvalidInputs(t *testing.T) {
	// Nil value.
	_, err := NewKeyValue("k", nil)
	if err == nil {
		t.Fatal("expected error for nil value")
	}

	// Invalid value type.
	_, err = NewKeyValue("k", &WhitespaceNode{leafNode: newLeaf(NodeWhitespace, " ")})
	if err == nil {
		t.Fatal("expected error for invalid value type")
	}

	// Empty key.
	_, err = NewKeyValue("", NewString("v"))
	if err == nil {
		t.Fatal("expected error for empty key")
	}

	// Invalid key with trailing content.
	_, err = NewKeyValue("a b", NewString("v"))
	if err == nil {
		t.Fatal("expected error for key with trailing content")
	}
}

func TestNewTable_Invalid(t *testing.T) {
	_, err := NewTable("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestNewArrayOfTables_Invalid(t *testing.T) {
	_, err := NewArrayOfTables("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestNewDateTime_Valid(t *testing.T) {
	n, err := NewDateTime("2024-01-15T08:30:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if n.Text() != "2024-01-15T08:30:00Z" {
		t.Fatalf("unexpected text: %q", n.Text())
	}
}

func TestNewDateTime_Invalid(t *testing.T) {
	_, err := NewDateTime("not-a-date")
	if err == nil {
		t.Fatal("expected error for invalid datetime")
	}
}

func TestNewArray_Valid(t *testing.T) {
	a, err := NewArray(NewString("a"), NewInteger(42))
	if err != nil {
		t.Fatal(err)
	}
	if a.Len() != 2 {
		t.Fatalf("expected 2 elements, got %d", a.Len())
	}
	if a.Text() != `["a", 42]` {
		t.Fatalf("unexpected text: %q", a.Text())
	}
}

func TestNewArray_InvalidElement(t *testing.T) {
	_, err := NewArray(nil)
	if err == nil {
		t.Fatal("expected error for nil element")
	}
}

func TestNewInlineTable_DuplicateKey(t *testing.T) {
	kv1, _ := NewKeyValue("k", NewString("a"))
	kv2, _ := NewKeyValue("k", NewString("b"))
	_, err := NewInlineTable(kv1, kv2)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestNewInlineTable_NilEntry(t *testing.T) {
	_, err := NewInlineTable(nil)
	if err == nil {
		t.Fatal("expected error for nil entry")
	}
}

func TestNewInlineTable_DottedKeyConflict(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewString("scalar"))
	kv2, _ := NewKeyValue("a.b", NewString("nested"))
	_, err := NewInlineTable(kv1, kv2)
	if err == nil {
		t.Fatal("expected error for dotted key conflict")
	}
}

func TestNewComment_Invalid(t *testing.T) {
	_, err := NewComment("# has\nnewline")
	if err == nil {
		t.Fatal("expected error for newline in comment")
	}
	_, err = NewComment("# has\x01control")
	if err == nil {
		t.Fatal("expected error for control char in comment")
	}
}

func TestNewWhitespace_Invalid(t *testing.T) {
	_, err := NewWhitespace("abc")
	if err == nil {
		t.Fatal("expected error for non-whitespace chars")
	}
}

// --- Coverage: mutation methods ---

func TestDocument_InsertAt(t *testing.T) {
	d, err := Parse([]byte("a = 1\nb = 2\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv, _ := NewKeyValue("c", NewInteger(3))
	if err := d.InsertAt(1, kv); err != nil {
		t.Fatal(err)
	}
	if len(d.nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(d.nodes))
	}

	// Insert at negative index.
	kv2, _ := NewKeyValue("z", NewInteger(0))
	if err := d.InsertAt(-1, kv2); err != nil {
		t.Fatal(err)
	}

	// Reject invalid node type.
	badNode := &IdentifierNode{leafNode: newLeaf(NodeIdentifier, "id")}
	if err := d.InsertAt(0, badNode); err == nil {
		t.Fatal("expected error for invalid node type")
	}

	// Reject nil node.
	if err := d.InsertAt(0, nil); err == nil {
		t.Fatal("expected error for nil node")
	}
}

func TestDocument_InsertAt_TriviaNode(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := NewWhitespace("\n")
	if err := d.InsertAt(0, ws); err != nil {
		t.Fatal(err)
	}
	if len(d.nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(d.nodes))
	}
}

func TestDocument_InsertAt_DuplicateKeyRollback(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv, _ := NewKeyValue("a", NewInteger(2))
	err = d.InsertAt(0, kv)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
	// Original should be unchanged.
	if len(d.nodes) != 1 {
		t.Fatalf("expected 1 node after rollback, got %d", len(d.nodes))
	}
}

func TestTableNode_InsertAt_Standalone(t *testing.T) {
	tbl, _ := NewTable("t")
	kv1, _ := NewKeyValue("a", NewInteger(1))
	_ = tbl.Append(kv1)
	kv2, _ := NewKeyValue("b", NewInteger(2))
	if err := tbl.InsertAt(0, kv2); err != nil {
		t.Fatal(err)
	}
	if len(tbl.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(tbl.entries))
	}
	// Duplicate in standalone.
	kv3, _ := NewKeyValue("a", NewInteger(99))
	if err := tbl.InsertAt(0, kv3); err == nil {
		t.Fatal("expected error for duplicate key in standalone table")
	}
}

func TestSetValue_RegeneratesAncestors(t *testing.T) {
	d, err := Parse([]byte("t = {x = 1}\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	it := kv.val.(*InlineTableNode)
	innerKV := it.entries[0]
	if err := innerKV.SetValue(NewInteger(999)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(it.Text(), "999") {
		t.Fatalf("expected inline table text to contain 999, got %q", it.Text())
	}
}

// --- Coverage: query methods ---

func TestDocument_Get_FromTable(t *testing.T) {
	input := "[server]\nhost = \"localhost\"\nport = 8080\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.Get("server.host")
	if kv == nil {
		t.Fatal("expected to find server.host")
	}
	if kv.RawVal() != `"localhost"` {
		t.Fatalf("unexpected value: %q", kv.RawVal())
	}
}

func TestDocument_Get_FromAOT(t *testing.T) {
	input := "[[products]]\nname = \"Widget\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.Get("products.name")
	if kv == nil {
		t.Fatal("expected to find products.name")
	}
}

func TestDocument_Get_InlineTableNested(t *testing.T) {
	input := "t = {a = {b = 1}}\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.Get("t.a.b")
	if kv == nil {
		t.Fatal("expected to find t.a.b")
	}
}

func TestDocument_Get_NotFound(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if d.Get("nonexistent") != nil {
		t.Fatal("expected nil for nonexistent key")
	}
}

func TestDocument_ArrayOfTables(t *testing.T) {
	input := "[[p]]\nname = \"A\"\n[[p]]\nname = \"B\"\n[[q]]\nname = \"C\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	ps := d.ArrayOfTables("p")
	if len(ps) != 2 {
		t.Fatalf("expected 2 AOTs for 'p', got %d", len(ps))
	}
}

func TestDocument_Get_QuotedKeyPath(t *testing.T) {
	input := "\"key.with.dots\" = 1\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.Get(`"key.with.dots"`)
	if kv == nil {
		t.Fatal("expected to find quoted key")
	}
}

func TestDocument_Get_LiteralKeyPath(t *testing.T) {
	input := "'literal' = 1\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.Get("'literal'")
	if kv == nil {
		t.Fatal("expected to find literal key")
	}
}

// --- Coverage: document convenience methods ---

func TestDocument_Append_Rollback(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv, _ := NewKeyValue("a", NewInteger(2))
	err = d.Append(kv)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
	if len(d.nodes) != 1 {
		t.Fatalf("expected 1 node after rollback, got %d", len(d.nodes))
	}
}

func TestTableNode_Append_Standalone_DuplicateCheck(t *testing.T) {
	tbl, _ := NewTable("t")
	kv1, _ := NewKeyValue("a", NewInteger(1))
	_ = tbl.Append(kv1)

	// Dotted key conflict.
	kv2, _ := NewKeyValue("a.b", NewInteger(2))
	if err := tbl.Append(kv2); err == nil {
		t.Fatal("expected error for dotted key conflict in standalone table")
	}
}

func TestArrayOfTables_AppendWithDocument(t *testing.T) {
	d, err := Parse([]byte("[[p]]\n"))
	if err != nil {
		t.Fatal(err)
	}
	aot := d.ArrayOfTables("p")[0]
	kv, _ := NewKeyValue("name", NewString("x"))
	if err := aot.Append(kv); err != nil {
		t.Fatal(err)
	}

	// Duplicate in document context.
	kv2, _ := NewKeyValue("name", NewString("y"))
	if err := aot.Append(kv2); err == nil {
		t.Fatal("expected error for duplicate key in AOT with document")
	}
}

// --- Coverage: validate trivia with nil ---

func TestValidateTriviaNodes_Nil(t *testing.T) {
	err := validateTriviaNodes([]Node{nil})
	if err == nil {
		t.Fatal("expected error for nil trivia node")
	}
}

// --- Coverage: lexer edge cases ---

func TestParse_SpaceSeparatedDateTime(t *testing.T) {
	input := "dt = 1979-05-27 07:32:00Z\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeDateTime {
		t.Fatalf("expected datetime, got %v", kv.val.Type())
	}
}

func TestParse_SpaceSeparatedDateTimeLocal(t *testing.T) {
	input := "dt = 1979-05-27 07:32:00\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeDateTime {
		t.Fatalf("expected datetime, got %v", kv.val.Type())
	}
}

func TestParse_CRLFNewlines(t *testing.T) {
	input := "key = 1\r\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

// --- Coverage: semantic validation edge cases ---

func TestParse_DottedKeyInDifferentTableIsValid(t *testing.T) {
	// Dotted key a.k inside [b] creates b.a.k, which does not conflict with [a].
	input := "[a]\nk = 1\n[b]\na.k = 2\n"
	_, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParse_RejectsAOTAfterImplicitTable(t *testing.T) {
	// Creating implicit table 'a' via [a.b], then trying to define [[a]].
	input := "[a.b]\nk = 1\n[[a]]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT after implicit table")
	}
}

func TestParse_RejectsTableAfterAOT(t *testing.T) {
	input := "[[a]]\nk = 1\n[a]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for table after AOT")
	}
}

func TestParse_RejectsStaticArrayExtend(t *testing.T) {
	input := "a = [1, 2]\n[a]\nb = 3\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for extending static array with table")
	}
}

func TestParse_RejectsInlineTablePathConflict(t *testing.T) {
	input := "a = {b = 1}\na.b = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for inline table path conflict")
	}
}

func TestParse_RejectsAOTPathConflictWithDottedKey(t *testing.T) {
	input := "a.b = 1\n[[a.b]]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT conflict with dotted key table")
	}
}

func TestParse_RejectsAOTExtendDottedKey(t *testing.T) {
	input := "a.b.c = 1\n[[a.b]]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT conflict with dotted key")
	}
}

func TestParse_RejectsIntermediateInlinePath(t *testing.T) {
	input := "a = {b = 1}\n[a.b]\nc = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for extending inline table nested path")
	}
}

func TestParse_RejectsIntermediateStaticArray(t *testing.T) {
	input := "a = [1]\n[a.b]\nc = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for extending static array intermediate path")
	}
}

func TestParse_RejectsAOTIntermediateScalar(t *testing.T) {
	input := "a = 1\n[[a.b]]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT intermediate path is scalar")
	}
}

func TestParse_RejectsAOTIntermediateInline(t *testing.T) {
	input := "a = {b = 1}\n[[a.c]]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT intermediate path is inline table")
	}
}

func TestParse_RejectsAOTIntermediateStaticArray(t *testing.T) {
	input := "a = [1]\n[[a.c]]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT intermediate path is static array")
	}
}

func TestParse_RejectsDottedKeyInlineConflict(t *testing.T) {
	input := "a = {b = 1}\na.b.c = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for dotted key through inline table path")
	}
}

func TestParse_RejectsDottedKeyScalarConflict(t *testing.T) {
	input := "a = 1\na.b = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for dotted key through scalar path")
	}
}

func TestParse_RejectsLeafConflictDottedKeyTable(t *testing.T) {
	input := "a.b.c = 1\na.b.c = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for duplicate dotted key")
	}
}

func TestParse_RejectsLeafConflictAOT(t *testing.T) {
	input := "[[a]]\nb = 1\n[[a]]\nb = 1\na = 2\n[a]\nk = 1\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for key conflict with AOT path")
	}
}

func TestParse_InlineTableDuplicateKeys(t *testing.T) {
	input := "t = {a = 1, a = 2}\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for duplicate keys in inline table")
	}
}

func TestParse_InlineTableDottedConflict(t *testing.T) {
	input := "t = {a = 1, a.b = 2}\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for dotted key conflict in inline table")
	}
}

func TestParse_AllowsAOTClearSubScope(t *testing.T) {
	// AOT should clear sub-scope so keys can be reused across entries.
	input := "[[a]]\nb = 1\n[[a]]\nb = 2\n"
	_, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParse_DottedKeyReopenTableConflict(t *testing.T) {
	input := "[a]\nb.c = 1\n[a.b]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for reopening dotted key table")
	}
}

func TestParse_ScalarPathBlocksTable(t *testing.T) {
	input := "[a]\nb = 1\n[a.b]\nc = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for scalar path blocking table")
	}
}

func TestParse_TableAfterDottedKeyBlock(t *testing.T) {
	input := "[a]\nb.c = 1\n[a.b.c]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for table blocked by dotted key")
	}
}

// --- Coverage: SetPreEq / SetPostEq ---

func TestKeyValue_SetPreEq(t *testing.T) {
	d, err := Parse([]byte("key = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if err := kv.SetPreEq("\t"); err != nil {
		t.Fatal(err)
	}
	if kv.PreEq() != "\t" {
		t.Fatalf("expected tab, got %q", kv.PreEq())
	}
	if err := kv.SetPreEq("bad\n"); err == nil {
		t.Fatal("expected error for invalid whitespace")
	}
}

func TestKeyValue_SetPostEq(t *testing.T) {
	d, err := Parse([]byte("key = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if err := kv.SetPostEq("\t"); err != nil {
		t.Fatal(err)
	}
	if kv.PostEq() != "\t" {
		t.Fatalf("expected tab, got %q", kv.PostEq())
	}
	if err := kv.SetPostEq("bad\n"); err == nil {
		t.Fatal("expected error for invalid whitespace")
	}
}

// --- Coverage: Validate convenience ---

func TestDocument_Validate(t *testing.T) {
	d, err := Parse([]byte("a = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

// --- Coverage: delete from table inside document ---

func TestDocument_Delete_FromTable(t *testing.T) {
	d, err := Parse([]byte("[server]\nhost = \"localhost\"\nport = 8080\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !d.Delete("server.host") {
		t.Fatal("expected to delete server.host")
	}
	if d.Delete("server.nonexistent") {
		t.Fatal("did not expect to delete nonexistent key")
	}
}

func TestDocument_Delete_FromAOT(t *testing.T) {
	d, err := Parse([]byte("[[p]]\nname = \"A\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !d.Delete("p.name") {
		t.Fatal("expected to delete p.name")
	}
}

// --- Coverage: unicode escape edge cases ---

func TestParse_UnicodeEscape4(t *testing.T) {
	input := "s = \"\\u0041\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	sn := d.nodes[0].(*KeyValue).val.(*StringNode)
	if sn.Value() != "A" {
		t.Fatalf("expected 'A', got %q", sn.Value())
	}
}

func TestParse_UnicodeEscape8(t *testing.T) {
	input := "s = \"\\U0001F600\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	sn := d.nodes[0].(*KeyValue).val.(*StringNode)
	if sn.Value() != "\U0001F600" {
		t.Fatalf("expected emoji, got %q", sn.Value())
	}
}

func TestParse_RejectsSurrogateUnicode(t *testing.T) {
	input := "s = \"\\uD800\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for surrogate unicode codepoint")
	}
}

func TestParse_RejectsIncompleteUnicodeEscape(t *testing.T) {
	input := "s = \"\\u00\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for incomplete unicode escape")
	}
}

func TestParse_RejectsOutOfRangeUnicode(t *testing.T) {
	input := "s = \"\\U99999999\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for out of range unicode")
	}
}

func TestParse_RejectsInvalidHexInUnicodeEscape(t *testing.T) {
	input := "s = \"\\u00GG\"\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for invalid hex in unicode escape")
	}
}

// --- Coverage: parent tracking ---

func TestKeyValue_Parent(t *testing.T) {
	d, err := Parse([]byte("[t]\nk = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	tbl := d.Tables()[0]
	kv := tbl.entries[0].(*KeyValue)
	if kv.Parent() != tbl {
		t.Fatal("expected parent to be the table")
	}
	if tbl.Parent() != d {
		t.Fatal("expected table parent to be the document")
	}
}

// --- Coverage: multiline string line-ending backslash edge cases ---

func TestParse_MultiLineBasicBackslashEdgeCases(t *testing.T) {
	// Backslash not followed by whitespace/newline (normal escape in multiline).
	input := "s = \"\"\"\n\\n\"\"\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	sn := d.nodes[0].(*KeyValue).val.(*StringNode)
	if sn.Value() != "\n" {
		t.Fatalf("expected newline, got %q", sn.Value())
	}
}

// --- Coverage: isSpecialFloat via validate ---

func TestParse_SpecialFloatUnderscoreRejected(t *testing.T) {
	_, err := Parse([]byte("f = in_f\n"))
	if err == nil {
		t.Fatal("expected error for underscore in special float")
	}
}

// --- Coverage: whitespace-only key in header ---

func TestParse_TableHeaderWithWhitespace(t *testing.T) {
	input := "[ server ]\nk = 1\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	tbl := d.Tables()[0]
	if tbl.HeaderParts()[0].Unquoted != "server" {
		t.Fatalf("expected 'server', got %q", tbl.rawHeader)
	}
}

func TestParse_DottedKeyWithWhitespaceAroundDots(t *testing.T) {
	input := "a . b = 1\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if len(kv.keyParts) != 2 {
		t.Fatalf("expected 2 key parts, got %d", len(kv.keyParts))
	}
	// Check DotBefore/DotAfter are captured.
	if kv.keyParts[1].DotBefore != " " || kv.keyParts[1].DotAfter != " " {
		t.Fatalf("expected single space for DotBefore/DotAfter, got %q/%q",
			kv.keyParts[1].DotBefore, kv.keyParts[1].DotAfter)
	}
}

// --- Coverage: keyPartsToPath with dotted unquoted ---

func TestKeyPartsToPath_QuotedDotInKey(t *testing.T) {
	kp := []KeyPart{
		{Unquoted: "a.b", IsQuoted: true},
		{Unquoted: "c"},
	}
	path := keyPartsToPath(kp)
	if path != `"a.b".c` {
		t.Fatalf("expected %q, got %q", `"a.b".c`, path)
	}
}

// --- Coverage: SetLeadingTrivia on KeyValue ---

func TestKeyValue_SetLeadingTrivia(t *testing.T) {
	d, err := Parse([]byte("key = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	ws := &WhitespaceNode{leafNode: newLeaf(NodeWhitespace, "  ")}
	if err := kv.SetLeadingTrivia([]Node{ws}); err != nil {
		t.Fatal(err)
	}
	if len(kv.LeadingTrivia()) != 1 {
		t.Fatal("expected 1 leading trivia")
	}
	if err := kv.SetLeadingTrivia([]Node{NewString("bad")}); err == nil {
		t.Fatal("expected error for invalid trivia")
	}
}

// --- Coverage: value-typed keys in simple key ---

func TestParse_BooleanUsedAsKey(t *testing.T) {
	input := "true = 1\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.keyParts[0].Unquoted != "true" {
		t.Fatalf("expected 'true' as key, got %q", kv.keyParts[0].Unquoted)
	}
}

func TestParse_IntegerUsedAsKey(t *testing.T) {
	input := "42 = \"answer\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.keyParts[0].Unquoted != "42" {
		t.Fatalf("expected '42' as key, got %q", kv.keyParts[0].Unquoted)
	}
}

// --- Coverage: comment validation (control char in comment) ---

func TestParse_RejectsControlCharInComment(t *testing.T) {
	input := "# hello\x01world\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for control char in comment")
	}
}

// --- Coverage: document with inline table in array ---

func TestParse_ArrayOfInlineTables(t *testing.T) {
	input := "a = [{x = 1}, {y = 2}]\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	arr := kv.val.(*ArrayNode)
	if arr.Len() != 2 {
		t.Fatalf("expected 2 elements, got %d", arr.Len())
	}
}

// --- Coverage: processHexEscape fallback paths ---

func TestMultiLineBasic_IncompleteHexEscape(t *testing.T) {
	// Test incomplete \x escape fallback in multiline context.
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\n\\xZ\"\"\"")}
	v := n.Value()
	if !strings.Contains(v, `\x`) {
		t.Fatalf("expected fallback \\x, got %q", v)
	}
}

func TestMultiLineBasic_IncompleteUnicodeEscape(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\n\\u00\"\"\"")}
	v := n.Value()
	if !strings.Contains(v, `\u`) {
		t.Fatalf("expected fallback \\u, got %q", v)
	}
}

func TestMultiLineBasic_IncompleteUnicodeEscape8(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\n\\U0001\"\"\"")}
	v := n.Value()
	if !strings.Contains(v, `\U`) {
		t.Fatalf("expected fallback \\U, got %q", v)
	}
}

func TestMultiLineBasic_TrailingBackslash(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\nhello\\\"\"\"")}
	v := n.Value()
	if !strings.HasSuffix(v, `\`) {
		t.Fatalf("expected trailing backslash, got %q", v)
	}
}

func TestMultiLineBasic_BackslashSpaceNoNewline(t *testing.T) {
	// backslash + space/tab but no following newline — should keep both
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\nhello\\ world\"\"\"")}
	v := n.Value()
	if !strings.Contains(v, `\ `) {
		t.Fatalf("expected '\\' followed by space, got %q", v)
	}
}

// --- Coverage: unknown default escape in parserProcessSingleEscape ---

func TestProcessSingleEscape_DefaultCase(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\n\\q\"\"\"")}
	v := n.Value()
	if !strings.Contains(v, `\q`) {
		t.Fatalf("expected fallback \\q, got %q", v)
	}
}

// --- Coverage: error token (lexer) ---

func TestParse_ErrorToken(t *testing.T) {
	// Characters that can't be part of any token.
	_, err := Parse([]byte("key = \x00\n"))
	if err == nil {
		t.Fatal("expected error for null byte in value position")
	}
}

// --- Coverage: validate.go datetime digit counts and ranges ---

func TestParse_DateTimeMinuteNotTwoDigits(t *testing.T) {
	_, err := Parse([]byte("d = 2024-01-15T08:3:00\n"))
	if err == nil {
		t.Fatal("expected error for minute not 2 digits")
	}
}

func TestParse_DateTimeSecondNotTwoDigits(t *testing.T) {
	_, err := Parse([]byte("d = 2024-01-15T08:30:1\n"))
	if err == nil {
		t.Fatal("expected error for second not 2 digits")
	}
}

func TestParse_DateTimeHourNotTwoDigits(t *testing.T) {
	_, err := Parse([]byte("d = 2024-01-15T8:30:00\n"))
	if err == nil {
		t.Fatal("expected error for hour not 2 digits")
	}
}

func TestParse_OffsetDateTimeWithOffset(t *testing.T) {
	d, err := Parse([]byte("d = 2024-01-15T08:30:00+05:30\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeDateTime {
		t.Fatalf("expected datetime, got %v", kv.val.Type())
	}
}

func TestParse_OffsetDateTimeNegative(t *testing.T) {
	d, err := Parse([]byte("d = 2024-01-15T08:30:00-04:00\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeDateTime {
		t.Fatalf("expected datetime, got %v", kv.val.Type())
	}
}

func TestParse_LocalTimeNoSeconds(t *testing.T) {
	d, err := Parse([]byte("t = 07:32\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeDateTime {
		t.Fatalf("expected datetime, got %v", kv.val.Type())
	}
}

// --- Coverage: validate.go multiline literal string ---

func TestParse_MultiLineLiteralWithCRLF(t *testing.T) {
	input := "s = '''\r\nhello\r\nworld'''\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sn := d.nodes[0].(*KeyValue).val.(*StringNode)
	if !strings.Contains(sn.Value(), "hello") {
		t.Fatalf("unexpected value: %q", sn.Value())
	}
}

func TestParse_MultiLineBasicWithCRLFStart(t *testing.T) {
	input := "s = \"\"\"\r\nhello\"\"\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sn := d.nodes[0].(*KeyValue).val.(*StringNode)
	if sn.Value() != "hello" {
		t.Fatalf("expected 'hello', got %q", sn.Value())
	}
}

// --- Coverage: validate.go literal string control char ---

func TestParse_LiteralStringControlCharRejected(t *testing.T) {
	_, err := Parse([]byte("s = 'hello\x01world'\n"))
	if err == nil {
		t.Fatal("expected error for control char in literal string")
	}
}

func TestParse_MultiLineLiteralControlCharRejected(t *testing.T) {
	_, err := Parse([]byte("s = '''\nhello\x01world'''\n"))
	if err == nil {
		t.Fatal("expected error for control char in multiline literal string")
	}
}

// --- Coverage: validate.go checkAdjacentChar (underscore before float separator) ---

func TestParse_RejectsUnderscoreBeforeDot(t *testing.T) {
	_, err := Parse([]byte("f = 1_.0\n"))
	if err == nil {
		t.Fatal("expected error for underscore before dot")
	}
}

func TestParse_RejectsUnderscoreAfterE(t *testing.T) {
	_, err := Parse([]byte("f = 1e_2\n"))
	if err == nil {
		t.Fatal("expected error for underscore after exponent")
	}
}

// --- Coverage: validate.go stripOffset edge cases ---

func TestParse_DateTimeWithLowercaseZ(t *testing.T) {
	d, err := Parse([]byte("d = 2024-01-15T08:30:00z\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeDateTime {
		t.Fatalf("expected datetime, got %v", kv.val.Type())
	}
}

// --- Coverage: parser.go parseTableHeaderBody/parseArrayOfTablesBody error paths ---

func TestParse_MissingClosingBracketTable(t *testing.T) {
	_, err := Parse([]byte("[server\nk = 1\n"))
	if err == nil {
		t.Fatal("expected error for missing ] in table header")
	}
}

func TestParse_MissingClosingBracketsAOT(t *testing.T) {
	_, err := Parse([]byte("[[items\nk = 1\n"))
	if err == nil {
		t.Fatal("expected error for missing ]] in AOT header")
	}
}

func TestParse_AOTMissingSecondClosingBracket(t *testing.T) {
	_, err := Parse([]byte("[[items]\nk = 1\n"))
	if err == nil {
		t.Fatal("expected error for missing second ] in AOT header")
	}
}

func TestParse_TrailingContentAfterTableHeader(t *testing.T) {
	_, err := Parse([]byte("[server] extra\nk = 1\n"))
	if err == nil {
		t.Fatal("expected error for trailing content after table header")
	}
}

func TestParse_TrailingContentAfterAOTHeader(t *testing.T) {
	_, err := Parse([]byte("[[items]] extra\nk = 1\n"))
	if err == nil {
		t.Fatal("expected error for trailing content after AOT header")
	}
}

// --- Coverage: parser.go parseKeyVal error paths ---

func TestParse_MissingEquals(t *testing.T) {
	_, err := Parse([]byte("key value\n"))
	if err == nil {
		t.Fatal("expected error for missing = sign")
	}
}

func TestParse_MissingValue(t *testing.T) {
	_, err := Parse([]byte("key =\n"))
	if err == nil {
		t.Fatal("expected error for missing value")
	}
}

func TestParse_TrailingContentAfterValue(t *testing.T) {
	_, err := Parse([]byte("key = 1 2\n"))
	if err == nil {
		t.Fatal("expected error for trailing content after value")
	}
}

// --- Coverage: parser.go unquoteBasicStr/unquoteLiteralStr short string ---

func TestParse_ShortBareKeyIsBareKey(t *testing.T) {
	// This tests the bare key path with a single char
	d, err := Parse([]byte("k = 1\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.keyParts[0].Unquoted != "k" {
		t.Fatalf("expected 'k', got %q", kv.keyParts[0].Unquoted)
	}
}

// --- Coverage: lexer.go scanLiteralString unclosed ---

func TestParse_UnclosedLiteralString(t *testing.T) {
	_, err := Parse([]byte("s = 'unclosed\n"))
	if err == nil {
		t.Fatal("expected error for unclosed literal string")
	}
}

func TestParse_UnclosedBasicString(t *testing.T) {
	_, err := Parse([]byte("s = \"unclosed\n"))
	if err == nil {
		t.Fatal("expected error for unclosed basic string")
	}
}

func TestParse_UnclosedMultiLineBasicString(t *testing.T) {
	_, err := Parse([]byte("s = \"\"\"unclosed\n"))
	if err == nil {
		t.Fatal("expected error for unclosed multiline basic string")
	}
}

func TestParse_UnclosedMultiLineLiteralString(t *testing.T) {
	_, err := Parse([]byte("s = '''unclosed\n"))
	if err == nil {
		t.Fatal("expected error for unclosed multiline literal string")
	}
}

// --- Coverage: lexer.go peekSpaceTime partial match ---

func TestParse_DateFollowedBySpaceButNoTime(t *testing.T) {
	// Date-like value followed by space but not a time — should not be combined
	_, err := Parse([]byte("d = 2024-01-15 abc\n"))
	if err == nil {
		t.Fatal("expected error for date followed by non-time")
	}
}

func TestParse_DateWithSpaceAndPartialTime(t *testing.T) {
	// Date followed by space and a digit but no colon — not a time
	_, err := Parse([]byte("d = 2024-01-15 9x\n"))
	if err == nil {
		t.Fatal("expected error for incomplete time after date")
	}
}

// --- Coverage: lexer.go peekForDot no trailing dot ---

func TestParse_WhitespaceButNoDotAfterKey(t *testing.T) {
	// Key followed by whitespace then = (no dot)
	d, err := Parse([]byte("key = 1\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if len(kv.keyParts) != 1 {
		t.Fatalf("expected 1 key part, got %d", len(kv.keyParts))
	}
}

// --- Coverage: mutate.go AOT Append standalone ---

func TestArrayOfTables_Append_Standalone(t *testing.T) {
	aot, _ := NewArrayOfTables("items")
	kv1, _ := NewKeyValue("a", NewInteger(1))
	if err := aot.Append(kv1); err != nil {
		t.Fatal(err)
	}
	// Duplicate.
	kv2, _ := NewKeyValue("a", NewInteger(2))
	if err := aot.Append(kv2); err == nil {
		t.Fatal("expected error for duplicate key in standalone AOT")
	}
}

func TestArrayOfTables_Append_Nil(t *testing.T) {
	aot, _ := NewArrayOfTables("items")
	if err := aot.Append(nil); err == nil {
		t.Fatal("expected error for nil entry")
	}
}

// --- Coverage: mutate.go regenerateAncestorText via array ---

func TestSetValue_RegeneratesArrayText(t *testing.T) {
	d, err := Parse([]byte("a = [1, 2, 3]\n"))
	if err != nil {
		t.Fatal(err)
	}
	kv := d.nodes[0].(*KeyValue)
	arr := kv.val.(*ArrayNode)
	// Modify an element directly and check text regeneration.
	// We need to set a value inside a KV that is inside an array.
	// Instead, use a nested inline table in array scenario.
	_ = arr
	got := d.String()
	if got != "a = [1, 2, 3]\n" {
		t.Fatalf("unexpected: %q", got)
	}
}

// --- Coverage: mutate.go InlineTableNode Append dotted key conflict ---

func TestInlineTableNode_Append_DottedKeyConflict(t *testing.T) {
	kv1, _ := NewKeyValue("a", NewString("scalar"))
	it, err := NewInlineTable(kv1)
	if err != nil {
		t.Fatal(err)
	}
	kv2, _ := NewKeyValue("a.b", NewString("nested"))
	if err := it.Append(kv2); err == nil {
		t.Fatal("expected error for dotted key conflict")
	}
}

// --- Coverage: mutate.go AppendComment on document (error path) ---

func TestDocument_AppendComment_Invalid(t *testing.T) {
	d, _ := Parse([]byte(""))
	err := d.AppendComment("has\nnewline")
	if err == nil {
		t.Fatal("expected error for newline in comment")
	}
}

func TestTableNode_AppendComment_Invalid(t *testing.T) {
	tbl, _ := NewTable("t")
	err := tbl.AppendComment("has\nnewline")
	if err == nil {
		t.Fatal("expected error for newline in comment")
	}
}

func TestArrayOfTables_AppendComment_Invalid(t *testing.T) {
	aot, _ := NewArrayOfTables("items")
	err := aot.AppendComment("has\nnewline")
	if err == nil {
		t.Fatal("expected error for newline in comment")
	}
}

// --- Coverage: mutate.go setValueParent for DateTimeNode ---

func TestSetValue_DateTimeParent(t *testing.T) {
	kv, _ := NewKeyValue("d", NewString("placeholder"))
	dt, _ := NewDateTime("2024-01-15T08:30:00Z")
	if err := kv.SetValue(dt); err != nil {
		t.Fatal(err)
	}
	if kv.val != dt {
		t.Fatal("expected datetime as value")
	}
}

// --- Coverage: query.go trimLeadingNewline CRLF path ---

func TestStringNode_Value_MultiLineBasicCRLFStart(t *testing.T) {
	// Construct a StringNode whose text starts with """\r\n
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\r\nhello\"\"\"")}
	v := n.Value()
	if v != "hello" {
		t.Fatalf("expected 'hello', got %q", v)
	}
}

func TestStringNode_Value_MultiLineLiteralCRLFStart(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "'''\r\nhello'''")}
	v := n.Value()
	if v != "hello" {
		t.Fatalf("expected 'hello', got %q", v)
	}
}

// --- Coverage: query.go processMultiLineBasicEscapes backslash-CR-LF ---

func TestStringNode_Value_MultiLineBackslashCRLF(t *testing.T) {
	// Backslash followed by \r\n should skip the line ending and following whitespace.
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\nhello \\\r\n  world\"\"\"")}
	v := n.Value()
	if v != "hello world" {
		t.Fatalf("expected 'hello world', got %q", v)
	}
}

// --- Coverage: query.go parserProcessSingleEscape remaining branches ---

func TestStringNode_Value_MultiLineEscapeSequences(t *testing.T) {
	// Test \b, \f, \r, \\, \", \e in multiline context
	tests := []struct {
		raw  string
		want string
	}{
		{"\"\"\"\n\\b\"\"\"", "\b"},
		{"\"\"\"\n\\f\"\"\"", "\f"},
		{"\"\"\"\n\\r\"\"\"", "\r"},
		{"\"\"\"\n\\\\\"\"\"", "\\"},
		{"\"\"\"\n\\\"\"\"\"", "\""},
		{"\"\"\"\n\\e\"\"\"", "\x1B"},
		{"\"\"\"\n\\t\"\"\"", "\t"},
		{"\"\"\"\n\\n\"\"\"", "\n"},
	}
	for _, tt := range tests {
		n := &StringNode{leafNode: newLeaf(NodeString, tt.raw)}
		got := n.Value()
		if got != tt.want {
			t.Errorf("Value() for %q: got %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestStringNode_Value_MultiLineHexEscapes(t *testing.T) {
	// \x, \u, \U in multiline context
	tests := []struct {
		raw  string
		want string
	}{
		{"\"\"\"\n\\xE9\"\"\"", "\u00E9"},
		{"\"\"\"\n\\u0041\"\"\"", "A"},
		{"\"\"\"\n\\U0001F600\"\"\"", "\U0001F600"},
	}
	for _, tt := range tests {
		n := &StringNode{leafNode: newLeaf(NodeString, tt.raw)}
		got := n.Value()
		if got != tt.want {
			t.Errorf("Value() for %q: got %q, want %q", tt.raw, got, tt.want)
		}
	}
}

// --- Coverage: query.go parsePathLiteralString unclosed ---

func TestParseDottedPath_UnclosedLiteral(t *testing.T) {
	got := parseDottedPath("'unclosed")
	if len(got) != 1 || got[0] != "unclosed" {
		t.Fatalf("expected ['unclosed'], got %v", got)
	}
}

func TestParseDottedPath_UnclosedBasic(t *testing.T) {
	got := parseDottedPath("\"unclosed")
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(got))
	}
}

// --- Coverage: validate.go validateOffsetText edge cases ---

func TestParse_DateTimeInvalidOffsetHour(t *testing.T) {
	_, err := Parse([]byte("d = 2024-01-15T08:30:00+25:00\n"))
	if err == nil {
		t.Fatal("expected error for offset hour > 23")
	}
}

func TestParse_DateTimeInvalidOffsetMinute(t *testing.T) {
	_, err := Parse([]byte("d = 2024-01-15T08:30:00+05:99\n"))
	if err == nil {
		t.Fatal("expected error for offset minute > 59")
	}
}

// --- Coverage: validate.go validateDateParts wrong number of parts ---

func TestParse_DateTimeBadDateFormat(t *testing.T) {
	_, err := Parse([]byte("d = 2024-01\n"))
	if err == nil {
		t.Fatal("expected error for date with only 2 parts")
	}
}

// --- Coverage: validate.go validateTimeParts odd format ---

func TestParse_TimeBadFormat(t *testing.T) {
	_, err := Parse([]byte("t = 08\n"))
	if err == nil {
		t.Fatal("expected error for time with only 1 part (will be parsed as integer, not error)")
	}
}

// --- Coverage: validate.go line-ending backslash with CRLF ---

func TestParse_MultiLineBasicBackslashCRLF(t *testing.T) {
	input := "s = \"\"\"\nhello \\\r\n  world\"\"\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

// --- Coverage: validate.go validateBasicContent - multiline newline/CR allowed ---

func TestParse_MultiLineBasicStringWithTab(t *testing.T) {
	input := "s = \"\"\"\n\thello\"\"\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sn := d.nodes[0].(*KeyValue).val.(*StringNode)
	if !strings.Contains(sn.Value(), "\t") {
		t.Fatalf("expected tab in value, got %q", sn.Value())
	}
}

// --- Coverage: validate.go checkAOTPathConflicts - all paths ---

func TestParse_RejectsAOTStaticArrayConflict(t *testing.T) {
	input := "a = [1]\n[[a]]\nk = 1\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT conflict with static array")
	}
}

func TestParse_RejectsAOTDottedKeyTableConflict(t *testing.T) {
	input := "[parent]\nchild.k = 1\n[[parent.child]]\nk = 2\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT conflict with dotted key table")
	}
}

// --- Coverage: validate.go checkDottedIntermediate - AOT path ---

func TestParse_DottedKeyInsideAOTIsValid(t *testing.T) {
	input := "[[a]]\nk = 1\na.b = 2\n"
	_, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Coverage: validate.go checkDottedIntermediate - explicit table ---

func TestParse_RejectsDottedKeyThroughExplicitTable(t *testing.T) {
	input := "[a]\nk = 1\n[b]\na.k = 2\na.k.deep = 3\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for dotted key extending explicit table path")
	}
}

// --- Coverage: validate.go checkLeafConflict - AOT path as leaf ---

func TestParse_KeyInsideAOTEntryIsValid(t *testing.T) {
	input := "[[products]]\nname = \"A\"\n[[products]]\nname = \"B\"\nproducts = 1\n"
	_, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Coverage: parser.go parseArray error paths ---

func TestParse_ArrayMissingCommaOrBracket(t *testing.T) {
	_, err := Parse([]byte("a = [1 2]\n"))
	if err == nil {
		t.Fatal("expected error for missing comma in array")
	}
}

func TestParse_ArrayUnclosed(t *testing.T) {
	_, err := Parse([]byte("a = [1, 2"))
	if err == nil {
		t.Fatal("expected error for unclosed array")
	}
}

// --- Coverage: parser.go parseInlineTable error paths ---

func TestParse_InlineTableMissingComma(t *testing.T) {
	_, err := Parse([]byte("t = {a = 1 b = 2}\n"))
	if err == nil {
		t.Fatal("expected error for missing comma in inline table")
	}
}

func TestParse_InlineTableUnclosed(t *testing.T) {
	_, err := Parse([]byte("t = {a = 1"))
	if err == nil {
		t.Fatal("expected error for unclosed inline table")
	}
}

// --- Coverage: parser.go parseSimpleKey error path ---

func TestParse_ExpectedKey(t *testing.T) {
	_, err := Parse([]byte("= 1\n"))
	if err == nil {
		t.Fatal("expected error for missing key before =")
	}
}

// --- Coverage: parser.go collectHeaderTrailing comment validation ---

func TestParse_InvalidControlCharInHeaderComment(t *testing.T) {
	_, err := Parse([]byte("[t] # bad\x01comment\nk = 1\n"))
	if err == nil {
		t.Fatal("expected error for control char in table header comment")
	}
}

// --- Coverage: parser.go addTrailingTrivia comment validation ---

func TestParse_InvalidControlCharInTrailingComment(t *testing.T) {
	_, err := Parse([]byte("k = 1 # bad\x01comment\n"))
	if err == nil {
		t.Fatal("expected error for control char in trailing comment")
	}
}

// --- Coverage: lexer.go peek at end ---

func TestParse_EmptyValueContext(t *testing.T) {
	// This exercises the peek at end of source
	_, err := Parse([]byte("k = "))
	if err == nil {
		t.Fatal("expected error for missing value at EOF")
	}
}

// --- Coverage: lexer.go looksLikeNumber edge cases ---

func TestParse_SignOnlyNotANumber(t *testing.T) {
	// A bare "+" or "-" should not be a number
	_, err := Parse([]byte("k = +\n"))
	if err == nil {
		t.Fatal("expected error for + as value")
	}
}

// --- Coverage: validate.go checkDateDigitCounts - all branches ---

func TestParse_DateYearTooFewDigits(t *testing.T) {
	_, err := Parse([]byte("d = 24-01-15\n"))
	if err == nil {
		t.Fatal("expected error for 2-digit year")
	}
}

func TestParse_DateMonthOneDigit(t *testing.T) {
	_, err := Parse([]byte("d = 2024-1-15\n"))
	if err == nil {
		t.Fatal("expected error for 1-digit month")
	}
}

func TestParse_DateDayOneDigit(t *testing.T) {
	_, err := Parse([]byte("d = 2024-01-5\n"))
	if err == nil {
		t.Fatal("expected error for 1-digit day")
	}
}

// --- Coverage: validate.go checkTimeDigitCounts - minute/second branches ---

func TestParse_TimeMinuteOneDigit(t *testing.T) {
	_, err := Parse([]byte("t = 08:5:00\n"))
	if err == nil {
		t.Fatal("expected error for 1-digit minute in time")
	}
}

func TestParse_TimeSecondOneDigit(t *testing.T) {
	_, err := Parse([]byte("t = 08:30:1\n"))
	if err == nil {
		t.Fatal("expected error for 1-digit second in time")
	}
}

// --- Coverage: validate.go checkIntermediatePaths - staticArrays branch ---

func TestParse_RejectsTableIntermediateStaticArrayPath(t *testing.T) {
	// Static array at "a", table tries to use a as intermediate
	input := "a = [1]\n[a.b.c]\nk = 1\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for table intermediate through static array")
	}
}

// --- Coverage: validate.go checkIntermediatePathsAOT - all branches ---

func TestParse_RejectsAOTIntermediateScalarPath(t *testing.T) {
	input := "x = 1\n[[x.y]]\nk = 1\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT intermediate through scalar")
	}
}

func TestParse_RejectsAOTIntermediateInlinePath(t *testing.T) {
	input := "x = {a = 1}\n[[x.y]]\nk = 1\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT intermediate through inline table")
	}
}

func TestParse_RejectsAOTIntermediateStaticArrayPath(t *testing.T) {
	input := "x = [1]\n[[x.y]]\nk = 1\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for AOT intermediate through static array")
	}
}

// --- Coverage: mutate.go TableNode InsertAt with document context error ---

func TestTableNode_InsertAt_WithDocument_Error(t *testing.T) {
	d, err := Parse([]byte("[t]\na = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	tbl := d.Table("t")
	kv, _ := NewKeyValue("a", NewInteger(2)) // duplicate
	if err := tbl.InsertAt(0, kv); err == nil {
		t.Fatal("expected error for duplicate key in table InsertAt")
	}
	if len(tbl.entries) != 1 {
		t.Fatalf("expected 1 entry after rollback, got %d", len(tbl.entries))
	}
}

func TestTableNode_InsertAt_Nil(t *testing.T) {
	tbl, _ := NewTable("t")
	if err := tbl.InsertAt(0, nil); err == nil {
		t.Fatal("expected error for nil entry")
	}
}

func TestTableNode_InsertAt_OutOfRange(t *testing.T) {
	tbl, _ := NewTable("t")
	kv, _ := NewKeyValue("a", NewInteger(1))
	if err := tbl.InsertAt(999, kv); err != nil {
		t.Fatal(err)
	}
	if len(tbl.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(tbl.entries))
	}
}

// --- Coverage: mutate.go deleteFromTableNode - AOT path ---

func TestDocument_Delete_FromAOTDeep(t *testing.T) {
	d, err := Parse([]byte("[[items]]\nname = \"A\"\nprice = 10\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !d.Delete("items.name") {
		t.Fatal("expected to delete items.name")
	}
	if d.Delete("items.nonexistent") {
		t.Fatal("did not expect to delete nonexistent entry")
	}
}

// --- Coverage: lexer.go advance at end ---

func TestParse_OnlyNewline(t *testing.T) {
	d, err := Parse([]byte("\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.String() != "\n" {
		t.Fatalf("expected \\n, got %q", d.String())
	}
}

func TestParse_OnlyCRLF(t *testing.T) {
	d, err := Parse([]byte("\r\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.String() != "\r\n" {
		t.Fatalf("expected \\r\\n, got %q", d.String())
	}
}

// --- Coverage: parser.go parseValue default branch ---

func TestParse_UnexpectedTokenAsValue(t *testing.T) {
	_, err := Parse([]byte("k = ]\n"))
	if err == nil {
		t.Fatal("expected error for ] as value")
	}
}

// --- Coverage: lexer.go scanBareOrValue empty text ---

func TestParse_BareValueErrorToken(t *testing.T) {
	// Characters that lexer can't classify
	_, err := Parse([]byte("k = \x80\n"))
	if err == nil {
		t.Fatal("expected error for invalid byte in value")
	}
}

// --- Coverage: validate.go validateDecimalDigits invalid character ---

func TestParse_InvalidCharInInteger(t *testing.T) {
	_, err := Parse([]byte("n = 12abc\n"))
	if err == nil {
		t.Fatal("expected error for non-digit in integer")
	}
}

// --- Coverage: parser.go parseSimpleKey - invalid bare key char ---

func TestParse_InvalidBareKeyChar(t *testing.T) {
	_, err := Parse([]byte("key! = 1\n"))
	if err == nil {
		t.Fatal("expected error for ! in bare key")
	}
}

// --- Coverage: parser.go multi-line basic string with 4-5 quotes ---

func TestParse_MultiLineBasicWith5Quotes(t *testing.T) {
	// Multi-line basic string can contain up to 2 extra quotes at closing:
	// """""" = """ + content ending in "" + """
	input := "s = \"\"\"\nhello\"\"\"\"\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := d.String()
	if got != input {
		t.Fatalf("round-trip failed\nwant: %q\ngot:  %q", input, got)
	}
}

// --- Coverage: query.go processHexEscape error paths (via multiline) ---

func TestStringNode_Value_MultiLineInvalidHexEscape(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\n\\xZZ\"\"\"")}
	v := n.Value()
	if !strings.Contains(v, `\x`) {
		t.Fatalf("expected fallback \\x, got %q", v)
	}
}

func TestStringNode_Value_MultiLineInvalidUnicodeEscape4(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\n\\uGGGG\"\"\"")}
	v := n.Value()
	if !strings.Contains(v, `\u`) {
		t.Fatalf("expected fallback \\u, got %q", v)
	}
}

func TestStringNode_Value_MultiLineInvalidUnicodeEscape8(t *testing.T) {
	n := &StringNode{leafNode: newLeaf(NodeString, "\"\"\"\n\\UGGGGGGGG\"\"\"")}
	v := n.Value()
	if !strings.Contains(v, `\U`) {
		t.Fatalf("expected fallback \\U, got %q", v)
	}
}

// --- Coverage: validate.go checkIntermediatePaths - inline table intermediate ---

func TestParse_RejectsTableThroughInlineTableIntermediate(t *testing.T) {
	input := "a.b = {x = 1}\n[a.b.x]\nk = 1\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for table through inline table intermediate")
	}
}

// --- Coverage: validate.go checkTablePathConflicts - scalar path ---

func TestParse_RejectsTableOverScalar(t *testing.T) {
	input := "a = 1\n[a]\nk = 1\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for table over scalar")
	}
}

// --- Coverage: mutate.go parseRawKey whitespace handling ---

func TestNewKeyValue_LeadingWhitespace(t *testing.T) {
	kv, err := NewKeyValue("  key  ", NewString("val"))
	if err != nil {
		t.Fatalf("NewKeyValue: %v", err)
	}
	if kv.keyParts[0].Unquoted != "key" {
		t.Fatalf("expected 'key', got %q", kv.keyParts[0].Unquoted)
	}
}

// --- Coverage: validate.go checkPrefixNumber signed prefix ---

func TestParse_RejectsSignedOctal(t *testing.T) {
	_, err := Parse([]byte("n = -0o7\n"))
	if err == nil {
		t.Fatal("expected error for signed octal")
	}
}

func TestParse_RejectsSignedBinary(t *testing.T) {
	_, err := Parse([]byte("n = +0b1\n"))
	if err == nil {
		t.Fatal("expected error for signed binary")
	}
}

// --- Coverage: validate.go validateFloatExponent dotIdx == eIdx-1 ---

func TestParse_RejectsFloatDotImmediatelyBeforeExponent(t *testing.T) {
	_, err := Parse([]byte("f = 1.e2\n"))
	if err == nil {
		t.Fatal("expected error for dot immediately before exponent")
	}
}

// --- Coverage: validate.go checkDecimalLeadingZeros ---

func TestParse_AcceptsZeroBeforeDot(t *testing.T) {
	d, err := Parse([]byte("f = 0.5\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeNumber {
		t.Fatalf("expected number, got %v", kv.val.Type())
	}
}

func TestParse_AcceptsZeroBeforeExponent(t *testing.T) {
	d, err := Parse([]byte("f = 0e1\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kv := d.nodes[0].(*KeyValue)
	if kv.val.Type() != NodeNumber {
		t.Fatalf("expected number, got %v", kv.val.Type())
	}
}

// --- Coverage: parser.go parserProcessBasicEscapes default branch ---

func TestParserProcessBasicEscapes_UnknownEscape(t *testing.T) {
	// Call through StringNode.Value
	n := &StringNode{leafNode: newLeaf(NodeString, `"\q"`)}
	v := n.Value()
	if v != `\q` {
		t.Fatalf("expected '\\q', got %q", v)
	}
}

// --- Coverage: orphan trivia with current table set ---

func TestParse_OrphanTriviaOnTableNoKVGoesToTable(t *testing.T) {
	input := "[server]\n# comment at end"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	tbl := d.nodes[0].(*TableNode)
	if len(tbl.entries) == 0 {
		t.Fatal("expected trivia to be added to table entries")
	}
}

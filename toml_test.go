package toml

import (
	"errors"
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

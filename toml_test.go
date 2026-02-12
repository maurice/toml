package toml

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty document",
			input:   []byte(""),
			wantErr: false,
		},
		{
			name:    "simple key-value",
			input:   []byte(`key = "value"`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Fatalf("Parse() returned nil document")
			}
			// basic round-trip equivalence (ignore trailing newline normalization)
			if strings.TrimSpace(got.String()) != strings.TrimSpace(string(tt.input)) {
				t.Fatalf("round-trip mismatch\nwant: %q\n got: %q", string(tt.input), got.String())
			}
		})
	}
}

func TestParse_PreservesTrailingComment(t *testing.T) {
	input := "key = \"value\"  # tail comment"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	kvs := d.KeyValues()
	if len(kvs) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(kvs))
	}
	kv := kvs[0]
	// trailing raw string preserved
	if !strings.HasPrefix(strings.TrimSpace(kv.Trailing), "# tail comment") {
		t.Fatalf("trailing comment not preserved: %q", kv.Trailing)
	}
	// trailing trivia nodes expose a CommentNode followed by no newline
	found := false
	for _, n := range kv.TrailingNodes {
		if n.Type() == NodeComment && strings.HasPrefix(n.Text(), "# tail comment") {
			found = true
		}
	}
	if !found {
		t.Fatalf("trailing CommentNode not found in TrailingNodes")
	}
}

func TestParse_PreservesLeadingTrivia_Raw(t *testing.T) {
	input := "\n# comment before\n\nkey = \"v\""
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	kvs := d.KeyValues()
	if len(kvs) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(kvs))
	}
	kv := kvs[0]
	// raw leading preserved
	if !strings.Contains(kv.Leading, "# comment before") {
		t.Fatalf("leading comment not preserved: %q", kv.Leading)
	}
}

func TestParse_PreservesLeadingTrivia_Nodes(t *testing.T) {
	input := "\n# comment before\n\nkey = \"v\""
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	kvs := d.KeyValues()
	if len(kvs) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(kvs))
	}
	kv := kvs[0]
	// parsed leading nodes include a CommentNode and WhitespaceNodes
	hasComment := false
	hasBlank := false
	for _, n := range kv.LeadingNodes {
		if n.Type() == NodeComment {
			hasComment = true
		}
		if n.Type() == NodeWhitespace && (strings.Contains(n.Text(), "\n\n") || strings.Contains(n.Text(), "\n")) {
			hasBlank = true
		}
	}
	if !hasComment || !hasBlank {
		t.Fatalf("expected leading comment and whitespace nodes; got: %+v", kv.LeadingNodes)
	}
}

func TestRoundTrip_PreservesFormatting(t *testing.T) {
	input := "# top comment\nkey = \"v\"  # inline\n\nother = 1"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	out := d.String()
	if strings.TrimSpace(out) != strings.TrimSpace(input) {
		t.Fatalf("round-trip formatting changed\nwant:\n%q\nget:\n%q", input, out)
	}
}

func TestCST_WalkAndFindTrivia(t *testing.T) {
	input := "# top\nkey = 1  # tail\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var comments []string
	var whites []string
	d.Walk(func(n Node) bool {
		if n.Type() == NodeComment {
			comments = append(comments, n.Text())
		}
		if n.Type() == NodeWhitespace {
			whites = append(whites, n.Text())
		}
		return true
	})

	if len(comments) < 2 { // top comment + inline tail
		t.Fatalf("expected at least 2 comments, got %d (%v)", len(comments), comments)
	}
	if len(whites) < 2 {
		t.Fatalf("expected whitespace nodes, got %d (%v)", len(whites), whites)
	}
}

func tokenKindHelper(t *testing.T, input string, want NodeType) {
	t.Helper()
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	kvs := d.KeyValues()
	if len(kvs) != 1 {
		t.Fatalf("expected 1 kv, got %d", len(kvs))
	}
	kv := kvs[0]
	if kv.KeyTok == nil || kv.KeyTok.Type() != NodeIdentifier {
		t.Fatalf("expected key token to be Identifier, got %v", kv.KeyTok)
	}
	if kv.ValueTok == nil {
		t.Fatalf("expected value token, got nil")
	}
	if kv.ValueTok.Type() != want {
		t.Fatalf("value kind mismatch for %q: want %v got %v", input, want, kv.ValueTok.Type())
	}
}

func TestCST_TokenKinds_Values(t *testing.T) {
	tokenKindHelper(t, `s = "hello"`, NodeString)
	tokenKindHelper(t, `n = 42`, NodeNumber)
	tokenKindHelper(t, `b = true`, NodeBoolean)
	tokenKindHelper(t, `id = bareword`, NodeIdentifier)
}

func TestCST_TokenKinds_TableHeader(t *testing.T) {
	input := "[section]"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var found bool
	for _, n := range d.Nodes {
		if tnode, ok := n.(*TableNode); ok {
			if tnode.HeaderTok == nil || tnode.HeaderTok.Type() != NodeIdentifier {
				t.Fatalf("expected table header token to be Identifier, got %v", tnode.HeaderTok)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("table node not found")
	}
}

func TestParse_TableHeaderPreserved(t *testing.T) {
	input := "# before\n[server.settings]  # header comment\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// find first TableNode
	var found *TableNode
	for _, n := range d.Nodes {
		if tnode, ok := n.(*TableNode); ok {
			found = tnode
			break
		}
	}
	if found == nil {
		t.Fatalf("expected TableNode, none found; nodes=%v", d.Nodes)
	}
	if found.Header != "server.settings" {
		t.Fatalf("unexpected header: %q", found.Header)
	}
	if !strings.Contains(found.Leading, "# before") {
		t.Fatalf("leading trivia not preserved: %q", found.Leading)
	}
	if !strings.Contains(found.Trailing, "header comment") {
		t.Fatalf("trailing trivia not preserved: %q", found.Trailing)
	}
	// round-trip
	if strings.TrimSpace(d.String()) != strings.TrimSpace(input) {
		t.Fatalf("round-trip changed\nwant:%q\nget:%q", input, d.String())
	}
}

func TestParse_ArrayOfTablesHeaderPreserved(t *testing.T) {
	input := "[[products]]  # arr\nname = \"prod\"\n"
	d, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	var found *ArrayOfTables
	for _, n := range d.Nodes {
		if an, ok := n.(*ArrayOfTables); ok {
			found = an
			break
		}
	}
	if found == nil {
		t.Fatalf("expected ArrayOfTables, none found; nodes=%v", d.Nodes)
	}
	if found.Header != "products" {
		t.Fatalf("unexpected array header: %q", found.Header)
	}
	if !strings.Contains(found.Trailing, "arr") {
		t.Fatalf("trailing trivia not preserved: %q", found.Trailing)
	}
}

/* next: design and add a Node interface, table nodes, arrays, and
   higher-level query APIs (Get/Set/Delete) built on top of this CST */

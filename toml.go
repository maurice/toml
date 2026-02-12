package toml

import (
	"bytes"
	"errors"
	"strings"
)

// sentinel errors.
var (
	ErrNilInput = errors.New("nil input")
)

// NodeType identifies node kinds in the CST.
type NodeType int

const (
	NodeDocument NodeType = iota
	NodeKeyValue
	NodeTable
	NodeArrayOfTables
	NodeInlineTable
	NodeIdentifier
	NodeString
	NodeNumber
	NodeBoolean
	NodeDateTime
	NodePunctuation
	NodeTrivia
	NodeComment
	NodeWhitespace
)

// Node is the public CST node interface. It deliberately does NOT expose
// mutation helpers — mutations should be provided by separate APIs on
// Document (Insert/Remove/Replace) so callers cannot accidentally
// corrupt trivia state by changing node text directly.
type Node interface {
	Type() NodeType
	Parent() Node
	Children() []Node
	Text() string
}

// Concrete leaf token node types (Identifier, String, Number, Boolean,
// DateTime, Punctuation). These are clearer for clients than a single
// generic token type and make it explicit what kind of token each leaf is.

type IdentifierNode struct {
	parent Node
	text   string
}

func (n *IdentifierNode) Type() NodeType   { return NodeIdentifier }
func (n *IdentifierNode) Parent() Node     { return n.parent }
func (n *IdentifierNode) Children() []Node { return nil }
func (n *IdentifierNode) Text() string     { return n.text }

type StringNode struct {
	parent Node
	text   string
}

func (n *StringNode) Type() NodeType   { return NodeString }
func (n *StringNode) Parent() Node     { return n.parent }
func (n *StringNode) Children() []Node { return nil }
func (n *StringNode) Text() string     { return n.text }

type NumberNode struct {
	parent Node
	text   string
}

func (n *NumberNode) Type() NodeType   { return NodeNumber }
func (n *NumberNode) Parent() Node     { return n.parent }
func (n *NumberNode) Children() []Node { return nil }
func (n *NumberNode) Text() string     { return n.text }

type BooleanNode struct {
	parent Node
	text   string
}

func (n *BooleanNode) Type() NodeType   { return NodeBoolean }
func (n *BooleanNode) Parent() Node     { return n.parent }
func (n *BooleanNode) Children() []Node { return nil }
func (n *BooleanNode) Text() string     { return n.text }

type DateTimeNode struct {
	parent Node
	text   string
}

func (n *DateTimeNode) Type() NodeType   { return NodeDateTime }
func (n *DateTimeNode) Parent() Node     { return n.parent }
func (n *DateTimeNode) Children() []Node { return nil }
func (n *DateTimeNode) Text() string     { return n.text }

type PunctNode struct {
	parent Node
	text   string
}

func (n *PunctNode) Type() NodeType   { return NodePunctuation }
func (n *PunctNode) Parent() Node     { return n.parent }
func (n *PunctNode) Children() []Node { return nil }
func (n *PunctNode) Text() string     { return n.text }

// CommentNode represents a comment (without trailing newline).
type CommentNode struct {
	parent Node
	text   string // includes leading '#'
}

func (c *CommentNode) Type() NodeType   { return NodeComment }
func (c *CommentNode) Parent() Node     { return c.parent }
func (c *CommentNode) Children() []Node { return nil }
func (c *CommentNode) Text() string     { return c.text }

// WhitespaceNode represents spaces/newlines/tabs between tokens.
type WhitespaceNode struct {
	parent Node
	text   string
}

func (w *WhitespaceNode) Type() NodeType   { return NodeWhitespace }
func (w *WhitespaceNode) Parent() Node     { return w.parent }
func (w *WhitespaceNode) Children() []Node { return nil }
func (w *WhitespaceNode) Text() string     { return w.text }

// KeyValue is a CST node for a single TOML key/value pair. It preserves
// leading/trailing trivia as both raw strings and parsed trivia nodes so
// clients can either read raw trivia or iterate trivia nodes directly.
type KeyValue struct {
	Leading      string // raw leading trivia
	LeadingNodes []Node

	Key      string // raw key text (as written)
	KeyTok   *IdentifierNode
	PreEq    string // whitespace between key and '='
	PostEq   string // whitespace between '=' and value
	Value    string // raw value text (as written)
	ValueTok Node   // concrete leaf node (StringNode/NumberNode/BooleanNode/...)

	Trailing      string // raw trailing trivia (comment+spaces)
	TrailingNodes []Node

	parent Node
}

// TableNode represents a table header like [table] and holds child nodes
// (key/value pairs) that belong to the table. For now child population
// is not performed by parser (future work) but header trivia is preserved.
type TableNode struct {
	Leading       string
	LeadingNodes  []Node
	Header        string // raw header text (e.g. "a.b")
	HeaderTok     *IdentifierNode
	Trailing      string
	TrailingNodes []Node
	parent        Node
}

// ArrayOfTables represents a header like [[array.of.tables]].
type ArrayOfTables struct {
	Leading       string
	LeadingNodes  []Node
	Header        string
	HeaderTok     *IdentifierNode
	Trailing      string
	TrailingNodes []Node
	parent        Node
}

func (t *TableNode) Type() NodeType { return NodeTable }
func (t *TableNode) Parent() Node   { return t.parent }
func (t *TableNode) Children() []Node {
	out := make([]Node, 0, len(t.LeadingNodes)+1+len(t.TrailingNodes))
	out = append(out, t.LeadingNodes...)
	if t.HeaderTok != nil {
		out = append(out, t.HeaderTok)
	}
	out = append(out, t.TrailingNodes...)
	return out
}
func (t *TableNode) Text() string { return "[" + t.Header + "]" + t.Trailing }

func (a *ArrayOfTables) Type() NodeType { return NodeArrayOfTables }
func (a *ArrayOfTables) Parent() Node   { return a.parent }
func (a *ArrayOfTables) Children() []Node {
	out := make([]Node, 0, len(a.LeadingNodes)+1+len(a.TrailingNodes))
	out = append(out, a.LeadingNodes...)
	if a.HeaderTok != nil {
		out = append(out, a.HeaderTok)
	}
	out = append(out, a.TrailingNodes...)
	return out
}
func (a *ArrayOfTables) Text() string { return "[[" + a.Header + "]]" + a.Trailing }

// Document represents a parsed TOML document as a sequence of CST nodes.
// For now we only model top-level KeyValue entries needed for initial tests.
type Document struct {
	Nodes []Node
}

// KeyValues returns the top-level KeyValue nodes in document order.
func (d *Document) KeyValues() []*KeyValue {
	out := make([]*KeyValue, 0)
	for _, n := range d.Nodes {
		if kv, ok := n.(*KeyValue); ok {
			out = append(out, kv)
		}
	}
	return out
}

// parseTrivia splits a trivia string into a sequence of CommentNode and
// WhitespaceNode instances. Comments exclude their trailing newline; the
// newline (if present) is emitted as a separate WhitespaceNode so callers
// can treat comments and line breaks independently.
func parseTrivia(s string, parent Node) []Node {
	var out []Node
	for i := 0; i < len(s); {
		ch := s[i]
		if ch == '#' {
			// comment until end of line (excluding newline)
			j := i
			for j < len(s) && s[j] != '\n' {
				j++
			}
			c := &CommentNode{parent: parent, text: s[i:j]}
			out = append(out, c)
			if j < len(s) && s[j] == '\n' {
				out = append(out, &WhitespaceNode{parent: parent, text: "\n"})
				j++
			}
			i = j
			continue
		}

		// run of whitespace (spaces/tabs/newlines)
		j := i
		for j < len(s) && s[j] != '#' {
			j++
		}
		ws := s[i:j]
		out = append(out, &WhitespaceNode{parent: parent, text: ws})
		i = j
	}
	return out
}

// parseKeyValueLine parses a left/right pair and appends a KeyValue node
// to the document. Returns true on success.
func parseKeyValueLine(left, right, pending string, d *Document) bool {
	key := strings.TrimSpace(left)
	pos := strings.Index(left, key)
	wsBeforeEq := left[pos+len(key):]

	val := right
	trailing := ""
	if idx := strings.Index(right, "#"); idx >= 0 {
		j := idx - 1
		for j >= 0 && (right[j] == ' ' || right[j] == '\t') {
			j--
		}
		trailing = right[j+1:]
		val = right[:j+1]
	}

	valTrim := strings.TrimLeft(val, " \t")
	wsAfterEq := val[:len(val)-len(valTrim)]
	value := strings.TrimSpace(val)

	kv := &KeyValue{
		Leading:  pending,
		Key:      key,
		PreEq:    wsBeforeEq,
		PostEq:   wsAfterEq,
		Value:    value,
		Trailing: trailing,
	}
	kv.parent = d
	kv.KeyTok = &IdentifierNode{parent: kv, text: kv.Key}
	kv.ValueTok = detectValueNode(kv.Value, kv)
	if kv.Leading != "" {
		kv.LeadingNodes = parseTrivia(kv.Leading, kv)
	}
	if kv.Trailing != "" {
		kv.TrailingNodes = parseTrivia(kv.Trailing, kv)
	}
	d.Nodes = append(d.Nodes, kv)
	return true
}

// isNumberLike reports whether s looks like a TOML numeric token. This
// uses a simple allowed-character check (sufficient for token-kind
// exposure in tests).
func isNumberLike(s string) bool {
	if s == "" {
		return false
	}
	allowed := "0123456789._+-eEoxabcfABCDEF"
	for _, r := range s {
		if !strings.ContainsRune(allowed, r) {
			return false
		}
	}
	return true
}

// helper to attach header node (table or array-of-tables) and parse trivia.
func attachHeaderNode(isArray bool, header, trailing, pending string, d *Document) {
	if isArray {
		a := &ArrayOfTables{
			Leading:  pending,
			Header:   header,
			Trailing: trailing,
			parent:   d,
		}
		a.HeaderTok = &IdentifierNode{parent: a, text: header}
		if a.Leading != "" {
			a.LeadingNodes = parseTrivia(a.Leading, a)
		}
		if a.Trailing != "" {
			a.TrailingNodes = parseTrivia(a.Trailing, a)
		}
		d.Nodes = append(d.Nodes, a)
		return
	}
	// table
	t := &TableNode{
		Leading:  pending,
		Header:   header,
		Trailing: trailing,
		parent:   d,
	}
	t.HeaderTok = &IdentifierNode{parent: t, text: header}
	if t.Leading != "" {
		t.LeadingNodes = parseTrivia(t.Leading, t)
	}
	if t.Trailing != "" {
		t.TrailingNodes = parseTrivia(t.Trailing, t)
	}
	d.Nodes = append(d.Nodes, t)
}

// parseHeaderLine attempts to parse a table or array-of-tables header
// from the provided raw line. When successful it appends the node to d
// and returns true.
func parseHeaderLine(line, pending string, d *Document) bool {
	trimLeft := strings.TrimLeft(line, " \t")
	// array-of-tables
	if strings.HasPrefix(trimLeft, "[[") {
		start := strings.Index(line, "[[")
		end := strings.LastIndex(line, "]]")
		if start < 0 || end <= start {
			return false
		}
		header := strings.TrimSpace(line[start+2 : end])
		attachHeaderNode(true, header, line[end+2:], pending, d)
		return true
	}

	// regular table header
	if strings.HasPrefix(trimLeft, "[") {
		start := strings.Index(line, "[")
		end := strings.LastIndex(line, "]")
		if start < 0 || end <= start {
			return false
		}
		header := strings.TrimSpace(line[start+1 : end])
		attachHeaderNode(false, header, line[end+1:], pending, d)
		return true
	}

	return false
}

func isBoolToken(v string) bool { return v == "true" || v == "false" }

func isStringToken(v string) bool {
	return len(v) > 0 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\''))
}

func isDateTimeLike(v string) bool {
	return strings.Contains(v, "T") || strings.Count(v, "-") >= 2
}

// detectValueNode returns a concrete leaf node for a raw value string
// using lightweight heuristics (good enough for token-kind exposure).
func detectValueNode(val string, parent Node) Node {
	v := strings.TrimSpace(val)
	if isBoolToken(v) {
		return &BooleanNode{parent: parent, text: v}
	}
	if isStringToken(v) {
		return &StringNode{parent: parent, text: v}
	}
	// number-like
	if isNumberLike(v) && strings.ContainsAny(v, "0123456789") {
		return &NumberNode{parent: parent, text: v}
	}
	if isDateTimeLike(v) {
		return &DateTimeNode{parent: parent, text: v}
	}
	// fallback to identifier-like node
	return &IdentifierNode{parent: parent, text: v}
}

// Parse reads a TOML document from bytes. This is a small, incremental
// implementation intended to validate the CST/trivia model and support
// tests. It currently supports single-line top-level key/value pairs and
// preserves leading/trailing trivia (comments & blank lines) for those.
func processLine(raw string, idx, total int, pending *bytes.Buffer, d *Document) {
	line := raw
	// blank line -> accumulate as leading trivia
	if strings.TrimSpace(line) == "" {
		pending.WriteString(line)
		if idx < total-1 {
			pending.WriteString("\n")
		}
		return
	}

	trim := strings.TrimSpace(line)
	if strings.HasPrefix(trim, "#") {
		// full-line comment -> keep as leading trivia
		pending.WriteString(line)
		if idx < total-1 {
			pending.WriteString("\n")
		}
		return
	}

	// Attempt to parse a simple key = value [# comment] line.
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		// maybe a table header ([table] or [[array]])
		if parseHeaderLine(line, pending.String(), d) {
			pending.Reset()
			return
		}

		// unknown/unsupported line — for now treat as leading trivia
		pending.WriteString(line)
		if idx < total-1 {
			pending.WriteString("\n")
		}
		return
	}

	// delegate key/value parsing to helper
	if parseKeyValueLine(parts[0], parts[1], pending.String(), d) {
		pending.Reset()
		return
	}

	// fallback: treat as unknown line
	pending.Reset()
}

func Parse(b []byte) (*Document, error) {
	if b == nil {
		return nil, ErrNilInput
	}

	s := string(b)
	if s == "" {
		return &Document{}, nil
	}

	lines := strings.Split(s, "\n")
	d := &Document{}
	var pendingLeading bytes.Buffer

	for i, raw := range lines {
		processLine(raw, i, len(lines), &pendingLeading, d)
	}

	return d, nil
}

// String renders the document back to source. The serializer uses the
// preserved trivia and token text to produce a source-preserving output
// for the constructs we currently model.
func (d *Document) String() string {
	var b strings.Builder
	for i, n := range d.Nodes {
		switch v := n.(type) {
		case *KeyValue:
			serializeKeyValue(&b, v, i == len(d.Nodes)-1)
		case *TableNode:
			serializeTableNode(&b, v, i == len(d.Nodes)-1)
		case *ArrayOfTables:
			serializeArrayOfTables(&b, v, i == len(d.Nodes)-1)
		default:
			b.WriteString(n.Text())
		}
	}
	return b.String()
}

func serializeKeyValue(b *strings.Builder, v *KeyValue, isLast bool) {
	if v.Leading != "" {
		b.WriteString(v.Leading)
		if !strings.HasSuffix(v.Leading, "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString(v.Key)
	b.WriteString(v.PreEq)
	b.WriteString("=")
	b.WriteString(v.PostEq)
	b.WriteString(v.Value)
	b.WriteString(v.Trailing)
	if !isLast || !strings.HasSuffix(v.Trailing, "\n") {
		b.WriteString("\n")
	}
}

func serializeTableNode(b *strings.Builder, v *TableNode, isLast bool) {
	if v.Leading != "" {
		b.WriteString(v.Leading)
		if !strings.HasSuffix(v.Leading, "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString("[")
	b.WriteString(v.Header)
	b.WriteString("]")
	b.WriteString(v.Trailing)
	if !isLast || !strings.HasSuffix(v.Trailing, "\n") {
		b.WriteString("\n")
	}
}

func serializeArrayOfTables(b *strings.Builder, v *ArrayOfTables, isLast bool) {
	if v.Leading != "" {
		b.WriteString(v.Leading)
		if !strings.HasSuffix(v.Leading, "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString("[[")
	b.WriteString(v.Header)
	b.WriteString("]]")
	b.WriteString(v.Trailing)
	if !isLast || !strings.HasSuffix(v.Trailing, "\n") {
		b.WriteString("\n")
	}
}

// Node interface implementations ------------------------------------------------

func (d *Document) Type() NodeType   { return NodeDocument }
func (d *Document) Parent() Node     { return nil }
func (d *Document) Children() []Node { return append([]Node(nil), d.Nodes...) }
func (d *Document) Text() string     { return d.String() }

func (k *KeyValue) Type() NodeType { return NodeKeyValue }
func (k *KeyValue) Parent() Node   { return k.parent }
func (k *KeyValue) Children() []Node {
	out := make([]Node, 0, 4+len(k.LeadingNodes)+len(k.TrailingNodes))
	// leading trivia
	out = append(out, k.LeadingNodes...)
	// key token
	if k.KeyTok != nil {
		out = append(out, k.KeyTok)
	}
	// value token
	if k.ValueTok != nil {
		out = append(out, k.ValueTok)
	}
	// trailing trivia
	out = append(out, k.TrailingNodes...)
	return out
}
func (k *KeyValue) Text() string {
	// return source-like rendering for this node
	var b strings.Builder
	b.WriteString(k.Key)
	b.WriteString(k.PreEq)
	b.WriteString("=")
	b.WriteString(k.PostEq)
	b.WriteString(k.Value)
	b.WriteString(k.Trailing)
	return b.String()
}

// Walk traverses the CST in pre-order. Visitor returns false to stop.
func (d *Document) Walk(visitor func(Node) bool) {
	var walk func(Node) bool
	walk = func(n Node) bool {
		if !visitor(n) {
			return false
		}
		for _, c := range n.Children() {
			if !walk(c) {
				return false
			}
		}
		return true
	}
	walk(d)
}

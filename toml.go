package toml

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors.
var (
	ErrNilInput          = errors.New("nil input")
	ErrEmptyKey          = errors.New("empty key")
	ErrUnexpectedContent = errors.New("unexpected content after key")
	ErrNilValue          = errors.New("nil value")
	ErrInvalidValueType  = errors.New("invalid value type")
	ErrNilNode           = errors.New("nil node")
	ErrInvalidNodeType   = errors.New("invalid document node type")
	ErrInvalidDateTime   = errors.New("invalid datetime")
	ErrNilEntry          = errors.New("nil key-value")
	ErrDuplicateKey      = errors.New("duplicate key")
	ErrKeyConflict       = errors.New("key conflicts with dotted key")
	ErrIndexOutOfRange   = errors.New("index out of range")
	ErrInvalidWhitespace = errors.New("invalid whitespace: must contain only spaces and tabs")
	ErrInvalidNewline    = errors.New("invalid newline: must be empty, \\n, or \\r\\n")
	ErrInvalidTrivia     = errors.New("invalid trivia node: must be *CommentNode or *WhitespaceNode")
	ErrCommentNewline    = errors.New("comment text must not contain newlines")
	ErrCommentControl    = errors.New("comment text contains invalid control character")
	ErrInvalidWsChar     = errors.New("whitespace text contains non-whitespace character")
)

// ParseError represents a parsing error with location information.
type ParseError struct {
	Message string
	Line    int
	Column  int
	Source  string
}

func (e *ParseError) Error() string {
	lines := strings.Split(e.Source, "\n")
	if e.Line < 1 || e.Line > len(lines) {
		return fmt.Sprintf("parse error at line %d: %s", e.Line, e.Message)
	}
	lineContent := lines[e.Line-1]
	var buf strings.Builder
	fmt.Fprintf(&buf, "parse error at line %d, column %d: %s\n", e.Line, e.Column, e.Message)
	fmt.Fprintf(&buf, "  %d | %s\n", e.Line, lineContent)
	buf.WriteString("    | ")
	for i := 1; i < e.Column; i++ {
		if i-1 < len(lineContent) && lineContent[i-1] == '\t' {
			buf.WriteByte('\t')
		} else {
			buf.WriteByte(' ')
		}
	}
	buf.WriteString("^\n")
	return buf.String()
}

// NodeType identifies node kinds in the CST.
type NodeType int

const (
	NodeDocument NodeType = iota
	NodeKeyValue
	NodeTable
	NodeArrayOfTables
	NodeArray
	NodeInlineTable
	NodeIdentifier
	NodeString
	NodeNumber
	NodeBoolean
	NodeDateTime
	NodePunctuation
	NodeComment
	NodeWhitespace
)

// Node is the CST node interface.
type Node interface {
	Type() NodeType
	Parent() Node
	Children() []Node
	Text() string
}

// baseNode provides shared parent tracking for all nodes.
type baseNode struct {
	parent   Node
	nodeType NodeType
	line     int
	col      int
}

func (b *baseNode) Type() NodeType   { return b.nodeType }
func (b *baseNode) Parent() Node     { return b.parent }
func (b *baseNode) setParent(p Node) { b.parent = p }

// leafNode is the common implementation for all terminal/leaf nodes.
type leafNode struct {
	baseNode
	text string
}

func (n *leafNode) Children() []Node { return nil }
func (n *leafNode) Text() string     { return n.text }

// Concrete leaf node types.

type IdentifierNode struct{ leafNode }
type StringNode struct{ leafNode }
type NumberNode struct{ leafNode }
type BooleanNode struct{ leafNode }
type DateTimeNode struct{ leafNode }
type PunctNode struct{ leafNode }
type CommentNode struct{ leafNode }
type WhitespaceNode struct{ leafNode }

func newLeaf(nodeType NodeType, text string) leafNode {
	return leafNode{baseNode: baseNode{nodeType: nodeType}, text: text}
}

// KeyPart represents one segment of a potentially dotted key.
type KeyPart struct {
	Text      string // raw text including quotes if quoted
	Unquoted  string // the bare key name (quotes and escapes resolved)
	IsQuoted  bool
	DotBefore string // whitespace before the preceding dot (empty for first part)
	DotAfter  string // whitespace after the preceding dot
}

// KeyValue is a CST node for key = value pairs.
type KeyValue struct {
	baseNode
	leadingTrivia  []Node    // whitespace/comments before the key
	keyParts       []KeyPart // parsed segments of (possibly dotted) key
	rawKey         string    // full raw key text as written (e.g. "a . b")
	preEq          string    // whitespace between key and =
	postEq         string    // whitespace between = and value
	val            Node      // typed value node
	rawVal         string    // raw value text as written
	trailingTrivia []Node    // trailing comment/whitespace on same line
	newline        string    // the line-ending newline if present
}

// KeyParts returns a copy of the parsed key segments.
func (kv *KeyValue) KeyParts() []KeyPart {
	return append([]KeyPart(nil), kv.keyParts...)
}

// RawKey returns the full raw key text as written.
func (kv *KeyValue) RawKey() string {
	return kv.rawKey
}

// Val returns the typed value node.
func (kv *KeyValue) Val() Node {
	return kv.val
}

// RawVal returns the raw value text as written.
func (kv *KeyValue) RawVal() string {
	return kv.rawVal
}

// LeadingTrivia returns a copy of the leading trivia nodes.
func (kv *KeyValue) LeadingTrivia() []Node {
	return append([]Node(nil), kv.leadingTrivia...)
}

// TrailingTrivia returns a copy of the trailing trivia nodes.
func (kv *KeyValue) TrailingTrivia() []Node {
	return append([]Node(nil), kv.trailingTrivia...)
}

// PreEq returns the whitespace between key and =.
func (kv *KeyValue) PreEq() string { return kv.preEq }

// PostEq returns the whitespace between = and value.
func (kv *KeyValue) PostEq() string { return kv.postEq }

// Newline returns the line-ending newline.
func (kv *KeyValue) Newline() string { return kv.newline }

// SetLeadingTrivia sets the leading trivia nodes.
// Each node must be a *CommentNode or *WhitespaceNode.
func (kv *KeyValue) SetLeadingTrivia(nodes []Node) error {
	if err := validateTriviaNodes(nodes); err != nil {
		return err
	}
	kv.leadingTrivia = append([]Node(nil), nodes...)
	return nil
}

// SetTrailingTrivia sets the trailing trivia nodes.
// Each node must be a *CommentNode or *WhitespaceNode.
func (kv *KeyValue) SetTrailingTrivia(nodes []Node) error {
	if err := validateTriviaNodes(nodes); err != nil {
		return err
	}
	kv.trailingTrivia = append([]Node(nil), nodes...)
	return nil
}

// SetPreEq sets the whitespace between key and =.
// Must contain only spaces and tabs.
func (kv *KeyValue) SetPreEq(s string) error {
	if !isHorizWhitespace(s) {
		return ErrInvalidWhitespace
	}
	kv.preEq = s
	regenerateAncestorText(kv)
	return nil
}

// SetPostEq sets the whitespace between = and value.
// Must contain only spaces and tabs.
func (kv *KeyValue) SetPostEq(s string) error {
	if !isHorizWhitespace(s) {
		return ErrInvalidWhitespace
	}
	kv.postEq = s
	regenerateAncestorText(kv)
	return nil
}

// SetNewline sets the line-ending newline.
// Must be "", "\n", or "\r\n".
func (kv *KeyValue) SetNewline(s string) error {
	if !isValidNewline(s) {
		return ErrInvalidNewline
	}
	kv.newline = s
	return nil
}

func (kv *KeyValue) Children() []Node {
	var out []Node
	out = append(out, kv.leadingTrivia...)
	if kv.val != nil {
		out = append(out, kv.val)
	}
	out = append(out, kv.trailingTrivia...)
	return out
}

func (kv *KeyValue) Text() string {
	var b strings.Builder
	b.WriteString(kv.rawKey)
	b.WriteString(kv.preEq)
	b.WriteString("=")
	b.WriteString(kv.postEq)
	if kv.val != nil {
		b.WriteString(kv.val.Text())
	}
	return b.String()
}

// TableNode represents [table.header] and holds child entries.
type TableNode struct {
	baseNode
	leadingTrivia  []Node
	rawHeader      string // full raw header text between brackets
	headerParts    []KeyPart
	trailingTrivia []Node // trivia after ] on the header line
	newline        string
	entries        []Node // child KeyValue nodes
}

// RawHeader returns the full raw header text between brackets.
func (t *TableNode) RawHeader() string {
	return t.rawHeader
}

// HeaderParts returns a copy of the parsed header key segments.
func (t *TableNode) HeaderParts() []KeyPart {
	return append([]KeyPart(nil), t.headerParts...)
}

// Entries returns a copy of the child entry nodes.
func (t *TableNode) Entries() []Node {
	return append([]Node(nil), t.entries...)
}

// LeadingTrivia returns a copy of the leading trivia nodes.
func (t *TableNode) LeadingTrivia() []Node {
	return append([]Node(nil), t.leadingTrivia...)
}

// TrailingTrivia returns a copy of the trailing trivia nodes.
func (t *TableNode) TrailingTrivia() []Node {
	return append([]Node(nil), t.trailingTrivia...)
}

// Newline returns the line-ending newline.
func (t *TableNode) Newline() string { return t.newline }

// SetLeadingTrivia sets the leading trivia nodes.
func (t *TableNode) SetLeadingTrivia(nodes []Node) error {
	if err := validateTriviaNodes(nodes); err != nil {
		return err
	}
	t.leadingTrivia = append([]Node(nil), nodes...)
	return nil
}

// SetTrailingTrivia sets the trailing trivia nodes.
func (t *TableNode) SetTrailingTrivia(nodes []Node) error {
	if err := validateTriviaNodes(nodes); err != nil {
		return err
	}
	t.trailingTrivia = append([]Node(nil), nodes...)
	return nil
}

// SetNewline sets the line-ending newline.
func (t *TableNode) SetNewline(s string) error {
	if !isValidNewline(s) {
		return ErrInvalidNewline
	}
	t.newline = s
	return nil
}

func (t *TableNode) Children() []Node {
	var out []Node
	out = append(out, t.leadingTrivia...)
	out = append(out, t.entries...)
	out = append(out, t.trailingTrivia...)
	return out
}

func (t *TableNode) Text() string {
	return "[" + t.rawHeader + "]"
}

// ArrayOfTables represents [[array.of.tables]] and holds child entries.
type ArrayOfTables struct {
	baseNode
	leadingTrivia  []Node
	rawHeader      string
	headerParts    []KeyPart
	trailingTrivia []Node
	newline        string
	entries        []Node
}

// RawHeader returns the full raw header text between brackets.
func (a *ArrayOfTables) RawHeader() string {
	return a.rawHeader
}

// HeaderParts returns a copy of the parsed header key segments.
func (a *ArrayOfTables) HeaderParts() []KeyPart {
	return append([]KeyPart(nil), a.headerParts...)
}

// Entries returns a copy of the child entry nodes.
func (a *ArrayOfTables) Entries() []Node {
	return append([]Node(nil), a.entries...)
}

// LeadingTrivia returns a copy of the leading trivia nodes.
func (a *ArrayOfTables) LeadingTrivia() []Node {
	return append([]Node(nil), a.leadingTrivia...)
}

// TrailingTrivia returns a copy of the trailing trivia nodes.
func (a *ArrayOfTables) TrailingTrivia() []Node {
	return append([]Node(nil), a.trailingTrivia...)
}

// Newline returns the line-ending newline.
func (a *ArrayOfTables) Newline() string { return a.newline }

// SetLeadingTrivia sets the leading trivia nodes.
func (a *ArrayOfTables) SetLeadingTrivia(nodes []Node) error {
	if err := validateTriviaNodes(nodes); err != nil {
		return err
	}
	a.leadingTrivia = append([]Node(nil), nodes...)
	return nil
}

// SetTrailingTrivia sets the trailing trivia nodes.
func (a *ArrayOfTables) SetTrailingTrivia(nodes []Node) error {
	if err := validateTriviaNodes(nodes); err != nil {
		return err
	}
	a.trailingTrivia = append([]Node(nil), nodes...)
	return nil
}

// SetNewline sets the line-ending newline.
func (a *ArrayOfTables) SetNewline(s string) error {
	if !isValidNewline(s) {
		return ErrInvalidNewline
	}
	a.newline = s
	return nil
}

func (a *ArrayOfTables) Children() []Node {
	var out []Node
	out = append(out, a.leadingTrivia...)
	out = append(out, a.entries...)
	out = append(out, a.trailingTrivia...)
	return out
}

func (a *ArrayOfTables) Text() string {
	return "[[" + a.rawHeader + "]]"
}

// ArrayNode represents [val1, val2, ...].
type ArrayNode struct {
	baseNode
	elements []Node
	text     string // raw source text
}

// Elements returns a copy of the array element nodes.
func (a *ArrayNode) Elements() []Node {
	return append([]Node(nil), a.elements...)
}

func (a *ArrayNode) Children() []Node { return append([]Node(nil), a.elements...) }
func (a *ArrayNode) Text() string     { return a.text }

// InlineTableNode represents { key = val, ... }.
type InlineTableNode struct {
	baseNode
	entries []*KeyValue
	text    string
}

// Entries returns a copy of the inline table entries.
func (n *InlineTableNode) Entries() []*KeyValue {
	return append([]*KeyValue(nil), n.entries...)
}

func (n *InlineTableNode) Children() []Node {
	out := make([]Node, 0, len(n.entries))
	for _, e := range n.entries {
		out = append(out, e)
	}
	return out
}

func (n *InlineTableNode) Text() string { return n.text }

// Document represents a parsed TOML document.
type Document struct {
	nodes []Node // top-level nodes: KeyValue, TableNode, ArrayOfTables
}

// Nodes returns a copy of the top-level nodes.
func (d *Document) Nodes() []Node {
	return append([]Node(nil), d.nodes...)
}

func (d *Document) Type() NodeType   { return NodeDocument }
func (d *Document) Parent() Node     { return nil }
func (d *Document) Children() []Node { return append([]Node(nil), d.nodes...) }
func (d *Document) Text() string     { return d.String() }

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

// Tables returns all TableNode nodes in document order.
func (d *Document) Tables() []*TableNode {
	var out []*TableNode
	for _, n := range d.nodes {
		if t, ok := n.(*TableNode); ok {
			out = append(out, t)
		}
	}
	return out
}

// ArraysOfTables returns all ArrayOfTables nodes in document order.
func (d *Document) ArraysOfTables() []*ArrayOfTables {
	var out []*ArrayOfTables
	for _, n := range d.nodes {
		if a, ok := n.(*ArrayOfTables); ok {
			out = append(out, a)
		}
	}
	return out
}

// String renders the document back to source, preserving formatting.
func (d *Document) String() string {
	var b strings.Builder
	for _, n := range d.nodes {
		serializeNode(&b, n)
	}
	return b.String()
}

func serializeNode(b *strings.Builder, n Node) {
	switch v := n.(type) {
	case *KeyValue:
		serializeKeyValue(b, v)
	case *TableNode:
		serializeTableNode(b, v)
	case *ArrayOfTables:
		serializeArrayOfTables(b, v)
	default:
		b.WriteString(n.Text())
	}
}

func serializeTrivia(b *strings.Builder, nodes []Node) {
	for _, n := range nodes {
		b.WriteString(n.Text())
	}
}

func serializeKeyValue(b *strings.Builder, kv *KeyValue) {
	serializeTrivia(b, kv.leadingTrivia)
	b.WriteString(kv.rawKey)
	b.WriteString(kv.preEq)
	b.WriteString("=")
	b.WriteString(kv.postEq)
	if kv.val != nil {
		b.WriteString(kv.val.Text())
	}
	serializeTrivia(b, kv.trailingTrivia)
	b.WriteString(kv.newline)
}

func serializeTableNode(b *strings.Builder, t *TableNode) {
	serializeTrivia(b, t.leadingTrivia)
	b.WriteString("[")
	b.WriteString(t.rawHeader)
	b.WriteString("]")
	serializeTrivia(b, t.trailingTrivia)
	b.WriteString(t.newline)
	for _, entry := range t.entries {
		serializeNode(b, entry)
	}
}

func serializeArrayOfTables(b *strings.Builder, a *ArrayOfTables) {
	serializeTrivia(b, a.leadingTrivia)
	b.WriteString("[[")
	b.WriteString(a.rawHeader)
	b.WriteString("]]")
	serializeTrivia(b, a.trailingTrivia)
	b.WriteString(a.newline)
	for _, entry := range a.entries {
		serializeNode(b, entry)
	}
}

// Parse reads a TOML document from bytes.
func Parse(b []byte) (*Document, error) {
	if b == nil {
		return nil, ErrNilInput
	}
	if msg := validateUTF8(b); msg != "" {
		return nil, &ParseError{Message: msg, Line: 1, Column: 1, Source: string(b)}
	}
	s := string(b)
	if s == "" {
		return &Document{}, nil
	}
	p := newParser(s)
	doc, err := p.parse()
	if err != nil {
		return nil, err
	}
	if err := validateDocument(doc, s); err != nil {
		return nil, err
	}
	return doc, nil
}

// --- Validation helpers for setters ---

// validateTriviaNodes checks that each node is a *CommentNode or *WhitespaceNode.
func validateTriviaNodes(nodes []Node) error {
	for _, n := range nodes {
		if n == nil {
			return ErrInvalidTrivia
		}
		switch n.(type) {
		case *CommentNode, *WhitespaceNode:
			// ok
		default:
			return ErrInvalidTrivia
		}
	}
	return nil
}

// isHorizWhitespace returns true if s contains only spaces and tabs (or is empty).
func isHorizWhitespace(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return false
		}
	}
	return true
}

// isValidNewline returns true if s is "", "\n", or "\r\n".
func isValidNewline(s string) bool {
	return s == "" || s == "\n" || s == "\r\n"
}

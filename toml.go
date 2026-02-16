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
	LeadingTrivia  []Node    // whitespace/comments before the key
	keyParts       []KeyPart // parsed segments of (possibly dotted) key
	rawKey         string    // full raw key text as written (e.g. "a . b")
	PreEq          string    // whitespace between key and =
	PostEq         string    // whitespace between = and value
	val            Node      // typed value node
	rawVal         string    // raw value text as written
	TrailingTrivia []Node    // trailing comment/whitespace on same line
	Newline        string    // the line-ending newline if present
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

func (kv *KeyValue) Children() []Node {
	var out []Node
	out = append(out, kv.LeadingTrivia...)
	if kv.val != nil {
		out = append(out, kv.val)
	}
	out = append(out, kv.TrailingTrivia...)
	return out
}

func (kv *KeyValue) Text() string {
	var b strings.Builder
	b.WriteString(kv.rawKey)
	b.WriteString(kv.PreEq)
	b.WriteString("=")
	b.WriteString(kv.PostEq)
	if kv.val != nil {
		b.WriteString(kv.val.Text())
	}
	return b.String()
}

// TableNode represents [table.header] and holds child entries.
type TableNode struct {
	baseNode
	LeadingTrivia  []Node
	rawHeader      string // full raw header text between brackets
	headerParts    []KeyPart
	TrailingTrivia []Node // trivia after ] on the header line
	Newline        string
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

func (t *TableNode) Children() []Node {
	var out []Node
	out = append(out, t.LeadingTrivia...)
	out = append(out, t.entries...)
	out = append(out, t.TrailingTrivia...)
	return out
}

func (t *TableNode) Text() string {
	return "[" + t.rawHeader + "]"
}

// ArrayOfTables represents [[array.of.tables]] and holds child entries.
type ArrayOfTables struct {
	baseNode
	LeadingTrivia  []Node
	rawHeader      string
	headerParts    []KeyPart
	TrailingTrivia []Node
	Newline        string
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

func (a *ArrayOfTables) Children() []Node {
	var out []Node
	out = append(out, a.LeadingTrivia...)
	out = append(out, a.entries...)
	out = append(out, a.TrailingTrivia...)
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
	serializeTrivia(b, kv.LeadingTrivia)
	b.WriteString(kv.rawKey)
	b.WriteString(kv.PreEq)
	b.WriteString("=")
	b.WriteString(kv.PostEq)
	if kv.val != nil {
		b.WriteString(kv.val.Text())
	}
	serializeTrivia(b, kv.TrailingTrivia)
	b.WriteString(kv.Newline)
}

func serializeTableNode(b *strings.Builder, t *TableNode) {
	serializeTrivia(b, t.LeadingTrivia)
	b.WriteString("[")
	b.WriteString(t.rawHeader)
	b.WriteString("]")
	serializeTrivia(b, t.TrailingTrivia)
	b.WriteString(t.Newline)
	for _, entry := range t.entries {
		serializeNode(b, entry)
	}
}

func serializeArrayOfTables(b *strings.Builder, a *ArrayOfTables) {
	serializeTrivia(b, a.LeadingTrivia)
	b.WriteString("[[")
	b.WriteString(a.rawHeader)
	b.WriteString("]]")
	serializeTrivia(b, a.TrailingTrivia)
	b.WriteString(a.Newline)
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

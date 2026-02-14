package toml

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors.
var (
	ErrNilInput = errors.New("nil input")
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
	buf.WriteString(fmt.Sprintf("parse error at line %d, column %d: %s\n", e.Line, e.Column, e.Message))
	buf.WriteString(fmt.Sprintf("  %d | %s\n", e.Line, lineContent))
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
	KeyParts       []KeyPart // parsed segments of (possibly dotted) key
	RawKey         string    // full raw key text as written (e.g. "a . b")
	PreEq          string    // whitespace between key and =
	PostEq         string    // whitespace between = and value
	Val            Node      // typed value node
	RawVal         string    // raw value text as written
	TrailingTrivia []Node    // trailing comment/whitespace on same line
	Newline        string    // the line-ending newline if present
}

func (k *KeyValue) Children() []Node {
	var out []Node
	out = append(out, k.LeadingTrivia...)
	if k.Val != nil {
		out = append(out, k.Val)
	}
	out = append(out, k.TrailingTrivia...)
	return out
}

func (k *KeyValue) Text() string {
	var b strings.Builder
	b.WriteString(k.RawKey)
	b.WriteString(k.PreEq)
	b.WriteString("=")
	b.WriteString(k.PostEq)
	b.WriteString(k.RawVal)
	return b.String()
}

// TableNode represents [table.header] and holds child entries.
type TableNode struct {
	baseNode
	LeadingTrivia  []Node
	RawHeader      string // full raw header text between brackets
	HeaderParts    []KeyPart
	TrailingTrivia []Node // trivia after ] on the header line
	Newline        string
	Entries        []Node // child KeyValue nodes
}

func (t *TableNode) Children() []Node {
	var out []Node
	out = append(out, t.LeadingTrivia...)
	out = append(out, t.Entries...)
	out = append(out, t.TrailingTrivia...)
	return out
}

func (t *TableNode) Text() string {
	return "[" + t.RawHeader + "]"
}

// ArrayOfTables represents [[array.of.tables]] and holds child entries.
type ArrayOfTables struct {
	baseNode
	LeadingTrivia  []Node
	RawHeader      string
	HeaderParts    []KeyPart
	TrailingTrivia []Node
	Newline        string
	Entries        []Node
}

func (a *ArrayOfTables) Children() []Node {
	var out []Node
	out = append(out, a.LeadingTrivia...)
	out = append(out, a.Entries...)
	out = append(out, a.TrailingTrivia...)
	return out
}

func (a *ArrayOfTables) Text() string {
	return "[[" + a.RawHeader + "]]"
}

// ArrayNode represents [val1, val2, ...].
type ArrayNode struct {
	baseNode
	Elements []Node
	text     string // raw source text
}

func (a *ArrayNode) Children() []Node { return append([]Node(nil), a.Elements...) }
func (a *ArrayNode) Text() string     { return a.text }

// InlineTableNode represents { key = val, ... }.
type InlineTableNode struct {
	baseNode
	Entries []*KeyValue
	text    string
}

func (n *InlineTableNode) Children() []Node {
	out := make([]Node, 0, len(n.Entries))
	for _, e := range n.Entries {
		out = append(out, e)
	}
	return out
}

func (n *InlineTableNode) Text() string { return n.text }

// Document represents a parsed TOML document.
type Document struct {
	Nodes []Node // top-level nodes: KeyValue, TableNode, ArrayOfTables
}

func (d *Document) Type() NodeType   { return NodeDocument }
func (d *Document) Parent() Node     { return nil }
func (d *Document) Children() []Node { return append([]Node(nil), d.Nodes...) }
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
	for _, n := range d.Nodes {
		if t, ok := n.(*TableNode); ok {
			out = append(out, t)
		}
	}
	return out
}

// ArraysOfTables returns all ArrayOfTables nodes in document order.
func (d *Document) ArraysOfTables() []*ArrayOfTables {
	var out []*ArrayOfTables
	for _, n := range d.Nodes {
		if a, ok := n.(*ArrayOfTables); ok {
			out = append(out, a)
		}
	}
	return out
}

// String renders the document back to source, preserving formatting.
func (d *Document) String() string {
	var b strings.Builder
	for _, n := range d.Nodes {
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
	b.WriteString(kv.RawKey)
	b.WriteString(kv.PreEq)
	b.WriteString("=")
	b.WriteString(kv.PostEq)
	b.WriteString(kv.RawVal)
	serializeTrivia(b, kv.TrailingTrivia)
	b.WriteString(kv.Newline)
}

func serializeTableNode(b *strings.Builder, t *TableNode) {
	serializeTrivia(b, t.LeadingTrivia)
	b.WriteString("[")
	b.WriteString(t.RawHeader)
	b.WriteString("]")
	serializeTrivia(b, t.TrailingTrivia)
	b.WriteString(t.Newline)
	for _, entry := range t.Entries {
		serializeNode(b, entry)
	}
}

func serializeArrayOfTables(b *strings.Builder, a *ArrayOfTables) {
	serializeTrivia(b, a.LeadingTrivia)
	b.WriteString("[[")
	b.WriteString(a.RawHeader)
	b.WriteString("]]")
	serializeTrivia(b, a.TrailingTrivia)
	b.WriteString(a.Newline)
	for _, entry := range a.Entries {
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

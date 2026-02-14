package toml

import (
	"fmt"
	"math"
	"strings"
)

// --- Key helpers ---

func isBareKeyStr(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !isBareKeyChar(r) {
			return false
		}
	}
	return true
}

// escapeBasicString escapes a Go string for use inside TOML double quotes.
func escapeBasicString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			escapeDefaultRune(&b, r)
		}
	}
	return b.String()
}

func escapeDefaultRune(b *strings.Builder, r rune) {
	switch {
	case r < 0x20 || r == 0x7F:
		b.WriteString(fmt.Sprintf(`\u%04X`, r))
	case r > 0xFFFF:
		b.WriteString(fmt.Sprintf(`\U%08X`, r))
	default:
		b.WriteRune(r)
	}
}

func makeKeyPart(key string) KeyPart {
	if isBareKeyStr(key) {
		return KeyPart{Text: key, Unquoted: key}
	}
	return KeyPart{
		Text:     `"` + escapeBasicString(key) + `"`,
		Unquoted: key,
		IsQuoted: true,
	}
}

func makeKeyParts(keys []string) []KeyPart {
	parts := make([]KeyPart, len(keys))
	for i, k := range keys {
		parts[i] = makeKeyPart(k)
	}
	return parts
}

func makeRawKey(parts []KeyPart) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(p.Text)
	}
	return b.String()
}

func makeRawHeader(parts []KeyPart) string {
	return makeRawKey(parts)
}

// --- Constructor functions ---

// NewString creates a new StringNode with the given Go string value,
// properly escaped and quoted for TOML.
func NewString(s string) *StringNode {
	return &StringNode{leafNode: newLeaf(NodeString, `"`+escapeBasicString(s)+`"`)}
}

// NewInteger creates a new NumberNode with a decimal integer representation.
func NewInteger(v int64) *NumberNode {
	return &NumberNode{leafNode: newLeaf(NodeNumber, fmt.Sprintf("%d", v))}
}

// NewFloat creates a new NumberNode with a float representation.
// Handles inf and nan values.
func NewFloat(v float64) *NumberNode {
	var text string
	switch {
	case math.IsInf(v, 1):
		text = "inf"
	case math.IsInf(v, -1):
		text = "-inf"
	case math.IsNaN(v):
		text = "nan"
	default:
		text = fmt.Sprintf("%v", v)
		if !strings.Contains(text, ".") && !strings.Contains(text, "e") {
			text += ".0"
		}
	}
	return &NumberNode{leafNode: newLeaf(NodeNumber, text)}
}

// NewBool creates a new BooleanNode.
func NewBool(v bool) *BooleanNode {
	text := "false"
	if v {
		text = "true"
	}
	return &BooleanNode{leafNode: newLeaf(NodeBoolean, text)}
}

// NewKeyValue creates a new KeyValue node with standard formatting (key = val\n).
// The key is automatically quoted if needed.
func NewKeyValue(key string, val Node) *KeyValue {
	segs := parseDottedPath(key)
	parts := makeKeyParts(segs)
	rawKey := makeRawKey(parts)
	rawVal := ""
	if val != nil {
		rawVal = val.Text()
	}
	return &KeyValue{
		baseNode: baseNode{nodeType: NodeKeyValue},
		KeyParts: parts,
		RawKey:   rawKey,
		PreEq:    " ",
		PostEq:   " ",
		Val:      val,
		RawVal:   rawVal,
		Newline:  "\n",
	}
}

// NewTable creates a new TableNode with the given header key segments.
// Each segment is automatically quoted if needed.
func NewTable(keys ...string) *TableNode {
	parts := makeKeyParts(keys)
	return &TableNode{
		baseNode:    baseNode{nodeType: NodeTable},
		RawHeader:   makeRawHeader(parts),
		HeaderParts: parts,
		Newline:     "\n",
	}
}

// --- Mutation methods ---

// SetValue updates the value of a KeyValue node.
func (kv *KeyValue) SetValue(val Node) {
	kv.Val = val
	if val != nil {
		kv.RawVal = val.Text()
	} else {
		kv.RawVal = ""
	}
}

// --- Document mutation ---

// Delete removes the first KeyValue matching the dotted path from the document.
// Returns true if a key was found and removed.
func (d *Document) Delete(path string) bool {
	segs := parseDottedPath(path)

	// Check top-level KVs.
	if idx := findTopLevelKV(d.Nodes, segs); idx >= 0 {
		d.Nodes = append(d.Nodes[:idx], d.Nodes[idx+1:]...)
		return true
	}

	// Check inside tables.
	return d.deleteFromTables(segs)
}

func findTopLevelKV(nodes []Node, segs []string) int {
	for i, n := range nodes {
		if kv, ok := n.(*KeyValue); ok {
			if matchKeyParts(kv.KeyParts, segs) {
				return i
			}
		}
	}
	return -1
}

func (d *Document) deleteFromTables(segs []string) bool {
	for prefixLen := len(segs) - 1; prefixLen >= 1; prefixLen-- {
		tableSegs := segs[:prefixLen]
		keySegs := segs[prefixLen:]
		for _, n := range d.Nodes {
			if deleteFromTableNode(n, tableSegs, keySegs) {
				return true
			}
		}
	}
	return false
}

func deleteFromTableNode(n Node, tableSegs, keySegs []string) bool {
	switch t := n.(type) {
	case *TableNode:
		if matchKeyParts(t.HeaderParts, tableSegs) {
			return deleteFromEntries(&t.Entries, keySegs)
		}
	case *ArrayOfTables:
		if matchKeyParts(t.HeaderParts, tableSegs) {
			return deleteFromEntries(&t.Entries, keySegs)
		}
	}
	return false
}

// DeleteTable removes the first TableNode matching the header path.
// Returns true if a table was found and removed.
func (d *Document) DeleteTable(path string) bool {
	segs := parseDottedPath(path)
	for i, n := range d.Nodes {
		if t, ok := n.(*TableNode); ok {
			if matchKeyParts(t.HeaderParts, segs) {
				d.Nodes = append(d.Nodes[:i], d.Nodes[i+1:]...)
				return true
			}
		}
	}
	return false
}

// Append adds a node to the end of the document's top-level nodes.
func (d *Document) Append(node Node) {
	d.Nodes = append(d.Nodes, node)
}

// InsertAt inserts a node at position i in the document's top-level nodes.
// If i is out of range, the node is appended.
func (d *Document) InsertAt(i int, node Node) {
	if i < 0 {
		i = 0
	}
	if i >= len(d.Nodes) {
		d.Nodes = append(d.Nodes, node)
		return
	}
	d.Nodes = append(d.Nodes[:i], append([]Node{node}, d.Nodes[i:]...)...)
}

// --- TableNode mutation ---

// Delete removes the first KeyValue matching the key from the table.
// Returns true if a key was found and removed.
func (t *TableNode) Delete(key string) bool {
	segs := parseDottedPath(key)
	return deleteFromEntries(&t.Entries, segs)
}

// Append adds a node to the end of the table's entries.
func (t *TableNode) Append(node Node) {
	t.Entries = append(t.Entries, node)
}

// InsertAt inserts a node at position i in the table's entries.
// If i is out of range, the node is appended.
func (t *TableNode) InsertAt(i int, node Node) {
	if i < 0 {
		i = 0
	}
	if i >= len(t.Entries) {
		t.Entries = append(t.Entries, node)
		return
	}
	t.Entries = append(t.Entries[:i], append([]Node{node}, t.Entries[i:]...)...)
}

// --- ArrayOfTables mutation ---

// Delete removes the first KeyValue matching the key from the array of tables.
// Returns true if a key was found and removed.
func (a *ArrayOfTables) Delete(key string) bool {
	segs := parseDottedPath(key)
	return deleteFromEntries(&a.Entries, segs)
}

// Append adds a node to the end of the array-of-tables' entries.
func (a *ArrayOfTables) Append(node Node) {
	a.Entries = append(a.Entries, node)
}

func deleteFromEntries(entries *[]Node, segs []string) bool {
	for i, e := range *entries {
		if kv, ok := e.(*KeyValue); ok {
			if matchKeyParts(kv.KeyParts, segs) {
				*entries = append((*entries)[:i], (*entries)[i+1:]...)
				return true
			}
		}
	}
	return false
}

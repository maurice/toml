package toml

import (
	"fmt"
	"math"
	"strings"
)

// --- Validation helpers ---

// parseRawKey parses a raw TOML key expression (bare, quoted, or dotted)
// and returns the parsed key parts and the trimmed raw key text.
// It reuses the lexer/parser infrastructure for full syntax validation.
func parseRawKey(raw string) ([]KeyPart, string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, "", ErrEmptyKey
	}
	p := newParser(raw)

	// Skip leading whitespace.
	if p.at(TokWhitespace) {
		p.advance()
	}

	parts, keyRaw, err := p.parseKey()
	if err != nil {
		return nil, "", err
	}

	// Skip trailing whitespace.
	if p.at(TokWhitespace) {
		p.advance()
	}

	if !p.at(TokEOF) {
		return nil, "", fmt.Errorf("%w: %q", ErrUnexpectedContent, p.cur.Text)
	}

	return parts, keyRaw, nil
}

// validateValueType checks that val is a valid TOML value node.
func validateValueType(val Node) error {
	if val == nil {
		return ErrNilValue
	}
	switch val.(type) {
	case *StringNode, *NumberNode, *BooleanNode, *DateTimeNode, *ArrayNode, *InlineTableNode:
		return nil
	default:
		return fmt.Errorf("%w: %T; expected string, number, bool, datetime, array, or inline table", ErrInvalidValueType, val)
	}
}

// validateDocumentNode checks that node is a valid top-level document node.
func validateDocumentNode(node Node) error {
	if node == nil {
		return ErrNilNode
	}
	switch node.(type) {
	case *KeyValue, *TableNode, *ArrayOfTables, *CommentNode, *WhitespaceNode:
		return nil
	default:
		return fmt.Errorf("%w: %T; expected *KeyValue, *TableNode, *ArrayOfTables, *CommentNode, or *WhitespaceNode", ErrInvalidNodeType, node)
	}
}

// --- Key helpers ---

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
// The rawKey is validated as a TOML key expression (bare, quoted, or dotted).
// The raw key text is preserved verbatim for serialization.
func NewKeyValue(rawKey string, val Node) (*KeyValue, error) {
	if err := validateValueType(val); err != nil {
		return nil, err
	}
	parts, keyRaw, err := parseRawKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}
	kv := &KeyValue{
		baseNode: baseNode{nodeType: NodeKeyValue},
		keyParts: parts,
		rawKey:   keyRaw,
		preEq:    " ",
		postEq:   " ",
		val:      val,
		rawVal:   val.Text(),
		newline:  "\n",
	}
	setValueParent(val, kv)
	return kv, nil
}

// NewTable creates a new TableNode.
// The rawKey is validated as a TOML key expression (bare, quoted, or dotted)
// and stored verbatim as the header content between [ and ].
func NewTable(rawKey string) (*TableNode, error) {
	parts, _, err := parseRawKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("invalid table key: %w", err)
	}
	return &TableNode{
		baseNode:    baseNode{nodeType: NodeTable},
		rawHeader:   rawKey,
		headerParts: parts,
		newline:     "\n",
	}, nil
}

// NewArrayOfTables creates a new ArrayOfTables node.
// The rawKey is validated as a TOML key expression and stored verbatim
// as the header content between [[ and ]].
func NewArrayOfTables(rawKey string) (*ArrayOfTables, error) {
	parts, _, err := parseRawKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("invalid array-of-tables key: %w", err)
	}
	return &ArrayOfTables{
		baseNode:    baseNode{nodeType: NodeArrayOfTables},
		rawHeader:   rawKey,
		headerParts: parts,
		newline:     "\n",
	}, nil
}

// NewDateTime creates a new DateTimeNode with the given TOML datetime string.
// The string is validated against TOML datetime formats (offset datetime,
// local datetime, local date, or local time).
func NewDateTime(s string) (*DateTimeNode, error) {
	if msg := validateDateTimeText(s); msg != "" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidDateTime, msg)
	}
	return &DateTimeNode{leafNode: newLeaf(NodeDateTime, s)}, nil
}

// NewArray creates a new ArrayNode with the given elements.
// Each element must be a valid TOML value node.
func NewArray(elements ...Node) (*ArrayNode, error) {
	for i, elem := range elements {
		if err := validateValueType(elem); err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
	}
	elems := make([]Node, len(elements))
	copy(elems, elements)
	a := &ArrayNode{
		baseNode: baseNode{nodeType: NodeArray},
		elements: elems,
	}
	a.text = generateArrayText(a.elements)
	return a, nil
}

// NewInlineTable creates a new InlineTableNode with the given key-value entries.
// Validates that entries are non-nil and that there are no duplicate keys.
func NewInlineTable(entries ...*KeyValue) (*InlineTableNode, error) {
	for i, kv := range entries {
		if kv == nil {
			return nil, fmt.Errorf("entry %d: %w", i, ErrNilEntry)
		}
	}
	seen := make(map[string]bool)
	for _, kv := range entries {
		path := keyPartsToPath(kv.keyParts)
		if seen[path] {
			return nil, fmt.Errorf("%w: %q in inline table", ErrDuplicateKey, path)
		}
		seen[path] = true
		for i := 1; i < len(kv.keyParts); i++ {
			prefix := keyPartsToPath(kv.keyParts[:i])
			if seen[prefix] {
				return nil, fmt.Errorf("%w: %q in inline table", ErrKeyConflict, prefix)
			}
		}
	}
	kvs := make([]*KeyValue, len(entries))
	copy(kvs, entries)
	n := &InlineTableNode{
		baseNode: baseNode{nodeType: NodeInlineTable},
		entries:  kvs,
	}
	for _, kv := range kvs {
		kv.setParent(n)
	}
	n.text = generateInlineTableText(n.entries)
	return n, nil
}

// generateArrayText produces the TOML text for an array from its elements.
func generateArrayText(elements []Node) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, elem := range elements {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(elem.Text())
	}
	b.WriteByte(']')
	return b.String()
}

// generateInlineTableText produces the TOML text for an inline table from its entries.
func generateInlineTableText(entries []*KeyValue) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, kv := range entries {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(kv.rawKey)
		b.WriteString(kv.preEq)
		b.WriteByte('=')
		b.WriteString(kv.postEq)
		if kv.val != nil {
			b.WriteString(kv.val.Text())
		}
	}
	b.WriteByte('}')
	return b.String()
}

// --- Parent tracking helpers ---

// setNodeParent sets the parent for any node type that embeds baseNode.
func setNodeParent(n Node, parent Node) {
	switch v := n.(type) {
	case *KeyValue:
		v.setParent(parent)
	case *TableNode:
		v.setParent(parent)
	case *ArrayOfTables:
		v.setParent(parent)
	case *CommentNode:
		v.setParent(parent)
	case *WhitespaceNode:
		v.setParent(parent)
	}
}

// setValueParent sets the parent for value nodes that embed baseNode.
func setValueParent(n Node, parent Node) {
	if n == nil {
		return
	}
	switch v := n.(type) {
	case *InlineTableNode:
		v.setParent(parent)
	case *ArrayNode:
		v.setParent(parent)
	case *StringNode:
		v.setParent(parent)
	case *NumberNode:
		v.setParent(parent)
	case *BooleanNode:
		v.setParent(parent)
	case *DateTimeNode:
		v.setParent(parent)
	}
}

// findDocument walks up the parent chain to find the containing Document.
func findDocument(n Node) *Document {
	for n != nil {
		if d, ok := n.(*Document); ok {
			return d
		}
		n = n.Parent()
	}
	return nil
}

// localDuplicateCheck checks for duplicate keys within a slice of entries.
func localDuplicateCheck(entries []Node) error {
	seen := make(map[string]bool)
	for _, e := range entries {
		kv, ok := e.(*KeyValue)
		if !ok {
			continue
		}
		path := keyPartsToPath(kv.keyParts)
		if seen[path] {
			return fmt.Errorf("%w: %q", ErrDuplicateKey, path)
		}
		seen[path] = true
		for i := 1; i < len(kv.keyParts); i++ {
			prefix := keyPartsToPath(kv.keyParts[:i])
			if seen[prefix] {
				return fmt.Errorf("%w: %q", ErrKeyConflict, prefix)
			}
		}
	}
	return nil
}

// --- Mutation methods ---

// SetValue updates the value of a KeyValue node.
// Returns an error if val is nil or not a valid TOML value type.
// If the KeyValue is inside an InlineTableNode or ArrayNode, the ancestor's
// text representation is regenerated.
func (kv *KeyValue) SetValue(val Node) error {
	if err := validateValueType(val); err != nil {
		return err
	}
	kv.val = val
	kv.rawVal = val.Text()
	setValueParent(val, kv)
	regenerateAncestorText(kv)
	return nil
}

// regenerateAncestorText walks up the parent chain and regenerates text
// for any InlineTableNode or ArrayNode ancestors.
func regenerateAncestorText(n Node) {
	for p := n.Parent(); p != nil; p = p.Parent() {
		switch v := p.(type) {
		case *InlineTableNode:
			v.text = generateInlineTableText(v.entries)
		case *ArrayNode:
			v.text = generateArrayText(v.elements)
		}
	}
}

// --- Document mutation ---

// Delete removes the first KeyValue matching the dotted path from the document.
// Returns true if a key was found and removed.
func (d *Document) Delete(path string) bool {
	segs := parseDottedPath(path)

	// Check top-level KVs.
	if idx := findTopLevelKV(d.nodes, segs); idx >= 0 {
		d.nodes = append(d.nodes[:idx], d.nodes[idx+1:]...)
		return true
	}

	// Check inside tables.
	return d.deleteFromTables(segs)
}

func findTopLevelKV(nodes []Node, segs []string) int {
	for i, n := range nodes {
		if kv, ok := n.(*KeyValue); ok {
			if matchKeyParts(kv.keyParts, segs) {
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
		for _, n := range d.nodes {
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
		if matchKeyParts(t.headerParts, tableSegs) {
			return deleteFromEntries(&t.entries, keySegs)
		}
	case *ArrayOfTables:
		if matchKeyParts(t.headerParts, tableSegs) {
			return deleteFromEntries(&t.entries, keySegs)
		}
	}
	return false
}

// DeleteTable removes the first TableNode matching the header path.
// Returns true if a table was found and removed.
func (d *Document) DeleteTable(path string) bool {
	segs := parseDottedPath(path)
	for i, n := range d.nodes {
		if t, ok := n.(*TableNode); ok {
			if matchKeyParts(t.headerParts, segs) {
				d.nodes = append(d.nodes[:i], d.nodes[i+1:]...)
				return true
			}
		}
	}
	return false
}

// Append adds a node to the end of the document's top-level nodes.
// The node must be a *KeyValue, *TableNode, *ArrayOfTables, *CommentNode,
// or *WhitespaceNode.
// Returns an error if the node would create an invalid document
// (e.g., duplicate keys, duplicate tables, or structural conflicts).
// Comment and whitespace nodes skip structural validation.
func (d *Document) Append(node Node) error {
	if err := validateDocumentNode(node); err != nil {
		return err
	}
	// Trivia nodes don't affect TOML structure; skip validation.
	if isTriviaNode(node) {
		d.nodes = append(d.nodes, node)
		setNodeParent(node, d)
		return nil
	}
	// Tentatively add.
	d.nodes = append(d.nodes, node)
	setNodeParent(node, d)
	// Full structural validation.
	if err := d.Validate(); err != nil {
		// Rollback.
		d.nodes = d.nodes[:len(d.nodes)-1]
		setNodeParent(node, nil)
		return err
	}
	return nil
}

// InsertAt inserts a node at position i in the document's top-level nodes.
// If i is out of range, the node is appended.
// Returns an error if the node would create an invalid document.
// Comment and whitespace nodes skip structural validation.
func (d *Document) InsertAt(i int, node Node) error {
	if err := validateDocumentNode(node); err != nil {
		return err
	}
	if i < 0 {
		i = 0
	}
	if i >= len(d.nodes) {
		return d.Append(node)
	}
	// Trivia nodes don't affect TOML structure; skip validation.
	if isTriviaNode(node) {
		d.nodes = append(d.nodes[:i], append([]Node{node}, d.nodes[i:]...)...)
		setNodeParent(node, d)
		return nil
	}
	// Tentatively insert.
	d.nodes = append(d.nodes[:i], append([]Node{node}, d.nodes[i:]...)...)
	setNodeParent(node, d)
	// Full structural validation.
	if err := d.Validate(); err != nil {
		// Rollback.
		d.nodes = append(d.nodes[:i], d.nodes[i+1:]...)
		setNodeParent(node, nil)
		return err
	}
	return nil
}

// isTriviaNode returns true if n is a *CommentNode or *WhitespaceNode.
func isTriviaNode(n Node) bool {
	switch n.(type) {
	case *CommentNode, *WhitespaceNode:
		return true
	}
	return false
}

// --- TableNode mutation ---

// Delete removes the first KeyValue matching the key from the table.
// Returns true if a key was found and removed.
func (t *TableNode) Delete(key string) bool {
	segs := parseDottedPath(key)
	return deleteFromEntries(&t.entries, segs)
}

// Append adds a key-value pair to the end of the table's entries.
// Returns an error if the key-value is nil, would create duplicate keys,
// or would create structural conflicts in the parent document.
func (t *TableNode) Append(kv *KeyValue) error {
	if kv == nil {
		return ErrNilEntry
	}
	// Tentatively add.
	t.entries = append(t.entries, kv)
	kv.setParent(t)
	// If attached to a document, run full validation.
	doc := findDocument(t)
	if doc != nil {
		if err := doc.Validate(); err != nil {
			// Rollback.
			t.entries = t.entries[:len(t.entries)-1]
			kv.setParent(nil)
			return err
		}
	} else {
		// Standalone table: local duplicate check only.
		if err := localDuplicateCheck(t.entries); err != nil {
			t.entries = t.entries[:len(t.entries)-1]
			kv.setParent(nil)
			return err
		}
	}
	return nil
}

// InsertAt inserts a key-value pair at position i in the table's entries.
// If i is out of range, the key-value is appended.
// Returns an error if it would create duplicate keys or structural conflicts.
func (t *TableNode) InsertAt(i int, kv *KeyValue) error {
	if kv == nil {
		return ErrNilEntry
	}
	if i < 0 {
		i = 0
	}
	if i >= len(t.entries) {
		return t.Append(kv)
	}
	// Tentatively insert.
	t.entries = append(t.entries[:i], append([]Node{kv}, t.entries[i:]...)...)
	kv.setParent(t)
	doc := findDocument(t)
	if doc != nil {
		if err := doc.Validate(); err != nil {
			t.entries = append(t.entries[:i], t.entries[i+1:]...)
			kv.setParent(nil)
			return err
		}
	} else {
		if err := localDuplicateCheck(t.entries); err != nil {
			t.entries = append(t.entries[:i], t.entries[i+1:]...)
			kv.setParent(nil)
			return err
		}
	}
	return nil
}

// --- ArrayOfTables mutation ---

// Delete removes the first KeyValue matching the key from the array of tables.
// Returns true if a key was found and removed.
func (a *ArrayOfTables) Delete(key string) bool {
	segs := parseDottedPath(key)
	return deleteFromEntries(&a.entries, segs)
}

// Append adds a key-value pair to the end of the array-of-tables' entries.
// Returns an error if the key-value is nil, would create duplicate keys,
// or would create structural conflicts in the parent document.
func (a *ArrayOfTables) Append(kv *KeyValue) error {
	if kv == nil {
		return ErrNilEntry
	}
	a.entries = append(a.entries, kv)
	kv.setParent(a)
	doc := findDocument(a)
	if doc != nil {
		if err := doc.Validate(); err != nil {
			a.entries = a.entries[:len(a.entries)-1]
			kv.setParent(nil)
			return err
		}
	} else {
		if err := localDuplicateCheck(a.entries); err != nil {
			a.entries = a.entries[:len(a.entries)-1]
			kv.setParent(nil)
			return err
		}
	}
	return nil
}

func deleteFromEntries(entries *[]Node, segs []string) bool {
	for i, e := range *entries {
		if kv, ok := e.(*KeyValue); ok {
			if matchKeyParts(kv.keyParts, segs) {
				*entries = append((*entries)[:i], (*entries)[i+1:]...)
				return true
			}
		}
	}
	return false
}

// --- ArrayNode mutation ---

// Append adds an element to the end of the array.
// The element must be a valid TOML value node.
// The array's text representation is regenerated.
func (a *ArrayNode) Append(elem Node) error {
	if err := validateValueType(elem); err != nil {
		return err
	}
	a.elements = append(a.elements, elem)
	a.text = generateArrayText(a.elements)
	return nil
}

// Delete removes the element at index i from the array.
// Returns an error if the index is out of bounds.
// The array's text representation is regenerated.
func (a *ArrayNode) Delete(i int) error {
	if i < 0 || i >= len(a.elements) {
		return fmt.Errorf("%w: index %d (array has %d elements)", ErrIndexOutOfRange, i, len(a.elements))
	}
	a.elements = append(a.elements[:i], a.elements[i+1:]...)
	a.text = generateArrayText(a.elements)
	return nil
}

// --- InlineTableNode mutation ---

// Append adds a key-value entry to the end of the inline table.
// Returns an error if the entry is nil or would create duplicate keys.
// The inline table's text representation is regenerated.
func (n *InlineTableNode) Append(kv *KeyValue) error {
	if kv == nil {
		return ErrNilEntry
	}
	path := keyPartsToPath(kv.keyParts)
	for _, existing := range n.entries {
		if keyPartsToPath(existing.keyParts) == path {
			return fmt.Errorf("%w: %q in inline table", ErrDuplicateKey, path)
		}
	}
	// Check dotted key conflicts.
	for i := 1; i < len(kv.keyParts); i++ {
		prefix := keyPartsToPath(kv.keyParts[:i])
		for _, existing := range n.entries {
			if keyPartsToPath(existing.keyParts) == prefix {
				return fmt.Errorf("%w: %q in inline table", ErrKeyConflict, prefix)
			}
		}
	}
	n.entries = append(n.entries, kv)
	kv.setParent(n)
	n.text = generateInlineTableText(n.entries)
	return nil
}

// Delete removes the first entry matching the key from the inline table.
// Returns true if a key was found and removed.
// The inline table's text representation is regenerated.
func (n *InlineTableNode) Delete(key string) bool {
	segs := parseDottedPath(key)
	for i, kv := range n.entries {
		if matchKeyParts(kv.keyParts, segs) {
			n.entries = append(n.entries[:i], n.entries[i+1:]...)
			n.text = generateInlineTableText(n.entries)
			return true
		}
	}
	return false
}

// --- Convenience constructors ---

// NewComment creates a CommentNode with the given text.
// The text should be the full comment including the leading "#".
// Returns an error if the text contains newlines or control characters
// other than tab.
func NewComment(text string) (*CommentNode, error) {
	for _, r := range text {
		if r == '\n' || r == '\r' {
			return nil, ErrCommentNewline
		}
		if r < 0x09 || (r > 0x0A && r < 0x0D) || (r > 0x0D && r < 0x20) || r == 0x7F {
			return nil, fmt.Errorf("%w: U+%04X", ErrCommentControl, r)
		}
	}
	return &CommentNode{leafNode: newLeaf(NodeComment, text)}, nil
}

// NewWhitespace creates a WhitespaceNode from the given string.
// The string must contain only spaces, tabs, newlines (\n), or carriage
// returns (\r).
func NewWhitespace(text string) (*WhitespaceNode, error) {
	for _, c := range text {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return nil, fmt.Errorf("%w: %q", ErrInvalidWsChar, c)
		}
	}
	return &WhitespaceNode{leafNode: newLeaf(NodeWhitespace, text)}, nil
}

// --- Document convenience methods ---

// AppendComment appends a "# text" comment followed by a newline to the
// document. The text parameter is the comment content without the leading
// "# ".
func (d *Document) AppendComment(text string) error {
	cn, err := NewComment("# " + text)
	if err != nil {
		return err
	}
	if err := d.Append(cn); err != nil {
		return err
	}
	ws, _ := NewWhitespace("\n")
	return d.Append(ws)
}

// AppendBlankLine appends a blank line ("\n") to the document.
func (d *Document) AppendBlankLine() error {
	ws, _ := NewWhitespace("\n")
	return d.Append(ws)
}

// --- TableNode convenience methods ---

// AppendComment appends a "# text" comment followed by a newline to the
// table's entries.
func (t *TableNode) AppendComment(text string) error {
	cn, err := NewComment("# " + text)
	if err != nil {
		return err
	}
	t.addEntry(cn)
	ws, _ := NewWhitespace("\n")
	t.addEntry(ws)
	return nil
}

// AppendBlankLine appends a blank line ("\n") to the table's entries.
func (t *TableNode) AppendBlankLine() {
	ws, _ := NewWhitespace("\n")
	t.addEntry(ws)
}

// --- ArrayOfTables convenience methods ---

// AppendComment appends a "# text" comment followed by a newline to the
// array-of-tables' entries.
func (a *ArrayOfTables) AppendComment(text string) error {
	cn, err := NewComment("# " + text)
	if err != nil {
		return err
	}
	a.addEntry(cn)
	ws, _ := NewWhitespace("\n")
	a.addEntry(ws)
	return nil
}

// AppendBlankLine appends a blank line ("\n") to the array-of-tables'
// entries.
func (a *ArrayOfTables) AppendBlankLine() {
	ws, _ := NewWhitespace("\n")
	a.addEntry(ws)
}

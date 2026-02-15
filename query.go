package toml

import (
	"math"
	"strconv"
	"strings"
)

// --- Path helpers ---

func parseDottedPath(path string) []string {
	var segs []string
	i := 0
	for i < len(path) {
		i = skipPathWs(path, i)
		if i >= len(path) {
			break
		}
		var seg string
		seg, i = parsePathSegment(path, i)
		segs = append(segs, seg)
		i = skipPathWs(path, i)
		if i < len(path) && path[i] == '.' {
			i++
		}
	}
	return segs
}

func skipPathWs(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

func parsePathSegment(path string, i int) (string, int) {
	if i >= len(path) {
		return "", i
	}
	switch path[i] {
	case '"':
		return parsePathBasicString(path, i)
	case '\'':
		return parsePathLiteralString(path, i)
	default:
		return parsePathBareKey(path, i)
	}
}

func parsePathBasicString(path string, i int) (string, int) {
	i++ // skip opening "
	start := i
	for i < len(path) {
		if path[i] == '\\' && i+1 < len(path) {
			i += 2 // skip escape sequence
			continue
		}
		if path[i] == '"' {
			return parserProcessBasicEscapes(path[start:i]), i + 1
		}
		i++
	}
	// Unclosed quote — return what we have.
	return parserProcessBasicEscapes(path[start:]), i
}

func parsePathLiteralString(path string, i int) (string, int) {
	i++ // skip opening '
	start := i
	for i < len(path) {
		if path[i] == '\'' {
			return path[start:i], i + 1
		}
		i++
	}
	// Unclosed quote — return what we have.
	return path[start:], i
}

func parsePathBareKey(path string, i int) (string, int) {
	start := i
	for i < len(path) && isBareKeyChar(rune(path[i])) {
		i++
	}
	return path[start:i], i
}

func matchKeyParts(parts []KeyPart, segs []string) bool {
	if len(parts) != len(segs) {
		return false
	}
	for i, p := range parts {
		if p.Unquoted != segs[i] {
			return false
		}
	}
	return true
}

// --- Document query methods ---

// Get finds a KeyValue by dotted path (e.g. "server.host").
// It searches top-level key-values and entries within tables.
// Returns nil if no matching key is found.
func (d *Document) Get(path string) *KeyValue {
	segs := parseDottedPath(path)

	// Check top-level KVs for exact match and prefix match into inline tables.
	if kv := findInEntries(d.Nodes, segs); kv != nil {
		return kv
	}

	// Try table prefixes from longest to shortest.
	return d.getFromTables(segs)
}

func (d *Document) getFromTables(segs []string) *KeyValue {
	for prefixLen := len(segs) - 1; prefixLen >= 1; prefixLen-- {
		tableSegs := segs[:prefixLen]
		keySegs := segs[prefixLen:]
		for _, n := range d.Nodes {
			if kv := getFromTableNode(n, tableSegs, keySegs); kv != nil {
				return kv
			}
		}
	}
	return nil
}

func getFromTableNode(n Node, tableSegs, keySegs []string) *KeyValue {
	switch t := n.(type) {
	case *TableNode:
		if matchKeyParts(t.HeaderParts, tableSegs) {
			return findInEntries(t.Entries, keySegs)
		}
	case *ArrayOfTables:
		if matchKeyParts(t.HeaderParts, tableSegs) {
			return findInEntries(t.Entries, keySegs)
		}
	}
	return nil
}

// Table finds the first TableNode whose header matches the given dotted path.
// Returns nil if no matching table is found.
func (d *Document) Table(path string) *TableNode {
	segs := parseDottedPath(path)
	for _, n := range d.Nodes {
		if t, ok := n.(*TableNode); ok {
			if matchKeyParts(t.HeaderParts, segs) {
				return t
			}
		}
	}
	return nil
}

func findInEntries(entries []Node, segs []string) *KeyValue {
	for _, e := range entries {
		if kv, ok := e.(*KeyValue); ok {
			if matchKeyParts(kv.KeyParts, segs) {
				return kv
			}
		}
	}
	// Prefix match into inline tables.
	for _, e := range entries {
		if kv, ok := e.(*KeyValue); ok {
			n := len(kv.KeyParts)
			if n < len(segs) && matchKeyParts(kv.KeyParts, segs[:n]) {
				if it, ok := kv.Val.(*InlineTableNode); ok {
					if found := findInKVEntries(it.Entries, segs[n:]); found != nil {
						return found
					}
				}
			}
		}
	}
	return nil
}

func findInKVEntries(entries []*KeyValue, segs []string) *KeyValue {
	for _, kv := range entries {
		if matchKeyParts(kv.KeyParts, segs) {
			return kv
		}
	}
	// Prefix match into nested inline tables.
	for _, kv := range entries {
		n := len(kv.KeyParts)
		if n < len(segs) && matchKeyParts(kv.KeyParts, segs[:n]) {
			if it, ok := kv.Val.(*InlineTableNode); ok {
				if found := findInKVEntries(it.Entries, segs[n:]); found != nil {
					return found
				}
			}
		}
	}
	return nil
}

// --- TableNode query methods ---

// Get finds a KeyValue within the table's entries by dotted key path.
// Returns nil if no matching key is found.
func (t *TableNode) Get(key string) *KeyValue {
	segs := parseDottedPath(key)
	return findInEntries(t.Entries, segs)
}

// --- ArrayOfTables query methods ---

// Get finds a KeyValue within the array-of-tables' entries by dotted key path.
// Returns nil if no matching key is found.
func (a *ArrayOfTables) Get(key string) *KeyValue {
	segs := parseDottedPath(key)
	return findInEntries(a.Entries, segs)
}

// --- InlineTableNode query methods ---

// Get finds a KeyValue within the inline table's entries by dotted key path.
// Returns nil if no matching key is found.
func (n *InlineTableNode) Get(key string) *KeyValue {
	segs := parseDottedPath(key)
	return findInKVEntries(n.Entries, segs)
}

// --- Value extraction methods ---

// Value returns the unquoted, unescaped string content.
func (n *StringNode) Value() string {
	raw := n.text
	if len(raw) < 2 {
		return raw
	}
	if strings.HasPrefix(raw, `"""`) && len(raw) >= 6 {
		return unquoteMultiLineBasic(raw)
	}
	if strings.HasPrefix(raw, "'''") && len(raw) >= 6 {
		return unquoteMultiLineLiteral(raw)
	}
	if raw[0] == '\'' {
		return raw[1 : len(raw)-1]
	}
	return parserProcessBasicEscapes(raw[1 : len(raw)-1])
}

func unquoteMultiLineBasic(raw string) string {
	inner := raw[3 : len(raw)-3]
	inner = trimLeadingNewline(inner)
	return processMultiLineBasicEscapes(inner)
}

func unquoteMultiLineLiteral(raw string) string {
	inner := raw[3 : len(raw)-3]
	return trimLeadingNewline(inner)
}

func trimLeadingNewline(s string) string {
	if len(s) > 0 && s[0] == '\n' {
		return s[1:]
	}
	if strings.HasPrefix(s, "\r\n") {
		return s[2:]
	}
	return s
}

// processMultiLineBasicEscapes handles basic string escapes including
// line-ending backslashes in multi-line strings.
func processMultiLineBasicEscapes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		i++
		if i >= len(s) {
			b.WriteByte('\\')
			break
		}
		switch s[i] {
		case '\n':
			i = skipMultiLineWs(s, i)
		case '\r':
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
			i = skipMultiLineWs(s, i)
		case ' ', '\t':
			if hasNewlineAfterWhitespace(s, i) {
				i = skipMultiLineWs(s, i)
			} else {
				b.WriteByte('\\')
				b.WriteByte(s[i])
			}
		default:
			i--
			result := parserProcessSingleEscape(s, &i)
			b.WriteString(result)
		}
	}
	return b.String()
}

func skipMultiLineWs(s string, i int) int {
	for i+1 < len(s) && (s[i+1] == ' ' || s[i+1] == '\t' || s[i+1] == '\n' || s[i+1] == '\r') {
		i++
	}
	return i
}

func hasNewlineAfterWhitespace(s string, pos int) bool {
	i := pos
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i < len(s) && (s[i] == '\n' || s[i] == '\r')
}

// parserProcessSingleEscape processes a single escape sequence starting at
// the backslash position. It advances *pos past the escape.
//
//nolint:gocyclo
func parserProcessSingleEscape(s string, pos *int) string {
	i := *pos
	i++ // skip backslash
	if i >= len(s) {
		*pos = i - 1
		return `\`
	}
	switch s[i] {
	case 'b':
		*pos = i
		return "\b"
	case 't':
		*pos = i
		return "\t"
	case 'n':
		*pos = i
		return "\n"
	case 'f':
		*pos = i
		return "\f"
	case 'r':
		*pos = i
		return "\r"
	case '"':
		*pos = i
		return `"`
	case '\\':
		*pos = i
		return `\`
	case 'e':
		*pos = i
		return "\x1B"
	case 'x':
		return processHexEscape(s, i, 2, pos)
	case 'u':
		return processHexEscape(s, i, 4, pos)
	case 'U':
		return processHexEscape(s, i, 8, pos)
	default:
		*pos = i
		return `\` + string(s[i])
	}
}

func processHexEscape(s string, i, digits int, pos *int) string {
	if i+digits < len(s) {
		if n, err := strconv.ParseUint(s[i+1:i+1+digits], 16, 32); err == nil {
			*pos = i + digits
			return string(rune(n))
		}
	}
	*pos = i
	labels := map[int]string{2: `\x`, 4: `\u`, 8: `\U`}
	return labels[digits]
}

// Int parses the number as an int64.
// Returns an error if the number is a float.
func (n *NumberNode) Int() (int64, error) {
	clean := strings.ReplaceAll(n.text, "_", "")
	if isSpecialFloat(clean) {
		return 0, strconv.ErrSyntax
	}
	// Check prefix integers before float detection, since hex digits
	// contain 'e'/'E' which would falsely trigger float classification.
	switch {
	case strings.HasPrefix(clean, "0x"):
		return strconv.ParseInt(clean[2:], 16, 64)
	case strings.HasPrefix(clean, "0o"):
		return strconv.ParseInt(clean[2:], 8, 64)
	case strings.HasPrefix(clean, "0b"):
		return strconv.ParseInt(clean[2:], 2, 64)
	}
	if strings.ContainsAny(clean, ".eE") {
		return 0, strconv.ErrSyntax
	}
	clean = strings.TrimPrefix(clean, "+")
	return strconv.ParseInt(clean, 10, 64)
}

// Float parses the number as a float64.
// Also works on integers, converting them to float64.
func (n *NumberNode) Float() (float64, error) {
	clean := strings.ReplaceAll(n.text, "_", "")
	switch clean {
	case "inf", "+inf":
		return math.Inf(1), nil
	case "-inf":
		return math.Inf(-1), nil
	case "nan", "+nan", "-nan":
		return math.NaN(), nil
	}
	// Integer prefixes — convert to float.
	switch {
	case strings.HasPrefix(clean, "0x"):
		v, err := strconv.ParseInt(clean[2:], 16, 64)
		return float64(v), err
	case strings.HasPrefix(clean, "0o"):
		v, err := strconv.ParseInt(clean[2:], 8, 64)
		return float64(v), err
	case strings.HasPrefix(clean, "0b"):
		v, err := strconv.ParseInt(clean[2:], 2, 64)
		return float64(v), err
	}
	clean = strings.TrimPrefix(clean, "+")
	return strconv.ParseFloat(clean, 64)
}

// Value returns the boolean value (true or false).
func (n *BooleanNode) Value() bool {
	return n.text == "true"
}

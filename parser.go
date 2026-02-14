package toml

import (
	"fmt"
	"strconv"
	"strings"
)

// parser builds a hierarchical CST from a token stream.
type parser struct {
	lex    *lexer
	cur    Token
	source string
}

func newParser(source string) *parser {
	p := &parser{
		lex:    newLexer(source),
		source: source,
	}
	p.cur = p.lex.Next()
	return p
}

func (p *parser) advance() Token {
	prev := p.cur
	p.cur = p.lex.Next()
	return prev
}

func (p *parser) at(t TokenType) bool { return p.cur.Type == t }

func (p *parser) parseError(msg string) error {
	return &ParseError{
		Message: msg,
		Line:    p.cur.Line,
		Column:  p.cur.Col,
		Source:  p.source,
	}
}

func (p *parser) tokError(msg string, tok Token) error {
	return &ParseError{
		Message: msg,
		Line:    tok.Line,
		Column:  tok.Col,
		Source:  p.source,
	}
}

// tableTarget is something that can hold child entries.
type tableTarget interface {
	addEntry(Node)
}

func (p *parser) parse() (*Document, error) {
	doc := &Document{}
	var ct tableTarget // current table receiving entries

	for !p.at(TokEOF) {
		trivia, err := p.collectLeadingTrivia()
		if err != nil {
			return nil, err
		}

		if p.at(TokEOF) {
			p.attachOrphanTrivia(doc, ct, trivia)
			break
		}

		if p.at(TokLBracket) {
			node, err := p.parseTableOrArrayHeader(trivia)
			if err != nil {
				return nil, err
			}
			doc.Nodes = append(doc.Nodes, node)
			if t, ok := node.(tableTarget); ok {
				ct = t
			}
			continue
		}

		kv, err := p.parseKeyVal(trivia)
		if err != nil {
			return nil, err
		}
		if err := p.addTrailingTrivia(kv); err != nil {
			return nil, err
		}

		if ct != nil {
			kv.setParent(nil) // parent will be the table
			ct.addEntry(kv)
		} else {
			kv.setParent(doc)
			doc.Nodes = append(doc.Nodes, kv)
		}
	}

	return doc, nil
}

// addEntry methods for table types.
func (t *TableNode) addEntry(n Node)     { t.Entries = append(t.Entries, n) }
func (a *ArrayOfTables) addEntry(n Node) { a.Entries = append(a.Entries, n) }

func (p *parser) attachOrphanTrivia(doc *Document, ct tableTarget, trivia []Node) {
	if len(trivia) == 0 {
		return
	}
	if attachTriviaToLast(doc, trivia) {
		return
	}
	for _, t := range trivia {
		if ct != nil {
			ct.addEntry(t)
		} else {
			doc.Nodes = append(doc.Nodes, t)
		}
	}
}

func attachTriviaToLast(doc *Document, trivia []Node) bool {
	if len(doc.Nodes) == 0 {
		return false
	}
	last := doc.Nodes[len(doc.Nodes)-1]
	switch v := last.(type) {
	case *TableNode:
		if kv := lastKV(v.Entries); kv != nil {
			kv.TrailingTrivia = append(kv.TrailingTrivia, trivia...)
			return true
		}
	case *ArrayOfTables:
		if kv := lastKV(v.Entries); kv != nil {
			kv.TrailingTrivia = append(kv.TrailingTrivia, trivia...)
			return true
		}
	case *KeyValue:
		v.TrailingTrivia = append(v.TrailingTrivia, trivia...)
		return true
	}
	return false
}

func lastKV(entries []Node) *KeyValue {
	if len(entries) == 0 {
		return nil
	}
	kv, _ := entries[len(entries)-1].(*KeyValue)
	return kv
}

// collectLeadingTrivia gathers whitespace, newlines, and comments.
func (p *parser) collectLeadingTrivia() ([]Node, error) {
	var nodes []Node
	for p.at(TokWhitespace) || p.at(TokNewline) || p.at(TokComment) {
		tok := p.advance()
		switch tok.Type { //nolint:exhaustive
		case TokComment:
			if msg := validateCommentText(tok.Text); msg != "" {
				return nil, p.tokError(msg, tok)
			}
			nodes = append(nodes, &CommentNode{leafNode: newLeaf(NodeComment, tok.Text)})
		default:
			nodes = append(nodes, &WhitespaceNode{leafNode: newLeaf(NodeWhitespace, tok.Text)})
		}
	}
	return nodes, nil
}

// addTrailingTrivia collects whitespace and comment after a value on the same line.
// It also enforces that a newline or EOF follows.
func (p *parser) addTrailingTrivia(kv *KeyValue) error {
	if p.at(TokWhitespace) {
		tok := p.advance()
		kv.TrailingTrivia = append(kv.TrailingTrivia,
			&WhitespaceNode{leafNode: newLeaf(NodeWhitespace, tok.Text)})
	}
	if p.at(TokComment) {
		tok := p.advance()
		if msg := validateCommentText(tok.Text); msg != "" {
			return p.tokError(msg, tok)
		}
		kv.TrailingTrivia = append(kv.TrailingTrivia,
			&CommentNode{leafNode: newLeaf(NodeComment, tok.Text)})
	}
	if p.at(TokNewline) {
		tok := p.advance()
		kv.Newline = tok.Text
		return nil
	}
	if p.at(TokEOF) {
		return nil
	}
	return p.parseError("expected newline or end of file after value")
}

// parseTableOrArrayHeader handles [ and [[ disambiguation.
func (p *parser) parseTableOrArrayHeader(trivia []Node) (Node, error) {
	headerLine, headerCol := p.cur.Line, p.cur.Col
	p.advance() // first [

	// Check for [[ (array of tables)
	if p.at(TokLBracket) {
		p.advance() // second [
		return p.parseArrayOfTablesBody(trivia, headerLine, headerCol)
	}

	return p.parseTableHeaderBody(trivia, headerLine, headerCol)
}

func (p *parser) parseTableHeaderBody(trivia []Node, hdrLine, hdrCol int) (*TableNode, error) {
	rawHeader, parts, err := p.parseKeyInHeader()
	if err != nil {
		return nil, err
	}

	if !p.at(TokRBracket) {
		return nil, p.parseError("expected ']' to close table header")
	}
	p.advance()

	trailing, nl, err2 := p.collectHeaderTrailing()
	if err2 != nil {
		return nil, err2
	}

	return &TableNode{
		baseNode:       baseNode{nodeType: NodeTable, line: hdrLine, col: hdrCol},
		LeadingTrivia:  trivia,
		RawHeader:      rawHeader,
		HeaderParts:    parts,
		TrailingTrivia: trailing,
		Newline:        nl,
	}, nil
}

func (p *parser) parseArrayOfTablesBody(trivia []Node, hdrLine, hdrCol int) (*ArrayOfTables, error) {
	rawHeader, parts, err := p.parseKeyInHeader()
	if err != nil {
		return nil, err
	}

	if !p.at(TokRBracket) {
		return nil, p.parseError("expected ']]' to close array of tables header")
	}
	p.advance()
	if !p.at(TokRBracket) {
		return nil, p.parseError("expected ']]' to close array of tables header")
	}
	p.advance()

	trailing, nl, err2 := p.collectHeaderTrailing()
	if err2 != nil {
		return nil, err2
	}

	return &ArrayOfTables{
		baseNode:       baseNode{nodeType: NodeArrayOfTables, line: hdrLine, col: hdrCol},
		LeadingTrivia:  trivia,
		RawHeader:      rawHeader,
		HeaderParts:    parts,
		TrailingTrivia: trailing,
		Newline:        nl,
	}, nil
}

func (p *parser) collectHeaderTrailing() ([]Node, string, error) {
	var nodes []Node
	if p.at(TokWhitespace) {
		tok := p.advance()
		nodes = append(nodes, &WhitespaceNode{leafNode: newLeaf(NodeWhitespace, tok.Text)})
	}
	if p.at(TokComment) {
		tok := p.advance()
		if msg := validateCommentText(tok.Text); msg != "" {
			return nil, "", p.tokError(msg, tok)
		}
		nodes = append(nodes, &CommentNode{leafNode: newLeaf(NodeComment, tok.Text)})
	}
	nl := ""
	if p.at(TokNewline) {
		tok := p.advance()
		nl = tok.Text
	} else if !p.at(TokEOF) {
		return nil, "", p.parseError("expected newline or end of file after table header")
	}
	return nodes, nl, nil
}

// parseKeyInHeader parses a key inside [ ] or [[ ]], returning raw text and parts.
func (p *parser) parseKeyInHeader() (string, []KeyPart, error) {
	var raw strings.Builder

	if p.at(TokWhitespace) {
		raw.WriteString(p.cur.Text)
		p.advance()
	}

	parts, keyRaw, err := p.parseKey()
	if err != nil {
		return "", nil, err
	}
	raw.WriteString(keyRaw)

	if p.at(TokWhitespace) {
		raw.WriteString(p.cur.Text)
		p.advance()
	}

	return raw.String(), parts, nil
}

// parseKey parses a simple or dotted key.
func (p *parser) parseKey() ([]KeyPart, string, error) {
	var parts []KeyPart
	var raw strings.Builder

	part, err := p.parseSimpleKey()
	if err != nil {
		return nil, "", err
	}
	raw.WriteString(part.Text)
	parts = append(parts, part)

	for p.at(TokDot) || (p.at(TokWhitespace) && p.lex.peekForDot()) {
		dotBefore := ""
		if p.at(TokWhitespace) {
			dotBefore = p.cur.Text
			raw.WriteString(dotBefore)
			p.advance()
		}
		if !p.at(TokDot) {
			break
		}
		raw.WriteString(".")
		p.advance()

		dotAfter := ""
		if p.at(TokWhitespace) {
			dotAfter = p.cur.Text
			raw.WriteString(dotAfter)
			p.advance()
		}

		part, err = p.parseSimpleKey()
		if err != nil {
			return nil, "", err
		}
		part.DotBefore = dotBefore
		part.DotAfter = dotAfter
		raw.WriteString(part.Text)
		parts = append(parts, part)
	}

	return parts, raw.String(), nil
}

func (p *parser) parseSimpleKey() (KeyPart, error) {
	switch p.cur.Type { //nolint:exhaustive
	case TokBareKey:
		tok := p.advance()
		for _, r := range tok.Text {
			if !isBareKeyChar(r) {
				return KeyPart{}, &ParseError{
					Message: fmt.Sprintf("invalid character %q in bare key %q", r, tok.Text),
					Line:    tok.Line,
					Column:  tok.Col,
					Source:  p.source,
				}
			}
		}
		return KeyPart{Text: tok.Text, Unquoted: tok.Text}, nil
	case TokBoolean, TokInteger, TokFloat, TokDateTime:
		tok := p.advance()
		return KeyPart{Text: tok.Text, Unquoted: tok.Text}, nil
	case TokBasicString:
		tok := p.advance()
		if msg := validateStringText(tok.Text); msg != "" {
			return KeyPart{}, p.tokError(msg, tok)
		}
		return KeyPart{Text: tok.Text, Unquoted: unquoteBasicStr(tok.Text), IsQuoted: true}, nil
	case TokLiteralString:
		tok := p.advance()
		if msg := validateStringText(tok.Text); msg != "" {
			return KeyPart{}, p.tokError(msg, tok)
		}
		return KeyPart{Text: tok.Text, Unquoted: unquoteLiteralStr(tok.Text), IsQuoted: true}, nil
	default:
		return KeyPart{}, p.parseError("expected key")
	}
}

func isBareKeyChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '-' || r == '_'
}

func (p *parser) parseKeyVal(trivia []Node) (*KeyValue, error) {
	kvLine, kvCol := p.cur.Line, p.cur.Col
	parts, rawKey, err := p.parseKey()
	if err != nil {
		return nil, err
	}

	preEq := ""
	if p.at(TokWhitespace) {
		preEq = p.cur.Text
		p.advance()
	}

	if !p.at(TokEquals) {
		return nil, p.parseError("expected '='")
	}
	p.lex.valueMode = true // switch to value context so . is part of floats
	p.advance()

	postEq := ""
	if p.at(TokWhitespace) {
		postEq = p.cur.Text
		p.advance()
	}

	val, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.lex.valueMode = false // back to key context

	return &KeyValue{
		baseNode:      baseNode{nodeType: NodeKeyValue, line: kvLine, col: kvCol},
		LeadingTrivia: trivia,
		KeyParts:      parts,
		RawKey:        rawKey,
		PreEq:         preEq,
		PostEq:        postEq,
		Val:           val,
		RawVal:        val.Text(),
	}, nil
}

// parseValue parses a TOML value.
func (p *parser) parseValue() (Node, error) {
	switch p.cur.Type { //nolint:exhaustive
	case TokBasicString, TokMultiLineBasicStr, TokLiteralString, TokMultiLineLiteralStr:
		return p.parseStringValue()
	case TokInteger, TokFloat:
		return p.parseNumberValue()
	case TokBoolean:
		tok := p.advance()
		return &BooleanNode{leafNode: newLeaf(NodeBoolean, tok.Text)}, nil
	case TokDateTime:
		return p.parseDateTimeValue()
	case TokLBracket:
		return p.parseArray()
	case TokLBrace:
		return p.parseInlineTable()
	default:
		return nil, p.parseError("expected value")
	}
}

func (p *parser) parseStringValue() (Node, error) {
	tok := p.advance()
	if msg := validateStringText(tok.Text); msg != "" {
		return nil, p.tokError(msg, tok)
	}
	return &StringNode{leafNode: newLeaf(NodeString, tok.Text)}, nil
}

func (p *parser) parseNumberValue() (Node, error) {
	tok := p.advance()
	if msg := validateNumberText(tok.Text); msg != "" {
		return nil, p.tokError(msg, tok)
	}
	return &NumberNode{leafNode: newLeaf(NodeNumber, tok.Text)}, nil
}

func (p *parser) parseDateTimeValue() (Node, error) {
	tok := p.advance()
	if msg := validateDateTimeText(tok.Text); msg != "" {
		return nil, p.tokError(msg, tok)
	}
	return &DateTimeNode{leafNode: newLeaf(NodeDateTime, tok.Text)}, nil
}

func (p *parser) parseArray() (Node, error) {
	startPos := p.cur.Pos
	p.advance() // [

	var elements []Node
	p.skipWsCommentNewline()

	for !p.at(TokRBracket) && !p.at(TokEOF) {
		p.lex.valueMode = true // array elements are values
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		elements = append(elements, val)
		p.lex.valueMode = true // restore after parseValue (inline table may unset it)
		p.skipWsCommentNewline()

		if p.at(TokComma) {
			p.advance()
			p.skipWsCommentNewline()
		} else if !p.at(TokRBracket) {
			return nil, p.parseError("expected ',' or ']' in array")
		}
	}

	if !p.at(TokRBracket) {
		return nil, p.parseError("expected ']' to close array")
	}
	closeTok := p.advance()
	endPos := closeTok.Pos + len(closeTok.Text)

	return &ArrayNode{
		baseNode: baseNode{nodeType: NodeArray},
		Elements: elements,
		text:     p.source[startPos:endPos],
	}, nil
}

func (p *parser) parseInlineTable() (Node, error) {
	startPos := p.cur.Pos
	p.lex.valueMode = false // keys inside inline table
	p.advance()             // {

	var entries []*KeyValue
	p.skipWsCommentNewline()

	for !p.at(TokRBrace) && !p.at(TokEOF) {
		kv, err := p.parseKeyVal(nil)
		if err != nil {
			return nil, err
		}
		entries = append(entries, kv)
		p.skipWsCommentNewline()

		if p.at(TokComma) {
			p.advance()
			p.skipWsCommentNewline()
		} else if !p.at(TokRBrace) {
			return nil, p.parseError("expected ',' or '}' in inline table")
		}
	}

	if !p.at(TokRBrace) {
		return nil, p.parseError("expected '}' to close inline table")
	}
	closeTok := p.advance()
	endPos := closeTok.Pos + len(closeTok.Text)

	return &InlineTableNode{
		baseNode: baseNode{nodeType: NodeInlineTable},
		Entries:  entries,
		text:     p.source[startPos:endPos],
	}, nil
}

func (p *parser) skipWsCommentNewline() {
	for p.at(TokWhitespace) || p.at(TokComment) || p.at(TokNewline) {
		p.advance()
	}
}

func unquoteBasicStr(s string) string {
	if len(s) < 2 {
		return s
	}
	return parserProcessBasicEscapes(s[1 : len(s)-1])
}

func unquoteLiteralStr(s string) string {
	if len(s) < 2 {
		return s
	}
	return s[1 : len(s)-1]
}

//nolint:gocyclo
func parserProcessBasicEscapes(s string) string {
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
		case 'b':
			b.WriteByte('\b')
		case 't':
			b.WriteByte('\t')
		case 'n':
			b.WriteByte('\n')
		case 'f':
			b.WriteByte('\f')
		case 'r':
			b.WriteByte('\r')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		case 'e':
			b.WriteByte(0x1B)
		case 'x':
			if i+2 < len(s) {
				if n, err := strconv.ParseUint(s[i+1:i+3], 16, 32); err == nil {
					b.WriteRune(rune(n))
					i += 2
					continue
				}
			}
			b.WriteString(`\x`)
		case 'u':
			if i+4 < len(s) {
				if n, err := strconv.ParseUint(s[i+1:i+5], 16, 32); err == nil {
					b.WriteRune(rune(n))
					i += 4
					continue
				}
			}
			b.WriteString(`\u`)
		case 'U':
			if i+8 < len(s) {
				if n, err := strconv.ParseUint(s[i+1:i+9], 16, 32); err == nil {
					b.WriteRune(rune(n))
					i += 8
					continue
				}
			}
			b.WriteString(`\U`)
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// suppress unused import errors.
var _ = fmt.Sprintf

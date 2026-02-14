package toml

import "strings"

// TokenType identifies lexer token kinds.
type TokenType int

const (
	TokError TokenType = iota - 1

	TokEOF TokenType = iota
	TokNewline
	TokWhitespace
	TokComment

	TokEquals
	TokDot
	TokComma
	TokLBracket
	TokRBracket
	TokLBrace
	TokRBrace

	TokBareKey
	TokBasicString
	TokMultiLineBasicStr
	TokLiteralString
	TokMultiLineLiteralStr
	TokInteger
	TokFloat
	TokBoolean
	TokDateTime
)

// Token is a lexer token with position.
type Token struct {
	Type TokenType
	Text string
	Pos  int // byte offset in source
	Line int // 1-indexed
	Col  int // 1-indexed
}

// lexer scans TOML source into tokens. It always emits single brackets
// (never [[/]]); the parser handles array-of-tables disambiguation.
type lexer struct {
	src       string
	pos       int
	line      int
	col       int
	valueMode bool // when true, dot is part of numeric tokens (value context)
}

func newLexer(src string) *lexer {
	return &lexer{src: src, pos: 0, line: 1, col: 1}
}

func (l *lexer) atEnd() bool { return l.pos >= len(l.src) }

func (l *lexer) peek() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *lexer) advance() {
	if l.pos >= len(l.src) {
		return
	}
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
}

func (l *lexer) makeToken(typ TokenType, start, startLine, startCol int) Token {
	return Token{Type: typ, Text: l.src[start:l.pos], Pos: start, Line: startLine, Col: startCol}
}

func (l *lexer) errToken(start, startLine, startCol int) Token {
	return Token{Type: TokError, Text: l.src[start:l.pos], Pos: start, Line: startLine, Col: startCol}
}

// Next returns the next token.
//
//nolint:gocyclo
func (l *lexer) Next() Token {
	if l.atEnd() {
		return Token{Type: TokEOF, Pos: l.pos, Line: l.line, Col: l.col}
	}

	ch := l.peek()
	sLine, sCol, sPos := l.line, l.col, l.pos

	switch {
	case ch == '\n' || (ch == '\r' && l.peekNext() == '\n'):
		return l.scanNewline()
	case ch == ' ' || ch == '\t':
		return l.scanWhitespace()
	case ch == '#':
		return l.scanComment()
	case ch == '=':
		l.advance()
		return l.makeToken(TokEquals, sPos, sLine, sCol)
	case ch == '.':
		l.advance()
		return l.makeToken(TokDot, sPos, sLine, sCol)
	case ch == ',':
		l.advance()
		return l.makeToken(TokComma, sPos, sLine, sCol)
	case ch == '[':
		l.advance()
		return l.makeToken(TokLBracket, sPos, sLine, sCol)
	case ch == ']':
		l.advance()
		return l.makeToken(TokRBracket, sPos, sLine, sCol)
	case ch == '{':
		l.advance()
		return l.makeToken(TokLBrace, sPos, sLine, sCol)
	case ch == '}':
		l.advance()
		return l.makeToken(TokRBrace, sPos, sLine, sCol)
	case ch == '"':
		return l.scanBasicStringStart()
	case ch == '\'':
		return l.scanLiteralStringStart()
	default:
		return l.scanBareOrValue()
	}
}

func (l *lexer) peekNext() byte {
	p := l.pos + 1
	if p >= len(l.src) {
		return 0
	}
	return l.src[p]
}

func (l *lexer) scanNewline() Token {
	sPos, sLine, sCol := l.pos, l.line, l.col
	if l.peek() == '\r' {
		l.advance()
	}
	l.advance() // \n
	return l.makeToken(TokNewline, sPos, sLine, sCol)
}

func (l *lexer) scanWhitespace() Token {
	sPos, sLine, sCol := l.pos, l.line, l.col
	for !l.atEnd() && (l.peek() == ' ' || l.peek() == '\t') {
		l.advance()
	}
	return l.makeToken(TokWhitespace, sPos, sLine, sCol)
}

func (l *lexer) scanComment() Token {
	sPos, sLine, sCol := l.pos, l.line, l.col
	for !l.atEnd() && l.peek() != '\n' && l.peek() != '\r' {
		l.advance()
	}
	return l.makeToken(TokComment, sPos, sLine, sCol)
}

func (l *lexer) scanBasicStringStart() Token {
	sPos, sLine, sCol := l.pos, l.line, l.col
	l.advance() // first "
	if l.peek() == '"' && l.peekNext() == '"' {
		l.advance() // second "
		l.advance() // third "
		return l.scanMultiLineBasicStr(sPos, sLine, sCol)
	}
	return l.scanBasicString(sPos, sLine, sCol)
}

func (l *lexer) scanBasicString(sPos, sLine, sCol int) Token {
	for !l.atEnd() {
		ch := l.peek()
		if ch == '\n' || ch == '\r' {
			return l.errToken(sPos, sLine, sCol)
		}
		if ch == '\\' {
			l.advance()
			if !l.atEnd() {
				l.advance()
			}
			continue
		}
		if ch == '"' {
			l.advance()
			return l.makeToken(TokBasicString, sPos, sLine, sCol)
		}
		l.advance()
	}
	return l.errToken(sPos, sLine, sCol)
}

func (l *lexer) scanMultiLineBasicStr(sPos, sLine, sCol int) Token {
	for !l.atEnd() {
		ch := l.peek()
		if ch == '\\' {
			l.advance()
			if !l.atEnd() {
				l.advance()
			}
			continue
		}
		if ch == '"' {
			count := 0
			for !l.atEnd() && l.peek() == '"' && count < 5 {
				l.advance()
				count++
			}
			if count >= 3 {
				return l.makeToken(TokMultiLineBasicStr, sPos, sLine, sCol)
			}
			continue
		}
		l.advance()
	}
	return l.errToken(sPos, sLine, sCol)
}

func (l *lexer) scanLiteralStringStart() Token {
	sPos, sLine, sCol := l.pos, l.line, l.col
	l.advance() // first '
	if l.peek() == '\'' && l.peekNext() == '\'' {
		l.advance() // second '
		l.advance() // third '
		return l.scanMultiLineLiteralStr(sPos, sLine, sCol)
	}
	return l.scanLiteralString(sPos, sLine, sCol)
}

func (l *lexer) scanLiteralString(sPos, sLine, sCol int) Token {
	for !l.atEnd() {
		ch := l.peek()
		if ch == '\n' || ch == '\r' {
			return l.errToken(sPos, sLine, sCol)
		}
		if ch == '\'' {
			l.advance()
			return l.makeToken(TokLiteralString, sPos, sLine, sCol)
		}
		l.advance()
	}
	return l.errToken(sPos, sLine, sCol)
}

func (l *lexer) scanMultiLineLiteralStr(sPos, sLine, sCol int) Token {
	for !l.atEnd() {
		ch := l.peek()
		if ch == '\'' {
			count := 0
			for !l.atEnd() && l.peek() == '\'' && count < 5 {
				l.advance()
				count++
			}
			if count >= 3 {
				return l.makeToken(TokMultiLineLiteralStr, sPos, sLine, sCol)
			}
			continue
		}
		l.advance()
	}
	return l.errToken(sPos, sLine, sCol)
}

// scanBareOrValue scans bare keys, booleans, numbers, dates, and special floats.
func (l *lexer) scanBareOrValue() Token {
	sPos, sLine, sCol := l.pos, l.line, l.col

	// In numeric context (starts with digit or sign+digit), dot is part of the
	// token (floats/datetimes), not a key separator.
	numCtx := l.startsNumeric()

	for !l.atEnd() && !l.isTokenDelimiter(l.peek(), numCtx) {
		l.advance()
	}

	text := l.src[sPos:l.pos]
	if text == "" {
		l.advance()
		return l.errToken(sPos, sLine, sCol)
	}

	// Space-separated datetime: "1979-05-27 07:32:00Z"
	// If we scanned a date-like token and next char is space followed by time
	if numCtx && l.isDateLikePrefix(text) && l.peekSpaceTime() {
		l.advance() // consume space
		for !l.atEnd() && !l.isTokenDelimiter(l.peek(), true) {
			l.advance()
		}
		text = l.src[sPos:l.pos]
	}

	typ := classifyBareToken(text)
	return Token{Type: typ, Text: text, Pos: sPos, Line: sLine, Col: sCol}
}

func (l *lexer) startsNumeric() bool {
	if !l.valueMode {
		return false
	}
	ch := l.peek()
	if isDigit(ch) {
		return true
	}
	if (ch == '+' || ch == '-') && isDigit(l.peekNext()) {
		return true
	}
	return false
}

func (l *lexer) isTokenDelimiter(ch byte, numericContext bool) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', '#', '=', ',', '[', ']', '{', '}', '"', '\'':
		return true
	case '.':
		return !numericContext
	}
	return false
}

// classifyBareToken determines the token type for an unquoted token string.
func classifyBareToken(s string) TokenType {
	if s == "true" || s == "false" {
		return TokBoolean
	}
	if isSpecialFloat(s) {
		return TokFloat
	}
	if isDateTimeLikeToken(s) {
		return TokDateTime
	}
	if looksLikeNumber(s) {
		return classifyNumber(s)
	}
	return TokBareKey
}

func isSpecialFloat(s string) bool {
	switch s {
	case "inf", "+inf", "-inf", "nan", "+nan", "-nan":
		return true
	}
	return false
}

func isDateTimeLikeToken(s string) bool {
	if len(s) < 5 {
		return false
	}
	// Time-only: starts with a digit and contains ':'
	if isDigit(s[0]) && strings.ContainsRune(s, ':') {
		return true
	}
	// Date forms: digits followed by '-' with at least two '-' separators
	// This catches both well-formed (YYYY-MM-DD) and malformed dates
	// (e.g. 1987-7-05) so the decoder can reject them.
	if isDigit(s[0]) && strings.Count(s, "-") >= 2 {
		return true
	}
	return false
}

func looksLikeNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '+' || s[0] == '-' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	return isDigit(s[start]) || (s[start] == '0' && start+1 < len(s) && isBasePrefix(s[start+1]))
}

func isBasePrefix(ch byte) bool {
	return ch == 'x' || ch == 'o' || ch == 'b'
}

func classifyNumber(s string) TokenType {
	clean := s
	if len(clean) > 0 && (clean[0] == '+' || clean[0] == '-') {
		clean = clean[1:]
	}
	if len(clean) > 1 && clean[0] == '0' && isBasePrefix(clean[1]) {
		return TokInteger
	}
	if containsFloatChar(clean) {
		return TokFloat
	}
	return TokInteger
}

func containsFloatChar(s string) bool {
	for _, ch := range s {
		if ch == '.' || ch == 'e' || ch == 'E' {
			return true
		}
	}
	return false
}

func isDigit(ch byte) bool { return ch >= '0' && ch <= '9' }

// isDateLikePrefix checks if text looks like YYYY-MM-DD.
func (l *lexer) isDateLikePrefix(s string) bool {
	return len(s) == 10 && isDigit(s[0]) && s[4] == '-' && s[7] == '-'
}

// peekSpaceTime checks if the next chars are space followed by HH:MM.
func (l *lexer) peekSpaceTime() bool {
	if l.pos >= len(l.src) || l.src[l.pos] != ' ' {
		return false
	}
	if l.pos+1 >= len(l.src) || !isDigit(l.src[l.pos+1]) {
		return false
	}
	if l.pos+2 >= len(l.src) || !isDigit(l.src[l.pos+2]) {
		return false
	}
	if l.pos+3 >= len(l.src) || l.src[l.pos+3] != ':' {
		return false
	}
	return true
}

// peekForDot checks if the source at the current lexer position (past the
// already-lexed current token) has a dot, optionally preceded by whitespace.
func (l *lexer) peekForDot() bool {
	pos := l.pos
	for pos < len(l.src) && (l.src[pos] == ' ' || l.src[pos] == '\t') {
		pos++
	}
	return pos < len(l.src) && l.src[pos] == '.'
}

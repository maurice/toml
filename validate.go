package toml

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// --- UTF-8 validation ---

// validateUTF8 checks that data contains only valid UTF-8.
func validateUTF8(data []byte) string {
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			return fmt.Sprintf("invalid UTF-8 byte at position %d", i)
		}
		i += size
	}
	return ""
}

// --- Comment validation ---

// validateCommentText checks a comment for invalid chars.
func validateCommentText(s string) string {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return "invalid UTF-8 in comment"
		}
		if r != '\t' && isControlChar(r) {
			return fmt.Sprintf("control character U+%04X in comment", r)
		}
		i += size
	}
	return ""
}

func isControlChar(r rune) bool {
	return (r >= 0 && r <= 0x1F) || r == 0x7F
}

// --- String validation ---

// validateStringText validates a TOML string token (with quotes).
func validateStringText(raw string) string {
	if len(raw) < 2 {
		return "invalid string"
	}
	if strings.HasPrefix(raw, `"""`) {
		return validateMultiLineBasicStr(raw)
	}
	if strings.HasPrefix(raw, "'''") {
		return validateMultiLineLiteralStr(raw)
	}
	if raw[0] == '\'' {
		return validateLiteralContent(raw[1:len(raw)-1], false)
	}
	return validateBasicContent(raw[1:len(raw)-1], false)
}

func validateMultiLineBasicStr(raw string) string {
	inner := raw[3 : len(raw)-3]
	if len(inner) > 0 && inner[0] == '\n' {
		inner = inner[1:]
	} else if strings.HasPrefix(inner, "\r\n") {
		inner = inner[2:]
	}
	return validateBasicContent(inner, true)
}

func validateMultiLineLiteralStr(raw string) string {
	inner := raw[3 : len(raw)-3]
	if len(inner) > 0 && inner[0] == '\n' {
		inner = inner[1:]
	} else if strings.HasPrefix(inner, "\r\n") {
		inner = inner[2:]
	}
	return validateLiteralContent(inner, true)
}

func validateBasicContent(s string, multiline bool) string {
	for i := 0; i < len(s); {
		if s[i] == '\\' {
			i++
			if i >= len(s) {
				return "trailing backslash in string"
			}
			newI, msg := validateBasicEscape(s, i, multiline)
			if msg != "" {
				return msg
			}
			i = newI
			continue
		}
		if msg := checkBareCarriageReturn(s, i, multiline); msg != "" {
			return msg
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return "invalid UTF-8 in string"
		}
		if msg := checkStringControlChar(r, multiline); msg != "" {
			return msg
		}
		i += size
	}
	return ""
}

func checkBareCarriageReturn(s string, i int, multiline bool) string {
	if multiline && s[i] == '\r' && (i+1 >= len(s) || s[i+1] != '\n') {
		return "bare carriage return in multi-line string"
	}
	return ""
}

func checkStringControlChar(r rune, multiline bool) string {
	if r == '\t' {
		return ""
	}
	if isControlChar(r) {
		if multiline && (r == '\n' || r == '\r') {
			return ""
		}
		return fmt.Sprintf("control character U+%04X in string", r)
	}
	return ""
}

func validateBasicEscape(s string, i int, multiline bool) (int, string) {
	switch s[i] {
	case 'b', 't', 'n', 'f', 'r', '"', '\\', 'e':
		return i + 1, ""
	case 'x':
		return validateUnicodeEscape(s, i, 2)
	case 'u':
		return validateUnicodeEscape(s, i, 4)
	case 'U':
		return validateUnicodeEscape(s, i, 8)
	case '\n', '\r':
		if !multiline {
			return 0, "invalid escape sequence"
		}
		return skipLineEndingBackslash(s, i), ""
	case ' ', '\t':
		return validateWsBackslash(s, i, multiline)
	default:
		return 0, fmt.Sprintf("invalid escape sequence '\\%c'", s[i])
	}
}

func validateUnicodeEscape(s string, i, digits int) (int, string) {
	label := `\u`
	switch digits {
	case 2:
		label = `\x`
	case 8:
		label = `\U`
	}
	if i+digits >= len(s) {
		return 0, fmt.Sprintf("incomplete %s escape", label)
	}
	for j := 1; j <= digits; j++ {
		if !isHexDigit(s[i+j]) {
			return 0, fmt.Sprintf("invalid %s escape", label)
		}
	}
	n, _ := strconv.ParseUint(s[i+1:i+1+digits], 16, 32)
	if n >= 0xD800 && n <= 0xDFFF {
		return 0, fmt.Sprintf("invalid unicode scalar U+%04X", n)
	}
	if n > 0x10FFFF {
		return 0, fmt.Sprintf("unicode codepoint U+%04X out of range", n)
	}
	return i + 1 + digits, ""
}

func skipLineEndingBackslash(s string, i int) int {
	if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
		i++
	}
	i++
	for i < len(s) && isWhitespaceOrNewline(s[i]) {
		i++
	}
	return i
}

func isWhitespaceOrNewline(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func validateWsBackslash(s string, i int, multiline bool) (int, string) {
	if multiline && hasNewlineAfterWs(s, i) {
		return skipToNextNonWs(s, i) + 1, ""
	}
	return 0, fmt.Sprintf("invalid escape sequence '\\%c'", s[i])
}

func validateLiteralContent(s string, multiline bool) string {
	for i := 0; i < len(s); {
		if msg := checkBareCarriageReturn(s, i, multiline); msg != "" {
			return msg
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return "invalid UTF-8 in literal string"
		}
		if msg := checkLiteralControlChar(r, multiline); msg != "" {
			return msg
		}
		i += size
	}
	return ""
}

func checkLiteralControlChar(r rune, multiline bool) string {
	if r == '\t' {
		return ""
	}
	if isControlChar(r) {
		if multiline && (r == '\n' || r == '\r') {
			return ""
		}
		return fmt.Sprintf("control character U+%04X in literal string", r)
	}
	return ""
}

func hasNewlineAfterWs(s string, pos int) bool {
	i := pos
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i < len(s) && (s[i] == '\n' || s[i] == '\r')
}

func skipToNextNonWs(s string, pos int) int {
	i := pos
	for i < len(s) && isWhitespaceOrNewline(s[i]) {
		i++
	}
	return i - 1
}

// --- Number validation ---

// validateNumberText validates a TOML number token.
func validateNumberText(text string) string {
	raw := text
	clean := strings.ReplaceAll(raw, "_", "")

	if isSpecialFloat(clean) {
		return validateUnderscores(raw)
	}
	if hasUnsignedPrefix(clean) || hasSignedPrefix(clean) {
		return checkPrefixNumber(raw, clean)
	}
	if msg := checkDecimalLeadingZeros(raw, clean); msg != "" {
		return msg
	}
	if strings.ContainsAny(clean, ".eE") {
		return validateFloatText(raw, clean)
	}
	return validateDecimalDigits(raw, clean)
}

func checkPrefixNumber(raw, clean string) string {
	if hasUnsignedPrefix(clean) {
		return checkUnsignedPrefix(raw, clean)
	}
	if hasSignedPrefix(clean) {
		return fmt.Sprintf("sign not allowed on %s integer", clean[1:3])
	}
	return ""
}

func hasUnsignedPrefix(clean string) bool {
	if len(clean) <= 1 {
		return false
	}
	return clean[0] == '0' && (clean[1] == 'x' || clean[1] == 'o' || clean[1] == 'b')
}

func hasSignedPrefix(clean string) bool {
	if len(clean) <= 2 {
		return false
	}
	if clean[0] != '+' && clean[0] != '-' {
		return false
	}
	return clean[1] == '0' && (clean[2] == 'x' || clean[2] == 'o' || clean[2] == 'b')
}

func checkUnsignedPrefix(raw, clean string) string {
	switch clean[1] {
	case 'x':
		return validatePrefixIntBody(raw, clean, "0x", isHexDigit)
	case 'o':
		return validatePrefixIntBody(raw, clean, "0o", isOctDigit)
	case 'b':
		return validatePrefixIntBody(raw, clean, "0b", isBinDigit)
	}
	return ""
}

func checkDecimalLeadingZeros(raw, clean string) string {
	num := stripSign(clean)
	if len(num) <= 1 {
		return ""
	}
	if num[0] == '0' && num[1] != '.' && num[1] != 'e' && num[1] != 'E' {
		return fmt.Sprintf("leading zeros not allowed: %s", raw)
	}
	return ""
}

func stripSign(s string) string {
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		return s[1:]
	}
	return s
}

func validateDecimalDigits(raw, clean string) string {
	num := stripSign(clean)
	for i := 0; i < len(num); i++ {
		if !isDecDigit(num[i]) {
			return fmt.Sprintf("invalid character in integer: %s", raw)
		}
	}
	return validateUnderscores(raw)
}

func validatePrefixIntBody(raw, clean, prefix string, validDigit func(byte) bool) string {
	body := clean[len(prefix):]
	if len(body) == 0 {
		return fmt.Sprintf("incomplete %s integer: %s", prefix, raw)
	}
	for i := 0; i < len(body); i++ {
		if !validDigit(body[i]) {
			return fmt.Sprintf("invalid digit in %s integer: %s", prefix, raw)
		}
	}
	return validateUnderscoresInBody(raw, len(prefix))
}

func validateFloatText(raw, clean string) string {
	if strings.Count(clean, ".") > 1 {
		return fmt.Sprintf("multiple dots in float: %s", raw)
	}
	eCount := strings.Count(clean, "e") + strings.Count(clean, "E")
	if eCount > 1 {
		return fmt.Sprintf("multiple exponents in float: %s", raw)
	}
	if msg := validateFloatUnderscores(raw); msg != "" {
		return msg
	}
	return validateFloatParts(raw, clean)
}

func validateFloatParts(raw, clean string) string {
	num := stripSign(clean)
	dotIdx := strings.Index(num, ".")
	eIdx := strings.IndexAny(num, "eE")

	if dotIdx >= 0 && eIdx >= 0 && dotIdx > eIdx {
		return fmt.Sprintf("dot after exponent: %s", raw)
	}
	if dotIdx >= 0 {
		if msg := validateFloatDotParts(raw, num, dotIdx, eIdx); msg != "" {
			return msg
		}
	}
	if eIdx >= 0 {
		if msg := validateFloatExponent(raw, clean, dotIdx, eIdx); msg != "" {
			return msg
		}
	}
	return ""
}

func validateFloatDotParts(raw, num string, dotIdx, eIdx int) string {
	if dotIdx == 0 || dotIdx == len(num)-1 {
		return fmt.Sprintf("invalid float: %s", raw)
	}
	afterDot := num[dotIdx+1:]
	if eIdx >= 0 {
		eRel := eIdx - dotIdx - 1
		afterDot = afterDot[:eRel]
	}
	if len(afterDot) == 0 {
		return fmt.Sprintf("no digits after decimal point: %s", raw)
	}
	return ""
}

func validateFloatExponent(raw, clean string, dotIdx, eIdx int) string {
	eClean := strings.IndexAny(clean, "eE")
	after := clean[eClean+1:]
	if len(after) > 0 && (after[0] == '+' || after[0] == '-') {
		after = after[1:]
	}
	if len(after) == 0 {
		return fmt.Sprintf("no digits in exponent: %s", raw)
	}
	if dotIdx >= 0 && dotIdx == eIdx-1 {
		return fmt.Sprintf("no digits between dot and exponent: %s", raw)
	}
	return ""
}

func validateFloatUnderscores(raw string) string {
	if msg := checkUnderscoreAdjacent(raw); msg != "" {
		return msg
	}
	return validateUnderscores(raw)
}

func checkUnderscoreAdjacent(raw string) string {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '_' {
			continue
		}
		if msg := checkAdjacentChar(raw, i); msg != "" {
			return msg
		}
	}
	return ""
}

func isFloatSeparator(c byte) bool {
	return c == '.' || c == 'e' || c == 'E'
}

func checkAdjacentChar(raw string, i int) string {
	if i > 0 && isFloatSeparator(raw[i-1]) {
		return fmt.Sprintf("underscore after %c: %s", raw[i-1], raw)
	}
	if i+1 < len(raw) && isFloatSeparator(raw[i+1]) {
		return fmt.Sprintf("underscore before %c: %s", raw[i+1], raw)
	}
	return ""
}

func validateUnderscores(raw string) string {
	start := 0
	if len(raw) > 0 && (raw[0] == '+' || raw[0] == '-') {
		start = 1
	}
	if start >= len(raw) {
		return ""
	}
	return validateUnderscoresInBody(raw, start)
}

func validateUnderscoresInBody(s string, start int) string {
	body := s[start:]
	if len(body) == 0 {
		return ""
	}
	if body[0] == '_' {
		return fmt.Sprintf("leading underscore: %s", s)
	}
	if body[len(body)-1] == '_' {
		return fmt.Sprintf("trailing underscore: %s", s)
	}
	for i := 1; i < len(body); i++ {
		if body[i] == '_' && body[i-1] == '_' {
			return fmt.Sprintf("double underscore: %s", s)
		}
	}
	return ""
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isDecDigit(c byte) bool { return c >= '0' && c <= '9' }
func isOctDigit(c byte) bool { return c >= '0' && c <= '7' }
func isBinDigit(c byte) bool { return c == '0' || c == '1' }

// --- DateTime validation ---

var (
	dtDateRe   = `(\d{4})-(\d{2})-(\d{2})`
	dtTimeRe   = `(\d{2}):(\d{2})(?::(\d{2})(\.\d+)?)?`
	dtOffsetRe = `([Zz]|[+-]\d{2}:\d{2})`

	dtReOffsetDT  = regexp.MustCompile(`^` + dtDateRe + `[T t]` + dtTimeRe + dtOffsetRe + `$`)
	dtReLocalDT   = regexp.MustCompile(`^` + dtDateRe + `[T t]` + dtTimeRe + `$`)
	dtReLocalDate = regexp.MustCompile(`^` + dtDateRe + `$`)
	dtReLocalTime = regexp.MustCompile(`^` + dtTimeRe + `$`)
)

// validateDateTimeText validates a TOML datetime token.
func validateDateTimeText(text string) string {
	if dtReOffsetDT.MatchString(text) {
		return validateDateTimeParts(text, true)
	}
	if dtReLocalDT.MatchString(text) {
		return validateDateTimeParts(text, false)
	}
	if dtReLocalDate.MatchString(text) {
		return validateDateParts(text)
	}
	if dtReLocalTime.MatchString(text) {
		return validateTimeParts(text)
	}
	return fmt.Sprintf("invalid datetime format: %s", text)
}

func validateDateTimeParts(text string, hasOffset bool) string {
	sep := strings.IndexAny(text, "Tt ")
	if sep < 0 {
		return fmt.Sprintf("invalid datetime: %s", text)
	}
	datePart := text[:sep]
	timePart := text[sep+1:]

	if hasOffset {
		timePart = stripOffset(timePart, text)
	}

	if msg := validateDateParts(datePart); msg != "" {
		return msg
	}
	return validateTimeParts(timePart)
}

func stripOffset(timePart, full string) string {
	if idx := strings.IndexAny(timePart, "Zz"); idx >= 0 {
		return timePart[:idx]
	}
	if idx := strings.LastIndexAny(timePart, "+-"); idx > 0 {
		offsetStr := timePart[idx+1:]
		if msg := validateOffsetText(offsetStr, full); msg != "" {
			return timePart // just return; error will be caught by time validation.
		}
		return timePart[:idx]
	}
	return timePart
}

func validateOffsetText(offset, full string) string {
	parts := strings.Split(offset, ":")
	if len(parts) != 2 {
		return fmt.Sprintf("invalid offset format: %s", full)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Sprintf("invalid offset hour: %s", full)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Sprintf("invalid offset minute: %s", full)
	}
	if h > 23 {
		return fmt.Sprintf("offset hour out of range: %s", full)
	}
	if m > 59 {
		return fmt.Sprintf("offset minute out of range: %s", full)
	}
	return ""
}

func validateDateParts(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return fmt.Sprintf("invalid date: %s", s)
	}
	if msg := checkDateDigitCounts(parts, s); msg != "" {
		return msg
	}
	return checkDateRanges(parts, s)
}

func checkDateDigitCounts(parts []string, s string) string {
	if len(parts[0]) != 4 {
		return fmt.Sprintf("year must be 4 digits: %s", s)
	}
	if len(parts[1]) != 2 {
		return fmt.Sprintf("month must be 2 digits: %s", s)
	}
	if len(parts[2]) != 2 {
		return fmt.Sprintf("day must be 2 digits: %s", s)
	}
	return ""
}

func checkDateRanges(parts []string, s string) string {
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])

	if month < 1 || month > 12 {
		return fmt.Sprintf("month out of range: %s", s)
	}
	if day < 1 {
		return fmt.Sprintf("day out of range: %s", s)
	}

	daysInMonth := [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if isLeapYear(year) {
		daysInMonth[2] = 29
	}
	if day > daysInMonth[month] {
		return fmt.Sprintf("day %d out of range for month %d: %s", day, month, s)
	}
	return ""
}

func isLeapYear(y int) bool {
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}

func validateTimeParts(s string) string {
	frac := strings.Index(s, ".")
	main := s
	if frac >= 0 {
		fracPart := s[frac+1:]
		if len(fracPart) == 0 {
			return fmt.Sprintf("trailing dot in time: %s", s)
		}
		main = s[:frac]
	}
	parts := strings.Split(main, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return fmt.Sprintf("time must have HH:MM or HH:MM:SS: %s", s)
	}
	if msg := checkTimeDigitCounts(parts, s); msg != "" {
		return msg
	}
	return checkTimeRanges(parts, s)
}

func checkTimeDigitCounts(parts []string, s string) string {
	if len(parts[0]) != 2 {
		return fmt.Sprintf("hour must be 2 digits: %s", s)
	}
	if len(parts[1]) != 2 {
		return fmt.Sprintf("minute must be 2 digits: %s", s)
	}
	if len(parts) == 3 && len(parts[2]) != 2 {
		return fmt.Sprintf("second must be 2 digits: %s", s)
	}
	return ""
}

func checkTimeRanges(parts []string, s string) string {
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	if hour > 23 {
		return fmt.Sprintf("hour out of range: %s", s)
	}
	if minute > 59 {
		return fmt.Sprintf("minute out of range: %s", s)
	}
	if len(parts) == 3 {
		sec, _ := strconv.Atoi(parts[2])
		if sec > 60 {
			return fmt.Sprintf("second out of range: %s", s)
		}
	}
	return ""
}

// --- Semantic validation ---

// tableState tracks semantics for TOML table/key validation.
type tableState struct {
	explicitTables  map[string]bool
	dottedKeyTables map[string]bool
	implicitTables  map[string]bool
	inlinePaths     map[string]bool
	staticArrays    map[string]bool
	aotPaths        map[string]bool
	scalarPaths     map[string]bool
}

func newTableState() *tableState {
	return &tableState{
		explicitTables:  make(map[string]bool),
		dottedKeyTables: make(map[string]bool),
		implicitTables:  make(map[string]bool),
		inlinePaths:     make(map[string]bool),
		staticArrays:    make(map[string]bool),
		aotPaths:        make(map[string]bool),
		scalarPaths:     make(map[string]bool),
	}
}

type docValidator struct {
	source string
	state  *tableState
}

func validateDocument(doc *Document, source string) error {
	v := &docValidator{
		source: source,
		state:  newTableState(),
	}
	return v.validate(doc)
}

func (v *docValidator) validate(doc *Document) error {
	for _, n := range doc.Nodes {
		switch node := n.(type) {
		case *KeyValue:
			if err := v.checkKeyValue(nil, node); err != nil {
				return err
			}
		case *TableNode:
			if err := v.checkTable(node); err != nil {
				return err
			}
		case *ArrayOfTables:
			if err := v.checkAOT(node); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *docValidator) errorAt(msg string, line, col int) error {
	return &ParseError{
		Message: msg,
		Line:    line,
		Column:  col,
		Source:  v.source,
	}
}

func keyPartsToPath(parts []KeyPart) string {
	var sb strings.Builder
	for i, p := range parts {
		if i > 0 {
			sb.WriteByte('.')
		}
		// If the unquoted key contains a dot, wrap it in quotes to
		// distinguish "a"."b.c" (2 parts) from "a"."b"."c" (3 parts).
		if strings.ContainsRune(p.Unquoted, '.') {
			sb.WriteByte('"')
			sb.WriteString(p.Unquoted)
			sb.WriteByte('"')
		} else {
			sb.WriteString(p.Unquoted)
		}
	}
	return sb.String()
}

func buildFullPath(baseParts, keyParts []KeyPart) string {
	all := make([]KeyPart, 0, len(baseParts)+len(keyParts))
	all = append(all, baseParts...)
	all = append(all, keyParts...)
	return keyPartsToPath(all)
}

func (v *docValidator) checkTable(node *TableNode) error {
	path := keyPartsToPath(node.HeaderParts)

	if msg := v.checkTablePathConflicts(path); msg != "" {
		return v.errorAt(msg, node.line, node.col)
	}
	if msg := v.checkIntermediatePaths(node.HeaderParts, path); msg != "" {
		return v.errorAt(msg, node.line, node.col)
	}

	v.state.explicitTables[path] = true
	v.markParentImplicit(node.HeaderParts)

	for _, entry := range node.Entries {
		if kv, ok := entry.(*KeyValue); ok {
			if err := v.checkKeyValue(node.HeaderParts, kv); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *docValidator) checkTablePathConflicts(path string) string {
	ts := v.state
	if ts.explicitTables[path] {
		return fmt.Sprintf("duplicate table: [%s]", path)
	}
	if ts.aotPaths[path] {
		return fmt.Sprintf("cannot define table [%s] already defined as array of tables", path)
	}
	if ts.dottedKeyTables[path] {
		return fmt.Sprintf("cannot reopen table [%s] defined via dotted keys", path)
	}
	if ts.scalarPaths[path] {
		return fmt.Sprintf("cannot define table [%s], key already defined as a value", path)
	}
	if ts.inlinePaths[path] {
		return fmt.Sprintf("cannot extend inline table/array [%s]", path)
	}
	if ts.staticArrays[path] {
		return fmt.Sprintf("cannot extend static array [%s]", path)
	}
	return ""
}

func (v *docValidator) checkIntermediatePaths(parts []KeyPart, path string) string {
	ts := v.state
	for i := 1; i < len(parts); i++ {
		parentPath := keyPartsToPath(parts[:i])
		if ts.scalarPaths[parentPath] {
			return fmt.Sprintf("cannot define table [%s], key %q already a value", path, parentPath)
		}
		if ts.inlinePaths[parentPath] {
			return fmt.Sprintf("cannot extend inline table/array at %q", parentPath)
		}
		if ts.staticArrays[parentPath] {
			return fmt.Sprintf("cannot extend static array at %q", parentPath)
		}
	}
	return ""
}

func (v *docValidator) markParentImplicit(parts []KeyPart) {
	ts := v.state
	for i := 1; i < len(parts); i++ {
		parentPath := keyPartsToPath(parts[:i])
		if !ts.explicitTables[parentPath] && !ts.aotPaths[parentPath] {
			ts.implicitTables[parentPath] = true
		}
	}
}

func (v *docValidator) checkAOT(node *ArrayOfTables) error {
	path := keyPartsToPath(node.HeaderParts)

	if msg := v.checkAOTPathConflicts(path); msg != "" {
		return v.errorAt(msg, node.line, node.col)
	}
	if msg := v.checkIntermediatePathsAOT(node.HeaderParts, path); msg != "" {
		return v.errorAt(msg, node.line, node.col)
	}

	v.state.aotPaths[path] = true
	v.markParentImplicit(node.HeaderParts)
	v.clearSubScope(path)

	for _, entry := range node.Entries {
		if kv, ok := entry.(*KeyValue); ok {
			if err := v.checkKeyValue(node.HeaderParts, kv); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *docValidator) checkAOTPathConflicts(path string) string {
	ts := v.state
	if ts.explicitTables[path] {
		return fmt.Sprintf("cannot define array of tables [[%s]] already defined as table", path)
	}
	if ts.scalarPaths[path] {
		return fmt.Sprintf("cannot define array [[%s]], key already a value", path)
	}
	if ts.inlinePaths[path] {
		return fmt.Sprintf("cannot extend inline table/array [[%s]]", path)
	}
	if ts.staticArrays[path] {
		return fmt.Sprintf("cannot extend static array [[%s]]", path)
	}
	if ts.dottedKeyTables[path] {
		return fmt.Sprintf("cannot define array [[%s]], key defined via dotted keys", path)
	}
	if ts.implicitTables[path] && !ts.aotPaths[path] {
		return fmt.Sprintf("cannot define array [[%s]], key already implicitly a table", path)
	}
	return ""
}

func (v *docValidator) checkIntermediatePathsAOT(parts []KeyPart, path string) string {
	ts := v.state
	for i := 1; i < len(parts); i++ {
		parentPath := keyPartsToPath(parts[:i])
		if ts.scalarPaths[parentPath] {
			return fmt.Sprintf("cannot define array [[%s]], key %q already a value", path, parentPath)
		}
		if ts.inlinePaths[parentPath] {
			return fmt.Sprintf("cannot extend inline table/array at %q", parentPath)
		}
		if ts.staticArrays[parentPath] {
			return fmt.Sprintf("cannot extend static array at %q", parentPath)
		}
	}
	return ""
}

func (v *docValidator) clearSubScope(path string) {
	prefix := path + "."
	clearPrefix(v.state.explicitTables, prefix)
	clearPrefix(v.state.dottedKeyTables, prefix)
	clearPrefix(v.state.scalarPaths, prefix)
	clearPrefix(v.state.inlinePaths, prefix)
	clearPrefix(v.state.staticArrays, prefix)
}

func clearPrefix(m map[string]bool, prefix string) {
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			delete(m, k)
		}
	}
}

func (v *docValidator) checkKeyValue(baseParts []KeyPart, kv *KeyValue) error {
	ts := v.state

	for i := 0; i < len(kv.KeyParts)-1; i++ {
		intermediatePath := buildFullPath(baseParts, kv.KeyParts[:i+1])
		if msg := v.checkDottedIntermediate(intermediatePath); msg != "" {
			return v.errorAt(msg, kv.line, kv.col)
		}
		ts.dottedKeyTables[intermediatePath] = true
	}

	leafPath := buildFullPath(baseParts, kv.KeyParts)

	// Check for duplicate/conflicting key BEFORE marking the path.
	if msg := v.checkLeafConflict(leafPath); msg != "" {
		return v.errorAt(msg, kv.line, kv.col)
	}

	v.markLeafPath(leafPath, kv.Val)

	// Check inline table entries for duplicate keys.
	if it, ok := kv.Val.(*InlineTableNode); ok {
		if err := v.checkInlineTableKeys(leafPath, it, kv.line, kv.col); err != nil {
			return err
		}
	}

	return nil
}

func (v *docValidator) checkDottedIntermediate(path string) string {
	ts := v.state
	if ts.inlinePaths[path] {
		return fmt.Sprintf("cannot extend inline table at %q", path)
	}
	if ts.scalarPaths[path] {
		return fmt.Sprintf("key %q already defined as a value", path)
	}
	if ts.explicitTables[path] {
		return fmt.Sprintf("cannot add to explicitly defined table %q via dotted keys", path)
	}
	if ts.aotPaths[path] {
		return fmt.Sprintf("cannot extend array of tables %q via dotted keys", path)
	}
	return ""
}

func (v *docValidator) markLeafPath(path string, val Node) {
	ts := v.state
	switch val.(type) {
	case *InlineTableNode:
		v.markInlinePaths(path, val)
	case *ArrayNode:
		v.markInlinePaths(path, val)
		ts.staticArrays[path] = true
	default:
		ts.scalarPaths[path] = true
	}
}

func (v *docValidator) markInlinePaths(path string, val Node) {
	v.state.inlinePaths[path] = true
	switch n := val.(type) {
	case *InlineTableNode:
		for _, kv := range n.Entries {
			subPath := path + "." + keyPartsToPath(kv.KeyParts)
			v.markInlinePaths(subPath, kv.Val)
		}
	case *ArrayNode:
		for _, elem := range n.Elements {
			if it, ok := elem.(*InlineTableNode); ok {
				for _, kv := range it.Entries {
					subPath := path + "." + keyPartsToPath(kv.KeyParts)
					v.markInlinePaths(subPath, kv.Val)
				}
			}
		}
	}
}

func (v *docValidator) checkLeafConflict(path string) string {
	ts := v.state
	if ts.scalarPaths[path] {
		return fmt.Sprintf("duplicate key %q", path)
	}
	if ts.inlinePaths[path] {
		return fmt.Sprintf("duplicate key %q", path)
	}
	if ts.dottedKeyTables[path] {
		return fmt.Sprintf("key %q already used as a table via dotted keys", path)
	}
	if ts.aotPaths[path] {
		return fmt.Sprintf("key %q already defined as array of tables", path)
	}
	return ""
}

func (v *docValidator) checkInlineTableKeys(_ string, it *InlineTableNode, line, col int) error {
	seen := make(map[string]bool)
	for _, kv := range it.Entries {
		fullKey := keyPartsToPath(kv.KeyParts)
		if seen[fullKey] {
			return v.errorAt(fmt.Sprintf("duplicate key %q in inline table", fullKey), line, col)
		}
		seen[fullKey] = true
		for i := 1; i < len(kv.KeyParts); i++ {
			prefix := keyPartsToPath(kv.KeyParts[:i])
			if seen[prefix] {
				return v.errorAt(fmt.Sprintf("key %q conflicts with dotted key in inline table", prefix), line, col)
			}
		}
	}
	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/maurice/toml"
)

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	doc, err := toml.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	result := documentToTaggedJSON(doc)

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonBytes))
	os.Exit(0)
}

func documentToTaggedJSON(doc *toml.Document) map[string]any {
	root := make(map[string]any)
	for _, n := range doc.Nodes() {
		switch v := n.(type) {
		case *toml.KeyValue:
			setKeyValue(root, v)
		case *toml.TableNode:
			processTableNode(root, v)
		case *toml.ArrayOfTables:
			processAOTNode(root, v)
		}
	}
	return root
}

func setKeyValue(tbl map[string]any, kv *toml.KeyValue) {
	val := valueToTagged(kv.Val())
	setNestedKey(tbl, kv.KeyParts(), val)
}

func processTableNode(root map[string]any, v *toml.TableNode) {
	tbl := resolveTablePath(root, v.HeaderParts())
	for _, entry := range v.Entries() {
		if kv, ok := entry.(*toml.KeyValue); ok {
			setKeyValue(tbl, kv)
		}
	}
}

func processAOTNode(root map[string]any, v *toml.ArrayOfTables) {
	parts := v.HeaderParts()
	parent := resolveTablePath(root, parts[:len(parts)-1])
	lastKey := parts[len(parts)-1].Unquoted
	arr, _ := parent[lastKey].([]any)
	entry := make(map[string]any)
	for _, e := range v.Entries() {
		if kv, ok := e.(*toml.KeyValue); ok {
			setKeyValue(entry, kv)
		}
	}
	parent[lastKey] = append(arr, entry)
}

// resolveTablePath navigates a path, following arrays-of-tables to their last element.
func resolveTablePath(root map[string]any, parts []toml.KeyPart) map[string]any {
	cur := root
	for _, p := range parts {
		key := p.Unquoted
		existing := cur[key]
		switch v := existing.(type) {
		case []any:
			if len(v) == 0 {
				m := make(map[string]any)
				cur[key] = []any{m}
				cur = m
			} else {
				if m, ok := v[len(v)-1].(map[string]any); ok {
					cur = m
				}
			}
		case map[string]any:
			cur = v
		default:
			sub := make(map[string]any)
			cur[key] = sub
			cur = sub
		}
	}
	return cur
}

func setNestedKey(m map[string]any, parts []toml.KeyPart, value any) {
	cur := m
	for i, p := range parts {
		key := p.Unquoted
		if i == len(parts)-1 {
			cur[key] = value
		} else {
			if sub, ok := cur[key].(map[string]any); ok {
				cur = sub
			} else {
				sub := make(map[string]any)
				cur[key] = sub
				cur = sub
			}
		}
	}
}

func valueToTagged(node toml.Node) any {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *toml.StringNode:
		return tagged("string", unquoteString(n.Text()))
	case *toml.NumberNode:
		return numberToTagged(n.Text())
	case *toml.BooleanNode:
		return tagged("bool", n.Text())
	case *toml.DateTimeNode:
		return datetimeToTagged(n.Text())
	case *toml.ArrayNode:
		result := make([]any, 0, len(n.Elements()))
		for _, elem := range n.Elements() {
			result = append(result, valueToTagged(elem))
		}
		return result
	case *toml.InlineTableNode:
		result := make(map[string]any)
		for _, kv := range n.Entries() {
			setNestedKey(result, kv.KeyParts(), valueToTagged(kv.Val()))
		}
		return result
	default:
		return tagged("string", n.Text())
	}
}

func tagged(typ, val string) map[string]string {
	return map[string]string{"type": typ, "value": val}
}

func numberToTagged(text string) map[string]string {
	clean := strings.ReplaceAll(text, "_", "")
	switch clean {
	case "inf", "+inf":
		return tagged("float", "+inf")
	case "-inf":
		return tagged("float", "-inf")
	case "nan", "+nan", "-nan":
		return tagged("float", "nan")
	}
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0o") || strings.HasPrefix(clean, "0b") {
		return tagged("integer", parseInteger(clean))
	}
	if strings.ContainsAny(clean, ".eE") {
		return tagged("float", parseFloat(clean))
	}
	return tagged("integer", parseInteger(clean))
}

func datetimeToTagged(text string) map[string]string {
	return tagged(detectDateTimeType(text), normalizeDatetime(text))
}

// normalizeDatetime normalizes space separators to T and adds :00 when seconds are omitted.
func normalizeDatetime(val string) string {
	// Replace space separator with T
	if spaceIdx := strings.Index(val, " "); spaceIdx > 0 {
		// Only replace if it looks like a date-time separator (digit before space, digit after)
		if spaceIdx+1 < len(val) && val[spaceIdx-1] >= '0' && val[spaceIdx-1] <= '9' &&
			val[spaceIdx+1] >= '0' && val[spaceIdx+1] <= '9' {
			val = val[:spaceIdx] + "T" + val[spaceIdx+1:]
		}
	}
	// Normalize lowercase t to T
	if tIdx := strings.Index(val, "t"); tIdx > 0 && val[tIdx-1] >= '0' && val[tIdx-1] <= '9' {
		val = val[:tIdx] + "T" + val[tIdx+1:]
	}
	// Add :00 seconds if missing
	val = addMissingSeconds(val)
	return val
}

func addMissingSeconds(val string) string {
	colonCount := strings.Count(val, ":")
	if colonCount == 0 {
		return val
	}
	// For time-local (no date part): exactly one colon means HH:MM
	if !strings.Contains(val, "-") && !strings.Contains(val, "T") {
		if colonCount == 1 {
			return val + ":00"
		}
		return val
	}
	// For date-time: find the time part and check colon count there
	tIdx := strings.Index(val, "T")
	if tIdx < 0 {
		return val
	}
	timePart := val[tIdx+1:]
	// Strip offset for analysis
	offsetStart := -1
	if zIdx := strings.IndexAny(timePart, "Zz"); zIdx >= 0 {
		offsetStart = zIdx
	} else if pIdx := strings.LastIndexAny(timePart, "+-"); pIdx > 0 {
		offsetStart = pIdx
	}
	timeCore := timePart
	suffix := ""
	if offsetStart >= 0 {
		timeCore = timePart[:offsetStart]
		suffix = timePart[offsetStart:]
	}
	if strings.Count(timeCore, ":") == 1 {
		return val[:tIdx+1] + timeCore + ":00" + suffix
	}
	return val
}

//nolint:gocyclo
func detectDateTimeType(val string) string {
	if strings.Contains(val, "Z") || strings.Contains(val, "z") {
		return "datetime"
	}
	hasT := strings.Contains(val, "T") || strings.Contains(val, "t")
	hasDash := strings.Count(val, "-") >= 2
	hasColon := strings.Count(val, ":") >= 1

	if hasT && hasDash && hasColon {
		tPos := strings.IndexAny(val, "Tt")
		timePart := val[tPos+1:]
		if strings.Contains(timePart, "+") || lastDashIsOffset(timePart) {
			return "datetime"
		}
		return "datetime-local"
	}
	if hasDash && hasColon && strings.Contains(val, " ") {
		parts := strings.SplitN(val, " ", 2)
		if len(parts) == 2 && strings.Count(parts[0], "-") >= 2 {
			timePart := parts[1]
			if strings.Contains(timePart, "+") || lastDashIsOffset(timePart) || strings.HasSuffix(timePart, "Z") {
				return "datetime"
			}
			return "datetime-local"
		}
	}
	if hasDash && !hasColon && !hasT {
		return "date-local"
	}
	if hasColon && !hasDash {
		return "time-local"
	}
	return "datetime"
}

func lastDashIsOffset(timePart string) bool {
	idx := strings.LastIndex(timePart, "-")
	if idx <= 0 {
		return false
	}
	return idx+1 < len(timePart) && timePart[idx+1] >= '0' && timePart[idx+1] <= '9'
}

//nolint:gocyclo
func unquoteString(val string) string {
	if len(val) < 2 {
		return val
	}
	if strings.HasPrefix(val, `"""`) && strings.HasSuffix(val, `"""`) && len(val) >= 6 {
		inner := val[3 : len(val)-3]
		if len(inner) > 0 && inner[0] == '\n' {
			inner = inner[1:]
		} else if strings.HasPrefix(inner, "\r\n") {
			inner = inner[2:]
		}
		return processBasicEscapes(inner)
	}
	if strings.HasPrefix(val, "'''") && strings.HasSuffix(val, "'''") && len(val) >= 6 {
		inner := val[3 : len(val)-3]
		if len(inner) > 0 && inner[0] == '\n' {
			inner = inner[1:]
		} else if strings.HasPrefix(inner, "\r\n") {
			inner = inner[2:]
		}
		return inner
	}
	if val[0] == '\'' && val[len(val)-1] == '\'' {
		return val[1 : len(val)-1]
	}
	if val[0] == '"' && val[len(val)-1] == '"' {
		return processBasicEscapes(val[1 : len(val)-1])
	}
	return val
}

//nolint:gocyclo
func processBasicEscapes(s string) string {
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
		case ' ', '\t':
			if hasNewlineAfterWs(s, i) {
				i = skipToNextNonWs(s, i)
				continue
			}
			b.WriteByte('\\')
			b.WriteByte(s[i])
		case '\n':
			i = skipMultiLineWhitespace(s, i)
		case '\r':
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
			i = skipMultiLineWhitespace(s, i)
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func skipMultiLineWhitespace(s string, i int) int {
	for i+1 < len(s) && (s[i+1] == ' ' || s[i+1] == '\t' || s[i+1] == '\n' || s[i+1] == '\r') {
		i++
	}
	return i
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
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i - 1
}

func parseInteger(val string) string {
	clean := strings.ReplaceAll(val, "_", "")
	var num int64
	var err error

	switch {
	case strings.HasPrefix(clean, "0x"):
		num, err = strconv.ParseInt(clean[2:], 16, 64)
	case strings.HasPrefix(clean, "0o"):
		num, err = strconv.ParseInt(clean[2:], 8, 64)
	case strings.HasPrefix(clean, "0b"):
		num, err = strconv.ParseInt(clean[2:], 2, 64)
	default:
		clean = strings.TrimPrefix(clean, "+")
		num, err = strconv.ParseInt(clean, 10, 64)
	}

	if err != nil {
		return val
	}
	return strconv.FormatInt(num, 10)
}

func parseFloat(val string) string {
	clean := strings.ReplaceAll(val, "_", "")
	clean = strings.TrimPrefix(clean, "+")
	num, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return val
	}
	if math.IsInf(num, 0) || math.IsNaN(num) {
		return val
	}
	result := strconv.FormatFloat(num, 'G', -1, 64)
	result = strings.ReplaceAll(result, "E+", "e+")
	result = strings.ReplaceAll(result, "E-", "e-")
	if !strings.Contains(result, ".") && !strings.Contains(result, "e") {
		result += ".0"
	}
	return result
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	var input map[string]any
	err = json.Unmarshal(data, &input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	var b strings.Builder
	encodeTable(&b, input, nil)
	fmt.Print(b.String())
	os.Exit(0)
}

func encodeTable(b *strings.Builder, m map[string]any, path []string) {
	var scalarKeys, tableKeys, aotKeys []string

	for k, v := range m {
		switch categorize(v) {
		case catScalar:
			scalarKeys = append(scalarKeys, k)
		case catTable:
			tableKeys = append(tableKeys, k)
		case catArrayOfTables:
			aotKeys = append(aotKeys, k)
		case catArray:
			scalarKeys = append(scalarKeys, k)
		}
	}

	sort.Strings(scalarKeys)
	sort.Strings(tableKeys)
	sort.Strings(aotKeys)

	emitScalars(b, m, scalarKeys)
	emitSubTables(b, m, path, tableKeys)
	emitArraysOfTables(b, m, path, aotKeys)
}

func emitScalars(b *strings.Builder, m map[string]any, keys []string) {
	for _, k := range keys {
		b.WriteString(quoteKey(k))
		b.WriteString(" = ")
		encodeValue(b, m[k])
		b.WriteString("\n")
	}
}

func emitSubTables(b *strings.Builder, m map[string]any, path, keys []string) {
	for _, k := range keys {
		subPath := makePath(path, k)
		sub := m[k].(map[string]any)
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[")
		b.WriteString(encodePath(subPath))
		b.WriteString("]\n")
		encodeTable(b, sub, subPath)
	}
}

func emitArraysOfTables(b *strings.Builder, m map[string]any, path, keys []string) {
	for _, k := range keys {
		subPath := makePath(path, k)
		arr := m[k].([]any)
		for _, elem := range arr {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("[[")
			b.WriteString(encodePath(subPath))
			b.WriteString("]]\n")
			if tbl, ok := elem.(map[string]any); ok {
				encodeTable(b, tbl, subPath)
			}
		}
	}
}

func makePath(base []string, key string) []string {
	out := make([]string, len(base)+1)
	copy(out, base)
	out[len(base)] = key
	return out
}

type category int

const (
	catScalar category = iota
	catTable
	catArrayOfTables
	catArray
)

func categorize(v any) category {
	switch val := v.(type) {
	case map[string]any:
		if isTaggedValue(val) {
			return catScalar
		}
		return catTable
	case []any:
		if isArrayOfTables(val) {
			return catArrayOfTables
		}
		return catArray
	default:
		return catScalar
	}
}

func isTaggedValue(m map[string]any) bool {
	_, hasType := m["type"]
	_, hasValue := m["value"]
	return hasType && hasValue && len(m) == 2
}

func isArrayOfTables(arr []any) bool {
	if len(arr) == 0 {
		return false
	}
	for _, elem := range arr {
		m, ok := elem.(map[string]any)
		if !ok {
			return false
		}
		if isTaggedValue(m) {
			return false
		}
	}
	return true
}

func encodeValue(b *strings.Builder, v any) {
	switch val := v.(type) {
	case map[string]any:
		if isTaggedValue(val) {
			encodeTaggedValue(b, val["type"].(string), fmt.Sprint(val["value"]))
			return
		}
		encodeInlineTable(b, val)
	case []any:
		encodeInlineArray(b, val)
	default:
		b.WriteString(fmt.Sprint(v))
	}
}

func encodeInlineTable(b *strings.Builder, val map[string]any) {
	b.WriteString("{")
	keys := sortedKeys(val)
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteKey(k))
		b.WriteString(" = ")
		encodeValue(b, val[k])
	}
	b.WriteString("}")
}

func encodeInlineArray(b *strings.Builder, val []any) {
	b.WriteString("[")
	for i, elem := range val {
		if i > 0 {
			b.WriteString(", ")
		}
		encodeValue(b, elem)
	}
	b.WriteString("]")
}

func encodeTaggedValue(b *strings.Builder, typ, val string) {
	switch typ {
	case "string":
		b.WriteString(`"`)
		b.WriteString(escapeString(val))
		b.WriteString(`"`)
	case "integer":
		b.WriteString(val)
	case "float":
		encodeFloat(b, val)
	case "bool":
		b.WriteString(val)
	case "datetime", "datetime-local", "date-local", "time-local":
		b.WriteString(val)
	default:
		b.WriteString(`"`)
		b.WriteString(escapeString(val))
		b.WriteString(`"`)
	}
}

func encodeFloat(b *strings.Builder, val string) {
	b.WriteString(val)
	if !strings.ContainsAny(val, ".eE") && !isSpecialFloat(val) {
		b.WriteString(".0")
	}
}

func isSpecialFloat(val string) bool {
	switch val {
	case "inf", "+inf", "-inf", "nan", "+nan", "-nan":
		return true
	}
	return false
}

func escapeString(s string) string {
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
			escapeRuneDefault(&b, r)
		}
	}
	return b.String()
}

func escapeRuneDefault(b *strings.Builder, r rune) {
	switch {
	case r < 0x20 || r == 0x7F:
		b.WriteString(fmt.Sprintf(`\u%04X`, r))
	case r > 0xFFFF:
		b.WriteString(fmt.Sprintf(`\U%08X`, r))
	default:
		b.WriteRune(r)
	}
}

func quoteKey(k string) string {
	if isBareKey(k) {
		return k
	}
	return `"` + escapeString(k) + `"`
}

func isBareKey(s string) bool {
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

func isBareKeyChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '-' || r == '_'
}

func encodePath(parts []string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteString(".")
		}
		b.WriteString(quoteKey(p))
	}
	return b.String()
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

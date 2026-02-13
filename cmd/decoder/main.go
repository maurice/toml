package main

import (
	"encoding/json"
	"fmt"
	"io"
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
		fmt.Fprintf(os.Stderr, "error parsing TOML: %v\n", err)
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

// documentToTaggedJSON converts a parsed TOML document to tagged JSON format.
func documentToTaggedJSON(doc *toml.Document) map[string]interface{} {
	result := make(map[string]interface{})

	for _, kv := range doc.KeyValues() {
		result[kv.Key] = kvToTaggedValue(kv)
	}

	return result
}

// kvToTaggedValue converts a KeyValue to tagged JSON format.
func kvToTaggedValue(kv *toml.KeyValue) map[string]string {
	typeStr, value := detectTypeAndValue(kv)
	return map[string]string{
		"type":  typeStr,
		"value": value,
	}
}

// detectTypeAndValue detects the type and value of a KeyValue.
func detectTypeAndValue(kv *toml.KeyValue) (string, string) {
	val := strings.TrimSpace(kv.Value)
	typeName := detectType(kv.ValueTok)
	convertedVal := convertRawValue(typeName, val)
	return typeName, convertedVal
}

// detectType determines the TOML type from a value token.
func detectType(tok toml.Node) string {
	if tok == nil {
		return "string"
	}

	switch tok.(type) {
	case *toml.StringNode:
		return "string"
	case *toml.NumberNode:
		return detectNumberType(tok.Text())
	case *toml.BooleanNode:
		return "bool"
	case *toml.DateTimeNode:
		return detectDateTimeType(tok.Text())
	default:
		return "string"
	}
}

// detectNumberType determines if a number is integer or float.
func detectNumberType(val string) string {
	val = strings.TrimSpace(val)
	if strings.ContainsAny(val, ".eE") {
		return "float"
	}
	return "integer"
}

// detectDateTimeType determines the datetime variant.
func detectDateTimeType(val string) string {
	val = strings.TrimSpace(val)

	// Check for offset datetime (has Z or ±HH:MM)
	if strings.Contains(val, "Z") ||
		(strings.Contains(val, "+") && strings.Contains(val, ":")) ||
		(strings.LastIndex(val, "-") > strings.LastIndex(val, "T")) {
		return "datetime"
	}

	if strings.Contains(val, "T") {
		return "datetime-local"
	}

	if strings.Count(val, "-") >= 2 {
		return "date-local"
	}

	if strings.Count(val, ":") >= 2 {
		return "time-local"
	}

	return "datetime"
}

// convertRawValue converts the raw TOML value to the appropriate JSON representation.
func convertRawValue(typeStr, val string) string {
	val = strings.TrimSpace(val)

	switch typeStr {
	case "string":
		return unquoteString(val)
	case "integer":
		return parseInteger(val)
	case "float":
		return parseFloat(val)
	case "bool":
		return val
	case "datetime", "datetime-local", "date-local", "time-local":
		return val
	default:
		return val
	}
}

// unquoteString removes quotes and handles escape sequences.
func unquoteString(val string) string {
	if len(val) < 2 {
		return val
	}

	// Check for triple quotes
	if (strings.HasPrefix(val, `"""`) && strings.HasSuffix(val, `"""`)) ||
		(strings.HasPrefix(val, "''''") && strings.HasSuffix(val, "'''")) {
		return val[3 : len(val)-3]
	}

	// Single quotes (literal strings)
	if val[0] == '\'' && val[len(val)-1] == '\'' {
		return val[1 : len(val)-1]
	}

	// Double quotes (basic strings)
	if val[0] == '"' && val[len(val)-1] == '"' {
		s, _ := strconv.Unquote(val)
		return s
	}

	return val
}

// parseInteger normalizes integer representation.
func parseInteger(val string) string {
	val = strings.TrimSpace(val)

	var num int64
	var err error

	switch {
	case strings.HasPrefix(val, "0x") || strings.HasPrefix(val, "0X"):
		num, err = strconv.ParseInt(val, 16, 64)
	case strings.HasPrefix(val, "0o") || strings.HasPrefix(val, "0O"):
		num, err = strconv.ParseInt(val[2:], 8, 64)
	case strings.HasPrefix(val, "0b") || strings.HasPrefix(val, "0B"):
		num, err = strconv.ParseInt(val[2:], 2, 64)
	default:
		num, err = strconv.ParseInt(val, 10, 64)
	}

	if err != nil {
		return val
	}
	return strconv.FormatInt(num, 10)
}

// parseFloat normalizes float representation.
func parseFloat(val string) string {
	val = strings.TrimSpace(val)
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return val
	}
	return strconv.FormatFloat(num, 'f', -1, 64)
}

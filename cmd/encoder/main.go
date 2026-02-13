package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/maurice/toml"
)

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	var input map[string]interface{}
	err = json.Unmarshal(data, &input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	doc := buildDocumentFromTaggedJSON(input)
	fmt.Print(doc.String())
	os.Exit(0)
}

// buildDocumentFromTaggedJSON constructs a Document from tagged JSON.
func buildDocumentFromTaggedJSON(data map[string]interface{}) *toml.Document {
	doc := &toml.Document{}

	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		kv := buildKeyValueFromTaggedJSON(key, data[key])
		if kv != nil {
			doc.Nodes = append(doc.Nodes, kv)
		}
	}

	return doc
}

// buildKeyValueFromTaggedJSON constructs a KeyValue from tagged JSON.
func buildKeyValueFromTaggedJSON(key string, value interface{}) *toml.KeyValue {
	switch v := value.(type) {
	case map[string]interface{}:
		// Check if it's a tagged value
		if len(v) == 2 {
			if typeVal, ok := v["type"].(string); ok {
				if valStr, ok := v["value"].(string); ok {
					// It's a tagged value - create a KeyValue
					return createKeyValue(key, typeVal, valStr)
				}
			}
		}
		// Not a tagged value, so skip (tables not yet supported)
	case []interface{}:
		// Arrays not yet supported
	}
	return nil
}

// createKeyValue creates a KeyValue node from tagged components.
func createKeyValue(key, typeStr, value string) *toml.KeyValue {
	formattedValue := formatValueForTOML(typeStr, value)

	kv := &toml.KeyValue{
		Key:      key,
		PreEq:    " ",
		PostEq:   " ",
		Value:    formattedValue,
		Trailing: "",
	}

	return kv
}

// formatValueForTOML formats a value for TOML output.
func formatValueForTOML(typeStr, value string) string {
	switch typeStr {
	case "string":
		// Properly quote and escape strings
		escaped := escapeString(value)
		return `"` + escaped + `"`
	case "integer", "float", "bool":
		// These don't need modification
		return value
	case "datetime", "datetime-local", "date-local", "time-local":
		// Date/time values are already formatted correctly
		return value
	default:
		// Fallback: quote as string
		escaped := escapeString(value)
		return `"` + escaped + `"`
	}
}

// escapeString escapes special characters in a string value.
func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

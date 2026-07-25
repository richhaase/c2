package jsonx

import (
	"bytes"
	"encoding/json"
)

func encode(v any, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func Compact(v any) ([]byte, error) {
	return encode(v, "")
}

func Indent(v any) ([]byte, error) {
	return encode(v, "  ")
}

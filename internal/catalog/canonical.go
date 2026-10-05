package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Canonicalize implements the JSON Canonicalization Scheme (RFC 8785) for the
// catalog's JSON-only data model. It rejects duplicate object member names and
// unsupported numeric forms before serializing deterministically.
func Canonicalize(data []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := decodeValue(decoder)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	var output bytes.Buffer
	if err := writeCanonical(&output, value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func decodeValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token := token.(type) {
	case json.Delim:
		switch token {
		case '{':
			object := make(map[string]any)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("object key is not a string")
				}
				if _, exists := object[key]; exists {
					return nil, fmt.Errorf("duplicate object key %q", key)
				}
				item, err := decodeValue(decoder)
				if err != nil {
					return nil, err
				}
				object[key] = item
			}
			if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
				return nil, fmt.Errorf("unterminated object")
			}
			return object, nil
		case '[':
			var array []any
			for decoder.More() {
				item, err := decodeValue(decoder)
				if err != nil {
					return nil, err
				}
				array = append(array, item)
			}
			if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
				return nil, fmt.Errorf("unterminated array")
			}
			return array, nil
		default:
			return nil, fmt.Errorf("unexpected delimiter %q", token)
		}
	case json.Number:
		return token, nil
	case string, bool, nil:
		return token, nil
	default:
		return nil, fmt.Errorf("unsupported JSON value %T", token)
	}
}

func writeCanonical(output *bytes.Buffer, value any) error {
	switch value := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		output.WriteString(strconv.FormatBool(value))
	case string:
		writeJSONString(output, value)
	case json.Number:
		canonical, err := canonicalNumber(string(value))
		if err != nil {
			return err
		}
		output.WriteString(canonical)
	case []any:
		output.WriteByte('[')
		for index, item := range value {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := writeCanonical(output, item); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return compareUTF16(keys[i], keys[j]) < 0 })
		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			writeJSONString(output, key)
			output.WriteByte(':')
			if err := writeCanonical(output, value[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical value %T", value)
	}
	return nil
}

func writeJSONString(output *bytes.Buffer, value string) {
	output.WriteByte('"')
	for _, runeValue := range value {
		switch runeValue {
		case '"':
			output.WriteString(`\"`)
		case '\\':
			output.WriteString(`\\`)
		case '\b':
			output.WriteString(`\b`)
		case '\f':
			output.WriteString(`\f`)
		case '\n':
			output.WriteString(`\n`)
		case '\r':
			output.WriteString(`\r`)
		case '\t':
			output.WriteString(`\t`)
		default:
			if runeValue < 0x20 {
				fmt.Fprintf(output, `\u%04x`, runeValue)
			} else {
				output.WriteRune(runeValue)
			}
		}
	}
	output.WriteByte('"')
}

func canonicalNumber(value string) (string, error) {
	// Catalog v1 permits only integral JSON numbers; catalog_version and artifact
	// sizes are integers. Restricting the model avoids lossy float conversions.
	if strings.ContainsAny(value, ".eE") {
		return "", fmt.Errorf("non-integral JSON number %q is unsupported", value)
	}
	integer, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid integer %q", value)
	}
	return strconv.FormatInt(integer, 10), nil
}

func compareUTF16(left, right string) int {
	leftUnits := utf16Units(left)
	rightUnits := utf16Units(right)
	for index := 0; index < len(leftUnits) && index < len(rightUnits); index++ {
		if leftUnits[index] < rightUnits[index] {
			return -1
		}
		if leftUnits[index] > rightUnits[index] {
			return 1
		}
	}
	switch {
	case len(leftUnits) < len(rightUnits):
		return -1
	case len(leftUnits) > len(rightUnits):
		return 1
	default:
		return 0
	}
}

func utf16Units(value string) []rune {
	result := make([]rune, 0, len(value))
	for _, runeValue := range value {
		if runeValue <= 0xffff {
			result = append(result, runeValue)
			continue
		}
		runeValue -= 0x10000
		result = append(result, 0xd800+(runeValue>>10), 0xdc00+(runeValue&0x3ff))
	}
	return result
}

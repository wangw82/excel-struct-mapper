package mapper

import (
	"fmt"
	"strconv"
	"strings"
)

type tagOptions map[string]string

var blockOptions = map[string]bool{
	tagKeyKey: true, tagKeyWorkflow: true, tagKeyFormat: true,
	tagKeyStartRow: true, tagKeyEndRow: true, tagKeyTitle: true, tagKeyEndTitle: true,
	tagKeyMinRows: true, tagKeyBlankLine: true, tagKeyOptional: true,
	tagKeyBlockCodec: true, tagKeyDataRow: true, tagKeyLabelCol: true,
	tagKeyValueCol: true, tagKeyLabel: true, tagKeyMultiCell: true,
	tagKeyAllowEmpty: true, tagKeySeparator: true, tagKeyValueCodec: true,
	tagKeyCodecOptions: true, tagKeyIncludeEndBlock: true,
}
var fieldOptions = map[string]bool{
	tagKeyHeader: true, tagKeyRequired: true, tagKeyAllowEmpty: true,
	tagKeySkipDecode: true, tagKeySkipEncode: true, tagKeyMultiCell: true,
	tagKeySeparator: true, tagKeyValueCodec: true, tagKeyValidate: true,
	tagKeyCodecOptions: true,
}

func parseOptions(tag string, allowed map[string]bool) (tagOptions, error) {
	result := tagOptions{}
	parts, err := splitEscaped(tag, ';')
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("%w: empty option", ErrInvalidTag)
		}
		pair, err := splitPair(part)
		if err != nil {
			return nil, err
		}
		key, value := pair[0], pair[1]
		if !allowed[key] {
			return nil, fmt.Errorf("%w: unknown option %q", ErrInvalidTag, key)
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("%w: duplicate option %q", ErrInvalidTag, key)
		}
		if value == "" {
			return nil, fmt.Errorf("%w: option %q has no value", ErrInvalidTag, key)
		}
		result[key] = value
	}
	return result, nil
}

func splitEscaped(value string, separator rune) ([]string, error) {
	var result []string
	var current strings.Builder
	escaped := false
	for _, char := range value {
		if escaped {
			if char != ';' && char != '=' && char != '\\' {
				return nil, fmt.Errorf("%w: unsupported escape \\%c", ErrInvalidTag, char)
			}
			current.WriteRune(char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == separator {
			result = append(result, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(char)
	}
	if escaped {
		return nil, fmt.Errorf("%w: trailing escape", ErrInvalidTag)
	}
	return append(result, current.String()), nil
}

func splitPair(value string) ([2]string, error) {
	for i, char := range value {
		if char == '=' {
			return [2]string{value[:i], value[i+1:]}, nil
		}
	}
	return [2]string{}, fmt.Errorf("%w: option %q has no equals sign", ErrInvalidTag, value)
}

func optionBool(options tagOptions, key string, fallback bool) (bool, error) {
	value, exists := options[key]
	if !exists {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%w: option %q: %v", ErrInvalidTag, key, err)
	}
	return parsed, nil
}

func optionRow(options tagOptions, key string, fallback int) (int, error) {
	value, exists := options[key]
	if !exists {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%w: option %q must be a positive row number", ErrInvalidTag, key)
	}
	return parsed - 1, nil
}

func optionColumn(options tagOptions, key string, fallback int) (int, error) {
	value, exists := options[key]
	if !exists {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%w: option %q must be a positive column number", ErrInvalidTag, key)
	}
	return parsed - 1, nil
}

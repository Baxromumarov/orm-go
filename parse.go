package orm_go

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

const TagKey = "orm"

// ParseInsertColumns returns non-zero struct fields tagged with `orm` as columns and values.
func ParseInsertColumns(model any) ([]string, []any) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	var cols []string
	var vals []any

	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		sf := t.Field(i)

		tag := sf.Tag.Get(TagKey)
		if tag == "" {
			continue
		}

		if fv.IsZero() {
			continue
		}

		cols = append(cols, tag)
		vals = append(vals, fv.Interface())
	}

	return cols, vals
}

// ParseTags returns the \`orm\` struct tags for exported fields in the given input.
// It supports structs and pointers to structs. For unsupported or nil inputs it returns nil.
func ParseTags(input any) []string {
	if input == nil {
		return nil
	}

	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	out := make([]string, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if f.PkgPath != "" {
			continue
		}

		tag := f.Tag.Get(TagKey)
		if tag == "" {
			continue
		}

		out = append(out, tag)
	}

	return out
}

// ParseTableName derives a table name from a struct type (lower + snake_case + pluralize).
func ParseTableName(input any) string {
	if input == nil {
		return ""
	}

	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}
	if v.Kind() == reflect.Slice && v.IsNil() {
		return ""
	}

	t := v.Type()

	// 🔥 unwrap pointers, slices, arrays
	for {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array:
			t = t.Elem()
		default:
			goto DONE
		}
	}

DONE:
	if t.Kind() != reflect.Struct {
		return ""
	}

	name := t.Name()
	if name == "" {
		return ""
	}

	return pluralize(
		snakeCase(
			strings.ToLower(name),
		),
	)
}

func pluralize(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}

	// If it already looks plural (ends with \`s\`) keep as-is.
	if strings.HasSuffix(s, "s") {
		return s
	}

	// Common English pluralization rules (simple, deterministic).
	// - \`...y\` after consonant -> \`...ies\` (city -> cities)
	// - \`...s|x|z|ch|sh\` -> \`...es\` (class -> classes, box -> boxes)
	lowerS := strings.ToLower(s)

	if strings.HasSuffix(lowerS, "y") && len(s) >= 2 {
		prev, _ := utf8.DecodeLastRuneInString(s[:len(s)-1])
		if !strings.ContainsRune("aeiou", unicode.ToLower(prev)) {
			return s[:len(s)-1] + "ies"
		}
	}

	if strings.HasSuffix(lowerS, "s") ||
		strings.HasSuffix(lowerS, "x") ||
		strings.HasSuffix(lowerS, "z") ||
		strings.HasSuffix(lowerS, "ch") ||
		strings.HasSuffix(lowerS, "sh") {
		return s + "es"
	}

	return s + "s"
}

// snakeCase converts CamelCase / PascalCase (and common initialism) into snake_case.
func snakeCase(input string) string {
	if input == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(input) + 4)

	runes := []rune(input)

	isUpper := func(r rune) bool { return r >= 'A' && r <= 'Z' }
	isLower := func(r rune) bool { return r >= 'a' && r <= 'z' }
	isDigit := func(r rune) bool { return r >= '0' && r <= '9' }

	for i, r := range runes {
		if r == '_' {
			if b.Len() > 0 {
				prev, _ := utf8.DecodeLastRuneInString(b.String())
				if prev != '_' {
					b.WriteRune('_')
				}
			}
			continue
		}

		if r == ' ' || r == '-' {
			if b.Len() > 0 {
				prev, _ := utf8.DecodeLastRuneInString(b.String())
				if prev != '_' {
					b.WriteRune('_')
				}
			}
			continue
		}

		// Add underscore on word boundaries:
		// - lower/digit followed by upper: "userID" -> "user_id"
		// - multiple uppers then lower: "HTTPServer" -> "http_server"
		if i > 0 {
			prev := runes[i-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}

			if isUpper(r) && (isLower(prev) || isDigit(prev)) {
				b.WriteRune('_')
			} else if isUpper(r) && isUpper(prev) && next != 0 && isLower(next) {
				b.WriteRune('_')
			}
		}

		b.WriteRune(unicode.ToLower(r))
	}

	// Trim trailing underscores (if any).
	out := b.String()
	out = strings.Trim(out, "_")
	// Collapse repeated underscores.
	out = strings.Join(strings.FieldsFunc(out, func(r rune) bool { return r == '_' }), "_")
	return out
}

func lower(input string) string {
	return strings.ToLower(input)
}

package orm_go

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// FieldMeta holds the struct field index and corresponding database column name.
type FieldMeta struct {
	Index  int
	Column string
}

// ModelMeta maps column names to their struct field metadata.
type ModelMeta struct {
	ByColumn map[string]FieldMeta
}

// metaCache stores parsed model metadata, keyed by reflect.Type.
var metaCache sync.Map

// GetModelMeta returns cached metadata for a model type.
// It accepts a value, pointer, or reflect.Type.
func GetModelMeta(model any) *ModelMeta {
	var t reflect.Type
	switch v := model.(type) {
	case reflect.Type:
		t = v
	default:
		if model == nil {
			return &ModelMeta{ByColumn: make(map[string]FieldMeta)}
		}
		t = reflect.TypeOf(model)
	}

	// Unwrap pointers to get the underlying struct type.
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t == nil || t.Kind() != reflect.Struct {
		return &ModelMeta{ByColumn: make(map[string]FieldMeta)}
	}

	// Check cache first.
	if cached, ok := metaCache.Load(t); ok {
		return cached.(*ModelMeta)
	}

	// Build metadata.
	meta := &ModelMeta{
		ByColumn: make(map[string]FieldMeta),
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		raw := sf.Tag.Get(TagKey)
		if raw == "" {
			continue
		}

		col := strings.Split(raw, ",")[0]
		if col == "" {
			continue
		}

		meta.ByColumn[col] = FieldMeta{
			Index:  i,
			Column: col,
		}
	}

	// Store in cache.
	metaCache.Store(t, meta)
	return meta
}

// ScanArgsForColumns builds scan destination pointers for the given columns.
// Used by SELECT operations to scan query results into struct fields.
func ScanArgsForColumns(v reflect.Value, columns []string, meta *ModelMeta) ([]any, error) {
	scanArgs := make([]any, 0, len(columns))
	for _, col := range columns {
		fm, ok := meta.ByColumn[col]
		if !ok {
			return nil, fmt.Errorf("unknown column: %s", col)
		}

		field := v.Field(fm.Index)
		if !field.CanAddr() {
			return nil, fmt.Errorf("field %s is not addressable", col)
		}

		scanArgs = append(scanArgs, field.Addr().Interface())
	}
	return scanArgs, nil
}

// ScanArgsForReturning builds scan destination pointers for RETURNING clause columns.
// This is a thin wrapper for semantic clarity - INSERT/UPDATE/DELETE RETURNING
// may have subtle differences from SELECT scanning in the future.
func ScanArgsForReturning(v reflect.Value, columns []string, meta *ModelMeta) ([]any, error) {
	scanArgs := make([]any, 0, len(columns))
	for _, col := range columns {
		fm, ok := meta.ByColumn[col]
		if !ok {
			return nil, fmt.Errorf("unknown returning column: %s", col)
		}

		field := v.Field(fm.Index)
		if !field.CanAddr() {
			return nil, fmt.Errorf("field %s is not addressable", col)
		}

		scanArgs = append(scanArgs, field.Addr().Interface())
	}
	return scanArgs, nil
}

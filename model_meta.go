package orm_go

import (
	"reflect"
	"strings"
	"sync"
)

type fieldMeta struct {
	index    int
	column   string
	rawTag   string
	exported bool
}

type modelMeta struct {
	byColumn        map[string]fieldMeta
	fields          []fieldMeta
	exportedTags    []string
	exportedColumns []string
}

var modelMetaCache sync.Map

func buildModelMeta(model any) *modelMeta {
	switch v := model.(type) {
	case reflect.Type:
		return modelMetaForType(v)
	default:
		if model == nil {
			return emptyModelMeta()
		}
		return modelMetaForType(reflect.TypeOf(model))
	}
}

func modelMetaForType(t reflect.Type) *modelMeta {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return emptyModelMeta()
	}

	if cached, ok := modelMetaCache.Load(t); ok {
		return cached.(*modelMeta)
	}

	meta := parseModelMeta(t)
	actual, _ := modelMetaCache.LoadOrStore(t, meta)
	return actual.(*modelMeta)
}

func parseModelMeta(t reflect.Type) *modelMeta {
	meta := &modelMeta{
		byColumn: make(map[string]fieldMeta),
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		raw := sf.Tag.Get(TagKey)
		if raw == "" {
			continue
		}

		col := strings.Split(raw, ",")[0]
		fm := fieldMeta{
			index:    i,
			column:   col,
			rawTag:   raw,
			exported: sf.PkgPath == "",
		}

		meta.byColumn[col] = fm
		meta.fields = append(meta.fields, fm)

		if fm.exported {
			meta.exportedTags = append(meta.exportedTags, raw)
			if col != "" {
				meta.exportedColumns = append(meta.exportedColumns, col)
			}
		}
	}

	return meta
}

func emptyModelMeta() *modelMeta {
	return &modelMeta{
		byColumn: make(map[string]fieldMeta),
	}
}

func columnsFromType(t reflect.Type) []string {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}

	meta := modelMetaForType(t)
	cols := make([]string, len(meta.exportedColumns))
	copy(cols, meta.exportedColumns)
	return cols
}

func structTypeFromInput(input any) (reflect.Type, bool) {
	if input == nil {
		return nil, false
	}

	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, false
	}

	return v.Type(), true
}

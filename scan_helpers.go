package orm_go

import (
	"fmt"
	"reflect"
)

func scanArgsForColumnsValue(v reflect.Value, columns []string, meta *modelMeta, unknownFmt string) ([]any, error) {
	scanArgs := make([]any, 0, len(columns))
	for _, col := range columns {
		fm, ok := meta.byColumn[col]
		if !ok {
			return nil, fmt.Errorf(unknownFmt, col)
		}

		field := v.Field(fm.index)
		if !field.CanAddr() {
			return nil, fmt.Errorf("field %s is not addressable", col)
		}

		scanArgs = append(scanArgs, field.Addr().Interface())
	}
	return scanArgs, nil
}

func scanArgsForModel(model any, columns []string, meta *modelMeta, unknownFmt string) ([]any, error) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return scanArgsForColumnsValue(v, columns, meta, unknownFmt)
}

func scanArgsForReturning(model any, columns []string) ([]any, error) {
	meta := buildModelMeta(model)
	return scanArgsForModel(model, columns, meta, "unknown returning column: %s")
}

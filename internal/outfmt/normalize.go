package outfmt

import (
	"encoding/json"
	"reflect"
)

func normalizeJSONOutput(v any) any {
	if v == nil {
		return v
	}
	switch v.(type) {
	case []byte, json.RawMessage:
		return v
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return v
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return v
		}
		items := rv.Interface()
		// Nil slices serialize as null in JSON, which breaks jq .items[].
		// Coerce to an empty slice so the output is always "items": [].
		if rv.IsNil() {
			items = []any{}
		}
		return map[string]any{"items": items}
	default:
		return v
	}
}

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
		return map[string]any{"items": rv.Interface()}
	default:
		return v
	}
}

package plugin

import (
	"reflect"
	"strings"
)

func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return defaultValue
}

func parseCommaList(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// notLast returns true if i is not the last index of a slice/array.
// This is used from text/templates where the slice element type varies.
func notLast(i int, s any) bool {
	if i < 0 {
		return false
	}

	v := reflect.ValueOf(s)
	if !v.IsValid() {
		return false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		return i < v.Len()-1
	default:
		return false
	}
}

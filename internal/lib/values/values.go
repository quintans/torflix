package values

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

func Ptr[T any](v T) *T {
	return &v
}

type M = map[string]any

func ToMap(args []any) M {
	data := make(M, len(args)/2)
	var k string
	var v any
	for len(args) > 0 {
		k, v, args = argsToValues(args)
		data[k] = v
	}
	return data
}

const badKey = "!BADKEY"

func argsToValues(args []any) (string, any, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return badKey, x, nil
		}
		return x, args[1], args[2:]

	default:
		return badKey, x, args[1:]
	}
}

func ToStr(data M) string {
	if len(data) == 0 {
		return ""
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	buf := &strings.Builder{}
	buf.WriteString("(")
	for k, v := range keys {
		if k > 0 {
			buf.WriteString("; ")
		}
		val := data[v]
		switch t := val.(type) {
		case string:
			buf.WriteString(fmt.Sprintf("%s=%s", v, t))
		case bool, int, int8, int16, int64, uint, uint8, uint16, uint64, float32, float64:
			buf.WriteString(fmt.Sprintf("%s=%v", v, t))
		default:
			jsonData, err := json.Marshal(t)
			if err != nil {
				buf.WriteString(fmt.Sprintf("%s=%v", v, t))
			} else {
				buf.WriteString(fmt.Sprintf("%s=%s", v, string(jsonData)))
			}
		}
	}
	buf.WriteString(")")

	return buf.String()
}

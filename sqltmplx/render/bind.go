package render

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var (
	errSpreadParamEmpty = errors.New("sqltmplx: spread parameter is empty")
	errSpreadParamType  = errors.New("sqltmplx: spread parameter must be slice or array")
)

func bindText(input string, st *state) (string, error) {
	var out strings.Builder
	for i := 0; i < len(input); {
		if i+1 < len(input) && input[i] == '#' && input[i+1] == '{' {
			end := strings.IndexByte(input[i+2:], '}')
			if end < 0 {
				return "", fmt.Errorf("sqltmplx: unterminated parameter placeholder")
			}
			raw := strings.TrimSpace(input[i+2 : i+2+end])
			spread := strings.HasSuffix(raw, "*")
			name := strings.TrimSpace(strings.TrimSuffix(raw, "*"))
			valOpt := lookup(st.params, name)
			if valOpt.IsAbsent() {
				return "", fmt.Errorf("sqltmplx: parameter %q not found", name)
			}
			val := valOpt.MustGet()
			if !spread {
				out.WriteString(st.nextBind())
				st.args = append(st.args, val)
			} else {
				rv := reflect.ValueOf(val)
				for rv.IsValid() && rv.Kind() == reflect.Pointer {
					if rv.IsNil() {
						return "", errSpreadParamEmpty
					}
					rv = rv.Elem()
				}
				if !rv.IsValid() {
					return "", errSpreadParamEmpty
				}
				if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
					return "", errSpreadParamType
				}
				if rv.Len() == 0 {
					return "", errSpreadParamEmpty
				}
				for j := 0; j < rv.Len(); j++ {
					if j > 0 {
						out.WriteString(", ")
					}
					out.WriteString(st.nextBind())
					st.args = append(st.args, rv.Index(j).Interface())
				}
			}
			i += 2 + end + 1
			continue
		}
		out.WriteByte(input[i])
		i++
	}
	return out.String(), nil
}

package render

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

var (
	errSpreadParamEmpty = errors.New("sqltmplx: spread parameter is empty")
	errSpreadParamType  = errors.New("sqltmplx: spread parameter must be slice or array")
)

func bindText(input string, st *state) (string, error) {
	var out strings.Builder
	for i := 0; i < len(input); {
		if i+1 < len(input) && input[i] == '/' && input[i+1] == '*' {
			text, next, handled, err := bindCommentPlaceholder(input, i, st)
			if err != nil {
				return "", err
			}
			if handled {
				out.WriteString(text)
				i = next
				continue
			}
		}
		out.WriteByte(input[i])
		i++
	}
	return out.String(), nil
}

func bindCommentPlaceholder(input string, i int, st *state) (string, int, bool, error) {
	endComment := strings.Index(input[i+2:], "*/")
	if endComment < 0 {
		return "", 0, false, fmt.Errorf("sqltmplx: unterminated sql comment")
	}
	commentEnd := i + 2 + endComment + 2
	raw := strings.TrimSpace(input[i+2 : i+2+endComment])
	if raw == "" || strings.HasPrefix(raw, "%") {
		return "", commentEnd, false, nil
	}
	if !isParamPath(raw) {
		return "", commentEnd, false, nil
	}

	j := commentEnd
	for j < len(input) && unicode.IsSpace(rune(input[j])) {
		j++
	}
	if j >= len(input) {
		return "", 0, false, fmt.Errorf("sqltmplx: placeholder %q missing test literal", raw)
	}

	spread := input[j] == '(' || looksLikeCollectionSample(input, j)
	text, err := bindParam(raw, spread, st)
	if err != nil {
		return "", 0, false, err
	}

	k, err := skipPlaceholderSample(input, j)
	if err != nil {
		return "", 0, false, fmt.Errorf("sqltmplx: placeholder %q invalid test literal: %w", raw, err)
	}
	return text, k, true, nil
}

func bindParam(name string, spread bool, st *state) (string, error) {
	valOpt := lookup(st.params, name)
	if valOpt.IsAbsent() {
		return "", fmt.Errorf("sqltmplx: parameter %q not found", name)
	}
	val := valOpt.MustGet()
	if !spread {
		st.args = append(st.args, val)
		return st.nextBind(), nil
	}
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
	var out strings.Builder
	for j := 0; j < rv.Len(); j++ {
		if j > 0 {
			out.WriteString(", ")
		}
		out.WriteString(st.nextBind())
		st.args = append(st.args, rv.Index(j).Interface())
	}
	return out.String(), nil
}

func isParamPath(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '.':
			if i == 0 || i == len(s)-1 {
				return false
			}
		case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
		default:
			return false
		}
	}
	return true
}

func looksLikeCollectionSample(input string, start int) bool {
	i := start
	if i < len(input) && isIdentifierStart(rune(input[i])) {
		j := i + 1
		for j < len(input) && isIdentifierPart(rune(input[j])) {
			j++
		}
		return j < len(input) && input[j] == '['
	}
	return false
}

func skipPlaceholderSample(input string, start int) (int, error) {
	i := start
	switch {
	case input[i] == '(':
		var err error
		i, err = skipBalanced(input, i, '(', ')')
		if err != nil {
			return 0, err
		}
	case input[i] == '\'' || input[i] == '"':
		var err error
		i, err = skipQuoted(input, i)
		if err != nil {
			return 0, err
		}
	case isIdentifierStart(rune(input[i])):
		var err error
		i, err = skipIdentifierExpr(input, i)
		if err != nil {
			return 0, err
		}
	default:
		var err error
		i, err = skipScalarToken(input, i)
		if err != nil {
			return 0, err
		}
	}
	return skipExprSuffixes(input, i)
}

func skipIdentifierExpr(input string, start int) (int, error) {
	i := start + 1
	for i < len(input) && isIdentifierPart(rune(input[i])) {
		i++
	}
	for {
		j := i
		for j < len(input) && unicode.IsSpace(rune(input[j])) {
			j++
		}
		if j >= len(input) || input[j] != '(' {
			return i, nil
		}
		next, err := skipBalanced(input, j, '(', ')')
		if err != nil {
			return 0, err
		}
		i = next
	}
}

func skipExprSuffixes(input string, start int) (int, error) {
	i := start
	for {
		next := i
		for next < len(input) && unicode.IsSpace(rune(input[next])) {
			next++
		}
		if next+1 < len(input) && input[next] == ':' && input[next+1] == ':' {
			i = next + 2
			if i >= len(input) || !isIdentifierStart(rune(input[i])) {
				return 0, fmt.Errorf("invalid cast suffix")
			}
			i++
			for i < len(input) {
				r := rune(input[i])
				if isIdentifierPart(r) || r == '.' {
					i++
					continue
				}
				if i+1 < len(input) && input[i] == '[' && input[i+1] == ']' {
					i += 2
					continue
				}
				break
			}
			continue
		}
		if next < len(input) && input[next] == '[' {
			i = next
			nextIndex, err := skipBalanced(input, i, '[', ']')
			if err != nil {
				return 0, err
			}
			i = nextIndex
			continue
		}
		return i, nil
	}
}

func isIdentifierStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentifierPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func skipBalanced(input string, start int, open, close byte) (int, error) {
	depth := 0
	for i := start; i < len(input); i++ {
		switch input[i] {
		case '\'', '"':
			j, err := skipQuoted(input, i)
			if err != nil {
				return 0, err
			}
			i = j - 1
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated balanced literal")
}

func skipQuoted(input string, start int) (int, error) {
	quote := input[start]
	for i := start + 1; i < len(input); i++ {
		if input[i] != quote {
			continue
		}
		if i+1 < len(input) && input[i+1] == quote {
			i++
			continue
		}
		return i + 1, nil
	}
	return 0, fmt.Errorf("unterminated quoted literal")
}

func skipScalarToken(input string, start int) (int, error) {
	i := start
	for i < len(input) {
		r := rune(input[i])
		if unicode.IsSpace(r) || r == ',' || r == ')' || r == '(' || r == ']' {
			break
		}
		i++
	}
	if i == start {
		return 0, fmt.Errorf("empty scalar literal")
	}
	return i, nil
}

package scan

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

func Scan(input string) ([]Token, error) {
	var tokens []Token
	for len(input) > 0 {
		start := strings.Index(input, "/*")
		if start < 0 {
			if input != "" {
				tokens = append(tokens, Token{Kind: Text, Value: input})
			}
			break
		}
		if start > 0 {
			tokens = append(tokens, Token{Kind: Text, Value: input[:start]})
		}
		input = input[start+2:]
		end := strings.Index(input, "*/")
		if end < 0 {
			return nil, fmt.Errorf("sqltmplx: unterminated directive comment")
		}
		raw := strings.TrimSpace(input[:end])
		input = input[end+2:]
		if isTemplateDirective(raw) {
			tokens = append(tokens, Token{Kind: Directive, Value: raw})
			continue
		}
		// Preserve ordinary SQL comments as text.
		tokens = append(tokens, Token{Kind: Text, Value: "/*" + raw + "*/"})
	}
	return tokens, nil
}

func isTemplateDirective(s string) bool {
	return lo.Contains([]string{"where", "set", "end"}, s) || strings.HasPrefix(s, "if ")
}

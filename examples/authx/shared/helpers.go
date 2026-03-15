package shared

import (
	"strings"

	"github.com/samber/lo"
)

func ParseBearer(raw string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	return token, token != ""
}

func HasRole(roles []string, target string) bool {
	return lo.Contains(roles, target)
}

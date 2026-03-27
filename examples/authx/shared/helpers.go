package shared

import (
	"strings"

	"github.com/samber/lo"
)

// ParseBearer extracts a bearer token from an Authorization header value.
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

// HasRole reports whether roles contains target.
func HasRole(roles []string, target string) bool {
	return lo.Contains(roles, target)
}

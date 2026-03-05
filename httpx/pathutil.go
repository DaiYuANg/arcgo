package httpx

import "strings"

func joinRoutePath(basePath, path string) string {
	base := normalizeRoutePrefix(basePath)

	if path == "" {
		if base == "" {
			return "/"
		}
		return base
	}

	cleanPath := path
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	if base == "" {
		return cleanPath
	}

	if cleanPath == "/" {
		return base
	}

	return base + cleanPath
}

func normalizeRoutePrefix(prefix string) string {
	clean := strings.TrimSpace(prefix)
	if clean == "" || clean == "/" {
		return ""
	}
	return "/" + strings.Trim(clean, "/")
}

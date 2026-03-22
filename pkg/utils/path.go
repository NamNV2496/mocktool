package utils

import "strings"

// MatchPath reports whether actualPath matches pattern.
//
// Rules:
//   - A segment prefixed with ':' (e.g. :id, :userId) matches exactly one
//     non-empty path segment.
//   - A bare '*' segment matches one or more remaining segments (glob).
//   - Query strings are stripped from both sides before comparison.
//
// Examples:
//
//	MatchPath("/api/users/:id",          "/api/users/123")        → true
//	MatchPath("/api/orders/:id/items",   "/api/orders/abc/items") → true
//	MatchPath("/api/users/*",            "/api/users/123/profile")→ true
//	MatchPath("/api/users/:id",          "/api/users/123/profile")→ false
func MatchPath(pattern, actualPath string) bool {
	// Strip query strings.
	pattern = strings.SplitN(pattern, "?", 2)[0]
	actualPath = strings.SplitN(actualPath, "?", 2)[0]

	patSegs := splitPath(pattern)
	actSegs := splitPath(actualPath)

	i, j := 0, 0
	for i < len(patSegs) {
		seg := patSegs[i]
		if seg == "*" {
			// Wildcard: consume all remaining actual segments.
			return true
		}
		if j >= len(actSegs) {
			return false
		}
		if !strings.HasPrefix(seg, ":") && seg != actSegs[j] {
			return false
		}
		i++
		j++
	}
	return j == len(actSegs)
}

func splitPath(p string) []string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	// Filter empty strings that result from leading/trailing slashes.
	out := parts[:0]
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

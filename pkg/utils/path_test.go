package utils

import "testing"

func TestMatchPath(t *testing.T) {
	cases := []struct {
		pattern string
		actual  string
		want    bool
	}{
		// Exact match still works.
		{"/api/users", "/api/users", true},
		{"/api/users", "/api/orders", false},

		// Single path param.
		{"/api/users/:id", "/api/users/123", true},
		{"/api/users/:id", "/api/users/abc-def", true},
		{"/api/users/:id", "/api/users/123/profile", false},

		// Multiple path params.
		{"/api/:resource/:id", "/api/users/42", true},
		{"/api/:resource/:id", "/api/orders/42", true},
		{"/api/:resource/:id", "/api/users", false},

		// Wildcard segment.
		{"/api/users/*", "/api/users/123", true},
		{"/api/users/*", "/api/users/123/profile", true},
		{"/api/users/*", "/api/orders/123", false},

		// Query strings are ignored.
		{"/api/users/:id", "/api/users/123?foo=bar", true},
		{"/api/users/:id?foo=bar", "/api/users/123?baz=qux", true},
	}

	for _, tc := range cases {
		got := MatchPath(tc.pattern, tc.actual)
		if got != tc.want {
			t.Errorf("MatchPath(%q, %q) = %v, want %v", tc.pattern, tc.actual, got, tc.want)
		}
	}
}

/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMatchPatternAgreesWithTheRouter is the point of the pattern language: the
// audit decides which declarations apply to a route, and the router decides
// which declarations actually run, so both have to read a pattern identically.
// The cases below are the ones the audit can decide exactly; a declaration whose
// placeholder carries a regular expression is treated as a wildcard by the
// audit, which errs towards reporting a route as covered (see §12.4).
func TestMatchPatternAgreesWithTheRouter(t *testing.T) {
	patterns := []string{"/", "/*", "/api/*", "/hello", "/hello/", "/hello/{name}", "/hello/{name}/x"}
	paths := []string{
		"/", "/hello", "/hello/", "/hello/world", "/hello/world/x",
		"/api", "/api/", "/api/v1", "/apix", "/other",
	}

	for _, pattern := range patterns {
		c := &Components{}
		c.Reset()
		c.Intercept([]string{"GET"}, pattern, &auditStub{})

		for _, path := range paths {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			assert.Equal(t, matchPattern(pattern, path), c.Interceptors(req) != nil,
				"pattern %q against path %q", pattern, path)
		}
	}
}

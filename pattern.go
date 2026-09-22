/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// The pattern language a component declaration uses (interceptor, preprocessor,
// postprocessor, hijacker). It is mux's own path template syntax plus one
// explicit wildcard, so a declaration says exactly what it covers:
//
//	/hello                    only /hello
//	/hello/{name}             /hello followed by exactly one segment
//	/hello/{name:[a-zA-Z]+}   the same, as mux matches the regular expression
//	/hello/*                  /hello itself and everything below it
//	/*                        every path
//
// A pattern that ended in "/" used to mean "everything below it", which made
// `- GET / X` cover the whole service. That rule only existed in the body of
// setComponent: reading a configuration file could not tell you whether an
// interceptor was global, and a trailing slash silently widened a route to a
// prefix. It is gone -- write /* when you mean everything.
const wildcardSuffix = "/*"

// prefixPart reports whether pattern is a prefix pattern and returns the path it
// covers: "/api/*" covers "/api/" (and, for convenience, "/api" itself).
func prefixPart(pattern string) (string, bool) {
	if !strings.HasSuffix(pattern, wildcardSuffix) {
		return "", false
	}
	return strings.TrimSuffix(pattern, "*"), true
}

// matchPattern reports whether path is served by pattern. The audit uses it to
// decide which declarations apply to a route; setComponent registers the same
// language with mux, and pattern_test.go checks that the two agree.
func matchPattern(pattern, path string) bool {
	if base, ok := prefixPart(pattern); ok {
		if base == "/" {
			return true
		}
		return path == strings.TrimSuffix(base, "/") || strings.HasPrefix(path, base)
	}
	patternParts := pathSegments(pattern)
	pathParts := pathSegments(path)
	if len(patternParts) != len(pathParts) {
		return false
	}
	for i := range patternParts {
		if strings.Contains(patternParts[i], "{") {
			// a placeholder stands for one non-empty segment, which is what
			// mux's own "{name}" regular expression requires
			if pathParts[i] == "" {
				return false
			}
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}
	return true
}

// pathSegments splits a pattern or a path into segments without normalising
// anything: "/hello/" and "/hello" are different paths, and mux treats them as
// different, so the audit must too.
func pathSegments(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

// registerPattern adds one pattern to a mux router with the meaning documented
// on this file.
func registerPattern(m *mux.Router, pattern string, handler http.Handler) []*mux.Route {
	if base, ok := prefixPart(pattern); ok {
		if base == "/" {
			return []*mux.Route{m.PathPrefix("/").Handler(handler)}
		}
		// "/api/*" covers "/api" as well as everything below it, because a
		// prefix that stops one path short of its own root is the kind of
		// surprise this change exists to remove
		return []*mux.Route{
			m.Handle(strings.TrimSuffix(base, "/"), handler),
			m.PathPrefix(base).Handler(handler),
		}
	}
	return []*mux.Route{m.Handle(pattern, handler)}
}

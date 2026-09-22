/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// auditStub is an interceptor used only to check that the audit can map a
// component back to the name it was registered under.
type auditStub struct {
	BaseInterceptor
}

func auditMappings(names ...string) map[string][][4]string {
	declarations := make([][4]string, 0, len(names))
	for _, name := range names {
		declarations = append(declarations, [4]string{"GET", "/hello", name, ""})
	}
	return map[string][][4]string{
		urlServiceMaps: {{"GET", "/hello", "TestService", "SayHello"}},
		interceptors:   declarations,
	}
}

func authFor(names ...string) authConfig {
	return authConfig{interceptors: names, publicRoutes: map[string]bool{}}
}

func TestMatchPattern(t *testing.T) {
	// a pattern is literal unless it says otherwise: "/" is the root, not
	// everything. It used to be a prefix, which made `- GET / X` global.
	assert.True(t, matchPattern("/", "/"))
	assert.False(t, matchPattern("/", "/anything"))
	assert.False(t, matchPattern("/", "/hello/world"))

	// an explicit wildcard covers the path itself and everything below it
	assert.True(t, matchPattern("/*", "/"))
	assert.True(t, matchPattern("/*", "/anything/at/all"))
	assert.True(t, matchPattern("/api/*", "/api"))
	assert.True(t, matchPattern("/api/*", "/api/v1/thing"))
	assert.False(t, matchPattern("/api/*", "/apix"))
	assert.False(t, matchPattern("/api/*", "/other"))

	assert.True(t, matchPattern("/hello", "/hello"))
	assert.False(t, matchPattern("/hello", "/hello/world"))
	assert.False(t, matchPattern("/hello", "/other"))

	// a "{...}" segment stands for exactly one segment of the request path
	assert.True(t, matchPattern("/hello/{your_Name:[a-zA-Z0-9]+}", "/hello/vaporz"))
	assert.True(t, matchPattern("/q/{code}", "/q/A3"))
	assert.False(t, matchPattern("/q/{code}", "/q/A3/extra"))
	assert.False(t, matchPattern("/q/{code}/x", "/q/A3"))
	assert.False(t, matchPattern("/hello/{name}", "/other/vaporz"))
}

func TestInterceptorChainUsesTheFirstMatchingDeclaration(t *testing.T) {
	// the first declaration that matches is the one whose chain runs, so it is
	// also the one the audit has to judge
	declarations := [][4]string{
		{"GET", "/*", "BaseInterceptor", ""},
		{"GET", "/hello", "TestInterceptor", ""},
	}
	assert.Equal(t, []string{"BaseInterceptor"}, interceptorChain(declarations, "GET", "/hello"))

	// a method that no declaration covers
	assert.Nil(t, interceptorChain(declarations, "POST", "/hello"))

	// and a path that none matches once no declaration is a prefix
	exact := [][4]string{{"GET", "/hello", "TestInterceptor", ""}}
	assert.Nil(t, interceptorChain(exact, "GET", "/other"))

	// a declaration may list several interceptors, separated by commas
	declarations = [][4]string{{"GET,POST", "/hello", "FirstInterceptor, SecondInterceptor", ""}}
	assert.Equal(t, []string{"FirstInterceptor", "SecondInterceptor"},
		interceptorChain(declarations, "POST", "/hello"))
}

func TestRouteKeyNormalisesSpelling(t *testing.T) {
	assert.Equal(t, "GET /health", routeKey("GET /health"))
	assert.Equal(t, "GET /health", routeKey("get   /health"))
	assert.Equal(t, "GET /health", routeKey(" GET /health "))
}

func TestAuditRoutesVerdicts(t *testing.T) {
	publicHello := authFor("AuthInterceptor")
	publicHello.publicRoutes["GET /hello"] = true

	named := &auditStub{}
	registered := map[string]interface{}{"AuthInterceptor": named}

	cases := []struct {
		name         string
		mappings     map[string][][4]string
		common       []Interceptor
		registered   map[string]interface{}
		auth         authConfig
		expectRefuse bool
		expectText   string
	}{
		{
			name:     "a declared auth interceptor protects the route",
			mappings: auditMappings("AuthInterceptor"),
			auth:     authFor("AuthInterceptor"),
		},
		{
			name:         "an interceptor that is not an auth interceptor does not",
			mappings:     auditMappings("OtherInterceptor"),
			auth:         authFor("AuthInterceptor"),
			expectRefuse: true,
			expectText:   "declare an auth interceptor",
		},
		{
			name:         "a route nobody declared an interceptor for is unprotected",
			mappings:     auditMappings(),
			auth:         authFor("AuthInterceptor"),
			expectRefuse: true,
			expectText:   "no common interceptor",
		},
		{
			name:     "a public route is allowed to have none",
			mappings: auditMappings(),
			auth:     publicHello,
		},
		{
			name:     "without an auth section the audit only reports",
			mappings: auditMappings(),
			auth:     authConfig{publicRoutes: map[string]bool{}},
		},
		{
			name:       "a common interceptor counts once it can be named",
			mappings:   auditMappings(),
			common:     []Interceptor{named},
			registered: registered,
			auth:       authFor("AuthInterceptor"),
		},
		{
			name:       "a common auth interceptor covers a route whose own interceptor is not auth",
			mappings:   auditMappings("OtherInterceptor"),
			common:     []Interceptor{named},
			registered: registered,
			auth:       authFor("AuthInterceptor"),
		},
		{
			name:     "an unnamed common interceptor cannot be verified, so it is not refused",
			mappings: auditMappings(),
			common:   []Interceptor{&auditStub{}},
			auth:     authFor("AuthInterceptor"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			registered := c.registered
			if registered == nil {
				registered = map[string]interface{}{}
			}
			err := auditRoutes(c.mappings, c.common, registered, c.auth)
			if !c.expectRefuse {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
			assert.Contains(t, err.Error(), c.expectText)
			assert.Contains(t, err.Error(), authPublicRoutesKey)
		})
	}
}

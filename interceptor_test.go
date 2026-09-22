/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func interceptorRequest(c *Components, target string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Form = req.URL.Query()
	return req.WithContext(context.WithValue(req.Context(), componentsKey, c))
}

func componentsWith(common []Interceptor, declarations ...Interceptor) *Components {
	c := &Components{}
	c.Reset()
	if len(common) > 0 {
		c.SetCommonInterceptor(common...)
	}
	if len(declarations) > 0 {
		c.Intercept([]string{"GET"}, "/hello", declarations...)
	}
	return c
}

func TestGetInterceptorsRunsTheCommonOnesFirst(t *testing.T) {
	common, routeLevel := &auditStub{}, &auditStub{}
	c := componentsWith([]Interceptor{common}, routeLevel)

	// adding one line of configuration to a route must not switch off the
	// cross-cutting interceptors the service installed globally
	assert.Equal(t, []Interceptor{common, routeLevel}, getInterceptors(interceptorRequest(c, "/hello")))
}

func TestGetInterceptorsReturnsANewSlice(t *testing.T) {
	common := &auditStub{}
	c := componentsWith([]Interceptor{common}, &auditStub{})

	chain := getInterceptors(interceptorRequest(c, "/hello"))
	chain[0] = &auditStub{}

	// the chain must not alias what Components holds, or one request could
	// change what the next one runs
	assert.Equal(t, []Interceptor{common}, c.CommonInterceptors())
}

func TestGetInterceptorsForARouteWithNoDeclarationOfItsOwn(t *testing.T) {
	common := &auditStub{}
	c := componentsWith([]Interceptor{common}, &auditStub{})

	// a path the declaration does not match is served by the common ones alone
	assert.Equal(t, []Interceptor{common}, getInterceptors(interceptorRequest(c, "/other")))
}

func TestGetInterceptorsWithoutCommonOnes(t *testing.T) {
	routeLevel := &auditStub{}
	c := componentsWith(nil, routeLevel)
	assert.Equal(t, []Interceptor{routeLevel}, getInterceptors(interceptorRequest(c, "/hello")))
}

func TestGetInterceptorsWithNeither(t *testing.T) {
	c := componentsWith(nil)
	assert.Empty(t, getInterceptors(interceptorRequest(c, "/hello")))
}

func TestMatchingInterceptorChainsReportsEveryMatch(t *testing.T) {
	declarations := [][4]string{
		{"GET", "/*", "PrefixInterceptor", ""},
		{"GET", "/hello", "ExactInterceptor", ""},
	}
	assert.Equal(t, [][]string{{"PrefixInterceptor"}, {"ExactInterceptor"}},
		matchingInterceptorChains(declarations, "GET", "/hello"))

	// only the first one runs, which is what the audit warns about
	assert.Equal(t, []string{"PrefixInterceptor"}, interceptorChain(declarations, "GET", "/hello"))
}

func TestAuditWarnsWhenSeveralDeclarationsMatchOneRoute(t *testing.T) {
	logged := &bytes.Buffer{}
	previous := log.Out
	log.SetOutput(logged)
	defer log.SetOutput(previous)

	mappings := map[string][][4]string{
		urlServiceMaps: {{"GET", "/hello", "TestService", "SayHello"}},
		interceptors: {
			{"GET", "/*", "PrefixInterceptor", ""},
			{"GET", "/hello", "ExactInterceptor", ""},
		},
	}
	auth := authConfig{publicRoutes: map[string]bool{}}

	assert.NoError(t, auditRoutes(mappings, nil, map[string]interface{}{}, auth))
	// the second declaration is ignored, which used to be invisible
	assert.Contains(t, logged.String(), "is matched by 2 interceptor declarations")
	assert.Contains(t, logged.String(), "only the first one runs")
}

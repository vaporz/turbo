/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"fmt"
	"strings"
)

// An interceptor is registered against a method and a url pattern, and the first
// declaration that matches a request is the one that runs -- see
// Components.interceptors and setComponent. The audit below therefore resolves a
// route the same way: the first declaration that matches it is its chain.
//
// The point of the audit is the failure mode that a unit test cannot catch:
// adding a route to urlmapping and forgetting the matching interceptor line
// leaves that endpoint with no authentication at all, silently, because "not
// declared" was never the same thing as "not needed".
const (
	authInterceptorsKey = "auth.interceptors"
	authPublicRoutesKey = "auth.public_routes"
)

// authConfig is the optional auth section of service.yaml.
type authConfig struct {
	// interceptors names the components that authenticate a request. An empty
	// list means the deployment has not declared any, and the audit is advisory.
	interceptors []string
	// publicRoutes holds "METHOD /path" keys that are allowed to have no
	// authentication.
	publicRoutes map[string]bool
}

func (c *Config) authConfig() authConfig {
	auth := authConfig{
		interceptors: c.GetStringSlice(authInterceptorsKey),
		publicRoutes: make(map[string]bool),
	}
	for _, route := range c.GetStringSlice(authPublicRoutesKey) {
		auth.publicRoutes[routeKey(route)] = true
	}
	return auth
}

// enforcing reports whether the audit may refuse a configuration.
func (a authConfig) enforcing() bool {
	return len(a.interceptors) > 0
}

// routeKey normalises "get /health" and "GET   /health" to one key.
func routeKey(route string) string {
	fields := strings.Fields(route)
	if len(fields) < 2 {
		return strings.ToUpper(strings.TrimSpace(route))
	}
	return strings.ToUpper(fields[0]) + " " + fields[1]
}

// pathSegments splits a url pattern or a path into segments.
func pathSegments(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// patternMatchesPath mirrors setComponent: a pattern ending in "/" is a path
// prefix, and any other pattern matches a path of the same shape, where a
// "{...}" segment stands for exactly one segment of the request path.
func patternMatchesPath(pattern, path string) bool {
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}
	patternParts := pathSegments(pattern)
	pathParts := pathSegments(path)
	if len(patternParts) != len(pathParts) {
		return false
	}
	for i := range patternParts {
		if strings.HasPrefix(patternParts[i], "{") {
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}
	return true
}

func methodMatches(declared []string, method string) bool {
	if len(declared) == 0 {
		return true
	}
	for _, m := range declared {
		if strings.EqualFold(strings.TrimSpace(m), method) {
			return true
		}
	}
	return false
}

// interceptorChain returns the names of the interceptors that actually run for a
// route: the first declaration matching it, or nil when the configuration
// declares none and the common interceptors apply instead.
func interceptorChain(declarations [][4]string, method, path string) []string {
	for _, declaration := range declarations {
		if !methodMatches(strings.Split(declaration[0], ","), method) {
			continue
		}
		if !patternMatchesPath(declaration[1], path) {
			continue
		}
		names := make([]string, 0, 1)
		for _, name := range strings.Split(declaration[2], ",") {
			if name = strings.TrimSpace(name); name != "" {
				names = append(names, name)
			}
		}
		return names
	}
	return nil
}

// namesOf maps components back to the names they were registered under, so that
// the common interceptors -- which are installed from code, not from the
// configuration -- can be checked against the declared auth interceptors too.
// A component registered twice under different names maps to both.
func namesOf(registered map[string]interface{}, list []Interceptor) (names []string, complete bool) {
	complete = true
	for _, interceptor := range list {
		found := false
		for name, component := range registered {
			if component == interface{}(interceptor) {
				names = append(names, name)
				found = true
			}
		}
		if !found {
			complete = false
		}
	}
	return names, complete
}

func hasAuthInterceptor(chain, auth []string) bool {
	for _, name := range chain {
		for _, authName := range auth {
			if name == authName {
				return true
			}
		}
	}
	return false
}

// auditRoutes reports what the configuration says about every route that the
// router will serve, and, once an auth section declares which interceptors
// authenticate a request, refuses a configuration that would leave a route
// unprotected. Being refused at startup is the point: it is the only mechanism
// that survives somebody adding a route and forgetting its interceptor line.
func auditRoutes(mappings map[string][][4]string, common []Interceptor,
	registered map[string]interface{}, auth authConfig) error {
	commonNames, commonNamed := namesOf(registered, common)
	var violations []string

	for _, route := range mappings[urlServiceMaps] {
		for _, method := range strings.Split(route[0], ",") {
			method = strings.ToUpper(strings.TrimSpace(method))
			path, serviceName, methodName := route[1], route[2], route[3]
			chain := interceptorChain(mappings[interceptors], method, path)
			effective, source := chain, "declared"
			if len(chain) == 0 {
				effective, source = commonNames, "common"
			}
			public := auth.publicRoutes[method+" "+path]

			switch {
			case public:
				log.Infof("route audit: %s %s -> %s.%s [public, auth=%s:%v]",
					method, path, serviceName, methodName, source, effective)
				continue
			case !auth.enforcing():
				log.Infof("route audit: %s %s -> %s.%s [auth=%s:%v]",
					method, path, serviceName, methodName, source, effective)
				continue
			case hasAuthInterceptor(effective, auth.interceptors):
				log.Infof("route audit: %s %s -> %s.%s [auth=%s:%v, authenticated]",
					method, path, serviceName, methodName, source, effective)
				continue
			}

			reason := fmt.Sprintf("no interceptor is declared for it (auth=%s:%v)", source, effective)
			switch {
			case len(chain) > 0:
				reason = fmt.Sprintf("its interceptor chain %v declares no auth interceptor", chain)
			case len(common) == 0:
				reason = "it declares no interceptor and the server has no common interceptor"
			case !commonNamed:
				// The server does have common interceptors, but at least one of
				// them was registered anonymously, so nothing can be checked.
				log.Warnf("route audit: %s %s -> %s.%s [auth=common:%v, cannot be verified: "+
					"a common interceptor is not registered under a name]",
					method, path, serviceName, methodName, commonNames)
				continue
			}
			log.Errorf("route audit: %s %s -> %s.%s [UNPROTECTED: %s]",
				method, path, serviceName, methodName, reason)
			violations = append(violations, fmt.Sprintf("%s %s -> %s.%s: %s",
				method, path, serviceName, methodName, reason))
		}
	}

	if len(violations) == 0 {
		return nil
	}
	return fmt.Errorf("refusing this configuration: %d route(s) would be served without any "+
		"declared auth interceptor (list a route under %s if it is meant to be public):\n  %s",
		len(violations), authPublicRoutesKey, strings.Join(violations, "\n  "))
}

package turbo

import (
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
	"strings"
)

// Components holds all component mappings
type Components struct {
	commonInterceptors []Interceptor
	routers            map[int]*mux.Router
	convertorMap       map[reflect.Type]convertor
	errorHandler       errorHandlerFunc
}

const (
	rInterceptor   = iota
	rPreprocessor
	rPostprocessor
	rHijacker
)

// TODO setup component mappings via service.yaml
// TODO reload mappings on config change
// Interceptor -----------------

// Interceptor intercepts requests, can run a func before and after a request
type Interceptor interface {
	Before(http.ResponseWriter, *http.Request) (*http.Request, error)
	After(http.ResponseWriter, *http.Request) (*http.Request, error)
}

// BaseInterceptor implements an empty Before() and After()
type BaseInterceptor struct{}

// Before will run before a request performs
func (i *BaseInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	return req, nil
}

// After will run after a request performs
func (i *BaseInterceptor) After(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	return req, nil
}

func (c *Components) setCommonInterceptor(interceptors ...Interceptor) {
	c.commonInterceptors = interceptors
}

func (c *Components) commonInterceptor() []Interceptor {
	if c.commonInterceptors != nil {
		return c.commonInterceptors
	}
	return []Interceptor{}
}

type interceptors []Interceptor

// ServeHTTP is an empty func, only for implementing http.Handler
func (i interceptors) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (c *Components) intercept(methods []string, urlPattern string, list ...Interceptor) {
	if c.routers == nil {
		c.routers = make(map[int]*mux.Router)
	}
	c.routers[rInterceptor] = setComponent(c.routers[rInterceptor], methods, urlPattern, interceptors(list))
}

func (c *Components) interceptors(req *http.Request) interceptors {
	if cp := component(c.routers[rInterceptor], req); cp != nil {
		return cp.(interceptors)
	}
	return nil
}

// PreProcessor-------------
type preprocessor func(http.ResponseWriter, *http.Request) error

// ServeHTTP is an empty func, only for implementing http.Handler
func (p preprocessor) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (c *Components) setPreprocessor(methods []string, urlPattern string, pre preprocessor) {
	if c.routers == nil {
		c.routers = make(map[int]*mux.Router)
	}
	c.routers[rPreprocessor] = setComponent(c.routers[rPreprocessor], methods, urlPattern, pre)
}

func (c *Components) preprocessor(req *http.Request) preprocessor {
	if cp := component(c.routers[rPreprocessor], req); cp != nil {
		return cp.(preprocessor)
	}
	return nil
}

// PostProcessor--------------
type postprocessor func(http.ResponseWriter, *http.Request, interface{}, error)

// ServeHTTP is an empty func, only for implementing http.Handler
func (p postprocessor) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (c *Components) setPostprocessor(methods []string, urlPattern string, post postprocessor) {
	if c.routers == nil {
		c.routers = make(map[int]*mux.Router)
	}
	c.routers[rPostprocessor] = setComponent(c.routers[rPostprocessor], methods, urlPattern, post)
}

func (c *Components) postprocessor(req *http.Request) postprocessor {
	if cp := component(c.routers[rPostprocessor], req); cp != nil {
		return cp.(postprocessor)
	}
	return nil
}

// Hijacker-----------------
type hijacker func(http.ResponseWriter, *http.Request)

// ServeHTTP is an empty func, only for implementing http.Handler
func (h hijacker) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (c *Components) setHijacker(methods []string, urlPattern string, h hijacker) {
	if c.routers == nil {
		c.routers = make(map[int]*mux.Router)
	}
	c.routers[rHijacker] = setComponent(c.routers[rHijacker], methods, urlPattern, h)
}

func (c *Components) hijacker(req *http.Request) hijacker {
	if cp := component(c.routers[rHijacker], req); cp != nil {
		return cp.(hijacker)
	}
	return nil
}

func setComponent(m *mux.Router, methods []string, urlPattern string, handler http.Handler) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}
	var route *mux.Route
	if strings.HasSuffix(urlPattern, "/") {
		route = m.PathPrefix(urlPattern).Handler(handler)
	} else {
		route = m.Handle(urlPattern, handler)
	}
	if len(methods) > 0 {
		route.Methods(methods...)
	}
	return m
}

func component(r *mux.Router, req *http.Request) http.Handler {
	var m mux.RouteMatch
	if r != nil && r.Match(req, &m) {
		return m.Handler
	}
	return nil
}

// Convertor--------------
type convertor func(r *http.Request) reflect.Value

func (c *Components) registerMessageFieldConvertor(field interface{}, convertorFunc convertor) {
	if c.convertorMap == nil {
		c.convertorMap = make(map[reflect.Type]convertor)
	}
	c.convertorMap[reflect.TypeOf(field).Elem()] = convertorFunc
}

func (c *Components) messageFieldConvertor(theType reflect.Type) convertor {
	if c.convertorMap == nil {
		return nil
	}
	return c.convertorMap[theType]
}

// ErrorHandler----------
type errorHandlerFunc func(http.ResponseWriter, *http.Request, error)

func defaultErrorHandler(resp http.ResponseWriter, req *http.Request, err error) {
	http.Error(resp, err.Error(), http.StatusInternalServerError)
}

func (c *Components) errorHandlerFunc() errorHandlerFunc {
	if c.errorHandler == nil {
		return defaultErrorHandler
	}
	return c.errorHandler
}

// WithErrorHandler registers an errorHandler to handle errors
func WithErrorHandler(e errorHandlerFunc) {
	client.components.errorHandler = e
}

// SetCommonInterceptor assigns interceptors to all URLs, if the URL has no other interceptors assigned
func SetCommonInterceptor(interceptors ...Interceptor) {
	client.components.setCommonInterceptor(interceptors...)
}

// CommonInterceptors returns a list of interceptors which are default
func CommonInterceptors() []Interceptor {
	return client.components.commonInterceptor()
}

// Intercept registers a list of interceptors to an URL pattern at given HTTP methods
func Intercept(methods []string, urlPattern string, list ...Interceptor) {
	client.components.intercept(methods, urlPattern, list...)
}

// Interceptors returns a list of interceptors for this request
func Interceptors(req *http.Request) interceptors {
	return client.components.interceptors(req)
}

// SetPreprocessor registers a preprocessor to an URL pattern
func SetPreprocessor(methods []string, urlPattern string, pre preprocessor) {
	client.components.setPreprocessor(methods, urlPattern, pre)
}

// Preprocessor returns a preprocessor for this request
func Preprocessor(req *http.Request) preprocessor {
	return client.components.preprocessor(req)
}

// SetPostprocessor registers a postprocessor to an URL pattern
func SetPostprocessor(methods []string, urlPattern string, post postprocessor) {
	client.components.setPostprocessor(methods, urlPattern, post)
}

// Postprocessor returns a postprocessor for this request
func Postprocessor(req *http.Request) postprocessor {
	return client.components.postprocessor(req)
}

// SetHijacker registers a hijacker to an URL pattern
func SetHijacker(methods []string, urlPattern string, h hijacker) {
	client.components.setHijacker(methods, urlPattern, h)
}

// Hijacker returns a hijacker for this request
func Hijacker(req *http.Request) hijacker {
	return client.components.hijacker(req)
}

// RegisterMessageFieldConvertor registers a convertor on a type
// usage: RegisterMessageFieldConvertor(new(SomeInterface), convertorFunc)
func RegisterMessageFieldConvertor(field interface{}, convertorFunc convertor) {
	client.components.registerMessageFieldConvertor(field, convertorFunc)
}

// MessageFieldConvertor returns the convertor for this type
func MessageFieldConvertor(theType reflect.Type) convertor {
	return client.components.messageFieldConvertor(theType)
}

// ResetComponents reset all component mappings
func ResetComponents() {
	client.components = new(Components)
}

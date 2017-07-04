package turbo

import (
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
	"strings"
)

// Components holds all component mappings
type Components struct {
	commonInterceptors   Interceptors
	routers              map[int]*mux.Router
	convertorMap         map[string]Convertor
	errorHandler         ErrorHandlerFunc
	RegisteredComponents map[string]interface{}
}

func (c *Components) RegisterComponent(name string, component interface{}) {
	if c.RegisteredComponents == nil {
		c.RegisteredComponents = make(map[string]interface{})
	}
	c.RegisteredComponents[name] = component
}

func (c *Components) Component(name string) interface{} {
	if c.RegisteredComponents == nil {
		return nil
	}
	return c.RegisteredComponents[name]
}

// Reset resets all component mappings
func (c *Components) Reset() {
	c.commonInterceptors = Interceptors{}
	c.routers = make(map[int]*mux.Router)
	c.convertorMap = make(map[string]Convertor)
	c.errorHandler = nil
}

const (
	rInterceptor = iota
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

type Interceptors []Interceptor

func (c *Components) setCommonInterceptor(interceptors Interceptors) {
	c.commonInterceptors = interceptors
}

func (c *Components) commonInterceptor() Interceptors {
	if c.commonInterceptors != nil {
		return c.commonInterceptors
	}
	return Interceptors{}
}

// ServeHTTP is an empty func, only for implementing http.Handler
func (i Interceptors) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (c *Components) intercept(methods []string, urlPattern string, list ...Interceptor) {
	if c.routers == nil {
		c.routers = make(map[int]*mux.Router)
	}
	c.routers[rInterceptor] = setComponent(c.routers[rInterceptor], methods, urlPattern, Interceptors(list))
}

func (c *Components) interceptors(req *http.Request) Interceptors {
	if cp := component(c.routers[rInterceptor], req); cp != nil {
		return cp.(Interceptors)
	}
	return nil
}

// PreProcessor-------------
type Preprocessor func(http.ResponseWriter, *http.Request) error

// ServeHTTP is an empty func, only for implementing http.Handler
func (p Preprocessor) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (c *Components) setPreprocessor(methods []string, urlPattern string, pre Preprocessor) {
	if c.routers == nil {
		c.routers = make(map[int]*mux.Router)
	}
	c.routers[rPreprocessor] = setComponent(c.routers[rPreprocessor], methods, urlPattern, pre)
}

func (c *Components) preprocessor(req *http.Request) Preprocessor {
	if cp := component(c.routers[rPreprocessor], req); cp != nil {
		return cp.(Preprocessor)
	}
	return nil
}

// PostProcessor--------------
type Postprocessor func(http.ResponseWriter, *http.Request, interface{}, error)

// ServeHTTP is an empty func, only for implementing http.Handler
func (p Postprocessor) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (c *Components) setPostprocessor(methods []string, urlPattern string, post Postprocessor) {
	if c.routers == nil {
		c.routers = make(map[int]*mux.Router)
	}
	c.routers[rPostprocessor] = setComponent(c.routers[rPostprocessor], methods, urlPattern, post)
}

func (c *Components) postprocessor(req *http.Request) Postprocessor {
	if cp := component(c.routers[rPostprocessor], req); cp != nil {
		return cp.(Postprocessor)
	}
	return nil
}

// Hijacker-----------------
type Hijacker func(http.ResponseWriter, *http.Request)

// ServeHTTP is an empty func, only for implementing http.Handler
func (h Hijacker) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (c *Components) setHijacker(methods []string, urlPattern string, h Hijacker) {
	if c.routers == nil {
		c.routers = make(map[int]*mux.Router)
	}
	c.routers[rHijacker] = setComponent(c.routers[rHijacker], methods, urlPattern, h)
}

func (c *Components) hijacker(req *http.Request) Hijacker {
	if cp := component(c.routers[rHijacker], req); cp != nil {
		return cp.(Hijacker)
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
type Convertor func(r *http.Request) reflect.Value

func (c *Components) registerMessageFieldConvertor(field string, convertorFunc Convertor) {
	if c.convertorMap == nil {
		c.convertorMap = make(map[string]Convertor)
	}
	c.convertorMap[field] = convertorFunc
}

func (c *Components) messageFieldConvertor(theType string) Convertor {
	if c.convertorMap == nil {
		return nil
	}
	return c.convertorMap[theType]
}

// ErrorHandler----------
type ErrorHandlerFunc func(http.ResponseWriter, *http.Request, error)

func defaultErrorHandler(resp http.ResponseWriter, req *http.Request, err error) {
	http.Error(resp, err.Error(), http.StatusInternalServerError)
}

func (c *Components) errorHandlerFunc() ErrorHandlerFunc {
	if c.errorHandler == nil {
		return defaultErrorHandler
	}
	return c.errorHandler
}

// WithErrorHandler registers an errorHandler to handle errors
func (c *Components) WithErrorHandler(e ErrorHandlerFunc) {
	c.errorHandler = e
}

// SetCommonInterceptor assigns Interceptors to all URLs, if the URL has no other Interceptors assigned
func (c *Components) SetCommonInterceptor(interceptors ...Interceptor) {
	c.setCommonInterceptor(interceptors)
}

// CommonInterceptors returns a list of Interceptors which are default
func (c *Components) CommonInterceptors() []Interceptor {
	return c.commonInterceptor()
}

// Intercept registers a list of Interceptors to an URL pattern at given HTTP methods
func (c *Components) Intercept(methods []string, urlPattern string, list ...Interceptor) {
	c.intercept(methods, urlPattern, list...)
}

// Interceptors returns a list of Interceptors for this request
func (c *Components) Interceptors(req *http.Request) Interceptors {
	return c.interceptors(req)
}

// SetPreprocessor registers a preprocessor to an URL pattern
func (c *Components) SetPreprocessor(methods []string, urlPattern string, pre Preprocessor) {
	c.setPreprocessor(methods, urlPattern, pre)
}

// Preprocessor returns a preprocessor for this request
func (c *Components) Preprocessor(req *http.Request) Preprocessor {
	return c.preprocessor(req)
}

// SetPostprocessor registers a Postprocessor to an URL pattern
func (c *Components) SetPostprocessor(methods []string, urlPattern string, post Postprocessor) {
	c.setPostprocessor(methods, urlPattern, post)
}

// Postprocessor returns a Postprocessor for this request
func (c *Components) Postprocessor(req *http.Request) Postprocessor {
	return c.postprocessor(req)
}

// SetHijacker registers a Hijacker to an URL pattern
func (c *Components) SetHijacker(methods []string, urlPattern string, h Hijacker) {
	c.setHijacker(methods, urlPattern, h)
}

// Hijacker returns a Hijacker for this request
func (c *Components) Hijacker(req *http.Request) Hijacker {
	return c.hijacker(req)
}

// RegisterMessageFieldConvertor registers a Convertor on a type
// usage: RegisterMessageFieldConvertor(new(SomeInterface), convertorFunc)
func (c *Components) RegisterMessageFieldConvertor(field string, convertorFunc Convertor) {
	c.registerMessageFieldConvertor(field, convertorFunc)
}

// MessageFieldConvertor returns the Convertor for this type
func (c *Components) MessageFieldConvertor(theType string) Convertor {
	return c.messageFieldConvertor(theType)
}

package turbo

import (
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
	"strings"
)

// Interceptor -----------------

// Interceptor intercepts requests, can run a func before and after a request
type Interceptor interface {
	Before(http.ResponseWriter, *http.Request) (*http.Request, error)
	After(http.ResponseWriter, *http.Request) (*http.Request, error)
}

// BaseInterceptor implements an empty Before() and After()
type BaseInterceptor struct{}

// Before will run before a request performs
func (i BaseInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	return req, nil
}

// After will run after a request performs
func (i BaseInterceptor) After(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	return req, nil
}

var commonInterceptors []Interceptor = []Interceptor{}

// SetCommonInterceptor assigns interceptors to all URLs, if the URL has no other interceptors assigned
func SetCommonInterceptor(interceptors ...Interceptor) {
	commonInterceptors = interceptors
}

// CommonInterceptors returns a list of interceptors which are default
func CommonInterceptors() []Interceptor {
	return commonInterceptors
}

var interceptorMap *mux.Router = mux.NewRouter()

type interceptors []Interceptor

// ServeHTTP is an empty func, only for implementing http.Handler
func (i interceptors) ServeHTTP(http.ResponseWriter, *http.Request) {}

// Intercept registers a list of interceptors to an URL pattern at given HTTP methods
func Intercept(methods []string, urlPattern string, list ...Interceptor) {
	var route *mux.Route
	if strings.HasSuffix(urlPattern, "/") {
		route = interceptorMap.PathPrefix(urlPattern).Handler(interceptors(list))
	} else {
		route = interceptorMap.Handle(urlPattern, interceptors(list))
	}
	if len(methods) > 0 {
		route.Methods(methods...)
	}
}

// Interceptors returns a list of interceptors for this request
func Interceptors(req *http.Request) interceptors {
	var m mux.RouteMatch
	if interceptorMap.Match(req, &m) {
		return m.Handler.(interceptors)
	}
	return []Interceptor{}
}

// PreProcessor-------------
var preprocessorMap *mux.Router = mux.NewRouter()

type preprocessor func(http.ResponseWriter, *http.Request) error

// ServeHTTP is an empty func, only for implementing http.Handler
func (p preprocessor) ServeHTTP(http.ResponseWriter, *http.Request) {}

// SetPreprocessor registers a preprocessor to an URL pattern
func SetPreprocessor(urlPattern string, pre preprocessor) {
	preprocessorMap.Handle(urlPattern, pre)
}

// Preprocessor returns a preprocessor for this request
func Preprocessor(req *http.Request) preprocessor {
	var m mux.RouteMatch
	if preprocessorMap.Match(req, &m) {
		return m.Handler.(preprocessor)
	}
	return nil
}

// PostProcessor--------------
var postprocessorMap *mux.Router = mux.NewRouter()

type postprocessor func(http.ResponseWriter, *http.Request, interface{}, error)

// ServeHTTP is an empty func, only for implementing http.Handler
func (p postprocessor) ServeHTTP(http.ResponseWriter, *http.Request) {}

// SetPostprocessor registers a postprocessor to an URL pattern
func SetPostprocessor(urlPattern string, post postprocessor) {
	postprocessorMap.Handle(urlPattern, post)
}

// Postprocessor returns a postprocessor for this request
func Postprocessor(req *http.Request) postprocessor {
	var m mux.RouteMatch
	if postprocessorMap.Match(req, &m) {
		return m.Handler.(postprocessor)
	}
	return nil
}

// Hijacker-----------------
var hijackerMap *mux.Router = mux.NewRouter()

type hijacker func(http.ResponseWriter, *http.Request)

// ServeHTTP is an empty func, only for implementing http.Handler
func (h hijacker) ServeHTTP(http.ResponseWriter, *http.Request) {}

// SetHijacker registers a hijacker to an URL pattern
func SetHijacker(urlPattern string, h hijacker) {
	hijackerMap.Handle(urlPattern, h)
}

// Hijacker returns a hijacker for this request
func Hijacker(req *http.Request) hijacker {
	var m mux.RouteMatch
	if hijackerMap.Match(req, &m) {
		return m.Handler.(hijacker)
	}
	return nil
}

// Convertor--------------
type convertor func(r *http.Request) reflect.Value

var convertorMap map[reflect.Type]convertor = make(map[reflect.Type]convertor)

// RegisterMessageFieldConvertor registers a convertor on a type
func RegisterMessageFieldConvertor(field interface{}, convertorFunc convertor) {
	convertorMap[reflect.TypeOf(field).Elem()] = convertorFunc
}

// MessageFieldConvertor returns the convertor for this type
func MessageFieldConvertor(theType reflect.Type) convertor {
	return convertorMap[theType]
}

package turbo

import (
	"github.com/gorilla/mux"
	"github.com/kylelemons/go-gypsy/yaml"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
)

const grpcServiceName string = "grpc_service_name"
const grpcServiceAddress string = "grpc_service_address"
const thriftServiceName string = "thrift_service_name"
const thriftServiceAddress string = "thrift_service_address"
const port string = "port"

var rpcType string

var config yaml.File

var urlServiceMaps [][3]string

// absolute path
var serviceRootPath string

var servicePkgPath string

var configs map[string]string = make(map[string]string)

var fieldMappings map[string][]string = make(map[string][]string)

// InitRpcType initializes rpc type
func InitRpcType(r string) {
	rpcType = r
}

// LoadServiceConfigWith accepts a package path, then load service.yaml in that path
func LoadServiceConfigWith(pkgPath string) {
	InitPkgPath(pkgPath)
	loadServiceConfig()
}

// InitPkgPath parse package path, if $GOPATH contains multi paths, the first one will be used
func InitPkgPath(pkgPath string) {
	initPkgPath(pkgPath)
}

func initPkgPath(pkgPath string) {
	goPath := os.Getenv("GOPATH")
	paths := strings.Split(goPath, ":")
	serviceRootPath = paths[0] + "/src/" + pkgPath
	servicePkgPath = pkgPath
}

func loadServiceConfig() {
	// TODO use viper to load config file, https://github.com/spf13/viper
	conf, err := yaml.ReadFile(serviceRootPath + "/service.yaml")
	if err != nil {
		log.Fatalf("readfile(%q): %s", serviceRootPath+"/service.yaml", err)
	}
	config = *conf
	initUrlMap()
	initConfigs()
	initFieldMapping()
}

func initConfigs() {
	cnf, err := yaml.Child(config.Root, "config")
	if err != nil {
		log.Fatalf("parse config error: %s", err)
	}
	configMap := cnf.(yaml.Map)
	for k, v := range configMap {
		configs[k] = (v.(yaml.Scalar)).String()
	}
}

func initUrlMap() {
	node, err := yaml.Child(config.Root, "urlmapping")
	if err != nil {
		log.Fatalf("parse urlmapping error: %s", err)
	}
	urlMap := node.(yaml.List)
	for _, line := range urlMap {
		appendUrlServiceMap(strings.TrimSpace(yaml.Render(line)))
	}
}

func initFieldMapping() {
	node, err := yaml.Child(config.Root, rpcType+"-fieldmapping")
	if err != nil {
		return
	}
	fieldMappingMap := node.(yaml.Map)
	for k, v := range fieldMappingMap {
		valueStrList := make([]string, 0)
		if v != nil {
			valueList := v.(yaml.List)
			for _, line := range valueList {
				valueStrList = append(valueStrList, strings.TrimSpace(yaml.Render(line)))
			}
		}
		fieldMappings[k] = valueStrList
	}
}

func appendUrlServiceMap(line string) {
	values := strings.Split(line, " ")
	HTTPMethod := strings.TrimSpace(values[0])
	url := strings.TrimSpace(values[1])
	methodName := strings.TrimSpace(values[2])
	urlServiceMaps = append(urlServiceMaps, [3]string{HTTPMethod, url, methodName})
}

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

type postprocessor func(http.ResponseWriter, *http.Request, interface{})

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

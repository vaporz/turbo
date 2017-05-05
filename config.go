package turbo

import (
	"github.com/gorilla/mux"
	"github.com/kylelemons/go-gypsy/yaml"
	"log"
	"net/http"
	"os"
	"strings"
)

const SERVICE_NAME string = "service_name"
const SERVICE_ADDRESS string = "service_address"
const PORT string = "port"

var config yaml.File

var UrlServiceMap [][3]string

// absolute path
var serviceRootPath string

var servicePkgPath string

var configs map[string]string = make(map[string]string)

//var requestPreprocessor map[string]func(http.ResponseWriter, *http.Request) error = make(map[string]func(http.ResponseWriter, *http.Request) error)

type emptyHandler struct{}

func (e emptyHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

var commonInterceptors []Interceptor = []Interceptor{}

var interceptorMap *mux.Router = mux.NewRouter()

type interceptorList struct {
	emptyHandler
	interceptors []Interceptor
}

func Intercept(urlPattern string, list ...Interceptor) {
	interceptorMap.Handle(urlPattern, &interceptorList{interceptors: list})
}

func SetCommonInterceptor(interceptors ...Interceptor) {
	commonInterceptors = interceptors
}

func Interceptors(req *http.Request) []Interceptor {
	var m mux.RouteMatch
	if interceptorMap.Match(req, &m) {
		list, _ := m.Handler.(*interceptorList)
		return list.interceptors
	}
	return []Interceptor{}
}

func CommonInterceptors() []Interceptor {
	return commonInterceptors
}

func LoadServiceConfigWith(pkgPath string) {
	BeforeLoadConfig(pkgPath)
	LoadServiceConfig()
}

func BeforeLoadConfig(pkgPath string) {
	initPkgPath(pkgPath)
}

// initPkgPath parse package path, if gopath contains multi paths, the last one will be used
func initPkgPath(pkgPath string) {
	goPath := os.Getenv("GOPATH")
	paths := strings.Split(goPath, ":")
	l := len(paths)
	serviceRootPath = paths[l-1] + "/src/" + pkgPath
	servicePkgPath = pkgPath
}

func LoadServiceConfig() {
	// TODO use viper to load config file, https://github.com/spf13/viper
	conf, err := yaml.ReadFile(serviceRootPath + "/service.yaml")
	if err != nil {
		log.Fatalf("readfile(%q): %s", serviceRootPath+"/service.yaml", err)
	}
	config = *conf
	initUrlMap()
	initConfigs()
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

func appendUrlServiceMap(line string) {
	values := strings.Split(line, " ")
	HTTPMethod := strings.TrimSpace(values[0])
	url := strings.TrimSpace(values[1])
	methodName := strings.TrimSpace(values[2])
	UrlServiceMap = append(UrlServiceMap, [3]string{HTTPMethod, url, methodName})
}

var preprocessorMap *mux.Router = mux.NewRouter()

type preprocessor struct {
	emptyHandler
	value func(http.ResponseWriter, *http.Request) error
}

func SetPreprocessor(urlPattern string, pre func(http.ResponseWriter, *http.Request) error) {
	preprocessorMap.Handle(urlPattern, &preprocessor{value: pre})
}

func Preprocessor(req *http.Request) func(http.ResponseWriter, *http.Request) error {
	var m mux.RouteMatch
	if preprocessorMap.Match(req, &m) {
		p, _ := m.Handler.(*preprocessor)
		return p.value
	}
	return nil
}

var hijackerMap *mux.Router = mux.NewRouter()

type hijacker struct {
	emptyHandler
	value func(http.ResponseWriter, *http.Request)
}

func SetHijacker(urlPattern string, h func(http.ResponseWriter, *http.Request)) {
	hijackerMap.Handle(urlPattern, &hijacker{value: h})
}

func Hijacker(req *http.Request) func(http.ResponseWriter, *http.Request) {
	var m mux.RouteMatch
	if hijackerMap.Match(req, &m) {
		h, _ := m.Handler.(*hijacker)
		return h.value
	}
	return nil
}

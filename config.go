package turbo

import (
	"log"
	"strings"
	"os"
	"net/http"
	"github.com/kylelemons/go-gypsy/yaml"
)

const SERVICE_NAME string = "service_name"
const SERVICE_ADDRESS string = "service_address"
const PORT string = "port"

var config yaml.File

var UrlServiceMap [][3]string

var serviceRootPath string

var servicePkgPath string

var configs map[string]string = make(map[string]string)

var requestPreprocessor map[string]func(http.ResponseWriter, *http.Request) error = make(map[string]func(http.ResponseWriter, *http.Request) error)

var hijacker map[string]func(http.ResponseWriter, *http.Request) = make(map[string]func(http.ResponseWriter, *http.Request))

// initPkgPath parse package path, if gopath contains multi paths, the last one will be used
func initPkgPath(pkgPath string) {
	goPath := os.Getenv("GOPATH")
	paths := strings.Split(goPath, ":")
	l := len(paths)
	serviceRootPath = paths[l-1] + "/src/" + pkgPath
	servicePkgPath = pkgPath
}

func loadServiceConfig(pkgPath string) {
	initPkgPath(pkgPath)
	conf, err := yaml.ReadFile(serviceRootPath + "/service.yaml")
	if err != nil {
		log.Fatalf("readfile(%q): %s", pkgPath, err)
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

func SetPreprocessor(methodName string, pre func(http.ResponseWriter, *http.Request) error) {
	requestPreprocessor[methodName] = pre
}

func Preprocessor(methodName string) func(http.ResponseWriter, *http.Request) error {
	if p, ok := requestPreprocessor[methodName]; ok {
		return p
	}
	return nil
}

func SetHijacker(methodName string, h func(http.ResponseWriter, *http.Request)) {
	hijacker[methodName] = h
}

func Hijacker(methodName string) func(http.ResponseWriter, *http.Request) {
	if h, ok := hijacker[methodName]; ok {
		return h
	}
	return nil
}

package turbo

import (
	"log"
	"strings"
	"bufio"
	"io"
	"os"
	"net/http"
)

const SERVICE_NAME string = "service_name"
const ADDRESS string = "address"

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
	// TODO use one config file
	initUrlMap()
	initConfigs()
}

func initConfigs() {
	f, err := os.Open(serviceRootPath + "/service.config")
	defer f.Close()
	if err != nil {
		log.Println(err)
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				appendConfig(line)
				return
			}
			log.Println(err)
			break
		}
		appendConfig(line)
	}

}

func appendConfig(line string) {
	pair := strings.Split(line, "=")
	k := strings.TrimSpace(pair[0])
	v := strings.TrimSpace(pair[1])
	configs[k] = v
}

func initUrlMap() {
	f, err := os.Open(serviceRootPath + "/urlmap.config")
	defer f.Close()
	if err != nil {
		log.Println(err)
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				appendUrlServiceMap(strings.TrimSpace(line))
				return
			}
			log.Println(err)
			break
		}
		appendUrlServiceMap(strings.TrimSpace(line))
	}
}

func appendUrlServiceMap(line string) {
	pair := strings.Split(line, "=")
	urlPair := strings.Split(strings.TrimSpace(pair[0]), " ")
	methodName := strings.TrimSpace(pair[1])
	UrlServiceMap = append(UrlServiceMap, [3]string{urlPair[0], urlPair[1], methodName})
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

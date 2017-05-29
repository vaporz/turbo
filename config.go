package turbo

import (
	"github.com/kylelemons/go-gypsy/yaml"
	"log"
	"os"
	"strings"
)

// TODO export a Configs struct
const grpcServiceName string = "grpc_service_name"
const grpcServiceAddress string = "grpc_service_address"
const thriftServiceName string = "thrift_service_name"
const thriftServiceAddress string = "thrift_service_address"
const httpPort string = "http_port"
const filterProtoJson string = "filter_proto_json"
const filterProtoJsonEmitZeroValues string = "filter_proto_json_emit_zerovalues"
const filterProtoJsonInt64AsNumber string = "filter_proto_json_int64_as_number"

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

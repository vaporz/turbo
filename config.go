package turbo

import (
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"log"
	"os"
	"strconv"
	"strings"
)

const grpcServiceName string = "grpc_service_name"
const grpcServiceAddress string = "grpc_service_address"
const thriftServiceName string = "thrift_service_name"
const thriftServiceAddress string = "thrift_service_address"
const httpPort string = "http_port"
const filterProtoJson string = "filter_proto_json"
const filterProtoJsonEmitZeroValues string = "filter_proto_json_emit_zerovalues"
const filterProtoJsonInt64AsNumber string = "filter_proto_json_int64_as_number"

type configs map[string]string

var Config configs = make(map[string]string)

func (c *configs) GrpcServiceName() string {
	return Config[grpcServiceName]
}

func (c *configs) SetGrpcServiceName(name string) {
	Config[grpcServiceName] = name
}

func (c *configs) GrpcServiceAddress() string {
	return Config[grpcServiceAddress]
}

func (c *configs) SetGrpcServiceAddress(address string) {
	Config[grpcServiceAddress] = address
}

func (c *configs) ThriftServiceName() string {
	return Config[thriftServiceName]
}

func (c *configs) SetThriftServiceName(name string) {
	Config[thriftServiceName] = name
}

func (c *configs) ThriftServiceAddress() string {
	return Config[thriftServiceAddress]
}

func (c *configs) SetThriftServiceAddress(address string) {
	Config[thriftServiceAddress] = address
}

func (c *configs) HTTPPort() int64 {
	i, err := strconv.ParseInt(Config[httpPort], 10, 32)
	if err != nil {
		fmt.Println(err)
	}
	return i
}

func (c *configs) HTTPPortStr() string {
	return ":" + Config[httpPort]
}

func (c *configs) SetHTTPPort(p int64) {
	Config[httpPort] = strconv.FormatInt(p, 10)
}

func (c *configs) FilterProtoJson() bool {
	option, ok := Config[filterProtoJson]
	if !ok || option != "true" {
		return false
	}
	return true
}

func (c *configs) SetFilterProtoJson(filterJson bool) {
	Config[filterProtoJson] = strconv.FormatBool(filterJson)
}

func (c *configs) FilterProtoJsonEmitZeroValues() bool {
	option, ok := Config[filterProtoJson]
	if !ok || option != "true" {
		return false
	}
	option, ok = Config[filterProtoJsonEmitZeroValues]
	if ok && option == "false" {
		return false
	}
	return true
}

func (c *configs) SetFilterProtoJsonEmitZeroValues(emitZeroValues bool) {
	Config[filterProtoJsonEmitZeroValues] = strconv.FormatBool(emitZeroValues)
}

func (c *configs) FilterProtoJsonInt64AsNumber() bool {
	option, ok := Config[filterProtoJson]
	if !ok || option != "true" {
		return false
	}
	option, ok = Config[filterProtoJsonInt64AsNumber]
	if ok && option == "false" {
		return false
	}
	return true
}

func (c *configs) SetFilterProtoJsonInt64AsNumber(asNumber bool) {
	Config[filterProtoJsonInt64AsNumber] = strconv.FormatBool(asNumber)
}

var rpcType string

var configFile yaml.File

var urlServiceMaps [][3]string

// absolute path
var serviceRootPath string

var servicePkgPath string

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
	configFile = *conf
	initUrlMap()
	initConfigs()
	initFieldMapping()
}

func initConfigs() {
	cnf, err := yaml.Child(configFile.Root, "config")
	if err != nil {
		log.Fatalf("parse config error: %s", err)
	}
	configMap := cnf.(yaml.Map)
	for k, v := range configMap {
		Config[k] = (v.(yaml.Scalar)).String()
	}
}

func initUrlMap() {
	node, err := yaml.Child(configFile.Root, "urlmapping")
	if err != nil {
		log.Fatalf("parse urlmapping error: %s", err)
	}
	urlMap := node.(yaml.List)
	for _, line := range urlMap {
		appendUrlServiceMap(strings.TrimSpace(yaml.Render(line)))
	}
}

func initFieldMapping() {
	node, err := yaml.Child(configFile.Root, rpcType+"-fieldmapping")
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

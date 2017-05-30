package turbo

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"regexp"
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

var Config = &config{}

type config struct {
	GOPATH          string // the GOPATH used by Turbo
	RpcType         string // "grpc"/"thrift"
	ConfigFileName  string // yaml file name, exclude extension
	ServiceRootPath string // absolute path
	ServicePkgPath  string // package path, e.g. "github.com/vaporz/turbo"

	configs        map[string]string
	urlServiceMaps [][3]string
	fieldMappings  map[string][]string
}

func (c *config) GrpcServiceName() string {
	return c.configs[grpcServiceName]
}

func (c *config) SetGrpcServiceName(name string) {
	c.configs[grpcServiceName] = name
}

func (c *config) GrpcServiceAddress() string {
	return c.configs[grpcServiceAddress]
}

func (c *config) GrpcServicePortStr() string {
	addr := c.configs[grpcServiceAddress]
	i := strings.Index(addr, ":")
	if i <= 0 {
		panic("invalid grpc_service_address")
	}
	return addr[i:]
}

func (c *config) SetGrpcServiceAddress(address string) {
	c.configs[grpcServiceAddress] = address
}

func (c *config) ThriftServiceName() string {
	return c.configs[thriftServiceName]
}

func (c *config) ThriftServicePortStr() string {
	addr := c.configs[thriftServiceAddress]
	i := strings.Index(addr, ":")
	if i <= 0 {
		panic("invalid thrift_service_address")
	}
	return addr[i:]
}

func (c *config) SetThriftServiceName(name string) {
	c.configs[thriftServiceName] = name
}

func (c *config) ThriftServiceAddress() string {
	return c.configs[thriftServiceAddress]
}

func (c *config) SetThriftServiceAddress(address string) {
	c.configs[thriftServiceAddress] = address
}

func (c *config) HTTPPort() int64 {
	i, err := strconv.ParseInt(c.configs[httpPort], 10, 32)
	if err != nil {
		fmt.Println(err)
	}
	return i
}

func (c *config) HTTPPortStr() string {
	return ":" + c.configs[httpPort]
}

func (c *config) SetHTTPPort(p int64) {
	c.configs[httpPort] = strconv.FormatInt(p, 10)
}

func (c *config) FilterProtoJson() bool {
	option, ok := c.configs[filterProtoJson]
	if !ok || option != "true" {
		return false
	}
	return true
}

func (c *config) SetFilterProtoJson(filterJson bool) {
	c.configs[filterProtoJson] = strconv.FormatBool(filterJson)
}

func (c *config) FilterProtoJsonEmitZeroValues() bool {
	option, ok := c.configs[filterProtoJson]
	if !ok || option != "true" {
		return false
	}
	option, ok = c.configs[filterProtoJsonEmitZeroValues]
	if ok && option == "false" {
		return false
	}
	return true
}

func (c *config) SetFilterProtoJsonEmitZeroValues(emitZeroValues bool) {
	c.configs[filterProtoJsonEmitZeroValues] = strconv.FormatBool(emitZeroValues)
}

func (c *config) FilterProtoJsonInt64AsNumber() bool {
	option, ok := c.configs[filterProtoJson]
	if !ok || option != "true" {
		return false
	}
	option, ok = c.configs[filterProtoJsonInt64AsNumber]
	if ok && option == "false" {
		return false
	}
	return true
}

func (c *config) SetFilterProtoJsonInt64AsNumber(asNumber bool) {
	c.configs[filterProtoJsonInt64AsNumber] = strconv.FormatBool(asNumber)
}

// LoadServiceConfigWith accepts a package path, then load service.yaml in that path
func LoadServiceConfig(rpcType, pkgPath, configFileName string) {
	initRpcType(rpcType)
	initConfigFileName(configFileName)
	initPkgPath(pkgPath)
	loadServiceConfig()
}

func initConfigFileName(name string) {
	Config.ConfigFileName = name
}

func initRpcType(r string) {
	Config.RpcType = r
}

func initPkgPath(pkgPath string) {
	goPath := os.Getenv("GOPATH")
	paths := strings.Split(goPath, ":")
	Config.GOPATH = paths[0]
	Config.ServiceRootPath = Config.GOPATH + "/src/" + pkgPath
	Config.ServicePkgPath = pkgPath
}

func loadServiceConfig() {
	// TODO reload config at runtime
	viper.SetConfigName(Config.ConfigFileName)
	viper.AddConfigPath(Config.ServiceRootPath)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	initUrlMap()
	initConfigs()
	initFieldMapping()
}

func initUrlMap() {
	Config.urlServiceMaps = make([][3]string, 0)
	urlMap := viper.GetStringSlice("urlmapping")
	for _, line := range urlMap {
		appendUrlServiceMap(strings.TrimSpace(line))
	}
}

func appendUrlServiceMap(line string) {
	values := strings.Split(line, " ")
	HTTPMethod := strings.TrimSpace(values[0])
	url := strings.TrimSpace(values[1])
	methodName := strings.TrimSpace(values[2])
	Config.urlServiceMaps = append(Config.urlServiceMaps, [3]string{HTTPMethod, url, methodName})
}

func initConfigs() {
	Config.configs = viper.GetStringMapString("config")
	// TODO check required config item
}

var matchKey = regexp.MustCompile("^(.*)\\[")
var matchSlice = regexp.MustCompile("\\[(.*)\\]")

func initFieldMapping() {
	Config.fieldMappings = make(map[string][]string)
	mappings := viper.GetStringSlice(Config.RpcType + "-fieldmapping")
	for _, m := range mappings {
		keyStr := matchKey.FindStringSubmatch(m)
		key := m
		if len(keyStr) >= 2 {
			key = keyStr[1]
		}
		k := strings.TrimSpace(key)
		valueSliceStr := matchSlice.FindStringSubmatch(m)
		if len(valueSliceStr) >= 2 {
			fields := strings.Split(valueSliceStr[1], ",")
			for _, v := range fields {
				Config.fieldMappings[k] = append(Config.fieldMappings[k], strings.TrimSpace(v))
			}
		}
	}
}

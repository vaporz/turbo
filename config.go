package turbo

import (
	"github.com/spf13/viper"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const grpcServiceName string = "grpc_service_name"
const grpcServiceHost string = "grpc_service_host"
const grpcServicePort string = "grpc_service_port"
const thriftServiceName string = "thrift_service_name"
const thriftServiceHost string = "thrift_service_host"
const thriftServicePort string = "thrift_service_port"
const httpPort string = "http_port"
const filterProtoJson string = "filter_proto_json"
const filterProtoJsonEmitZeroValues string = "filter_proto_json_emit_zerovalues"
const filterProtoJsonInt64AsNumber string = "filter_proto_json_int64_as_number"
const turboLogPath string = "turbo_log_path"
const environment string = "environment"
const serviceRootPath string = "service_root_path"

// GOPATH inits the GOPATH turbo used.
func GOPATH() string {
	goPath := os.Getenv("GOPATH")
	paths := strings.Split(goPath, ":")
	return paths[0]
}

// Config holds the info in a config file
type Config struct {
	viper.Viper
	// RpcType should be "grpc" or "thrift"
	RpcType string
	// GOPATH is the GOPATH used by Turbo
	GOPATH string
	// File is the config file path
	File string

	configs        map[string]string
	urlServiceMaps [][3]string
	fieldMappings  map[string][]string

	interceptors   [][3]string
	preprocessors  [][3]string
	postprocessors [][3]string
	hijackers      [][3]string
	convertors     [][2]string
	errorhandler   string
}

func NewConfig(rpcType, configFilePath string) *Config {
	c := &Config{Viper: *viper.New(), RpcType: rpcType, GOPATH: GOPATH()}
	c.loadServiceConfig(configFilePath)
	return c
}

func (c *Config) loadServiceConfig(p string) {
	c.SetConfigFile(p)
	err := c.ReadInConfig()
	if err != nil {
		panic(err)
	}
	c.File = p
	c.loadUrlMap()
	c.loadConfigs()
	c.loadComponents()
}

func (c *Config) loadComponents() {
	c.interceptors = c.loadMappings("interceptor")
	c.preprocessors = c.loadMappings("preprocessor")
	c.postprocessors = c.loadMappings("postprocessor")
	c.hijackers = c.loadMappings("hijacker")
	c.convertors = c.loadConvertor()
	c.errorhandler = c.GetString("errorhandler")
}

func (c *Config) loadUrlMap() {
	c.urlServiceMaps = c.loadMappings("urlmapping")
}

func (c *Config) loadMappings(key string) [][3]string {
	mapping := make([][3]string, 0)
	lines := c.GetStringSlice(key)
	for _, line := range lines {
		mapping = appendMap(mapping, strings.TrimSpace(line))
	}
	return mapping
}

func appendMap(mapping [][3]string, line string) [][3]string {
	values := strings.Split(line, " ")
	HTTPMethod := strings.TrimSpace(values[0])
	url := strings.TrimSpace(values[1])
	value := strings.TrimSpace(values[2])
	return append(mapping, [3]string{HTTPMethod, url, value})
}

func (c *Config) loadConvertor() [][2]string {
	mapping := make([][2]string, 0)
	lines := c.GetStringSlice("convertor")
	for _, line := range lines {
		values := strings.Split(strings.TrimSpace(line), " ")
		name := strings.TrimSpace(values[0])
		convertorName := strings.TrimSpace(values[1])
		mapping = append(mapping, [2]string{name, convertorName})
	}
	return mapping
}

func (c *Config) loadConfigs() {
	c.configs = c.GetStringMapString("config")
}

var matchKey = regexp.MustCompile("^(.*)\\[")
var matchSlice = regexp.MustCompile("\\[(.+)\\]")

func (c *Config) loadFieldMapping() {
	c.SetConfigName(c.RpcType + "fields")
	c.AddConfigPath(c.ServiceRootPath() + "/gen")
	err := c.ReadInConfig()
	if err != nil {
		panic(err)
	}
	c.fieldMappings = make(map[string][]string)
	mappings := c.GetStringSlice(c.RpcType + "-fieldmapping")
	for _, m := range mappings {
		keyStr := matchKey.FindStringSubmatch(m)
		key := m
		if len(keyStr) >= 2 {
			key = keyStr[1]
		}
		k := strings.TrimSpace(key)
		valueSliceStr := matchSlice.FindStringSubmatch(m)
		c.fieldMappings[k] = parseSliceStr(valueSliceStr)
	}
}

func parseSliceStr(valueSliceStr []string) []string {
	result := make([]string, 0)
	if len(valueSliceStr) >= 2 {
		fields := strings.Split(valueSliceStr[1], ",")
		for _, v := range fields {
			result = append(result, strings.TrimSpace(v))
		}
	} else {
		result = []string{}
	}
	return result
}

func (c *Config) Env() string {
	return c.configs[environment]
}

// TODO should return raw value
func (c *Config) ServiceRootPath() string {
	p := c.configs[serviceRootPath]
	if path.IsAbs(p) {
		return p
	} else {
		return c.GOPATH + "/src/" + p
	}
}

func (c *Config) GrpcServiceName() string {
	return c.configs[grpcServiceName]
}

func (c *Config) GrpcServiceHost() string {
	return c.configs[grpcServiceHost]
}

func (c *Config) GrpcServicePort() string {
	return c.configs[grpcServicePort]
}

func (c *Config) ThriftServiceName() string {
	return c.configs[thriftServiceName]
}

func (c *Config) ThriftServiceHost() string {
	return c.configs[thriftServiceHost]
}

func (c *Config) ThriftServicePort() string {
	return c.configs[thriftServicePort]
}

func (c *Config) HTTPPort() int64 {
	p, ok := c.configs[httpPort]
	if !ok || len(strings.TrimSpace(p)) == 0 {
		panic("[http_port] is required!")
	}
	i, err := strconv.ParseInt(p, 10, 64)
	if err != nil {
		log.Error(err.Error())
	}
	return i
}

func (c *Config) FilterProtoJson() bool {
	option, ok := c.configs[filterProtoJson]
	if !ok || option != "true" {
		return false
	}
	return true
}

func (c *Config) FilterProtoJsonEmitZeroValues() bool {
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

func (c *Config) FilterProtoJsonInt64AsNumber() bool {
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

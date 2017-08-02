package turbo

import (
	"github.com/spf13/viper"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	grpcServiceName               = "grpc_service_name"
	grpcServiceHost               = "grpc_service_host"
	grpcServicePort               = "grpc_service_port"
	thriftServiceName             = "thrift_service_name"
	thriftServiceHost             = "thrift_service_host"
	thriftServicePort             = "thrift_service_port"
	httpPort                      = "http_port"
	filterProtoJson               = "filter_proto_json"
	filterProtoJsonEmitZeroValues = "filter_proto_json_emit_zerovalues"
	filterProtoJsonInt64AsNumber  = "filter_proto_json_int64_as_number"
	turboLogPath                  = "turbo_log_path"
	environment                   = "environment"
	serviceRootPath               = "service_root_path"

	urlServiceMaps = "urlServiceMaps"
	interceptors   = "interceptors"
	preprocessors  = "preprocessors"
	postprocessors = "postprocessors"
	hijackers      = "hijackers"
	convertors     = "convertors"
)

// GOPATH inits the GOPATH turbo used.
func GOPATH() string {
	goPath := os.Getenv("GOPATH")
	paths := strings.Split(goPath, ":")
	return paths[0]
}

// RpcType should be "grpc" or "thrift"
var RpcType string

// Config holds the info in a config file
type Config struct {
	viper.Viper
	// File is the config file path
	File          string
	configs       map[string]string
	fieldMappings map[string][]string
	mappings      map[string][][3]string
}

// NewConfig loads the config file at 'configFilePath', and returns a Config struct ptr
func NewConfig(rpcType, configFilePath string) *Config {
	RpcType = rpcType
	c := &Config{
		Viper:    *viper.New(),
		File:     configFilePath,
		mappings: make(map[string][][3]string)}
	c.loadServiceConfig()
	return c
}

func (c *Config) ErrorHandler() string {
	return c.GetString("errorhandler")
}

func (c *Config) loadServiceConfig() {
	c.SetConfigFile(c.File)
	err := c.ReadInConfig()
	panicIf(err)
	c.loadUrlMap()
	c.loadConfigs()
	c.loadComponents()
}

func (c *Config) loadComponents() {
	c.mappings[interceptors] = c.loadMappings("interceptor")
	c.mappings[preprocessors] = c.loadMappings("preprocessor")
	c.mappings[postprocessors] = c.loadMappings("postprocessor")
	c.mappings[hijackers] = c.loadMappings("hijacker")
	c.mappings[convertors] = c.loadConvertor()
}

func (c *Config) loadUrlMap() {
	c.mappings[urlServiceMaps] = c.loadMappings("urlmapping")
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

func (c *Config) loadConvertor() [][3]string {
	mapping := make([][3]string, 0)
	lines := c.GetStringSlice("convertor")
	for _, line := range lines {
		values := strings.Split(strings.TrimSpace(line), " ")
		name := strings.TrimSpace(values[0])
		convertorName := strings.TrimSpace(values[1])
		mapping = append(mapping, [3]string{name, convertorName})
	}
	return mapping
}

func (c *Config) loadConfigs() {
	c.configs = c.GetStringMapString("config")
}

var matchKey = regexp.MustCompile("^(.*)\\[")
var matchSlice = regexp.MustCompile("\\[(.+)\\]")

func (c *Config) loadFieldMapping() {
	c.SetConfigName(RpcType + "fields")
	c.AddConfigPath(c.ServiceRootPathAbsolute() + "/gen")
	err := c.ReadInConfig()
	panicIf(err)
	c.fieldMappings = make(map[string][]string)
	mappings := c.GetStringSlice(RpcType + "-fieldmapping")
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

// ServiceRootPath returns "service_root_path" in config file,
// "service_root_path" can be either an absolute path, or the package path of the service, e.g. github.com/xxx/yyy
func (c *Config) ServiceRootPath() string {
	return c.configs[serviceRootPath]
}

// ServiceRootPathAbsolute returns the absolute path to service's root,
// if "service_root_path" in config file is an absolute path, it's returned directly,
// if "service_root_path" is a relative path, $GOPATH+"/src/"+[service_root_path] is returned.
func (c *Config) ServiceRootPathAbsolute() string {
	p := c.configs[serviceRootPath]
	if len(strings.TrimSpace(p)) == 0 {
		panic("'service_root_path' in config file is not set!")
	}
	if path.IsAbs(p) {
		return p
	} else {
		return GOPATH() + "/src/" + p
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
	logErrorIf(err)
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

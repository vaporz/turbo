package turbo

import (
	"github.com/fsnotify/fsnotify"
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

// LoadServiceConfig accepts a package path, then load service.yaml in that path
func LoadServiceConfig(rpcType, configFilePath string) *Config {
	c := &Config{RpcType: rpcType, GOPATH: GOPATH()}
	c.loadServiceConfig(configFilePath)
	c.watchConfig(configFilePath)

	initLogger(c)
	return c
}

// GOPATH inits the GOPATH turbo used.
func GOPATH() string {
	goPath := os.Getenv("GOPATH")
	paths := strings.Split(goPath, ":")
	return paths[0]
}

// Config holds the info in a config file
type Config struct {
	// TODO add viper
	// RpcType should be "grpc" or "thrift"
	RpcType string
	// GOPATH is the GOPATH used by Turbo
	GOPATH string

	configs        map[string]string
	urlServiceMaps [][3]string
	fieldMappings  map[string][]string
}

func (c *Config) Env() string {
	return c.configs[environment]
}

func (c *Config) TurboLogPath() string {
	return c.configs[turboLogPath]
}

func (c *Config) ServiceRootPath() string {
	p := c.configs[serviceRootPath]
	if path.IsAbs(p) {
		return p
	} else {
		return c.GOPATH + "/src/" + p
	}
}

func (c *Config) SetServiceRootPath(p string) {
	c.configs[serviceRootPath] = p
}

func (c *Config) GrpcServiceName() string {
	return c.configs[grpcServiceName]
}

func (c *Config) SetGrpcServiceName(name string) {
	c.configs[grpcServiceName] = name
}

func (c *Config) GrpcServiceAddress() string {
	return c.GrpcServiceHost() + ":" + c.GrpcServicePort()
}

func (c *Config) GrpcServiceHost() string {
	return c.configs[grpcServiceHost]
}

func (c *Config) GrpcServicePort() string {
	return c.configs[grpcServicePort]
}

func (c *Config) SetGrpcServiceHost(host string) {
	c.configs[grpcServiceHost] = host
}

func (c *Config) SetGrpcServicePort(port string) {
	c.configs[grpcServicePort] = port
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

func (c *Config) ThriftServiceAddress() string {
	return c.ThriftServiceHost() + ":" + c.ThriftServicePort()
}

func (c *Config) SetThriftServiceHost(host string) {
	c.configs[thriftServiceHost] = host
}

func (c *Config) SetThriftServicePort(port string) {
	c.configs[thriftServicePort] = port
}

func (c *Config) SetThriftServiceName(name string) {
	c.configs[thriftServiceName] = name
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

func (c *Config) HTTPPortStr() string {
	return ":" + strconv.FormatInt(c.HTTPPort(), 10)
}

func (c *Config) SetHTTPPort(p int64) {
	c.configs[httpPort] = strconv.FormatInt(p, 10)
}

func (c *Config) FilterProtoJson() bool {
	option, ok := c.configs[filterProtoJson]
	if !ok || option != "true" {
		return false
	}
	return true
}

func (c *Config) SetFilterProtoJson(filterJson bool) {
	c.configs[filterProtoJson] = strconv.FormatBool(filterJson)
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

func (c *Config) SetFilterProtoJsonEmitZeroValues(emitZeroValues bool) {
	c.configs[filterProtoJsonEmitZeroValues] = strconv.FormatBool(emitZeroValues)
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

func (c *Config) SetFilterProtoJsonInt64AsNumber(asNumber bool) {
	c.configs[filterProtoJsonInt64AsNumber] = strconv.FormatBool(asNumber)
}

func (c *Config) watchConfig(configFilePath string) {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		c.loadServiceConfig(configFilePath)
		reloadConfig <- true
	})
}

func (c *Config) loadServiceConfig(configFilePath string) {
	viper.SetConfigFile(configFilePath)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	c.loadUrlMap()
	c.loadConfigs()
}

func (c *Config) loadUrlMap() {
	c.urlServiceMaps = make([][3]string, 0)
	urlMap := viper.GetStringSlice("urlmapping")
	for _, line := range urlMap {
		c.appendUrlServiceMap(strings.TrimSpace(line))
	}
}

func (c *Config) appendUrlServiceMap(line string) {
	values := strings.Split(line, " ")
	HTTPMethod := strings.TrimSpace(values[0])
	url := strings.TrimSpace(values[1])
	methodName := strings.TrimSpace(values[2])
	c.urlServiceMaps = append(c.urlServiceMaps, [3]string{HTTPMethod, url, methodName})
}

func (c *Config) loadConfigs() {
	c.configs = viper.GetStringMapString("config")
}

var matchKey = regexp.MustCompile("^(.*)\\[")
var matchSlice = regexp.MustCompile("\\[(.+)\\]")

func (c *Config) loadFieldMapping() {
	v := viper.New()
	v.SetConfigName(c.RpcType + "fields")
	v.AddConfigPath(c.ServiceRootPath() + "/gen")
	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}
	c.fieldMappings = make(map[string][]string)
	mappings := v.GetStringSlice(c.RpcType + "-fieldmapping")
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
				c.fieldMappings[k] = append(c.fieldMappings[k], strings.TrimSpace(v))
			}
		} else {
			c.fieldMappings[k] = []string{}
		}
	}
}

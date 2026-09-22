/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	jsonFieldNamesProto           = "proto"
	jsonFieldNamesCamel           = "camel"
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
	logLevel                      = "log_level"
	environment                   = "environment"
	fileRootPath                  = "file_root_path"
	packagePath                   = "package_path"

	jsonFieldNames = "json_field_names"
	urlServiceMaps = "urlServiceMaps"
	interceptors   = "interceptors"
	preprocessors  = "preprocessors"
	postprocessors = "postprocessors"
	hijackers      = "hijackers"
	convertors     = "convertors"
)

//
//// GOPATH inits the GOPATH turbo used.
//func GOPATH() string {
//	goPath := os.Getenv("GOPATH")
//	paths := strings.Split(goPath, ":")
//	return paths[0]
//}

func GetWD() string {
	currentDir, e := os.Getwd()
	if e != nil {
		panic(e)
	}
	return currentDir
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
	mappings      map[string][][4]string
}

// NewConfig loads the config file at 'configFilePath', and returns a Config struct ptr
func NewConfig(rpcType, configFilePath string) *Config {
	RpcType = rpcType
	c := &Config{
		Viper:    *viper.New(),
		File:     configFilePath,
		mappings: make(map[string][][4]string)}
	c.loadServiceConfig()
	return c
}

func (c *Config) ErrorHandler() string {
	return c.GetString("errorhandler")
}

func (c *Config) loadServiceConfig() {
	panicIf(c.loadServiceConfigErr())
}

// loadServiceConfigErr loads the configuration, reporting a failure instead of
// panicking. The same code runs when a running server reloads its configuration,
// and there a file that is half written or malformed must not take the server
// down: the caller decides whether to fail fast (startup) or to keep serving the
// configuration already in effect (reload).
func (c *Config) loadServiceConfigErr() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid configuration %s: %v", c.File, r)
		}
	}()
	c.SetConfigFile(c.File)
	if err := c.ReadInConfig(); err != nil {
		return err
	}
	c.loadUrlMap()
	c.loadConfigs()
	c.loadComponents()
	return c.validate()
}

// validate refuses a configuration turbo cannot work with: one whose values it
// would have to guess at, and one that lists no route at all.
func (c *Config) validate() error {
	if len(c.mappings[urlServiceMaps]) == 0 {
		// A gateway with no route cannot serve anything, so this is never a
		// configuration somebody meant to use. It is, however, exactly what a
		// reload reads when it catches the file while it is being written: the
		// writer truncates first, and an empty file is valid YAML. Accepting it
		// would replace a working routing table with an empty one, and every
		// request would answer 404 while the process still looked healthy.
		return fmt.Errorf("urlmapping is empty, so no route would be served at all " +
			"(a configuration file read while it is being written looks like this)")
	}
	if value := strings.TrimSpace(c.configs[jsonFieldNames]); value != "" &&
		!strings.EqualFold(value, jsonFieldNamesProto) && !strings.EqualFold(value, jsonFieldNamesCamel) {
		return fmt.Errorf("invalid %s: %q, expected %q or %q",
			jsonFieldNames, value, jsonFieldNamesProto, jsonFieldNamesCamel)
	}
	if value := c.LogLevel(); value != "" {
		if _, err := logger.ParseLevel(value); err != nil {
			return fmt.Errorf("invalid %s: %q, expected one of panic, fatal, error, warn, info, debug, trace",
				logLevel, value)
		}
	}
	return nil
}

// LogLevel returns the level turbo's own log should use, or "" when the
// configuration leaves the choice to the environment. It lives under "config"
// with the other server settings, which also keeps it apart from a service's own
// top level log_level.
func (c *Config) LogLevel() string {
	return strings.TrimSpace(c.configs[logLevel])
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

func (c *Config) loadMappings(key string) [][4]string {
	mapping := make([][4]string, 0)
	lines := c.GetStringSlice(key)
	for _, line := range lines {
		mapping = appendMap(mapping, strings.TrimSpace(line))
	}
	return mapping
}

func appendMap(mapping [][4]string, line string) [][4]string {
	values := strings.Split(line, " ")
	HTTPMethod := strings.TrimSpace(values[0])
	url := strings.TrimSpace(values[1])
	v1 := strings.TrimSpace(values[2])
	var v2 string
	if len(values) > 3 {
		v2 = strings.TrimSpace(values[3])
	}
	return append(mapping, [4]string{HTTPMethod, url, v1, v2})
}

func (c *Config) loadConvertor() [][4]string {
	mapping := make([][4]string, 0)
	lines := c.GetStringSlice("convertor")
	for _, line := range lines {
		values := strings.Split(strings.TrimSpace(line), " ")
		name := strings.TrimSpace(values[0])
		convertorName := strings.TrimSpace(values[1])
		mapping = append(mapping, [4]string{name, convertorName})
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
	c.AddConfigPath(c.ServiceRootPath() + "/gen")
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

// FileRootPath returns the absolute path to the root of packages,
func (c *Config) FileRootPath() string {
	p := c.configs[fileRootPath]
	if len(strings.TrimSpace(p)) == 0 {
		panic("'file_root_path' in config file is not set!")
	}
	if path.IsAbs(p) {
		return p
	} else {
		panic("fileRootPath MUST be an absolute path, got: " + p + " ")
	}
}

// PackagePath returns the base package name,
func (c *Config) PackagePath() string {
	p := c.configs[packagePath]
	if len(strings.TrimSpace(p)) == 0 {
		panic("'package_path' in config file is not set!")
	}
	return p
}

// ServiceRootPath returns the absolute path to service's root,
func (c *Config) ServiceRootPath() string {
	p := c.FileRootPath() + "/" + c.PackagePath()
	if path.IsAbs(p) {
		return p
	} else {
		panic("serviceRootPath MUST be an absolute path, got: " + p + " ")
	}
}

func (c *Config) GrpcServiceNames() []string {
	names := c.configs[grpcServiceName]
	return strings.Split(names, ",")
}

func (c *Config) GrpcServiceHost() string {
	return c.configs[grpcServiceHost]
}

func (c *Config) GrpcServicePort() string {
	return c.configs[grpcServicePort]
}

func (c *Config) ThriftServiceNames() []string {
	names := c.configs[thriftServiceName]
	return strings.Split(names, ",")
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

// JSONFieldNames reports whether proto JSON should use the proto field names
// ("proto", owner_openid) or the JSON names protobuf defines ("camel",
// ownerOpenid). Anything unrecognised is refused when the configuration loads,
// rather than quietly falling back to one of them.
func (c *Config) JSONFieldNames() string {
	if strings.EqualFold(strings.TrimSpace(c.configs[jsonFieldNames]), jsonFieldNamesCamel) {
		return jsonFieldNamesCamel
	}
	return jsonFieldNamesProto
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

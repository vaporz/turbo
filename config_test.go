package turbo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig(t *testing.T) {
	assert.Equal(t, "production", Config.Env())
	assert.Equal(t, "log", Config.TurboLogPath())
	assert.Equal(t, "grpc", Config.RpcType())
	assert.Equal(t, "service_test", Config.ConfigFileName())
	assert.Equal(t, Config.GOPATH()+"/src/github.com/vaporz/turbo/test", Config.ServiceRootPath())
	assert.Equal(t, "github.com/vaporz/turbo/test", Config.ServicePkgPath())

	assert.Equal(t, int64(8081), Config.HTTPPort())
	Config.SetHTTPPort(1234)
	assert.Equal(t, int64(1234), Config.HTTPPort())
	assert.Equal(t, ":1234", Config.HTTPPortStr())

	assert.Equal(t, "YourService", Config.GrpcServiceName())
	Config.SetGrpcServiceName("test")
	assert.Equal(t, "test", Config.GrpcServiceName())

	assert.Equal(t, "127.0.0.1", Config.GrpcServiceHost())
	assert.Equal(t, "50051", Config.GrpcServicePort())
	assert.Equal(t, "127.0.0.1:50051", Config.GrpcServiceAddress())
	Config.SetGrpcServiceHost("test host")
	Config.SetGrpcServicePort("test port")
	assert.Equal(t, "test host:test port", Config.GrpcServiceAddress())

	assert.Equal(t, "YourService", Config.ThriftServiceName())
	Config.SetThriftServiceName("test thrift")
	assert.Equal(t, "test thrift", Config.ThriftServiceName())

	assert.Equal(t, "127.0.0.1", Config.ThriftServiceHost())
	assert.Equal(t, "50052", Config.ThriftServicePort())
	assert.Equal(t, "127.0.0.1:50052", Config.ThriftServiceAddress())
	assert.Equal(t, "50052", Config.ThriftServicePort())
	Config.SetThriftServiceHost("test host")
	Config.SetThriftServicePort("test port")
	assert.Equal(t, "test host:test port", Config.ThriftServiceAddress())

	assert.Equal(t, true, Config.FilterProtoJson())
	Config.SetFilterProtoJson(false)
	assert.Equal(t, false, Config.FilterProtoJson())

	assert.Equal(t, false, Config.FilterProtoJsonInt64AsNumber())
	Config.SetFilterProtoJsonInt64AsNumber(true)
	assert.Equal(t, false, Config.FilterProtoJsonInt64AsNumber())
	Config.SetFilterProtoJson(true)
	assert.Equal(t, true, Config.FilterProtoJsonInt64AsNumber())

	Config.SetFilterProtoJson(false)
	assert.Equal(t, false, Config.FilterProtoJsonEmitZeroValues())
	Config.SetFilterProtoJsonEmitZeroValues(true)
	assert.Equal(t, false, Config.FilterProtoJsonEmitZeroValues())
	Config.SetFilterProtoJson(true)
	assert.Equal(t, true, Config.FilterProtoJsonEmitZeroValues())

	assert.Equal(t, "GET,POST", Config.urlServiceMaps[0][0])
	assert.Equal(t, "/hello", Config.urlServiceMaps[0][1])
	assert.Equal(t, "SayHello", Config.urlServiceMaps[0][2])
	assert.Equal(t, "GET", Config.urlServiceMaps[1][0])
	assert.Equal(t, "/eat_apple/{num:[0-9]+}", Config.urlServiceMaps[1][1])
	assert.Equal(t, "EatApple", Config.urlServiceMaps[1][2])

	loadFieldMapping()
	assert.Equal(t, "CommonValues values", Config.fieldMappings["SayHelloRequest"][0])
}

func TestHttpPortPanic(t *testing.T) {
	p := Config.HTTPPort()
	defer func() {
		Config.SetHTTPPort(p)
		if err := recover(); err != nil {
			assert.Equal(t, "[http_port] is required!", err)
		} else {
			t.Errorf("The code did not panic")
		}
	}()
	Config.configs[httpPort] = ""
	Config.HTTPPort()
}

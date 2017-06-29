package turbo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig(t *testing.T) {
	c := LoadServiceConfig("grpc", "test/service_test.yaml")
	assert.Equal(t, "production", c.Env())
	assert.Equal(t, "log", c.TurboLogPath())
	assert.Equal(t, "grpc", c.RpcType)
	assert.Equal(t, c.GOPATH+"/src/"+"github.com/vaporz/turbo/test", c.ServiceRootPath())

	assert.Equal(t, int64(8081), c.HTTPPort())
	c.SetHTTPPort(1234)
	assert.Equal(t, int64(1234), c.HTTPPort())
	assert.Equal(t, ":1234", c.HTTPPortStr())

	assert.Equal(t, "YourService", c.GrpcServiceName())
	c.SetGrpcServiceName("test")
	assert.Equal(t, "test", c.GrpcServiceName())

	assert.Equal(t, "127.0.0.1", c.GrpcServiceHost())
	assert.Equal(t, "50051", c.GrpcServicePort())
	assert.Equal(t, "127.0.0.1:50051", c.GrpcServiceAddress())
	c.SetGrpcServiceHost("test host")
	c.SetGrpcServicePort("test port")
	assert.Equal(t, "test host:test port", c.GrpcServiceAddress())

	assert.Equal(t, "YourService", c.ThriftServiceName())
	c.SetThriftServiceName("test thrift")
	assert.Equal(t, "test thrift", c.ThriftServiceName())

	assert.Equal(t, "127.0.0.1", c.ThriftServiceHost())
	assert.Equal(t, "50052", c.ThriftServicePort())
	assert.Equal(t, "127.0.0.1:50052", c.ThriftServiceAddress())
	assert.Equal(t, "50052", c.ThriftServicePort())
	c.SetThriftServiceHost("test host")
	c.SetThriftServicePort("test port")
	assert.Equal(t, "test host:test port", c.ThriftServiceAddress())

	assert.Equal(t, true, c.FilterProtoJson())
	c.SetFilterProtoJson(false)
	assert.Equal(t, false, c.FilterProtoJson())

	assert.Equal(t, false, c.FilterProtoJsonInt64AsNumber())
	c.SetFilterProtoJsonInt64AsNumber(true)
	assert.Equal(t, false, c.FilterProtoJsonInt64AsNumber())
	c.SetFilterProtoJson(true)
	assert.Equal(t, true, c.FilterProtoJsonInt64AsNumber())

	c.SetFilterProtoJson(false)
	assert.Equal(t, false, c.FilterProtoJsonEmitZeroValues())
	c.SetFilterProtoJsonEmitZeroValues(true)
	assert.Equal(t, false, c.FilterProtoJsonEmitZeroValues())
	c.SetFilterProtoJson(true)
	assert.Equal(t, true, c.FilterProtoJsonEmitZeroValues())

	assert.Equal(t, "GET,POST", c.urlServiceMaps[0][0])
	assert.Equal(t, "/hello", c.urlServiceMaps[0][1])
	assert.Equal(t, "SayHello", c.urlServiceMaps[0][2])
	assert.Equal(t, "GET", c.urlServiceMaps[1][0])
	assert.Equal(t, "/eat_apple/{num:[0-9]+}", c.urlServiceMaps[1][1])
	assert.Equal(t, "EatApple", c.urlServiceMaps[1][2])
	c.loadFieldMapping()
	assert.Equal(t, "CommonValues values", c.fieldMappings["SayHelloRequest"][0])
}

func TestHttpPortPanic(t *testing.T) {
	c := LoadServiceConfig("grpc", "test/service_test.yaml")
	p := c.HTTPPort()
	defer func() {
		c.SetHTTPPort(p)
		if err := recover(); err != nil {
			assert.Equal(t, "[http_port] is required!", err)
		} else {
			t.Errorf("The code did not panic")
		}
	}()
	c.configs[httpPort] = ""
	c.HTTPPort()
}

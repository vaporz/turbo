package turbo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig(t *testing.T) {
	//LoadServiceConfig("grpc", "github.com/vaporz/turbo", "service_test")
	assert.Equal(t, "grpc", Config.RpcType)
	assert.Equal(t, "service_test", Config.ConfigFileName)
	assert.Equal(t, Config.GOPATH+"/src/github.com/vaporz/turbo", Config.ServiceRootPath)
	assert.Equal(t, "github.com/vaporz/turbo", Config.ServicePkgPath)

	assert.Equal(t, int64(8081), Config.HTTPPort())
	assert.Equal(t, "YourService", Config.GrpcServiceName())
	assert.Equal(t, "127.0.0.1:50051", Config.GrpcServiceAddress())
	assert.Equal(t, "YourService", Config.ThriftServiceName())
	assert.Equal(t, "127.0.0.1:50052", Config.ThriftServiceAddress())
	assert.Equal(t, true, Config.FilterProtoJson())
	assert.Equal(t, true, Config.FilterProtoJsonInt64AsNumber())
	assert.Equal(t, true, Config.FilterProtoJsonEmitZeroValues())

	assert.Equal(t, "GET,POST", Config.urlServiceMaps[0][0])
	assert.Equal(t, "/hello", Config.urlServiceMaps[0][1])
	assert.Equal(t, "SayHello", Config.urlServiceMaps[0][2])
	assert.Equal(t, "GET", Config.urlServiceMaps[1][0])
	assert.Equal(t, "/eat_apple/{num:[0-9]+}", Config.urlServiceMaps[1][1])
	assert.Equal(t, "EatApple", Config.urlServiceMaps[1][2])
	assert.Equal(t, "CommonValues values", Config.fieldMappings["SayHelloRequest"][0])
}

package turbo

import (
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestGrpcInit(t *testing.T) {
	grpcServiceAddress := Config.GrpcServiceAddress()
	Config.SetGrpcServiceAddress("")

	client = &Client{gClient: new(grpcClient)}
	err := client.gClient.init(func(*grpc.ClientConn) interface{} { return nil })
	assert.NotNil(t, err)
	assert.Equal(t, "Error: missing [grpc_service_address] in config", err.Error())

	client.gClient.grpcService = ""
	err = client.gClient.init(func(*grpc.ClientConn) interface{} { return nil })
	assert.Nil(t, err)

	Config.SetGrpcServiceAddress(grpcServiceAddress)
}

func TestGrpcClose(t *testing.T) {
	client = &Client{gClient: new(grpcClient)}
	err := client.gClient.close()
	assert.Nil(t, err)
}

func TestGrpcService(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.Equal(t, "grpc connection not initiated!", err.(*logger.Entry).Message)
		} else {
			t.Errorf("The code did not panic")
		}
	}()
	client = &Client{gClient: new(grpcClient)}
	GrpcService()
}

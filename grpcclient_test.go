package turbo

import (
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestGrpcInit(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.Fail(t, "should not panic")
		}
	}()
	s := &GrpcServer{gClient: new(grpcClient)}
	s.gClient.grpcService = ""
	s.gClient.init("", func(*grpc.ClientConn) interface{} { return nil })
}

func TestGrpcClose(t *testing.T) {
	s := &GrpcServer{gClient: new(grpcClient)}
	err := s.gClient.close()
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
	s := &GrpcServer{gClient: new(grpcClient)}
	s.Service()
}

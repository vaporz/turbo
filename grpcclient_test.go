package turbo

import (
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestGrpcInit(t *testing.T) {
	s := &Server{gClient: new(grpcClient)}
	s.gClient.grpcService = ""
	err := s.gClient.init("", func(*grpc.ClientConn) interface{} { return nil })
	assert.Nil(t, err)
}

func TestGrpcClose(t *testing.T) {
	s := &Server{gClient: new(grpcClient)}
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
	GrpcService(nil)
}

package component

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	"google.golang.org/grpc"
)

// GrpcClient returns a grpc client
func GrpcClient(conn *grpc.ClientConn) map[string]interface{} {
	return map[string]interface{}{
		"TestService": proto.NewTestServiceClient(conn),
		"MinionsService": proto.NewMinionsServiceClient(conn),
	}
}

type ServiceInitializer struct {
}

// InitService from defaultInitializer does nothing
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	return nil
}

// StopService from defaultInitializer does nothing
func (i *ServiceInitializer) StopService(s turbo.Servable) {
}

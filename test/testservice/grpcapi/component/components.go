package component

import (
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	"google.golang.org/grpc"
	"github.com/vaporz/turbo"
)

// GrpcClient returns a grpc client
func GrpcClient(conn *grpc.ClientConn) interface{} {
	return proto.NewTestServiceClient(conn)
}

// RegisterComponents inits turbo components, such as interceptors, pre/postprocessors, errorHandlers, etc.
func RegisterComponents(s *turbo.GrpcServer) {
	 //s.RegisterComponent("name", component)
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

package component

import (
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	"google.golang.org/grpc"
)

// GrpcClient returns a grpc client
func GrpcClient(conn *grpc.ClientConn) interface{} {
	return proto.NewTestServiceClient(conn)
}

// InitComponents inits turbo components, such as interceptors, pre/postprocessors, errorHandlers, etc.
func InitComponents() {
}

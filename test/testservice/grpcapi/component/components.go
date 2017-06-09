package component

import (
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	"google.golang.org/grpc"
)

func GrpcClient(conn *grpc.ClientConn) interface{} {
	return proto.NewTestServiceClient(conn)
}

func InitComponents() {
}

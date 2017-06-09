package impl

import (
	"golang.org/x/net/context"
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	"google.golang.org/grpc"
)

func RegisterServer(s *grpc.Server) {
	proto.RegisterTestServiceServer(s, &TestService{})
}

type TestService struct {
}

func (s *TestService) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}

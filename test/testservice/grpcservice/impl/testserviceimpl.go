package impl

import (
	"encoding/json"
	"errors"
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func RegisterServer(s *grpc.Server) {
	proto.RegisterTestServiceServer(s, &TestService{})
}

type TestService struct {
}

func (s *TestService) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	if req.BoolValue {
		bytes, err := json.Marshal(req)
		if err != nil {
			return &proto.SayHelloResponse{}, err
		}
		return &proto.SayHelloResponse{Message: string(bytes)}, nil
	}
	if req.YourName == "error" {
		return &proto.SayHelloResponse{}, errors.New("grpc error")
	}
	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}

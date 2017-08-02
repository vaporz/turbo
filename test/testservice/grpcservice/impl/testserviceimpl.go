package impl

import (
	"encoding/json"
	"errors"
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// RegisterServer registers a service struct to a server
func RegisterServer(s *grpc.Server) {
	proto.RegisterTestServiceServer(s, &TestService{})
}

// TestService is the struct which implements generated interface
type TestService struct {
}

// SayHello is an example entry point
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
	trailer := metadata.Pairs("trailer-key", "trailerval")
	grpc.SetTrailer(ctx, trailer)
	header := metadata.Pairs("header-key", "headerval")
	grpc.SetHeader(ctx, header)
	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}

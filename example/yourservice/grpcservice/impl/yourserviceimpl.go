package impl

import (
	"golang.org/x/net/context"
	"turbo/example/yourservice/gen/proto"
	"fmt"
)

type YourService struct {
}

func (s *YourService) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	fmt.Println(req.Values.TransactionId)
	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}

func (s *YourService) EatApple(ctx context.Context, req *proto.EatAppleRequest) (*proto.EatAppleResponse, error) {
	msg := fmt.Sprintf("[grpc server]Good taste! Apple num=%d, string=%s, bool=%t", req.Num, req.StringValue, req.BoolValue)
	return &proto.EatAppleResponse{Message: msg}, nil
}

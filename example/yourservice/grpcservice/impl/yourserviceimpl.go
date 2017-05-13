package impl

import (
	"golang.org/x/net/context"
	"turbo/example/yourservice/gen"
	"fmt"
)

type YourService struct {
}

func (s *YourService) SayHello(ctx context.Context, req *gen.SayHelloRequest) (*gen.SayHelloResponse, error) {
	fmt.Println(req.Values.TransactionId)
	return &gen.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}

func (s *YourService) EatApple(ctx context.Context, req *gen.EatAppleRequest) (*gen.EatAppleResponse, error) {
	msg := fmt.Sprintf("[grpc server]Good taste! Apple num=%d, string=%s, bool=%t", req.Num, req.StringValue, req.BoolValue)
	return &gen.EatAppleResponse{Message: msg}, nil
}

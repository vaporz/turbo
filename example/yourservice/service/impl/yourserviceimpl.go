package impl

import (
	"golang.org/x/net/context"
	"turbo/example/yourservice/gen"
	"strconv"
)

type YourService struct {
}

func (s *YourService) SayHello(ctx context.Context, req *gen.SayHelloRequest) (*gen.SayHelloResponse, error) {
	return &gen.SayHelloResponse{Message: "Hello, " + req.YourName}, nil
}

func (s *YourService) EatApple(ctx context.Context, req *gen.EatAppleRequest) (*gen.EatAppleResponse, error) {
	msg := "Good taste! Apple num=" + strconv.Itoa(int(req.Num)) + ", string=" + req.StringValue + ", bool=" + strconv.FormatBool(req.BoolValue)
	return &gen.EatAppleResponse{Message: msg}, nil
}

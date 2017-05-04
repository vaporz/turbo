package impl

import (
	"golang.org/x/net/context"
	"turbo/example/yourservice/gen"
)

type YourService struct {
}

func (s *YourService) SayHello(ctx context.Context, req *gen.SayHelloRequest) (*gen.SayHelloResponse, error) {
	return &gen.SayHelloResponse{Message: "Hello, " + req.YourName}, nil
}

func (s *YourService) EatApple(ctx context.Context, req *gen.EatAppleRequest) (*gen.EatAppleResponse, error) {
	return &gen.EatAppleResponse{Message: "Good taste!"}, nil
}
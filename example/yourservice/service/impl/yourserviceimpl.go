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

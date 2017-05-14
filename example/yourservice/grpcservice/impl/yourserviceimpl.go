package impl

import (
	"golang.org/x/net/context"
	"turbo/example/yourservice/gen/proto"
	"strconv"
)

type YourService struct {
}

func (s *YourService) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	someId := strconv.FormatInt(req.Values.SomeId, 10)
	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName + ", someId=" + someId}, nil
}

func (s *YourService) EatApple(ctx context.Context, req *proto.EatAppleRequest) (*proto.EatAppleResponse, error) {
	return &proto.EatAppleResponse{Message: "Good taste! Apple num=" + strconv.FormatInt(int64(req.Num), 10)}, nil
}

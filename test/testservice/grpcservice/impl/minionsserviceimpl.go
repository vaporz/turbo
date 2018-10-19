package impl

import (
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	"golang.org/x/net/context"
)

type MinionsService struct {
}

func (s *MinionsService) Eat(ctx context.Context, req *proto.EatRequest) (*proto.EatResponse, error) {
	if req.Food != "banana" {
		return &proto.EatResponse{Message: "Uh..."}, nil
	}
	return &proto.EatResponse{Message: "Yummy!"}, nil
}

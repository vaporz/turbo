package impl

import (
	"context"
	"github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
)

type MinionsService struct {
}

// SayHello is an example entry point
func (m MinionsService) Eat(ctx context.Context, food string) (r *gen.EatResponse, err error) {
	if food != "banana" {
		return &gen.EatResponse{Message: "Uh..."}, nil
	}
	return &gen.EatResponse{Message: "Yummy!"}, nil
}

package impl

import (
	"github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/services"
)

type MinionsService struct {
}

// SayHello is an example entry point
func (m MinionsService) Eat(food string) (r *services.EatResponse, err error) {
	if food != "banana" {
		return &services.EatResponse{Message: "Uh..."}, nil
	}
	return &services.EatResponse{Message: "Yummy!"}, nil
}

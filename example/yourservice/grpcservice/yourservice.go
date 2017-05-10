package main

import (
	"net"
	"log"
	"google.golang.org/grpc"
	"github.com/vaporz/turbo/example/yourservice/grpcservice/impl"
	"github.com/vaporz/turbo/example/yourservice/gen"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	gen.RegisterYourServiceServer(grpcServer, &impl.YourService{})

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

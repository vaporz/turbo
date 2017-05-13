package main

import (
	"net"
	"log"
	"google.golang.org/grpc"
	"turbo/example/yourservice/grpcservice/impl"
	"turbo/example/yourservice/gen/proto"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	proto.RegisterYourServiceServer(grpcServer, &impl.YourService{})

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

package main

import (
	_ "fmt"
	"net"
	"log"
	"google.golang.org/grpc"
	"zx/demo/inventoryservice/service/impl"
	pb "zx/demo/proto/inventoryservice"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterInventoryServiceServer(grpcServer, &impl.InventoryService{})

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

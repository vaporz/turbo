package main

import (
	pb "zx/demo/framework/example/inventoryservice/proto"
	"google.golang.org/grpc"
	f "zx/demo/framework"
	"zx/demo/framework/example/inventoryservice/http/gen"
)

func main() {
	f.InitHandler(gen.Handler)
	f.StartGrpcHTTPServer(grpcClient)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return pb.NewInventoryServiceClient(conn)
}

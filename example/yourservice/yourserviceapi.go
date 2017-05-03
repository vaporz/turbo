package main

import (
	"turbo"
	"google.golang.org/grpc"
	"yourservice/gen"
)

func main() {
	turbo.StartGrpcHTTPServer("turbo/example/yourservice", grpcClient, gen.Switcher)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return gen.NewYourServiceClient(conn)
}

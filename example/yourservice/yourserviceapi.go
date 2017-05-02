package main

import (
	"flag"
	"fmt"
	"os"
	"turbo"
	"google.golang.org/grpc"
	"yourservice/gen"
)

var (
	pkgPath = flag.String("p", "", "package path")
)

func main() {
	flag.Parse()
	if len(*pkgPath) == 0 {
		fmt.Println("package path is empty")
		os.Exit(1)
	}

	turbo.StartGrpcHTTPServer(*pkgPath, grpcClient, gen.Switcher)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return gen.NewYourServiceClient(conn)
}

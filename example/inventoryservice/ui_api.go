package main

import (
	g "turbo/example/inventoryservice/gen"
	"google.golang.org/grpc"
	"flag"
	"os"
	"fmt"
	"turbo"
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
	turbo.LoadServiceConfig(*pkgPath)
	turbo.InitHandler(g.Handler)
	turbo.StartGrpcHTTPServer(grpcClient)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return g.NewInventoryServiceClient(conn)
}

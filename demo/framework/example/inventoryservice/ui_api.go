package main

import (
	g "zx/demo/framework/example/inventoryservice/gen"
	"google.golang.org/grpc"
	f "zx/demo/framework"
	"flag"
	"os"
	"fmt"
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
	f.LoadServiceConfig(*pkgPath)
	f.InitHandler(g.Handler)
	f.StartGrpcHTTPServer(grpcClient)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return g.NewInventoryServiceClient(conn)
}

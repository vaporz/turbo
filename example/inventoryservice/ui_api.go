package main

import (
	g "turbo/example/inventoryservice/gen"
	"google.golang.org/grpc"
	"flag"
	"os"
	"fmt"
	"turbo"
	"net/http"
	"errors"
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
	// TODO make a methodName list from .proto
	turbo.SetHijacker("GetVideo", hijackGetVideo)
	turbo.SetPreprocessor("GetVideo", checkGetVideo)

	turbo.StartGrpcHTTPServer(*pkgPath, grpcClient, g.Handler)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return g.NewInventoryServiceClient(conn)
}

func hijackGetVideo(resp http.ResponseWriter, req *http.Request) {
	client := turbo.GrpcService().(g.InventoryServiceClient)
	r := new(g.GetVideoRequest)
	r.Id = "hijacked id!!!"
	res, err := client.GetVideo(req.Context(), r)
	if err == nil {
		resp.Write([]byte(res.String() + "\n"))
	} else {
		resp.Write([]byte(err.Error() + "\n"))
	}
}

func checkGetVideo(resp http.ResponseWriter, req *http.Request) error {
	a := req.Form["a"][0]
	resp.Write([]byte("param a ==" + a + "!\n"))
	req.Form["id"][0] = "333333333"
	return errors.New("")
	//return nil
}

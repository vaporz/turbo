package main

import (
	client "zx/demo/inventoryservice/http/component/clients"
	c "zx/demo/inventoryservice/http/component"
	pb "zx/demo/proto/inventoryservice"
	"net/http"
	"log"
	"google.golang.org/grpc"
)

func main() {
	c.LoadServiceConfig()

	client.InitGrpcConnection(getClient)
	defer client.CloseGrpcConnection()
	s := &http.Server{
		Addr:    ":8081",
		Handler: c.Router(), // TODO interceptors: loginRequired, loggerContext, formatter
	}
	log.Fatal(s.ListenAndServe())
}

func getClient(conn *grpc.ClientConn) interface{} {
	return pb.NewInventoryServiceClient(conn)
}

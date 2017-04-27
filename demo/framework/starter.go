package framework

import (
	client "zx/demo/framework/clients"
	"net/http"
	"log"
	"google.golang.org/grpc"
)

func StartGrpcHTTPServer(clientCreator func(conn *grpc.ClientConn) interface{}) {
	LoadServiceConfig()

	client.InitGrpcConnection(clientCreator)
	defer client.CloseGrpcConnection()
	s := &http.Server{
		Addr:    ":8081",
		Handler: Router(), // TODO interceptors: loginRequired, loggerContext, formatter
	}
	log.Fatal(s.ListenAndServe())
}

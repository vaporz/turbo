package turbo

import (
	"net/http"
	"log"
	"google.golang.org/grpc"
)

func StartGrpcHTTPServer(clientCreator func(conn *grpc.ClientConn) interface{}) {

	InitGrpcConnection(clientCreator)
	defer CloseGrpcConnection()
	s := &http.Server{
		Addr:    ":8081",
		Handler: Router(), // TODO register interceptors: loginRequired, loggerContext, formatter
	}
	// TODO start a goroutine
	log.Fatal(s.ListenAndServe())
}

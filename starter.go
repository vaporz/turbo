package turbo

import (
	"net/http"
	"log"
	"google.golang.org/grpc"
)

func StartGrpcHTTPServer(pkgPath string, clientCreator func(conn *grpc.ClientConn) interface{}, h func(methodName string) func(http.ResponseWriter, *http.Request)) {
	loadServiceConfig(pkgPath)
	initHandler(h)
	initGrpcConnection(clientCreator)
	defer closeGrpcConnection()
	s := &http.Server{
		Addr:    ":" + configs[PORT],
		Handler: router(), // TODO register interceptors: loginRequired, loggerContext, formatter
	}
	// TODO start a goroutine, start multi http server at different port
	log.Fatal(s.ListenAndServe())
}

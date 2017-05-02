package turbo

import (
	"net/http"
	"log"
	"google.golang.org/grpc"
)

func StartGrpcHTTPServer(pkgPath string, clientCreator func(conn *grpc.ClientConn) interface{}, switcher func(string, http.ResponseWriter, *http.Request)) {
	loadServiceConfig(pkgPath)
	initSwitcher(switcher)
	initGrpcConnection(clientCreator)
	defer closeGrpcConnection()
	s := &http.Server{
		Addr:    ":" + configs[PORT],
		Handler: router(),
	}
	// TODO start a goroutine, start multi http server at different port
	log.Fatal(s.ListenAndServe())
}

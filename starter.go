package turbo

import (
	"fmt"
	"google.golang.org/grpc"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func StartGrpcHTTPServer(pkgPath string, clientCreator func(conn *grpc.ClientConn) interface{}, switcher func(string, http.ResponseWriter, *http.Request)) {
	initPkgPath(pkgPath)
	LoadServiceConfig()
	initGrpcConnection(clientCreator)
	defer closeGrpcConnection()
	startHTTPServer(configs[PORT], router(switcher))
}

func startHTTPServer(port string, router http.Handler) {
	s := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	go s.ListenAndServe()
	//wait for exit
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-exit:
		fmt.Println("Received CTRL-C")
		break
	}
	fmt.Println("Server exit")
}

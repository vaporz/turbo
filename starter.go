package turbo

import (
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"google.golang.org/grpc"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func StartGrpcHTTPServer(pkgPath string, clientCreator func(conn *grpc.ClientConn) interface{}, switcher func(string, http.ResponseWriter, *http.Request) (interface{}, error)) {
	initPkgPath(pkgPath)
	LoadServiceConfig()
	err := initGrpcService(clientCreator)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer closeGrpcService()
	startHTTPServer(configs[PORT], router(switcher))
}

func StartThriftHTTPServer(pkgPath string, clientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}, switcher func(string, http.ResponseWriter, *http.Request) (interface{}, error)) {
	initPkgPath(pkgPath)
	LoadServiceConfig()
	err := initThriftService(clientCreator)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer closeThriftService()
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

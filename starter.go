package turbo

import (
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// TODO start both http and grpc/thrift with one command

type grpcClientCreator func(conn *grpc.ClientConn) interface{}

// StartGrpcHTTPServer starts a HTTP server which sends requests via grpc
func StartGrpcHTTPServer(pkgPath, configFileName string, clientCreator grpcClientCreator, s switcher) {
	LoadServiceConfig("grpc", pkgPath, configFileName)
	err := initGrpcService(clientCreator)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer closeGrpcService()
	startHTTPServer(Config.HTTPPortStr(), router(s))
}

type thriftClientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func StartThriftHTTPServer(pkgPath, configFileName string, clientCreator thriftClientCreator, s switcher) {
	LoadServiceConfig("thrift", pkgPath, configFileName)
	err := initThriftService(clientCreator)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer closeThriftService()
	startHTTPServer(Config.HTTPPortStr(), router(s))
}

func startHTTPServer(portStr string, router http.Handler) {
	s := &http.Server{
		Addr:    portStr,
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

func StartGrpcService(port int, registerServer func(s *grpc.Server)) {
	portStr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	registerServer(grpcServer)
	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func StartThriftService(port int, registerTProcessor func() thrift.TProcessor) {
	portStr := fmt.Sprintf(":%d", port)
	transport, err := thrift.NewTServerSocket(portStr)
	if err != nil {
		log.Println("socket error")
		os.Exit(1)
	}
	server := thrift.NewTSimpleServer4(registerTProcessor(), transport,
		thrift.NewTTransportFactory(), thrift.NewTBinaryProtocolFactoryDefault())
	server.Serve()
}

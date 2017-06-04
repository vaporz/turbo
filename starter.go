package turbo

import (
	"context"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var serviceStarted = make(chan bool)

var httpServerQuit = make(chan bool)
var serviceQuit = make(chan bool)

var reloadConfig = make(chan bool)

func waitForQuit() {
	<-httpServerQuit
	<-serviceQuit
	//for _, c := range waitOnList {
	// error:
	// transport: http2Server.HandleStreams failed to read frame:
	// read tcp 127.0.0.1:50051->127.0.0.1:55313: use of closed network connection
	//	fmt.Printf(strconv.FormatBool(<-c))
	//}
}

type grpcClientCreator func(conn *grpc.ClientConn) interface{}

// StartGRPC starts both HTTP server and GRPC service
func StartGRPC(pkgPath, configFileName string, servicePort int, clientCreator grpcClientCreator, s switcher, registerServer func(s *grpc.Server)) {
	initLogger()
	log.Info("Starting Turbo...")
	LoadServiceConfig("grpc", pkgPath, configFileName)
	go startGrpcServiceInternal(servicePort, registerServer, false)
	<-serviceStarted
	go startGrpcHTTPServerInternal(clientCreator, s)
	waitForQuit()
	log.Info("Turbo exit, bye!")
}

// StartGrpcHTTPServer starts a HTTP server which sends requests via grpc
func StartGrpcHTTPServer(pkgPath, configFileName string, clientCreator grpcClientCreator, s switcher) {
	LoadServiceConfig("grpc", pkgPath, configFileName)
	startGrpcHTTPServerInternal(clientCreator, s)
}

func startGrpcHTTPServerInternal(clientCreator grpcClientCreator, s switcher) {
	log.Info("Starting HTTP Server...")
	switcherFunc = s
	err := initGrpcService(clientCreator)
	if err != nil {
		//fmt.Println(err.Error())
		//os.Exit(1)
		log.Fatal(err.Error())
	}
	defer closeGrpcService()
	startHTTPServer(Config.HTTPPortStr(), router())
}

type thriftClientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}

// StartTHRIFT starts both HTTP server and Thrift service
func StartTHRIFT(pkgPath, configFileName string, port int, clientCreator thriftClientCreator, s switcher, registerTProcessor func() thrift.TProcessor) {
	log.Info("Starting Turbo...")
	LoadServiceConfig("grpc", pkgPath, configFileName)
	go startThriftServiceInternal(port, registerTProcessor, false)
	<-serviceStarted
	go startThriftHTTPServerInternal(clientCreator, s)
	waitForQuit()
	log.Info("Turbo exit, bye!")
}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func StartThriftHTTPServer(pkgPath, configFileName string, clientCreator thriftClientCreator, s switcher) {
	LoadServiceConfig("thrift", pkgPath, configFileName)
	startThriftHTTPServerInternal(clientCreator, s)
}

func startThriftHTTPServerInternal(clientCreator thriftClientCreator, s switcher) {
	log.Info("Starting HTTP Server...")
	switcherFunc = s
	err := initThriftService(clientCreator)
	if err != nil {
		//fmt.Println(err.Error())
		//os.Exit(1)
		log.Fatal(err.Error())
	}
	defer closeThriftService()
	startHTTPServer(Config.HTTPPortStr(), router())
}

func startHTTPServer(portStr string, handler http.Handler) {
	s := &http.Server{
		Addr:    portStr,
		Handler: handler,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.ErrorF("HTTP Server failed to serve: %v", err)
		}
	}()
	log.Info("HTTP Server started")
	for {
		//wait for exit
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
		select {
		case <-exit:
			log.Info("Received CTRL-C, HTTP Server is shutting down...")
			shutDownHTTP(s)
			log.Info("HTTP Server stopped")
			close(httpServerQuit)
			return
		case <-reloadConfig:
			s.Handler = router()
			log.Info("HTTP Server ServeMux reloaded")
		}
	}
}

func shutDownHTTP(s *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	s.Shutdown(ctx)
}

// StartGrpcService starts a GRPC service
func StartGrpcService(port int, registerServer func(s *grpc.Server)) {
	initLogger()
	startGrpcServiceInternal(port, registerServer, true)
}

func startGrpcServiceInternal(port int, registerServer func(s *grpc.Server), alone bool) {
	log.Info("Starting GRPC Service...")
	portStr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", portStr)
	if err != nil {
		log.FatalF("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	registerServer(grpcServer)
	reflection.Register(grpcServer)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.ErrorF("GRPC Service failed to serve: %v", err)
		}
	}()
	log.Info("GRPC Service started")
	serviceStarted <- true

	if !alone {
		<-httpServerQuit // wait for http server quit
		log.Info("Stopping GRPC Service...")
		grpcServer.Stop()
		log.Info("GRPC Service stopped")
		close(serviceQuit)
	} else {
		//wait for exit
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
		select {
		case <-exit:
			log.Info("Received CTRL-C, GRPC Service is stopping...")
			grpcServer.Stop()
			log.Info("GRPC Service stopped")
		}
	}
}

// StartThriftService starts a Thrift service
func StartThriftService(port int, registerTProcessor func() thrift.TProcessor) {
	startThriftServiceInternal(port, registerTProcessor, true)
}

func startThriftServiceInternal(port int, registerTProcessor func() thrift.TProcessor, alone bool) {
	log.Info("Starting Thrift Service...")
	portStr := fmt.Sprintf(":%d", port)
	transport, err := thrift.NewTServerSocket(portStr)
	if err != nil {
		//log.Println("socket error")
		//os.Exit(1)
		log.Fatal("socket error")
	}
	server := thrift.NewTSimpleServer4(registerTProcessor(), transport,
		thrift.NewTTransportFactory(), thrift.NewTBinaryProtocolFactoryDefault())
	//go server.Serve()
	err = server.Listen()
	if err != nil {
		panic(err)
	}
	go server.AcceptLoop()
	log.Info("Thrift Service started")
	serviceStarted <- true

	if !alone {
		<-httpServerQuit // wait for http server quit
		log.Info("Stopping Thrift Service...")
		server.Stop()
		log.Info("Thrift Service stopped")
		close(serviceQuit)
	} else {
		//wait for exit
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
		select {
		case <-exit:
			log.Info("Received CTRL-C, Thrift Service is stopping...")
			server.Stop()
			log.Info("Thrift Service stopped")
		}
	}
}

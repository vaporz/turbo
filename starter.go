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

var serviceStarted = make(chan bool, 1)

var httpServerQuit = make(chan bool)
var serviceQuit = make(chan bool)

var reloadConfig = make(chan bool)
var stopHttp = make(chan string, 1)
var stopService = make(chan string, 1)

func ResetChans() {
	serviceStarted = make(chan bool, 1)

	httpServerQuit = make(chan bool)
	serviceQuit = make(chan bool)

	reloadConfig = make(chan bool)
	stopHttp = make(chan string, 1)
	stopService = make(chan string, 1)
}

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
func StartGRPC(pkgPath, configFileName string, servicePort int, clientCreator grpcClientCreator, s switcher,
	registerServer func(s *grpc.Server)) {
	LoadServiceConfig("grpc", pkgPath, configFileName)
	log.Info("Starting Turbo...")
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
	components = new(Components)
	gClient = new(grpcClient)
	err := gClient.init(clientCreator)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer gClient.close()
	startHTTPServer(Config.HTTPPortStr(), router())
}

type thriftClientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}

// StartTHRIFT starts both HTTP server and Thrift service
func StartTHRIFT(pkgPath, configFileName string, port int, clientCreator thriftClientCreator, s switcher,
	registerTProcessor func() thrift.TProcessor) {
	LoadServiceConfig("thrift", pkgPath, configFileName)
	log.Info("Starting Turbo...")
	go startThriftServiceInternal(port, registerTProcessor, false)
	<-serviceStarted
	time.Sleep(time.Second * 1)
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
	components = new(Components)
	tClient = new(thriftClient)
	err := tClient.init(clientCreator)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer tClient.close()
	startHTTPServer(Config.HTTPPortStr(), router())
}

func startHTTPServer(portStr string, handler http.Handler) {
	s := &http.Server{
		Addr:    portStr,
		Handler: handler,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Printf("HTTP Server failed to serve: %v", err)
		}
	}()
	log.Info("HTTP Server started")
	//wait for exit
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		select {
		case v := <-stopHttp:
			if v == "http" {
				shutDownHTTP(s)
				return
			}
		case <-exit:
			shutDownHTTP(s)
			stopService <- "service"
			return
		case <-reloadConfig:
			s.Handler = router()
			log.Info("HTTP Server ServeMux reloaded")
		}
	}
}

func shutDownHTTP(s *http.Server) {
	log.Info("HTTP Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	s.Shutdown(ctx)
	log.Info("HTTP Server stop")
	close(httpServerQuit)
}

// StartGrpcService starts a GRPC service
func StartGrpcService(port int, pkgPath, configFileName string, registerServer func(s *grpc.Server)) {
	LoadServiceConfig("grpc", pkgPath, configFileName)
	startGrpcServiceInternal(port, registerServer, true)
}

func startGrpcServiceInternal(port int, registerServer func(s *grpc.Server), alone bool) {
	log.Info("Starting GRPC Service...")
	portStr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	registerServer(grpcServer)
	reflection.Register(grpcServer)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("GRPC Service failed to serve: %v", err)
		}
	}()
	log.Info("GRPC Service started")
	serviceStarted <- true

	if alone {
		//TODO bug:exit can't log.
		//wait for exit
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	doSelect:
		select {
		case v := <-stopService:
			if v != "service" {
				goto doSelect
			}
			grpcServer.GracefulStop()
			log.Info("GRPC Service stop")
		case <-exit:
			log.Info("Received CTRL-C, GRPC Service is stopping...")
			grpcServer.GracefulStop()
			log.Info("GRPC Service stop")
		}
		close(serviceQuit)
	} else {
		fmt.Println("waiting for http server quit")
		<-stopService
		<-httpServerQuit // wait for http server quit
		log.Info("Stopping GRPC Service...")
		grpcServer.GracefulStop()
		log.Info("GRPC Service stop")
		close(serviceQuit)
	}
}

// StartThriftService starts a Thrift service
func StartThriftService(port int, pkgPath, configFileName string, registerTProcessor func() thrift.TProcessor) {
	LoadServiceConfig("thrift", pkgPath, configFileName)
	startThriftServiceInternal(port, registerTProcessor, true)
}

func startThriftServiceInternal(port int, registerTProcessor func() thrift.TProcessor, alone bool) {
	log.Infof("Starting Thrift Service at :%d...", port)
	portStr := fmt.Sprintf(":%d", port)
	transport, err := thrift.NewTServerSocket(portStr)
	if err != nil {
		log.Fatal("socket error")
	}
	server := thrift.NewTSimpleServer4(registerTProcessor(), transport,
		thrift.NewTTransportFactory(), thrift.NewTBinaryProtocolFactoryDefault())
	go server.Serve()
	log.Info("Thrift Service started")
	serviceStarted <- true

	if alone {
		//wait for exit
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	doSelect:
		select {
		case v := <-stopService:
			if v != "service" {
				goto doSelect
			}
			log.Info("stop, Thrift Service is stopping...")
			server.Stop()
			log.Info("Thrift Service stop")
		case <-exit:
			log.Info("Received CTRL-C, Thrift Service is stopping...")
			server.Stop()
			log.Info("Thrift Service stop")
		}
		close(serviceQuit)
	} else {
		fmt.Println("waiting for http server quit")
		<-stopService
		<-httpServerQuit // wait for http server quit
		log.Info("Stopping Thrift Service...")
		server.Stop()
		log.Info("Thrift Service stop")
		close(serviceQuit)
	}
}

func Stop() {
	stopHttp <- "http"
	<-httpServerQuit
	stopService <- "service"
	<-serviceQuit
}

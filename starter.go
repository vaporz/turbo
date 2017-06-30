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

var client *Client

// Client holds the data for a server
type Client struct {
	// TODO add config
	components   *Components
	gClient      *grpcClient
	tClient      *thriftClient
	switcherFunc switcher
}

var serviceStarted = make(chan bool, 1)

var httpServerQuit = make(chan bool)
var serviceQuit = make(chan bool)

var reloadConfig = make(chan bool)
var stopHttp = make(chan string, 1)
var stopService = make(chan string, 1)

// ResetChans resets chan vars
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
}

type grpcClientCreator func(conn *grpc.ClientConn) interface{}

// StartGRPC starts both HTTP server and GRPC service
func StartGRPC(configFilePath string, clientCreator grpcClientCreator, s switcher,
	registerServer func(s *grpc.Server)) {
	c := LoadServiceConfig("grpc", configFilePath)
	log.Info("Starting Turbo...")
	go startGrpcServiceInternal(c, registerServer, false)
	<-serviceStarted
	go startGrpcHTTPServerInternal(c, clientCreator, s)
	waitForQuit()
	log.Info("Turbo exit, bye!")
}

// StartGrpcHTTPServer starts a HTTP server which sends requests via grpc
func StartGrpcHTTPServer(configFilePath string, clientCreator grpcClientCreator, s switcher) {
	c := LoadServiceConfig("grpc", configFilePath)
	startGrpcHTTPServerInternal(c, clientCreator, s)
}

func startGrpcHTTPServerInternal(c *Config, clientCreator grpcClientCreator, s switcher) {
	log.Info("Starting HTTP Server...")
	client = &Client{
		components:   new(Components),
		gClient:      new(grpcClient),
		switcherFunc: s}
	err := client.gClient.init(c.GrpcServiceAddress(), clientCreator)
	if err != nil {
		log.Panic(err.Error())
	}
	defer client.gClient.close()
	startHTTPServer(c)
}

type thriftClientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}

// StartTHRIFT starts both HTTP server and Thrift service
func StartTHRIFT(configFilePath string, clientCreator thriftClientCreator, s switcher,
	registerTProcessor func() thrift.TProcessor) {
	c := LoadServiceConfig("thrift", configFilePath)
	log.Info("Starting Turbo...")
	go startThriftServiceInternal(c, registerTProcessor, false)
	<-serviceStarted
	time.Sleep(time.Second * 1)
	go startThriftHTTPServerInternal(c, clientCreator, s)
	waitForQuit()
	log.Info("Turbo exit, bye!")
}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func StartThriftHTTPServer(configFilePath string, clientCreator thriftClientCreator, s switcher) {
	c := LoadServiceConfig("thrift", configFilePath)
	startThriftHTTPServerInternal(c, clientCreator, s)
}

func startThriftHTTPServerInternal(c *Config, clientCreator thriftClientCreator, s switcher) {
	log.Info("Starting HTTP Server...")
	client = &Client{
		components:   new(Components),
		tClient:      new(thriftClient),
		switcherFunc: s}
	err := client.tClient.init(c.ThriftServiceAddress(), clientCreator)
	if err != nil {
		log.Panic(err.Error())
	}
	defer client.tClient.close()
	startHTTPServer(c)
}

func startHTTPServer(c *Config) {
	s := &http.Server{
		Addr:    c.HTTPPortStr(),
		Handler: router(c),
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
			log.Info("Config file changed!")
			s.Handler = router(c)
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
func StartGrpcService(configFilePath string, registerServer func(s *grpc.Server)) {
	c := LoadServiceConfig("grpc", configFilePath)
	startGrpcServiceInternal(c, registerServer, true)
}

func startGrpcServiceInternal(c *Config, registerServer func(s *grpc.Server), alone bool) {
	log.Info("Starting GRPC Service...")
	lis, err := net.Listen("tcp", ":"+c.GrpcServicePort())
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
func StartThriftService(configFilePath string, registerTProcessor func() thrift.TProcessor) {
	c := LoadServiceConfig("thrift", configFilePath)
	startThriftServiceInternal(c, registerTProcessor, true)
}

func startThriftServiceInternal(c *Config, registerTProcessor func() thrift.TProcessor, alone bool) {
	port := c.ThriftServicePort()
	log.Infof("Starting Thrift Service at :%d...", port)
	transport, err := thrift.NewTServerSocket(":" + port)
	if err != nil {
		log.Panic("socket error")
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

// Stop stops the server gracefully
func Stop() {
	stopHttp <- "http"
	<-httpServerQuit
	stopService <- "service"
	<-serviceQuit
}

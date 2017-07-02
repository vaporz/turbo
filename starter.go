package turbo

import (
	"context"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Client holds the data for a server
type Server struct {
	config       *Config
	Components   *Components
	gClient      *grpcClient
	tClient      *thriftClient
	switcherFunc switcher
	chans        map[int]chan bool
}

func NewServer(rpcType, configFilePath string) *Server {
	s := &Server{
		config:     NewConfig(rpcType, configFilePath),
		Components: new(Components),
		gClient:    new(grpcClient),
		tClient:    new(thriftClient),
		chans:      make(map[int]chan bool)}
	s.initChans()
	s.watchConfig()
	initLogger(s.config)
	return s
}

func (s *Server) watchConfig() {
	s.config.WatchConfig()
	s.config.OnConfigChange(func(e fsnotify.Event) {
		s.config.loadServiceConfig(s.config.File)
		s.chans[reloadConfig] <- true
	})
}

const (
	serviceStarted = iota
	httpServerQuit
	serviceQuit
	reloadConfig
	stopHttp
	stopService
)

// ResetChans resets chan vars
func (s *Server) initChans() {
	s.chans[serviceStarted] = make(chan bool, 1)
	s.chans[httpServerQuit] = make(chan bool)
	s.chans[serviceQuit] = make(chan bool)
	s.chans[reloadConfig] = make(chan bool)
	s.chans[stopHttp] = make(chan bool, 1)
	s.chans[stopService] = make(chan bool, 1)
}

func (s *Server) waitForQuit() {
	<-s.chans[httpServerQuit]
	<-s.chans[serviceQuit]
}

// ResetComponents reset all component mappings
func (s *Server) ResetComponents() {
	s.Components = new(Components)
}

type grpcClientCreator func(conn *grpc.ClientConn) interface{}

// StartGRPC starts both HTTP server and GRPC service
func (s *Server) StartGRPC(clientCreator grpcClientCreator, sw switcher,
	registerServer func(s *grpc.Server)) {
	log.Info("Starting Turbo...")
	go s.startGrpcServiceInternal(registerServer, false)
	<-s.chans[serviceStarted]
	go s.StartGrpcHTTPServer(clientCreator, sw)
	s.waitForQuit()
	log.Info("Turbo exit, bye!")
}

// StartGrpcHTTPServer starts a HTTP server which sends requests via grpc
func (s *Server) StartGrpcHTTPServer(clientCreator grpcClientCreator, sw switcher) {
	log.Info("Starting HTTP Server...")
	s.switcherFunc = sw
	err := s.gClient.init(s.config.GrpcServiceHost()+":"+s.config.GrpcServicePort(), clientCreator)
	if err != nil {
		log.Panic(err.Error())
	}
	defer s.gClient.close()
	s.startHTTPServer()
}

type thriftClientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}

// StartTHRIFT starts both HTTP server and Thrift service
func (s *Server) StartTHRIFT(clientCreator thriftClientCreator, sw switcher,
	registerTProcessor func() thrift.TProcessor) {
	log.Info("Starting Turbo...")
	go s.startThriftServiceInternal(registerTProcessor, false)
	<-s.chans[serviceStarted]
	time.Sleep(time.Second * 1)
	go s.StartThriftHTTPServer(clientCreator, sw)
	s.waitForQuit()
	log.Info("Turbo exit, bye!")
}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func (s *Server) StartThriftHTTPServer(clientCreator thriftClientCreator, sw switcher) {
	log.Info("Starting HTTP Server...")
	s.switcherFunc = sw
	err := s.tClient.init(s.config.ThriftServiceHost()+":"+s.config.ThriftServicePort(), clientCreator)
	if err != nil {
		log.Panic(err.Error())
	}
	defer s.tClient.close()
	s.startHTTPServer()
}

func (s *Server) startHTTPServer() {
	hs := &http.Server{
		Addr:    ":" + strconv.FormatInt(s.config.HTTPPort(), 10),
		Handler: router(s),
	}
	go func() {
		if err := hs.ListenAndServe(); err != nil {
			log.Printf("HTTP Server failed to serve: %v", err)
		}
	}()
	log.Info("HTTP Server started")
	//wait for exit
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		select {
		case <-s.chans[stopHttp]:
			s.shutDownHTTP(hs)
			return
		case <-exit:
			s.shutDownHTTP(hs)
			s.chans[stopService] <- true
			return
		case <-s.chans[reloadConfig]:
			log.Info("Config file changed!")
			hs.Handler = router(s)
			log.Info("HTTP Server ServeMux reloaded")
		}
	}
}

func (s *Server) shutDownHTTP(hs *http.Server) {
	log.Info("HTTP Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	hs.Shutdown(ctx)
	log.Info("HTTP Server stopped")
	close(s.chans[httpServerQuit])
}

// StartGrpcService starts a GRPC service
func (s *Server) StartGrpcService(registerServer func(s *grpc.Server)) {
	s.startGrpcServiceInternal(registerServer, true)
}

func (s *Server) startGrpcServiceInternal(registerServer func(s *grpc.Server), alone bool) {
	log.Info("Starting GRPC Service...")
	lis, err := net.Listen("tcp", ":"+s.config.GrpcServicePort())
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
	time.Sleep(time.Millisecond * 10)
	s.chans[serviceStarted] <- true
	if alone {
		//TODO bug:exit can't log.
		//wait for exit
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
		select {
		case <-s.chans[stopService]:
			grpcServer.GracefulStop()
			log.Info("GRPC Service stop")
		case <-exit:
			log.Info("Received CTRL-C, GRPC Service is stopping...")
			grpcServer.GracefulStop()
			log.Info("GRPC Service stopped")
		}
		close(s.chans[serviceQuit])
	} else {
		<-s.chans[stopService]
		<-s.chans[httpServerQuit] // wait for http server quit
		log.Info("Stopping GRPC Service...")
		grpcServer.GracefulStop()
		log.Info("GRPC Service stopped")
		close(s.chans[serviceQuit])
	}
}

// StartThriftService starts a Thrift service
func (s *Server) StartThriftService(registerTProcessor func() thrift.TProcessor) {
	s.startThriftServiceInternal(registerTProcessor, true)
}

func (s *Server) startThriftServiceInternal(registerTProcessor func() thrift.TProcessor, alone bool) {
	port := s.config.ThriftServicePort()
	log.Infof("Starting Thrift Service at :%d...", port)
	transport, err := thrift.NewTServerSocket(":" + port)
	if err != nil {
		log.Panic("socket error")
	}
	server := thrift.NewTSimpleServer4(registerTProcessor(), transport,
		thrift.NewTTransportFactory(), thrift.NewTBinaryProtocolFactoryDefault())
	go server.Serve()
	log.Info("Thrift Service started")
	s.chans[serviceStarted] <- true

	if alone {
		//wait for exit
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
		select {
		case <-s.chans[stopService]:
			log.Info("stop, Thrift Service is stopping...")
			server.Stop()
			log.Info("Thrift Service stop")
		case <-exit:
			log.Info("Received CTRL-C, Thrift Service is stopping...")
			server.Stop()
			log.Info("Thrift Service stopped")
		}
		close(s.chans[serviceQuit])
	} else {
		<-s.chans[stopService]
		<-s.chans[httpServerQuit] // wait for http server quit
		log.Info("Stopping Thrift Service...")
		server.Stop()
		log.Info("Thrift Service stopped")
		close(s.chans[serviceQuit])
	}
}

// Stop stops the server gracefully
func (s *Server) Stop() {
	s.chans[stopHttp] <- true
	<-s.chans[httpServerQuit]
	s.chans[stopService] <- true
	<-s.chans[serviceQuit]
}

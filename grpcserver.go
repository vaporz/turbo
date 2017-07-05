package turbo

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type GrpcServer struct {
	*Server
}

func NewGrpcServer(configFilePath string) *GrpcServer {
	s := &GrpcServer{
		Server: &Server{
			Config:     NewConfig("grpc", configFilePath),
			Components: new(Components),
			chans:      make(map[int]chan bool),
			gClient:    new(grpcClient),
		},
	}
	s.initChans()
	s.watchConfig()
	initLogger(s.Config)
	return s
}

type grpcClientCreator func(conn *grpc.ClientConn) interface{}

// StartGRPC starts both HTTP server and GRPC service
func (s *GrpcServer) StartGRPC(clientCreator grpcClientCreator, sw switcher,
	registerServer func(s *grpc.Server)) {
	log.Info("Starting Turbo...")
	go s.startGrpcServiceInternal(registerServer, false)
	<-s.chans[serviceStarted]
	go s.StartGrpcHTTPServer(clientCreator, sw)
	// TODO wait for exit here
	s.waitForQuit()
	log.Info("Turbo exit, bye!")
}

// StartGrpcHTTPServer starts a HTTP server which sends requests via grpc
func (s *GrpcServer) StartGrpcHTTPServer(clientCreator grpcClientCreator, sw switcher) {
	log.Info("Starting HTTP Server...")
	s.switcherFunc = sw
	err := s.gClient.init(s.Config.GrpcServiceHost()+":"+s.Config.GrpcServicePort(), clientCreator)
	if err != nil {
		log.Panic(err.Error())
	}
	defer s.gClient.close()
	s.startHTTPServer()
	// TODO wait for exit here
}

// StartGrpcService starts a GRPC service
func (s *GrpcServer) StartGrpcService(registerServer func(s *grpc.Server)) {
	s.startGrpcServiceInternal(registerServer, true)
	// TODO wait for exit here
}

func (s *GrpcServer) startGrpcServiceInternal(registerServer func(s *grpc.Server), alone bool) {
	log.Info("Starting GRPC Service...")
	lis, err := net.Listen("tcp", ":"+s.Config.GrpcServicePort())
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

package turbo

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
)

type GrpcServer struct {
	*Server
}

func NewGrpcServer(configFilePath string) *GrpcServer {
	s := &GrpcServer{
		Server: &Server{
			Config:       NewConfig("grpc", configFilePath),
			Components:   new(Components),
			reloadConfig: make(chan bool),
			gClient:      new(grpcClient),
			Initializer:  &defaultInitializer{},
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
	s.Initializer.InitService(s.Server)
	grpcServer := s.startGrpcServiceInternal(registerServer, false)
	httpServer := s.startGrpcHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, grpcServer, nil)
	log.Info("Turbo exit, bye!")
}

// StartGrpcHTTPServer starts a HTTP server which sends requests via grpc
func (s *GrpcServer) StartGrpcHTTPServer(clientCreator grpcClientCreator, sw switcher) {
	s.Initializer.InitService(s.Server)
	httpServer := s.startGrpcHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, nil, nil)
}

// StartGrpcService starts a GRPC service
func (s *GrpcServer) StartGrpcService(registerServer func(s *grpc.Server)) {
	s.Initializer.InitService(s.Server)
	grpcServer := s.startGrpcServiceInternal(registerServer, true)
	s.waitForQuit(nil, grpcServer, nil)
}

func (s *GrpcServer) startGrpcHTTPServerInternal(clientCreator grpcClientCreator, sw switcher) *http.Server {
	log.Info("Starting HTTP Server...")
	s.switcherFunc = sw
	err := s.gClient.init(s.Config.GrpcServiceHost()+":"+s.Config.GrpcServicePort(), clientCreator)
	if err != nil {
		log.Panic(err.Error())
	}
	return s.startHTTPServer()
}

func (s *GrpcServer) startGrpcServiceInternal(registerServer func(s *grpc.Server), alone bool) *grpc.Server {
	log.Info("Starting GRPC Service...")
	lis, err := net.Listen("tcp", ":"+s.Config.GrpcServicePort())
	if err != nil {
		log.Panic("failed to listen: %v", err)
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
	return grpcServer
}

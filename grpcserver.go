package turbo

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
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
	grpcServer := s.startGrpcServiceInternal(registerServer, false)
	httpServer := s.startGrpcHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, grpcServer)
	log.Info("Turbo exit, bye!")
}

// StartGrpcHTTPServer starts a HTTP server which sends requests via grpc
func (s *GrpcServer) StartGrpcHTTPServer(clientCreator grpcClientCreator, sw switcher) {
	httpServer := s.startGrpcHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, nil)
}

// StartGrpcService starts a GRPC service
func (s *GrpcServer) StartGrpcService(registerServer func(s *grpc.Server)) {
	grpcServer := s.startGrpcServiceInternal(registerServer, true)
	s.waitForQuit(nil, grpcServer)
}

func (s *GrpcServer) waitForQuit(httpServer *http.Server, grpcServer *grpc.Server) {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
Wait:
	select {
	case <-exit:
		log.Info("Received CTRL-C, Service is stopping...")
		if httpServer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			httpServer.Shutdown(ctx)
			log.Info("Http Server stopped")
		}
		if grpcServer != nil {
			s.gClient.close()
			grpcServer.GracefulStop()
			log.Info("Grpc Server stopped")
		}
	case <-s.chans[reloadConfig]:
		if httpServer == nil {
			goto Wait
		}
		log.Info("Reloading configuration...")
		httpServer.Handler = router(s.Server)
		log.Info("Router reloaded")
		s.Components = s.loadComponentsNoPanic()
		log.Info("Configuration reloaded")
	}
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
	return grpcServer
}

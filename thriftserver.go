package turbo

import (
	"context"
	"git.apache.org/thrift.git/lib/go/thrift"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ThriftServer struct {
	*Server
}

func NewThriftServer(configFilePath string) *ThriftServer {
	s := &ThriftServer{
		Server: &Server{
			Config:     NewConfig("thrift", configFilePath),
			Components: new(Components),
			chans:      make(map[int]chan bool),
			tClient:    new(thriftClient),
		},
	}
	s.initChans()
	s.watchConfig()
	initLogger(s.Config)
	return s
}

type thriftClientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}

// StartTHRIFT starts both HTTP server and Thrift service
func (s *ThriftServer) StartTHRIFT(clientCreator thriftClientCreator, sw switcher,
	registerTProcessor func() thrift.TProcessor) {
	log.Info("Starting Turbo...")
	thriftServer := s.startThriftServiceInternal(registerTProcessor, false)
	time.Sleep(time.Second * 1)
	httpServer := s.startThriftHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, thriftServer)
	log.Info("Turbo exit, bye!")
}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func (s *ThriftServer) StartThriftHTTPServer(clientCreator thriftClientCreator, sw switcher) {
	httpServer := s.startThriftHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, nil)
}

// StartThriftService starts a Thrift service
func (s *ThriftServer) StartThriftService(registerTProcessor func() thrift.TProcessor) {
	thriftServer := s.startThriftServiceInternal(registerTProcessor, true)
	s.waitForQuit(nil, thriftServer)
}

func (s *ThriftServer) waitForQuit(httpServer *http.Server, thriftServer *thrift.TSimpleServer) {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	// TODO split http server and grpc/thrift server
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
		if thriftServer != nil {
			s.tClient.close()
			thriftServer.Stop()
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

func (s *ThriftServer) startThriftHTTPServerInternal(clientCreator thriftClientCreator, sw switcher) *http.Server {
	log.Info("Starting HTTP Server...")
	s.switcherFunc = sw
	err := s.tClient.init(s.Config.ThriftServiceHost()+":"+s.Config.ThriftServicePort(), clientCreator)
	if err != nil {
		log.Panic(err.Error())
	}
	return s.startHTTPServer()
}

func (s *ThriftServer) startThriftServiceInternal(registerTProcessor func() thrift.TProcessor, alone bool) *thrift.TSimpleServer {
	port := s.Config.ThriftServicePort()
	log.Infof("Starting Thrift Service at :%d...", port)
	transport, err := thrift.NewTServerSocket(":" + port)
	if err != nil {
		log.Panic("socket error")
	}
	server := thrift.NewTSimpleServer4(registerTProcessor(), transport,
		thrift.NewTTransportFactory(), thrift.NewTBinaryProtocolFactoryDefault())
	go server.Serve()
	log.Info("Thrift Service started")
	return server
}

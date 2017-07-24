package turbo

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"net/http"
	"time"
)

type ThriftServer struct {
	*Server
}

func NewThriftServer(configFilePath string) *ThriftServer {
	s := &ThriftServer{
		Server: &Server{
			Config:       NewConfig("thrift", configFilePath),
			Components:   new(Components),
			reloadConfig: make(chan bool),
			tClient:      new(thriftClient),
			Initializer:  &defaultInitializer{},
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
	s.Initializer.InitService(s.Server)
	thriftServer := s.startThriftServiceInternal(registerTProcessor, false)
	time.Sleep(time.Second * 1)
	httpServer := s.startThriftHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, nil, thriftServer)
	log.Info("Turbo exit, bye!")
}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func (s *ThriftServer) StartThriftHTTPServer(clientCreator thriftClientCreator, sw switcher) {
	s.Initializer.InitService(s.Server)
	httpServer := s.startThriftHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, nil, nil)
	log.Info("Thrift HttpServer exit, bye!")
}

// StartThriftService starts a Thrift service
func (s *ThriftServer) StartThriftService(registerTProcessor func() thrift.TProcessor) {
	s.Initializer.InitService(s.Server)
	thriftServer := s.startThriftServiceInternal(registerTProcessor, true)
	s.waitForQuit(nil, nil, thriftServer)
	log.Info("Thrift Service exit, bye!")
}

func (s *ThriftServer) startThriftHTTPServerInternal(clientCreator thriftClientCreator, sw switcher) *http.Server {
	log.Info("Starting HTTP Server...")
	s.switcherFunc = sw
	s.tClient.init(s.Config.ThriftServiceHost()+":"+s.Config.ThriftServicePort(), clientCreator)
	return s.startHTTPServer()
}

func (s *ThriftServer) startThriftServiceInternal(registerTProcessor func() thrift.TProcessor, alone bool) *thrift.TSimpleServer {
	port := s.Config.ThriftServicePort()
	log.Infof("Starting Thrift Service at :%s...", port)
	transport, err := thrift.NewTServerSocket(":" + port)
	logPanicIf(err)
	server := thrift.NewTSimpleServer4(registerTProcessor(), transport,
		thrift.NewTTransportFactory(), thrift.NewTBinaryProtocolFactoryDefault())
	go server.Serve()
	log.Info("Thrift Service started")
	return server
}

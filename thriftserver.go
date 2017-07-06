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
	Initializer.InitService(s.Config)
	thriftServer := s.startThriftServiceInternal(registerTProcessor, false)
	time.Sleep(time.Second * 1)
	httpServer := s.startThriftHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, nil, thriftServer)
	log.Info("Turbo exit, bye!")
}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func (s *ThriftServer) StartThriftHTTPServer(clientCreator thriftClientCreator, sw switcher) {
	Initializer.InitService(s.Config)
	httpServer := s.startThriftHTTPServerInternal(clientCreator, sw)
	s.waitForQuit(httpServer, nil, nil)
}

// StartThriftService starts a Thrift service
func (s *ThriftServer) StartThriftService(registerTProcessor func() thrift.TProcessor) {
	Initializer.InitService(s.Config)
	thriftServer := s.startThriftServiceInternal(registerTProcessor, true)
	s.waitForQuit(nil, nil, thriftServer)
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

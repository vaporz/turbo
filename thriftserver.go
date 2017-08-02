package turbo

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"net/http"
	"time"
)

type ThriftServer struct {
	*Server
	tClient *thriftClient
}

func NewThriftServer(configFilePath string) *ThriftServer {
	s := &ThriftServer{
		Server: &Server{
			Config:       NewConfig("thrift", configFilePath),
			Components:   new(Components),
			reloadConfig: make(chan bool),
			Initializer:  &defaultInitializer{},
		},
		tClient: new(thriftClient),
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
	s.Initializer.InitService(s)
	thriftServer := s.startThriftServiceInternal(registerTProcessor, false)
	time.Sleep(time.Second * 1)
	httpServer := s.startThriftHTTPServerInternal(clientCreator, sw)
	waitForQuit(s, httpServer, nil, thriftServer)
	log.Info("Turbo exit, bye!")
}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func (s *ThriftServer) StartThriftHTTPServer(clientCreator thriftClientCreator, sw switcher) {
	s.Initializer.InitService(s)
	httpServer := s.startThriftHTTPServerInternal(clientCreator, sw)
	waitForQuit(s, httpServer, nil, nil)
	log.Info("Thrift HttpServer exit, bye!")
}

// StartThriftService starts a Thrift service
func (s *ThriftServer) StartThriftService(registerTProcessor func() thrift.TProcessor) {
	s.Initializer.InitService(s)
	thriftServer := s.startThriftServiceInternal(registerTProcessor, true)
	waitForQuit(s, nil, nil, thriftServer)
	log.Info("Thrift Service exit, bye!")
}

func (s *ThriftServer) startThriftHTTPServerInternal(clientCreator thriftClientCreator, sw switcher) *http.Server {
	log.Info("Starting HTTP Server...")
	switcherFunc = sw
	s.tClient.init(s.Config.ThriftServiceHost()+":"+s.Config.ThriftServicePort(), clientCreator)
	return startHTTPServer(s)
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

// ThriftService returns a Thrift client instance,
// example: client := turbo.ThriftService().(proto.YourServiceClient)
func (s *ThriftServer) Service() interface{} {
	if s == nil || s.tClient == nil || s.tClient.thriftService == nil {
		log.Panic("thrift connection not initiated!")
	}
	return s.tClient.thriftService
}

func (s *ThriftServer) ServerField() *Server { return s.Server }

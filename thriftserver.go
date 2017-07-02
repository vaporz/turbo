package turbo

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ThriftServer struct {
	Server
}

func NewThriftServer(configFilePath string) *ThriftServer {
	s := &ThriftServer{
		Server: Server{
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
	go s.startThriftServiceInternal(registerTProcessor, false)
	<-s.chans[serviceStarted]
	time.Sleep(time.Second * 1)
	go s.StartThriftHTTPServer(clientCreator, sw)
	s.waitForQuit()
	log.Info("Turbo exit, bye!")
}

// StartThriftHTTPServer starts a HTTP server which sends requests via Thrift
func (s *ThriftServer) StartThriftHTTPServer(clientCreator thriftClientCreator, sw switcher) {
	log.Info("Starting HTTP Server...")
	s.switcherFunc = sw
	err := s.tClient.init(s.Config.ThriftServiceHost()+":"+s.Config.ThriftServicePort(), clientCreator)
	if err != nil {
		log.Panic(err.Error())
	}
	defer s.tClient.close()
	s.startHTTPServer()
}

// StartThriftService starts a Thrift service
func (s *ThriftServer) StartThriftService(registerTProcessor func() thrift.TProcessor) {
	s.startThriftServiceInternal(registerTProcessor, true)
}

func (s *ThriftServer) startThriftServiceInternal(registerTProcessor func() thrift.TProcessor, alone bool) {
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

package turbo

import (
	"context"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Server holds the data for a server
type Server struct {
	// Config holds data read from config file
	Config *Config
	// Components holds the mappings of url to component
	Components           *Components
	switcherFunc         switcher
	gClient              *grpcClient
	tClient              *thriftClient
	registeredComponents map[string]interface{}
	reloadConfig         chan bool
	exit                 chan os.Signal
	// Initializer implements Initializable
	Initializer Initializable
}

// RegisterComponent registers a component,
// The convention is to register with the name of that component,
// the name is used in config file to look up for a component.
func (s *Server) RegisterComponent(name string, component interface{}) {
	if s.registeredComponents == nil {
		s.registeredComponents = make(map[string]interface{})
	}
	s.registeredComponents[name] = component
}

// Component returns a component by name.
func (s *Server) Component(name string) interface{} {
	if s.registeredComponents == nil {
		return nil
	}
	return s.registeredComponents[name]
}

func (s *Server) watchConfig() {
	s.Config.WatchConfig()
	s.Config.OnConfigChange(func(e fsnotify.Event) {
		s.Config.loadServiceConfig(s.Config.File)
		s.reloadConfig <- true
	})
}

func (s *Server) initChans() {
	s.reloadConfig = make(chan bool)
	s.exit = make(chan os.Signal, 1)
}

func (s *Server) startHTTPServer() *http.Server {
	s.Components = s.loadComponents()
	hs := &http.Server{
		Addr:    ":" + strconv.FormatInt(s.Config.HTTPPort(), 10),
		Handler: router(s),
	}
	go func() {
		if err := hs.ListenAndServe(); err != nil {
			log.Printf("HTTP Server failed to serve: %v", err)
		}
	}()
	log.Info("HTTP Server started")
	return hs
}

func (s *Server) loadComponentsNoPanic() *Components {
	defer func() {
		if err := recover(); err != nil {
			log.Error("reload components failed, err=", err)
		}
	}()
	return s.loadComponents()
}

func (s *Server) loadComponents() *Components {
	c := &Components{routers: make(map[int]*mux.Router)}
	for _, m := range s.Config.mappings[interceptors] {
		names := strings.Split(m[2], ",")
		components := make([]Interceptor, 0)
		for _, name := range names {
			components = append(components, s.Component(name).(Interceptor))
		}
		c.Intercept(strings.Split(m[0], ","), m[1], components...)
		log.Info("interceptor:", m)
	}
	for _, m := range s.Config.mappings[preprocessors] {
		c.SetPreprocessor(strings.Split(m[0], ","), m[1], s.Component(m[2]).(Preprocessor))
		log.Info("preprocessor:", m)
	}
	for _, m := range s.Config.mappings[postprocessors] {
		c.SetPostprocessor(strings.Split(m[0], ","), m[1], s.Component(m[2]).(Postprocessor))
		log.Info("postprocessor:", m)
	}
	for _, m := range s.Config.mappings[hijackers] {
		c.SetHijacker(strings.Split(m[0], ","), m[1], s.Component(m[2]).(Hijacker))
		log.Info("hijacker:", m)
	}
	for _, m := range s.Config.mappings[convertors] {
		c.SetMessageFieldConvertor(m[0], s.Component(m[1]).(Convertor))
		log.Info("convertor:", m)
	}
	if len(s.Config.ErrorHandler()) > 0 {
		c.WithErrorHandler(s.Component(s.Config.ErrorHandler()).(ErrorHandlerFunc))
		log.Info("errorhandler:", s.Config.ErrorHandler())
	}
	return c
}

func (s *Server) waitForQuit(httpServer *http.Server, grpcServer *grpc.Server, thriftServer *thrift.TSimpleServer) {
	signal.Notify(s.exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
Wait:
	select {
	case <-s.exit:
		log.Info("Received CTRL-C, Service is stopping...")
	case <-s.reloadConfig:
		if httpServer == nil {
			goto Wait
		}
		log.Info("Reloading configuration...")
		newComponents := s.loadComponentsNoPanic()
		newRouter := router(s)
		httpServer.Handler = newRouter
		log.Info("Router reloaded")
		s.Components = newComponents
		log.Info("Configuration reloaded")
		goto Wait
	}
	s.quit(httpServer, grpcServer, thriftServer)
}

func (s *Server) quit(httpServer *http.Server, grpcServer *grpc.Server, thriftServer *thrift.TSimpleServer) {
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
	if thriftServer != nil {
		s.tClient.close()
		thriftServer.Stop()
		log.Info("Grpc Server stopped")
	}
	s.Initializer.StopService(s)
}

// Stop stops the server gracefully
func (s *Server) Stop() {
	s.exit <- syscall.SIGQUIT
}

// GrpcService returns a grpc client instance,
// example: client := turbo.GrpcService().(proto.YourServiceClient)
func (s *Server) GrpcService() interface{} {
	if s == nil || s.gClient == nil || s.gClient.grpcService == nil {
		log.Panic("grpc connection not initiated!")
	}
	return s.gClient.grpcService
}

// ThriftService returns a Thrift client instance,
// example: client := turbo.ThriftService().(proto.YourServiceClient)
func (s *Server) ThriftService() interface{} {
	if s == nil || s.tClient == nil || s.tClient.thriftService == nil {
		log.Panic("thrift connection not initiated!")
	}
	return s.tClient.thriftService
}

// Initializable defines funcs run before service started and after service stopped
type Initializable interface {
	// InitService is run before the service is started, do initializing staffs for your service here
	InitService(s *Server) error

	// StopService is run after both grpc server and http server are stopped,
	// do your cleaning up work here.
	StopService(s *Server)
}

type defaultInitializer struct {
}

// InitService from defaultInitializer does nothing
func (d *defaultInitializer) InitService(s *Server) error {
	return nil
}

// StopService from defaultInitializer does nothing
func (d *defaultInitializer) StopService(s *Server) {
}

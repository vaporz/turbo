/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// TODO use dep

// TODO try to use sync.Once

// TODO Make Ctrl+C cancel the context.Context
// https://medium.com/@matryer/make-ctrl-c-cancel-the-context-context-bd006a8ad6ff

type Servable interface {
	Service(serviceName string) interface{}
	ServerField() *Server
	Stop()
	RegisterComponent(name string, component interface{})
}

// Server holds the data for a server
type Server struct {
	// Config holds data read from config file
	Config *Config
	// Components holds the mappings of url to component
	Components   *Components
	reloadConfig chan bool
	exit         chan os.Signal
	// Initializer implements Initializable
	Initializer Initializable
	httpServer  *http.Server
}

func (s *Server) Service() interface{} {
	fmt.Println("oh no!!!")
	return nil
}
func (s *Server) ServerField() *Server { return s }

// Stop stops the server gracefully
func (s *Server) Stop() { return }

// RegisterComponent registers a component,
// The convention is to register with the name of that component,
// the name is used in config file to look up for a component.
func (s *Server) RegisterComponent(name string, component interface{}) {
	if s.Components.registeredComponents == nil {
		s.Components.registeredComponents = make(map[string]interface{})
	}
	s.Components.registeredComponents[name] = component
}

// Component returns a component by name.
func (s *Server) Component(name string) (interface{}, error) {
	if s.Components.registeredComponents[name] == nil {
		return nil, errors.New("no such component: " + name + ", forget to register?")
	}
	return s.Components.registeredComponents[name], nil
}

func watchConfigReload(s Servable) {
	s.ServerField().watchConfig()
	go func() {
		for {
			select {
			case <-s.ServerField().reloadConfig:
				if s.ServerField().httpServer == nil {
					continue
				}
				log.Info("Reloading configuration...")
				newComponents := s.ServerField().loadComponentsNoPanic()
				newRouter := router(s)
				s.ServerField().httpServer.Handler = newRouter
				s.ServerField().Components = newComponents
				log.Info("Configuration reloaded")
			}
		}
	}()
}

func (s *Server) watchConfig() {
	s.Config.WatchConfig()
	s.Config.OnConfigChange(func(e fsnotify.Event) {
		c := &Config{
			Viper:    *viper.New(),
			File:     s.Config.File,
			mappings: make(map[string][][4]string)}
		c.loadServiceConfig()
		s.Config = c
		s.reloadConfig <- true
	})
}

func (s *Server) initChans() {
	s.reloadConfig = make(chan bool)
	s.exit = make(chan os.Signal, 1)
}

func startHTTPServer(s Servable) *http.Server {
	s.ServerField().Components = s.ServerField().loadComponents()
	hs := &http.Server{
		Addr:    ":" + strconv.FormatInt(s.ServerField().Config.HTTPPort(), 10),
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
			log.Error("reload Components failed, err=", err)
		}
	}()
	return s.loadComponents()
}

func (s *Server) loadComponents() *Components {
	c := &Components{routers: make(map[int]*mux.Router), registeredComponents: s.Components.registeredComponents}
	for _, m := range s.Config.mappings[interceptors] {
		names := strings.Split(m[2], ",")
		components := make([]Interceptor, 0)
		for _, name := range names {
			components = append(components, getComponentByName(s, name).(Interceptor))
		}
		c.Intercept(strings.Split(m[0], ","), m[1], components...)
		log.Info("interceptor:", m)
	}
	for _, m := range s.Config.mappings[preprocessors] {
		c.SetPreprocessor(strings.Split(m[0], ","), m[1], getComponentByName(s, m[2]).(Preprocessor))
		log.Info("preprocessor:", m)
	}
	for _, m := range s.Config.mappings[postprocessors] {
		c.SetPostprocessor(strings.Split(m[0], ","), m[1], getComponentByName(s, m[2]).(Postprocessor))
		log.Info("postprocessor:", m)
	}
	for _, m := range s.Config.mappings[hijackers] {
		c.SetHijacker(strings.Split(m[0], ","), m[1], getComponentByName(s, m[2]).(Hijacker))
		log.Info("hijacker:", m)
	}
	for _, m := range s.Config.mappings[convertors] {
		c.SetConvertor(m[0], getComponentByName(s, m[1]).(Convertor))
		log.Info("convertor:", m)
	}
	if len(s.Config.ErrorHandler()) > 0 {
		c.WithErrorHandler(getComponentByName(s, s.Config.ErrorHandler()).(ErrorHandlerFunc))
		log.Info("errorhandler:", s.Config.ErrorHandler())
	}
	return c
}

func getComponentByName(s *Server, name string) interface{} {
	com, err := s.Component(name)
	if err != nil {
		panic(err)
	}
	return com
}

func stop(s Servable, httpServer *http.Server, grpcServer *grpc.Server, thriftServer *thrift.TSimpleServer) {
	// if s.ServerField().exit is not closed, close it, return directly
	if httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		httpServer.Shutdown(ctx)
		log.Info("Http Server stopped")
	}
	if grpcServer != nil {
		s.(*GrpcServer).gClient.close()
		grpcServer.GracefulStop()
		log.Info("Grpc Server stopped")
	}
	if thriftServer != nil {
		s.(*ThriftServer).tClient.close()
		thriftServer.Stop()
		log.Info("Thrift Server stopped")
	}
	s.ServerField().Initializer.StopService(s)
}

// Initializable defines funcs run before service started and after service stopped
type Initializable interface {
	// InitService is run before the service is started, do initializing staffs for your service here
	InitService(s Servable) error

	// StopService is run after both grpc server and http server are stopped,
	// do your cleaning up work here.
	StopService(s Servable)
}

type defaultInitializer struct {
}

// InitService from defaultInitializer does nothing
func (d *defaultInitializer) InitService(s Servable) error {
	return nil
}

// StopService from defaultInitializer does nothing
func (d *defaultInitializer) StopService(s Servable) {
}

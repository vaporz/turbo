/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

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
	Components *Components
	// componentsLock guards Components and currentRouter: a request reads them
	// while a configuration reload replaces them.
	componentsLock sync.RWMutex
	// currentRouter is what the HTTP server is serving. net/http reads
	// http.Server.Handler for every request, so a reload must never write that
	// field; it swaps this value instead.
	currentRouter http.Handler
	// done is closed by Stop. After that no reload is started, and the reload
	// goroutine returns instead of waiting for a signal nobody will send.
	done         chan struct{}
	stopOnce     sync.Once
	reloadConfig chan *Config
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
func (s *Server) Stop() { s.shutdown() }

// shutdown closes done once, which stops the reload goroutine and keeps the
// configuration watcher from handing it any more work.
func (s *Server) shutdown() {
	s.stopOnce.Do(func() {
		if s.done != nil {
			close(s.done)
		}
	})
}

// RegisterComponent registers a component.
//
// The convention is to register with the name of that component, the name is
// used in config file to look up for a component.
//
// One instance serves every request, and it is called concurrently: see the note
// on Interceptor about request state.
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
	// say it once, where somebody looks when they change a setting and wonder why
	// nothing happened
	log.Info("turbo: a configuration change reloads urlmapping, components and " +
		"filter_proto_json; http_port, grpc_service_port, thrift_service_port, " +
		"environment, turbo_log_path and log_level need a restart")
	s.ServerField().watchConfig()
	go func() {
		for {
			select {
			case <-s.ServerField().done:
				return
			case previous := <-s.ServerField().reloadConfig:
				if s.ServerField().httpServer == nil {
					continue
				}
				log.Info("Reloading configuration...")
				if err := reloadComponents(s, previous); err != nil {
					// Never install a routing table that could not be built: the
					// server keeps serving the configuration already in effect,
					// and keeps watching for the next change.
					log.Error("turbo: configuration reload failed, keeping the running configuration: ", err)
					continue
				}
				log.Info("Configuration reloaded")
			}
		}
	}()
}

// How long a reload waits for a configuration file to stop changing, and how
// often it looks. See waitForStableFile.
const (
	fileSettlePoll  = 20 * time.Millisecond  // how often the file is looked at
	fileSettleQuiet = 100 * time.Millisecond // how long it has to hold still
	fileSettleWait  = 3 * time.Second        // the longest a reload waits
)

// waitForStableFile waits until a configuration file stops changing, so that a
// reload never reads one that is still being written.
//
// A configuration file is usually written in place: the writer truncates it and
// writes the new content after that, and the filesystem reports the truncation
// at once. A reader that reacts to that first event gets an empty or half
// written file, which is how one configuration change can replace a working
// routing table with an empty one.
//
// Two reads that return the same bytes, fileSettleQuiet apart, mean the writer
// has finished. An empty file does not count as settled, because a writer needs
// a moment to produce its first bytes -- a copy over a slow or remote path
// writes its first block only after a round trip. The wait is bounded, so a
// writer that never finishes cannot keep the reload from happening; whatever the
// file holds by then goes through the normal validation, which refuses a
// configuration with no route.
func waitForStableFile(path string) {
	deadline := time.Now().Add(fileSettleWait)
	var previous []byte
	var previousAt time.Time
	for {
		current, err := os.ReadFile(path)
		if err != nil {
			// there is no file to wait for; the loader reports what is wrong
			return
		}
		now := time.Now()
		if len(current) > 0 && bytes.Equal(current, previous) && now.Sub(previousAt) >= fileSettleQuiet {
			return
		}
		if !bytes.Equal(current, previous) {
			previous, previousAt = current, now
		}
		if !now.Before(deadline) {
			return
		}
		time.Sleep(fileSettlePoll)
	}
}

func (s *Server) watchConfig() {
	s.Config.WatchConfig()
	s.Config.OnConfigChange(func(e fsnotify.Event) {
		// Read the path under the lock and let it go before waiting: the wait can
		// last, and the reload path replaces this configuration while a request is
		// being served.
		s.componentsLock.RLock()
		path := s.Config.File
		s.componentsLock.RUnlock()

		// Never read a file somebody is still writing: see waitForStableFile.
		waitForStableFile(path)
		c := &Config{
			Viper:    *viper.New(),
			File:     path,
			mappings: make(map[string][][4]string)}
		if err := c.loadServiceConfigErr(); err != nil {
			// The change is ignored rather than fatal. It used to panic here, in
			// the goroutine that watches the file, where nothing recovered it and
			// the whole process died.
			log.Error("turbo: ignoring configuration change, it cannot be loaded: ", err)
			return
		}
		select {
		case <-s.done:
			// the server has been stopped: there is no reloader left to hand
			// this change to
			return
		default:
		}
		s.componentsLock.Lock()
		previous := s.Config
		s.Config = c
		s.componentsLock.Unlock()
		select {
		case s.reloadConfig <- previous:
		default:
			// a reload is already pending; it will read the newest config
		}
	})
}

func (s *Server) initChans() {
	// buffered and sent to without blocking: after Stop nobody is receiving, and
	// the callback that watches the file must not be left blocked on a send
	s.reloadConfig = make(chan *Config, 1)
	s.done = make(chan struct{})
	s.exit = make(chan os.Signal, 1)
}

func startHTTPServer(s Servable) *http.Server {
	s.ServerField().componentsLock.Lock()
	s.ServerField().Components = s.ServerField().loadComponents()
	s.ServerField().currentRouter = router(s)
	s.ServerField().componentsLock.Unlock()
	hs := &http.Server{
		Addr:    ":" + strconv.FormatInt(s.ServerField().Config.HTTPPort(), 10),
		Handler: currentHandler(s),
	}
	go func() {
		if err := hs.ListenAndServe(); err != nil {
			log.Printf("HTTP Server failed to serve: %v", err)
		}
	}()
	log.Info("HTTP Server started")
	return hs
}

// loadComponentsErr builds the components the current configuration describes,
// reporting an error instead of panicking. A configuration that names a
// component nobody registered cannot be turned into a working routing table, and
// the caller must be able to keep the one that already works.
func (s *Server) loadComponentsErr() (components *Components, err error) {
	defer func() {
		if r := recover(); r != nil {
			components, err = nil, fmt.Errorf("invalid configuration %s: %v", s.Config.File, r)
		}
	}()
	return s.loadComponents(), nil
}

// reloadComponents builds the components the configuration in effect describes,
// installs them together with the routing table built from that same
// configuration, and reports what stopped it when the configuration cannot be
// turned into components. previous is the configuration that was in effect before
// the change: a failed rebuild leaves it in place, so the server keeps serving
// what its components were built from.
//
// The rebuild runs under the write lock. It reads the configuration and the
// components the service registered, and both are written while the server serves
// requests -- the watcher replaces the configuration, and a service registers a
// component or installs a common interceptor. Without the lock those reads race
// with the writes, and a change arriving in the middle could be read half way:
// part of the routing table built from one configuration, the rest from another.
// A request takes the same lock only to look the handler up and releases it before
// serving, so holding it here costs one rebuild per configuration change.
func reloadComponents(s Servable, previous *Config) error {
	server := s.ServerField()
	server.componentsLock.Lock()
	defer server.componentsLock.Unlock()

	newComponents, err := server.loadComponentsErr()
	if err != nil {
		server.Config = previous
		return err
	}
	server.currentRouter = router(s)
	server.Components = newComponents
	return nil
}

func (s *Server) loadComponents() *Components {
	// Common interceptors are installed in code (SetCommonInterceptor), typically
	// before the server starts, while this rebuild runs at startup and on every
	// successful reload. Carrying the registered components over but not these
	// turned every cross-cutting interceptor off on the first configuration change
	// -- logging, metrics and, worst of all, a global authentication interceptor.
	c := &Components{
		routers:              make(map[int]*mux.Router),
		registeredComponents: s.Components.registeredComponents,
		commonInterceptors:   s.Components.commonInterceptors,
	}
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
	// The audit runs on every load: at startup a violation refuses to start, and
	// on reload it is an error that keeps the running configuration in place.
	panicIf(auditRoutes(s.Config.mappings, c.commonInterceptors, c.registeredComponents, s.Config.authConfig()))
	return c
}

func getComponentByName(s *Server, name string) interface{} {
	com, err := s.Component(name)
	if err != nil {
		panic(err)
	}
	return com
}

// currentHandler serves whatever router is in effect. It exists so that a reload
// never writes http.Server.Handler, which net/http reads for every request.
func currentHandler(s Servable) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		s.ServerField().componentsLock.RLock()
		handler := s.ServerField().currentRouter
		s.ServerField().componentsLock.RUnlock()
		if handler == nil {
			http.NotFound(resp, req)
			return
		}
		handler.ServeHTTP(resp, req)
	})
}

func stop(s Servable, httpServer *http.Server, grpcServer *grpc.Server, thriftServer *thrift.TSimpleServer) {
	s.ServerField().shutdown()
	s.ServerField().Initializer.StopService(s)
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

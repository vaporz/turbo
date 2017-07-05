package turbo

import (
	"github.com/fsnotify/fsnotify"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Server holds the data for a server
type Server struct {
	Config               *Config
	Components           *Components
	switcherFunc         switcher
	chans                map[int]chan bool
	gClient              *grpcClient
	tClient              *thriftClient
	registeredComponents map[string]interface{}
	exit                 chan os.Signal
}

func (s *Server) RegisterComponent(name string, component interface{}) {
	if s.registeredComponents == nil {
		s.registeredComponents = make(map[string]interface{})
	}
	s.registeredComponents[name] = component
}

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
		s.chans[reloadConfig] <- true
	})
}

const (
	reloadConfig = iota
)

func (s *Server) initChans() {
	s.chans[reloadConfig] = make(chan bool)
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
	c := &Components{}
	for _, m := range s.Config.interceptors {
		names := strings.Split(m[2], ",")
		components := make([]Interceptor, 0)
		for _, name := range names {
			components = append(components, s.Component(name).(Interceptor))
		}
		c.Intercept(strings.Split(m[0], ","), m[1], components...)
		log.Info("interceptor:", m)
	}
	for _, m := range s.Config.preprocessors {
		c.SetPreprocessor(strings.Split(m[0], ","), m[1], s.Component(m[2]).(Preprocessor))
		log.Info("preprocessor:", m)
	}
	for _, m := range s.Config.postprocessors {
		c.SetPostprocessor(strings.Split(m[0], ","), m[1], s.Component(m[2]).(Postprocessor))
		log.Info("postprocessor:", m)
	}
	for _, m := range s.Config.hijackers {
		c.SetHijacker(strings.Split(m[0], ","), m[1], s.Component(m[2]).(Hijacker))
		log.Info("hijacker:", m)
	}
	for _, m := range s.Config.convertors {
		c.SetMessageFieldConvertor(m[0], s.Component(m[1]).(Convertor))
		log.Info("convertor:", m)
	}
	if len(s.Config.errorhandler) > 0 {
		c.WithErrorHandler(s.Component(s.Config.errorhandler).(ErrorHandlerFunc))
		log.Info("errorhandler:", s.Config.errorhandler)
	}
	return c
}

// Stop stops the server gracefully
func (s *Server) Stop() {
	s.exit <- syscall.SIGQUIT
}

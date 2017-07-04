package turbo

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type TurboServer interface {
}

// Client holds the data for a server
type Server struct {
	Config       *Config
	Components   *Components
	switcherFunc switcher
	chans        map[int]chan bool
	gClient      *grpcClient
	tClient      *thriftClient
}

func (s *Server) RegisterComponent(name string, component interface{}) {
	s.Components.RegisterComponent(name, component)
}

func (s *Server) watchConfig() {
	s.Config.WatchConfig()
	s.Config.OnConfigChange(func(e fsnotify.Event) {
		s.Config.loadServiceConfig(s.Config.File)
		s.chans[reloadConfig] <- true
	})
}

const (
	serviceStarted = iota
	httpServerQuit
	serviceQuit
	reloadConfig
	stopHttp
	stopService
)

// ResetChans resets chan vars
func (s *Server) initChans() {
	s.chans[serviceStarted] = make(chan bool, 1)
	s.chans[httpServerQuit] = make(chan bool)
	s.chans[serviceQuit] = make(chan bool)
	s.chans[reloadConfig] = make(chan bool)
	s.chans[stopHttp] = make(chan bool, 1)
	s.chans[stopService] = make(chan bool, 1)
}

func (s *Server) waitForQuit() {
	<-s.chans[httpServerQuit]
	<-s.chans[serviceQuit]
}

func (s *Server) startHTTPServer() {
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
	//wait for exit
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		select {
		case <-s.chans[stopHttp]:
			s.shutDownHTTP(hs)
			return
		case <-exit:
			s.shutDownHTTP(hs)
			s.chans[stopService] <- true
			return
		case <-s.chans[reloadConfig]:
			log.Info("Config file changed!")
			hs.Handler = router(s)
			log.Info("HTTP Server ServeMux reloaded")
		}
	}
}

func (s *Server) shutDownHTTP(hs *http.Server) {
	log.Info("HTTP Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	hs.Shutdown(ctx)
	log.Info("HTTP Server stopped")
	close(s.chans[httpServerQuit])
}

// Stop stops the server gracefully
func (s *Server) Stop() {
	s.chans[stopHttp] <- true
	<-s.chans[httpServerQuit]
	s.chans[stopService] <- true
	<-s.chans[serviceQuit]
}

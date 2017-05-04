package interceptor

import (
	"turbo"
	"log"
	"net/http"
)

type LogInterceptor struct {
	// optional, BaseInterceptor allows you to create an interceptor which implements
	// Before() or After() only, or none of them.
	// If you were to implement both, you can remove this line.
	turbo.BaseInterceptor
}

func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	log.Println("[Before] Request URL:" + req.URL.Path)
	return nil
}

func (l LogInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	log.Println("[After] Request URL:" + req.URL.Path)
	return nil
}

package interceptor

import (
	"turbo"
	"log"
	"net/http"
)

type LogInterceptor struct {
	turbo.BaseInterceptor
}

func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	log.Println("loginterceptor before!!!!")
	return nil
}

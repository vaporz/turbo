package interceptor

import (
	"turbo"
	"log"
	"net/http"
)

type LoginInterceptor struct {
	turbo.BaseInterceptor
}

func (l LoginInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	log.Println("login interceptor before!!!!")
	//return errors.New("login interceptor error!")
	return nil
}

func (l LoginInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	log.Println("login interceptor after!!!!")
	return nil
}

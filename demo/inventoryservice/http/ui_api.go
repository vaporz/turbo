package main

import (
	c "zx/demo/inventoryservice/http/component"
	"net/http"
	"log"
)

func main() {
	c.InitGrpcConnection()
	defer c.CloseGrpcConnection()
	s := &http.Server{
		Addr:    ":8081",
		Handler: c.Router(), // TODO interceptors: loginRequired, loggerContext, formatter
	}
	log.Fatal(s.ListenAndServe())
}

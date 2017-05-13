package main

import (
	"turbo/example/yourservice/thriftservice/impl"
	"turbo/example/yourservice/gen/thrift/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
	"log"
	"os"
)

func main() {
	transport, err := thrift.NewTServerSocket(":50052")
	if err != nil {
		log.Println("socket error")
		os.Exit(1)
	}

	server := thrift.NewTSimpleServer4(gen.NewYourServiceProcessor(impl.YourService{}), transport,
		thrift.NewTTransportFactory(), thrift.NewTBinaryProtocolFactoryDefault())
	server.Serve()
}

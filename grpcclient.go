package turbo

import (
	"google.golang.org/grpc"
	"log"
)

var (
	gClient     = new(grpcClient)
	grpcService interface{}
)

type grpcClient struct {
	conn *grpc.ClientConn
}

func (g *grpcClient) dial(address string) (err error) {
	if g.conn, err = grpc.Dial(address, grpc.WithInsecure()); err != nil {
		log.Fatalln("connect error:" + err.Error())
	}
	return err
}

func (g *grpcClient) close() error {
	if g.conn == nil {
		return nil
	}
	return g.conn.Close()
}

func initGrpcService(clientCreator func(conn *grpc.ClientConn) interface{}) error {
	// TODO support multiple grpc clients
	// TODO support grpcservice discovery
	if grpcService != nil {
		return nil
	}
	addr := Config.GrpcServiceAddress()
	if len(addr) == 0 {
		panic("Error: missing [grpc_service_address] in config")
	}
	err := gClient.dial(addr)
	if err == nil {
		grpcService = clientCreator(gClient.conn)
	}
	return err
}

func closeGrpcService() error {
	return gClient.close()
}

// GrpcService returns a grpc client instance,
// example: client := turbo.GrpcService().(proto.YourServiceClient)
func GrpcService() interface{} {
	if grpcService == nil {
		log.Fatalln("grpc connection not initiated!")
	}
	return grpcService
}

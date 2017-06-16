package turbo

import (
	"google.golang.org/grpc"
)

type grpcClient struct {
	grpcService interface{}
	conn        *grpc.ClientConn
}

func (g *grpcClient) init(clientCreator func(conn *grpc.ClientConn) interface{}) error {
	// ???? support multiple grpc clients
	// ???? support grpcservice discovery
	if g.grpcService != nil {
		return nil
	}
	addr := Config.GrpcServiceAddress()
	if len(addr) == 0 {
		log.Panic("Error: missing [grpc_service_address] in config")
	}
	log.Info("[grpc]connecting addr:", addr)
	err := g.dial(addr)
	if err == nil {
		g.grpcService = clientCreator(g.conn)
	}
	return err
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

// GrpcService returns a grpc client instance,
// example: client := turbo.GrpcService().(proto.YourServiceClient)
func GrpcService() interface{} {
	if client.gClient.grpcService == nil {
		log.Fatalln("grpc connection not initiated!")
	}
	return client.gClient.grpcService
}

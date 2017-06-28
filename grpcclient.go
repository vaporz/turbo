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
	log.Info("[grpc]connecting addr:", addr)
	err := g.dial(addr)
	if err == nil {
		g.grpcService = clientCreator(g.conn)
	}
	return err
}

func (g *grpcClient) dial(address string) (err error) {
	g.conn, err = grpc.Dial(address, grpc.WithInsecure())
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
	if client == nil || client.gClient == nil || client.gClient.grpcService == nil {
		log.Panic("grpc connection not initiated!")
	}
	return client.gClient.grpcService
}

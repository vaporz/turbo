package turbo

import (
	"google.golang.org/grpc"
)

type grpcClient struct {
	grpcService interface{}
	conn        *grpc.ClientConn
}

func (g *grpcClient) init(addr string, clientCreator func(conn *grpc.ClientConn) interface{}) {
	// ???? support multiple grpc clients
	// ???? support grpcservice discovery
	if g.grpcService != nil {
		return
	}
	log.Info("[grpc]connecting addr:", addr)
	g.dial(addr)
	g.grpcService = clientCreator(g.conn)
}

func (g *grpcClient) dial(address string) {
	var err error
	g.conn, err = grpc.Dial(address, grpc.WithInsecure())
	logPanicIf(err)
}

func (g *grpcClient) close() error {
	if g.conn == nil {
		return nil
	}
	return g.conn.Close()
}

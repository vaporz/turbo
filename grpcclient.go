package turbo

import (
	"google.golang.org/grpc"
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
	// ???? support multiple grpc clients
	// ???? support grpcservice discovery
	if grpcService != nil {
		return nil
	}
	addr := Config.GrpcServiceAddress()
	if len(addr) == 0 {
		log.Panic("Error: missing [grpc_service_address] in config")
	}
	log.Info("[grpc]connecting addr:", addr)
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

// TODO refactor and remove all such Reset Funcs
func ResetGrpcClient(){
	gClient     = new(grpcClient)
	grpcService = nil
}

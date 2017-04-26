package clients

import (
	"google.golang.org/grpc"
	pb "zx/demo/proto/inventoryservice"
	"log"
)

var (
	client           = new(grpcClient)
	inventoryService pb.InventoryServiceClient
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

func InitGrpcConnection() error {
	// TODO support multiple grpc clients
	// TODO support service discovery
	if inventoryService != nil {
		return nil
	}
	// TODO read from config file
	if err := client.dial("127.0.0.1:50051"); err == nil {
		inventoryService = pb.NewInventoryServiceClient(client.conn)
	}
	return nil
}

func CloseGrpcConnection() error {
	return client.close()
}

func InventoryService() pb.InventoryServiceClient {
	if inventoryService == nil {
		log.Fatalln("grpc connection not initiated!")
	}
	return inventoryService
}

/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"google.golang.org/grpc"
)

type grpcClient struct {
	grpcServiceMap map[string]interface{}
	conn           *grpc.ClientConn
}

func (g *grpcClient) init(addr string, clientCreator grpcClientCreator) {
	// ???? support grpcservice discovery
	if g.grpcServiceMap != nil {
		return
	}
	log.Info("[grpc]connecting addr:", addr)
	g.dial(addr)
	g.grpcServiceMap = clientCreator(g.conn)
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

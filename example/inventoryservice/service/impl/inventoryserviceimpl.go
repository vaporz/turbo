package impl

import (
	"golang.org/x/net/context"
	p "turbo/example/inventoryservice/gen"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type InventoryService struct {
}

func (s *InventoryService) GetVideoList(ctx context.Context, req *p.GetVideoListRequest) (*p.GetVideoListResponse, error) {
	fmt.Println("[GetListVideo]incomming requeset!!! networkId=" + req.NetworkId)
	var list []*p.Video
	list = append(list, &p.Video{Id: 111, Name: "test video"})
	list = append(list, &p.Video{Id: 222, Name: "test video222"})
	return &p.GetVideoListResponse{Videos: list}, nil
}

func (s *InventoryService) GetVideo(ctx context.Context, req *p.GetVideoRequest) (*p.GetVideoResponse, error) {
	fmt.Println("[GetVideo]incomming requeset!!! video id=" + req.Id)
	video := &p.Video{Id: 123, Name: "videovideo"}
	err := grpc.Errorf(codes.Internal, "error=%d", 111)
	return &p.GetVideoResponse{Video: video}, err
}

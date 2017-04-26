package impl

import (
	"golang.org/x/net/context"
	is "zx/demo/proto/inventoryservice"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type InventoryService struct {
}

func (s *InventoryService) GetVideoList(ctx context.Context, req *is.GetVideoListRequest) (*is.GetVideoListResponse, error) {
	fmt.Println("[GetListVideo]incomming requeset!!! networkId=" + req.NetworkId)
	var list []*is.Video
	list = append(list, &is.Video{Id: 111, Name: "test video"})
	list = append(list, &is.Video{Id: 222, Name: "test video222"})
	return &is.GetVideoListResponse{Videos: list}, nil
}

func (s *InventoryService) GetVideo(ctx context.Context, req *is.GetVideoRequest) (*is.GetVideoResponse, error) {
	fmt.Println("[GetVideo]incomming requeset!!! video id=" + req.Id)
	video := &is.Video{Id: 123, Name: "videovideo"}
	err := grpc.Errorf(codes.Internal, "error=%d", 111)
	return &is.GetVideoResponse{Video: video}, err
}

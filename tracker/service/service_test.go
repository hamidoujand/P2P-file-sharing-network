package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/pb"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRegisterPeer(t *testing.T) {
	store := peerstore.New()
	service := service.New(store)

	in := pb.RegisterPeerRequest{
		Ip:   "127.0.0.1",
		Port: 50051,
		Files: []*pb.File{
			{
				Name:         "file1.txt",
				Size:         10,
				Checksum:     "some-hash",
				LastModified: timestamppb.New(time.Now()),
				FileType:     "txt",
			},
		},
	}

	resp, err := service.RegisterPeer(context.Background(), &in)
	if err != nil {
		t.Fatalf("expected to get a response: %s", err)
	}

	if resp.StatusCode != uint32(codes.OK) {
		t.Errorf("code=%d, got %d", codes.OK, resp.StatusCode)
	}
}

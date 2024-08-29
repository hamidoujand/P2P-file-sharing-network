package service_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/pb"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRegisterPeer(t *testing.T) {
	store := peerstore.New()
	service := service.New(store)

	in := pb.RegisterPeerRequest{
		Host: "127.0.0.1:50051",
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

func TestUnRegisterPeer(t *testing.T) {
	store := peerstore.New()
	service := service.New(store)

	host := "127.0.0.1:50051"
	in := pb.RegisterPeerRequest{
		Host: host,
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

	input := pb.UnRegisterPeerRequest{
		Host: host,
	}

	response, err := service.UnRegisterPeer(context.Background(), &input)
	if err != nil {
		t.Fatalf("expected to unregister peer from network: %s", err)
	}

	if response.StatusCode != uint32(codes.OK) {
		t.Errorf("code= %d, got %d", codes.OK, resp.StatusCode)
	}

	//not-found network
	input = pb.UnRegisterPeerRequest{
		Host: "0.0.0.0:9000",
	}
	_, err = service.UnRegisterPeer(context.Background(), &input)
	if err == nil {
		t.Fatal("expected to get an error while unregistering a not found peer")
	}

	status, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected to get a status from err")
	}
	if status.Code() != codes.NotFound {
		t.Fatalf("status=%d, got %d", codes.NotFound, status.Code())
	}
}

func TestGetPeers(t *testing.T) {
	peers := map[string]*pb.File{
		"127.0.0.1:50051": {
			Name:         "file1.txt",
			Size:         10,
			FileType:     "txt",
			Checksum:     "some-radom-hash",
			LastModified: timestamppb.Now(),
		},
		"198.168.2.1:50031": {
			Name:         "file2.txt",
			Size:         20,
			FileType:     "txt",
			Checksum:     "some-radom-hash",
			LastModified: timestamppb.Now(),
		},
	}

	//seed
	store := peerstore.New()
	service := service.New(store)

	for host, file := range peers {
		in := pb.RegisterPeerRequest{
			Host:  host,
			Files: []*pb.File{file},
		}

		_, err := service.RegisterPeer(context.Background(), &in)
		if err != nil {
			t.Fatalf("expected to seed: %s", err)
		}
	}

	fetchedPeers, err := service.GetPeers(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("expected to get all peers: %s", err)
	}

	if len(fetchedPeers.Peers) != len(peers) {
		t.Errorf("len=%d, got %d", len(peers), len(fetchedPeers.Peers))
	}

	for peer := range peers {
		found := slices.ContainsFunc(fetchedPeers.Peers, func(p *pb.Peer) bool {
			return p.Host == peer
		})
		if !found {
			t.Fatalf("expected %s to be in fetched peers", peer)
		}

	}
}

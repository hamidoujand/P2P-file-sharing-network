package service_test

import (
	"context"
	"reflect"
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

	if resp.StatusCode != int64(codes.OK) {
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

	if resp.StatusCode != int64(codes.OK) {
		t.Errorf("code=%d, got %d", codes.OK, resp.StatusCode)
	}

	input := pb.UnRegisterPeerRequest{
		Host: host,
	}

	response, err := service.UnRegisterPeer(context.Background(), &input)
	if err != nil {
		t.Fatalf("expected to unregister peer from network: %s", err)
	}

	if response.StatusCode != int64(codes.OK) {
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

	for _, peer := range fetchedPeers.Peers {
		expectedFile := peers[peer.Host]
		found := slices.ContainsFunc(peer.Files, func(f *pb.File) bool {
			return expectedFile.Name == f.Name && expectedFile.FileType == f.FileType
		})

		if !found {
			t.Fatalf("expected file %s to be inside of peers list", expectedFile.Name)
		}
	}
}

func TestGetPeersForFile(t *testing.T) {
	meta := pb.File{
		Name:         "file.txt",
		Size:         10,
		FileType:     "txt",
		Checksum:     "some-checksum",
		LastModified: timestamppb.Now(),
	}

	store := peerstore.New()
	service := service.New(store)
	p1 := pb.RegisterPeerRequest{
		Host:  "127.0.0.1:9000",
		Files: []*pb.File{&meta},
	}
	p2 := pb.RegisterPeerRequest{
		Host:  "127.0.0.1:8000",
		Files: []*pb.File{&meta},
	}
	pp := []*pb.RegisterPeerRequest{&p1, &p2}
	for _, p := range pp {
		_, err := service.RegisterPeer(context.Background(), p)
		if err != nil {
			t.Fatalf("expected to register peer %s", p.Host)
		}
	}

	req := pb.GetPeersForFileRequest{
		File: &meta,
	}
	fetchedPeers, err := service.GetPeersForFile(context.Background(), &req)
	if err != nil {
		t.Fatalf("expected to get list of peers for file %s", meta.Name)
	}

	length := len(fetchedPeers.Peers)
	expected := 2
	if length != expected {
		t.Fatalf("length= %d, got %d", expected, length)
	}

	peer1File := fetchedPeers.Peers[0].Files[0]
	peer2File := fetchedPeers.Peers[1].Files[0]

	if !reflect.DeepEqual(peer1File, peer2File) {
		t.Fatal("expected peers to return the same file.")
	}
}

func TestUpdatePeer(t *testing.T) {
	meta := pb.File{
		Name:         "file.txt",
		Size:         10,
		FileType:     "txt",
		Checksum:     "some-checksum",
		LastModified: timestamppb.Now(),
	}

	store := peerstore.New()
	service := service.New(store)

	host := "127.0.0.1:9000"

	p := pb.RegisterPeerRequest{
		Host:  host,
		Files: []*pb.File{&meta},
	}

	_, err := service.RegisterPeer(context.Background(), &p)
	if err != nil {
		t.Fatalf("expected to register peer %s", p.Host)
	}

	//update it
	meta2 := pb.File{
		Name:         "file2.txt",
		Size:         20,
		FileType:     "txt",
		Checksum:     "some-checksum",
		LastModified: timestamppb.Now(),
	}
	in := &pb.UpdatePeerRequest{
		Host:  host,
		Files: []*pb.File{&meta, &meta2},
	}

	resp, err := service.UpdatePeer(context.Background(), in)
	if err != nil {
		t.Fatalf("expected to update perr: %s", err)
	}

	if resp.StatusCode != int64(codes.OK) {
		t.Fatalf("code=%d, got %d", codes.OK, resp.StatusCode)
	}

	//fetch it
	files, err := store.GetPeerByHost(host)
	if err != nil {
		t.Fatalf("expected to fetch peer's files: %s", err)
	}

	length := len(files)
	expected := 2
	if length != expected {
		t.Fatalf("length=%d, got %d", expected, length)
	}

	expectedFileNames := []string{"file.txt", "file2.txt"}
	for filename := range files {
		found := slices.Contains(expectedFileNames, filename)
		if !found {
			t.Errorf("expected the file %s to be in updated peer", filename)
		}
	}
}

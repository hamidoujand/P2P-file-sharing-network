package service_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/pb/tracker"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRegisterPeer(t *testing.T) {
	store := peerstore.New()
	service := service.New(store)

	in := tracker.RegisterPeerRequest{
		Host: "127.0.0.1:50051",
		Files: []*tracker.File{
			{
				Name:     "file1.txt",
				Size:     10,
				Checksum: "some-hash",
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
	in := tracker.RegisterPeerRequest{
		Host: host,
		Files: []*tracker.File{
			{
				Name:     "file1.txt",
				Size:     10,
				Checksum: "some-hash",
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

	input := tracker.UnRegisterPeerRequest{
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
	input = tracker.UnRegisterPeerRequest{
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
	peers := map[string]*tracker.File{
		"127.0.0.1:50051": {
			Name:     "file1.txt",
			Size:     10,
			Checksum: "some-radom-hash",
		},
		"198.168.2.1:50031": {
			Name:     "file2.txt",
			Size:     20,
			Checksum: "some-radom-hash",
		},
	}

	//seed
	store := peerstore.New()
	service := service.New(store)

	for host, file := range peers {
		in := tracker.RegisterPeerRequest{
			Host:  host,
			Files: []*tracker.File{file},
		}

		_, err := service.RegisterPeer(context.Background(), &in)
		if err != nil {
			t.Fatalf("expected to seed: %s", err)
		}
	}

	fetchedPeers, err := service.GetPeers(context.Background(), &tracker.GetPeersRequest{})
	if err != nil {
		t.Fatalf("expected to get all peers: %s", err)
	}

	if len(fetchedPeers.Peers) != len(peers) {
		t.Errorf("len=%d, got %d", len(peers), len(fetchedPeers.Peers))
	}

	for peer := range peers {
		found := slices.ContainsFunc(fetchedPeers.Peers, func(p *tracker.Peer) bool {
			return p.Host == peer
		})
		if !found {
			t.Fatalf("expected %s to be in fetched peers", peer)
		}
	}

	for _, peer := range fetchedPeers.Peers {
		expectedFile := peers[peer.Host]
		found := slices.ContainsFunc(peer.Files, func(f *tracker.File) bool {
			return expectedFile.Name == f.Name
		})

		if !found {
			t.Fatalf("expected file %s to be inside of peers list", expectedFile.Name)
		}
	}
}

func TestGetPeersForFile(t *testing.T) {
	meta := tracker.File{
		Name:     "file.txt",
		Size:     10,
		Checksum: "some-checksum",
	}

	store := peerstore.New()
	service := service.New(store)
	p1 := tracker.RegisterPeerRequest{
		Host:  "127.0.0.1:9000",
		Files: []*tracker.File{&meta},
	}
	p2 := tracker.RegisterPeerRequest{
		Host:  "127.0.0.1:8000",
		Files: []*tracker.File{&meta},
	}
	pp := []*tracker.RegisterPeerRequest{&p1, &p2}
	for _, p := range pp {
		_, err := service.RegisterPeer(context.Background(), p)
		if err != nil {
			t.Fatalf("expected to register peer %s", p.Host)
		}
	}

	req := tracker.GetPeersForFileRequest{
		FileName: meta.Name,
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
	meta := tracker.File{
		Name:     "file.txt",
		Size:     10,
		Checksum: "some-checksum",
	}

	store := peerstore.New()
	service := service.New(store)

	host := "127.0.0.1:9000"

	p := tracker.RegisterPeerRequest{
		Host:  host,
		Files: []*tracker.File{&meta},
	}

	_, err := service.RegisterPeer(context.Background(), &p)
	if err != nil {
		t.Fatalf("expected to register peer %s", p.Host)
	}

	//update it
	meta2 := tracker.File{
		Name:     "file2.txt",
		Size:     20,
		Checksum: "some-checksum",
	}
	in := &tracker.UpdatePeerRequest{
		Host:  host,
		Files: []*tracker.File{&meta, &meta2},
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

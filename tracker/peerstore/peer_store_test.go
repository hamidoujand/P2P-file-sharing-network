package peerstore_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
)

func TestPeerStore(t *testing.T) {
	files := []peerstore.FileMetadata{
		{
			Name:     "file1.txt",
			Size:     10,
			Checksum: "some-radom-hash",
		},
		{
			Name:     "file2.txt",
			Size:     20,
			Checksum: "some-radom-hash",
		},
	}

	host := "127.0.0.1:50051"
	store := peerstore.New()
	store.RegisterPeer(host, files)

	//check
	peer, err := store.GetPeerByHost(host)
	if err != nil {
		t.Fatalf("expected to get the peer data from stor: %s", err)
	}

	for _, file := range files {
		if f, ok := peer[file.Name]; !ok {
			t.Fatalf("expected file %s to be in files", file.Name)
		} else {
			if f.Size != file.Size {
				t.Errorf("size=%d, got %d", file.Size, f.Size)
			}
		}
	}
}

func TestRemovePeer(t *testing.T) {
	files := []peerstore.FileMetadata{
		{
			Name:     "file1.txt",
			Size:     10,
			Checksum: "some-radom-hash",
		},
		{
			Name:     "file2.txt",
			Size:     20,
			Checksum: "some-radom-hash",
		},
	}

	host := "127.0.0.1:50051"
	store := peerstore.New()
	store.RegisterPeer(host, files)

	store.RegisterPeer(host, files)

	if err := store.RemovePeerByHost(host); err != nil {
		t.Fatalf("expected to delete the peer: %s", err)
	}

	err := store.RemovePeerByHost(host)
	if err == nil {
		t.Fatal("expected to get an error while removing already deleted peer")
	}

	if !errors.Is(err, peerstore.ErrPeerNotFound) {
		t.Fatalf("error=%v, got %v", peerstore.ErrPeerNotFound, err)
	}
}

func TestGetAllPeers(t *testing.T) {
	store := peerstore.New()
	peers := map[string]peerstore.FileMetadata{
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

	for k, v := range peers {
		store.RegisterPeer(k, []peerstore.FileMetadata{v})
	}

	//get all peers
	fetchedPeers := store.GetAllPeers()
	if len(fetchedPeers) != len(peers) {
		t.Fatalf("len=%d, got %d", len(peers), len(fetchedPeers))
	}

	for host, wanted := range peers {
		found := slices.ContainsFunc(fetchedPeers, func(p peerstore.Peer) bool {
			return p.Host == host && p.Files[0].Name == wanted.Name
		})
		if !found {
			t.Fatalf("expected %s to be one of the fetched peers", host)
		}

	}
}

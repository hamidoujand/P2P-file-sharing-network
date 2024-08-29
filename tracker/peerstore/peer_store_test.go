package peerstore_test

import (
	"testing"
	"time"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
)

func TestPeerStore(t *testing.T) {
	files := []peerstore.FileMetadata{
		{
			Name:         "file1.txt",
			Size:         10,
			Mime:         "txt",
			Checksum:     "some-radom-hash",
			LastModified: time.Now(),
		},
		{
			Name:         "file2.txt",
			Size:         20,
			Mime:         "txt",
			Checksum:     "some-radom-hash",
			LastModified: time.Now(),
		},
	}

	ip := "127.0.0.1"
	var port uint32 = 50051

	store := peerstore.New()
	store.RegisterPeer(ip, port, files)

	//check
	peer, err := store.GetPeerByHost(ip, port)
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

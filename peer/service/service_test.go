package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/service"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPing(t *testing.T) {
	store := store.New()
	service := service.New(store)
	in := &pb.PingRequest{
		Message: "ping",
	}

	out, err := service.Ping(context.Background(), in)
	if err != nil {
		t.Fatalf("expected to ping the peer: %s", err)
	}

	if out.Status != codes.OK.String() {
		t.Errorf("status=%s, got %s", codes.OK.String(), out.Status)
	}
}

func TestCheckFileExistence(t *testing.T) {
	s := store.New()
	file := store.FileMetadata{
		Name:         "file.txt",
		Size:         12,
		FileType:     "txt",
		Checksum:     "some-checksum",
		LastModified: time.Now(),
	}
	s.AddFileMetadata(file)

	service := service.New(s)

	in := &pb.CheckFileExistenceRequest{
		Name: "file.txt",
	}

	resp, err := service.CheckFileExistence(context.Background(), in)
	if err != nil {
		t.Fatalf("expected to get back respose: %s", err)
	}

	if !resp.Exists {
		t.Fatal("expected the file to exists")
	}

	if resp.Metadata.Name != file.Name {
		t.Errorf("name=%s, got %s", file.Name, resp.Metadata.GetName())
	}

	//not-found
	in = &pb.CheckFileExistenceRequest{
		Name: "not-found.txt",
	}
	_, err = service.CheckFileExistence(context.Background(), in)
	if err == nil {
		t.Fatal("expected to get an error while asking for not found file")
	}

	status, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected the error to be of type status error: %T", err)
	}

	if status.Code() != codes.NotFound {
		t.Errorf("code=%d, got %d", codes.NotFound, status.Code())
	}
}

func TestGetFileMetadata(t *testing.T) {
	s := store.New()

	file := store.FileMetadata{
		Name:         "file.txt",
		Size:         12,
		FileType:     "txt",
		Checksum:     "some-checksum",
		LastModified: time.Now(),
	}

	s.AddFileMetadata(file)

	service := service.New(s)

	in := pb.GetFileMetadataRequest{
		Name:     file.Name,
		FileType: file.FileType,
	}

	resp, err := service.GetFileMetadata(context.Background(), &in)
	if err != nil {
		t.Fatalf("expected to get the metadata: %s", err)
	}

	if resp.Metadata.Name != file.Name {
		t.Errorf("name= %s, got %s", file.Name, resp.Metadata.Name)
	}

	if resp.Metadata.FileType != file.FileType {
		t.Errorf("fileType= %s, got %s", file.FileType, resp.Metadata.FileType)
	}

	if resp.Metadata.NumChunks != 1 {
		t.Errorf("numberOfChunks=%d, got %d", 1, resp.Metadata.NumChunks)
	}
}

package service_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/peer"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/service"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestPing(t *testing.T) {
	store := store.New()
	const defaultChunk int64 = 512 * 1024
	service := service.New(store, defaultChunk)
	in := &peer.PingRequest{
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
		Name:     "file.txt",
		Size:     12,
		FileType: "txt",
		Checksum: "some-checksum",
	}
	s.AddFileMetadata(file)
	const defaultChunk int64 = 512 * 1024

	service := service.New(s, defaultChunk)

	in := &peer.CheckFileExistenceRequest{
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
	in = &peer.CheckFileExistenceRequest{
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
		Name:     "file.txt",
		Size:     12,
		FileType: "txt",
		Checksum: "some-checksum",
	}

	s.AddFileMetadata(file)
	const defaultChunk int64 = 512 * 1024

	service := service.New(s, defaultChunk)

	in := peer.GetFileMetadataRequest{
		Name: file.Name,
	}

	resp, err := service.GetFileMetadata(context.Background(), &in)
	if err != nil {
		t.Fatalf("expected to get the metadata: %s", err)
	}

	if resp.Metadata.Name != file.Name {
		t.Errorf("name= %s, got %s", file.Name, resp.Metadata.Name)
	}
}

func createTestServer(t *testing.T, defaultChunkSize int64, filename string) (*grpc.ClientConn, peer.PeerServiceServer) {
	lis := bufconn.Listen(1024 * 1024) //buffered conn
	server := grpc.NewServer()
	s := store.New()

	s.AddFileMetadata(store.FileMetadata{
		Name: filename,
	})

	service := service.New(s, defaultChunkSize)

	peer.RegisterPeerServiceServer(server, service)

	go func() {
		err := server.Serve(lis)
		if err != nil {
			panic("server couldn't start")
		}
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		t.Fatalf("failed to create a client connection: %s", err)
	}
	return conn, service
}

func createTempFile(t *testing.T, filename string, content []byte) {
	asset := "peer/static"
	filePath := filepath.Join(asset, filename)

	//create the path
	if err := os.MkdirAll(asset, 0755); err != nil {
		t.Fatalf("failed to create dir tree: %s", err)
	}

	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
}

func removeTempFile(t *testing.T) {
	asset := "peer"
	if err := os.RemoveAll(asset); err != nil {
		t.Fatalf("failed to remove all files: %s", err)
	}
}

func TestDownloadFile(t *testing.T) {
	const defaultChunkSize int64 = 10 * 1024
	filename := "file.txt"
	conn, _ := createTestServer(t, defaultChunkSize, filename)
	defer conn.Close()

	peerClient := peer.NewPeerServiceClient(conn)

	// fileSize := 100 * 1024
	fileSize := 30 * 1024
	buff := make([]byte, fileSize)
	n, err := rand.Read(buff)
	if err != nil {
		t.Fatalf("failed to randomly generate bytes: %s", err)
	}

	//hex
	var hexBytes bytes.Buffer
	_, err = hex.NewEncoder(&hexBytes).Write(buff[:n])
	if err != nil {
		t.Fatalf("failed to encode random bytes into hex: %s", err)
	}
	createTempFile(t, filename, hexBytes.Bytes())
	defer removeTempFile(t)

	input := &peer.DownloadFileRequest{
		FileName: filename,
	}
	stream, err := peerClient.DownloadFile(context.Background(), input)
	if err != nil {
		t.Fatalf("failed to download file: %s", err)
	}

	receivedData := make([]byte, 0)
	for {
		chunk, err := stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("failed to receive chunk: %s", err)
		}
		fmt.Printf("chunk: %d/%d\n", chunk.ChunkNumber, chunk.TotalChunks)
		receivedData = append(receivedData, chunk.Data...)
	}

	if string(receivedData) != hexBytes.String() {
		t.Fatal("expected the downloaded file to have same content as original file")
	}

}

func TestConcurrentDownload(t *testing.T) {
	const defaultChunkSize int64 = 10 * 1024
	filename := "file.txt"
	conn, _ := createTestServer(t, defaultChunkSize, filename)
	defer conn.Close()

	peerClient := peer.NewPeerServiceClient(conn)

	// fileSize := 100 * 1024
	fileSize := 30 * 1024
	buff := make([]byte, fileSize)
	n, err := rand.Read(buff)
	if err != nil {
		t.Fatalf("failed to randomly generate bytes: %s", err)
	}

	//hex
	var hexBytes bytes.Buffer
	_, err = hex.NewEncoder(&hexBytes).Write(buff[:n])
	if err != nil {
		t.Fatalf("failed to encode random bytes into hex: %s", err)
	}
	createTempFile(t, filename, hexBytes.Bytes())
	defer removeTempFile(t)

	numClients := 4
	var wg sync.WaitGroup
	wg.Add(numClients)

	errChan := make(chan error, numClients)

	for i := range numClients {
		go func() {
			//each goroutine will download the same file.
			defer wg.Done()
			input := &peer.DownloadFileRequest{
				FileName: filename,
			}

			stream, err := peerClient.DownloadFile(context.Background(), input)
			if err != nil {
				errChan <- fmt.Errorf("failed to download file: %s", err)
			}

			receivedData := make([]byte, 0)
			for {
				chunk, err := stream.Recv()

				if err == io.EOF {
					break
				}

				if err != nil {
					errChan <- fmt.Errorf("failed to receive chunk: %s", err)
				}
				t.Logf("goroutine[%d]===> chunk: %d/%d\n", i, chunk.ChunkNumber, chunk.TotalChunks)
				receivedData = append(receivedData, chunk.Data...)
			}

			if string(receivedData) != hexBytes.String() {
				errChan <- fmt.Errorf("expected the downloaded file to have same content as original file")
			}
			t.Logf("goroutine[%d] Done!\n", i)
			//till here we
			errChan <- nil
		}()
	}

	//watcher
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(errChan)
		close(done)
	}()

	<-done
	//check the error from all
	for range errChan {
		if err := <-errChan; err != nil {
			t.Fatal(err)
		}
	}
}

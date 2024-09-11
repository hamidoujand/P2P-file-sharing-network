package service_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/peer"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/tracker"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/service"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewService(t *testing.T) {
	//tracker client setup
	trackerClient := setupTrackerClient(t)
	const defaultDownloadChunk int64 = 512 * 1024

	//fs to load files from
	file := fstest.MapFile{
		Data: []byte("this is a test file."),
	}

	fs := fstest.MapFS{
		"file.txt": &file,
	}
	s := store.New()

	conf := service.Config{
		Host:             "0.0.0.0:50051",
		Store:            s,
		TrackerClient:    trackerClient,
		DefaultChunkSize: defaultDownloadChunk,
		Fs:               fs,
	}
	_, err := service.New(context.Background(), &conf)
	if err != nil {
		t.Fatalf("failed to create a new peer service: %s", err)
	}

	//check the store
	fm, err := s.GetFileMetadata("file.txt")
	if err != nil {
		t.Fatalf("expected the file.txt's metadata to be stored: %s", err)
	}

	if len(file.Data) != int(fm.Size) {
		t.Errorf("size=%d, got %d", len(file.Data), fm.Size)
	}
}

// when peer itself has the file.
func TestDownloadFile(t *testing.T) {
	const fileSize int64 = 30 * 1024
	const filename = "file.txt"

	buff := make([]byte, fileSize)
	n, err := rand.Read(buff)
	if err != nil {
		t.Fatalf("failed to randomly generate bytes: %s", err)
	}

	// hex
	var hexBytes bytes.Buffer
	_, err = hex.NewEncoder(&hexBytes).Write(buff[:n])
	if err != nil {
		t.Fatalf("failed to encode random bytes into hex: %s", err)
	}
	//file
	file := fstest.MapFile{
		Data: hexBytes.Bytes(),
	}

	fs := fstest.MapFS{
		filename: &file,
	}

	peerClient := setupPeerClient(t, fs)

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
		fmt.Printf("successfully received: chunk[%d/%d]\n", chunk.ChunkNumber, chunk.TotalChunks)
		receivedData = append(receivedData, chunk.Data...)
	}

	if string(receivedData) != hexBytes.String() {
		t.Fatal("expected the downloaded file to have same content as original file")
	}
}

func TestUploadFile(t *testing.T) {
	staticDir := "static"
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("failed to create static dir: %s", err)
	}

	file := fstest.MapFile{
		Data: []byte("this is a test file."),
	}
	fs := fstest.MapFS{
		"file.txt": &file,
	}

	peerClient := setupPeerClient(t, fs)

	bs := make([]byte, 100*1024)
	n, err := rand.Read(bs)
	if err != nil {
		t.Fatalf("generate random data: %s", err)
	}

	var buff bytes.Buffer
	_, err = hex.NewEncoder(&buff).Write(bs[:n])
	if err != nil {
		t.Fatalf("failed to encode random bytes into hex: %s", err)
	}

	hash := sha256.New()
	_, err = hash.Write(buff.Bytes())
	if err != nil {
		t.Fatalf("failed to write hash: %s", err)
	}

	checksum := fmt.Sprintf("%X", hash.Sum(nil))

	stream, err := peerClient.UploadFile(context.Background())
	if err != nil {
		t.Fatalf("failed to create stream: %s", err)
	}

	filename := "file2.txt"
	const defaultChunkSize = 20 * 1024
	totalChunks := ((buff.Len() + defaultChunkSize - 1) / defaultChunkSize)

	chunk := make([]byte, defaultChunkSize)
	chunkNumber := 1

	for {
		n, err := buff.Read(chunk)
		if err == io.EOF {
			break
		}

		chunk := peer.UploadFileChunk{
			ChunkNumber: int32(chunkNumber),
			TotalChunks: int32(totalChunks),
			FileName:    filename,
			Data:        chunk[:n],
		}

		if err := stream.Send(&chunk); err != nil {
			t.Fatalf("failed to send chunk[%d]: %s", chunkNumber, err)
		}
		fmt.Printf("successfully sent: chunk[%d/%d]\n", chunkNumber, totalChunks)
		chunkNumber++

	}

	//close stream
	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("failed to close stream: %s", err)
	}

	if !resp.Success {
		t.Fatal("expected the response to be a success")
	}

	//check file existence
	checkReq := peer.CheckFileExistenceRequest{
		Name: filename,
	}
	checkResp, err := peerClient.CheckFileExistence(context.Background(), &checkReq)
	if err != nil {
		t.Fatalf("expected the file we uploaded to be in store: %s", err)
	}

	if !checkResp.Exists {
		t.Fatalf("exisits=%t, got %t", true, checkResp.Exists)
	}

	//get file metadata
	in := peer.GetFileMetadataRequest{
		Name: filename,
	}
	meta, err := peerClient.GetFileMetadata(context.Background(), &in)
	if err != nil {
		t.Fatalf("expected to get the file metadata: %s", err)
	}
	if meta.Metadata.Name != filename {
		t.Fatalf("name=%s, got %s", filename, meta.Metadata.Name)
	}

	if meta.Metadata.Checksum != checksum {
		t.Fatalf("checksum=[%s], got [%s]", checksum, meta.Metadata.Checksum)
	}

	t.Cleanup(func() {
		//remove file
		if err := os.RemoveAll(staticDir); err != nil {
			t.Fatalf("expected to remove the file[%s]: %s", filename, err)
		}
	})

}

// =============================================================================
// utils
func setupTrackerClient(t *testing.T) tracker.TrackerServiceClient {
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create tracker conn: %s", err)
	}

	tracker.RegisterTrackerServiceServer(server, newTrackerServiceMock())

	go func() {
		if err := server.Serve(lis); err != nil {
			panic("failed to serve")
		}
	}()

	client := tracker.NewTrackerServiceClient(conn)
	return client
}

func setupPeerClient(t *testing.T, fsys fs.FS) peer.PeerServiceClient {
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create peer conn: %s", err)
	}
	service := setupPeerService(t, fsys)
	peer.RegisterPeerServiceServer(server, service)

	go func() {
		if err := server.Serve(lis); err != nil {
			panic("failed to serve")
		}
	}()

	//client
	client := peer.NewPeerServiceClient(conn)
	return client
}

func setupPeerService(t *testing.T, fsys fs.FS) peer.PeerServiceServer {
	//tracker client setup
	trackerClient := setupTrackerClient(t)
	const defaultDownloadChunk int64 = 5 * 1024 //for testing ,"5kb" chunk

	s := store.New()

	conf := service.Config{
		Host:             "0.0.0.0:50051",
		Store:            s,
		TrackerClient:    trackerClient,
		DefaultChunkSize: defaultDownloadChunk,
		Fs:               fsys,
	}
	service, err := service.New(context.Background(), &conf)
	if err != nil {
		t.Fatalf("failed to create a new peer service: %s", err)
	}
	return service
}

// ==============================================================================
// mocks
type trackerServiceMock struct {
	tracker.UnimplementedTrackerServiceServer
	peers map[string]*tracker.Peer
	mu    sync.RWMutex
}

func newTrackerServiceMock() *trackerServiceMock {
	return &trackerServiceMock{
		peers: make(map[string]*tracker.Peer),
	}
}

func (ts *trackerServiceMock) RegisterPeer(ctx context.Context, in *tracker.RegisterPeerRequest) (*tracker.RegisterPeerResponse, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.peers[in.Host] = &tracker.Peer{
		Host:  in.Host,
		Files: in.Files,
	}

	return &tracker.RegisterPeerResponse{StatusCode: int64(codes.OK), Message: codes.OK.String()}, nil
}

func (ts *trackerServiceMock) UnRegisterPeer(ctx context.Context, in *tracker.UnRegisterPeerRequest) (*tracker.UnRegisterPeerResponse, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.peers, in.Host)

	return &tracker.UnRegisterPeerResponse{StatusCode: int64(codes.OK), Message: codes.OK.String()}, nil
}
func (ts *trackerServiceMock) GetPeers(ctx context.Context, in *tracker.GetPeersRequest) (*tracker.GetPeersResponse, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	results := make([]*tracker.Peer, 0, len(ts.peers))
	for _, v := range ts.peers {
		results = append(results, v)
	}

	return &tracker.GetPeersResponse{Peers: results}, nil
}
func (ts *trackerServiceMock) GetPeersForFile(ctx context.Context, in *tracker.GetPeersForFileRequest) (*tracker.GetPeersResponse, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	peerList := []*tracker.Peer{}
	for _, peer := range ts.peers {
		for _, file := range peer.Files {
			if file.Name == in.FileName {
				peerList = append(peerList, peer)
				break
			}
		}
	}

	return &tracker.GetPeersResponse{Peers: peerList}, nil
}
func (ts *trackerServiceMock) UpdatePeer(ctx context.Context, in *tracker.UpdatePeerRequest) (*tracker.UpdatePeerResponse, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.peers[in.Host] = &tracker.Peer{
		Host:  in.Host,
		Files: in.Files,
	}
	return &tracker.UpdatePeerResponse{StatusCode: int64(codes.OK), Message: codes.OK.String()}, nil
}

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
	"log"
	"net"
	"os"
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

// when peer needs to download file from another peer first.
func TestPeerToPeerDownload(t *testing.T) {
	trackerLis := bufconn.Listen(1024 * 1024)
	peer1Lis := bufconn.Listen(1024 * 1024)
	peer2Lis := bufconn.Listen(1024 * 1024)

	trackerDialer := func(context.Context, string) (net.Conn, error) {
		return trackerLis.Dial()
	}

	peer1Dialer := func(context.Context, string) (net.Conn, error) {
		return peer1Lis.Dial()
	}

	peer2Dialer := func(context.Context, string) (net.Conn, error) {
		return peer2Lis.Dial()
	}

	trackerConn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(trackerDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create tracker conn: %s", err)
	}
	defer trackerConn.Connect()

	peer1Conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(peer1Dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create peer1 conn: %s", err)
	}
	defer peer1Conn.Close()

	peer2Conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(peer2Dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create peer2 conn: %s", err)
	}
	defer peer2Conn.Close()

	filename := "file2.txt"
	trackerService := trackerServiceMock{
		peer: &tracker.Peer{
			Host: peer2Lis.Addr().String(),
			Files: []*tracker.File{
				{
					Name:     filename,
					Size:     10,
					Checksum: "some-checksom",
				},
			},
		},
	}

	trackerServer := grpc.NewServer()
	peer1Server := grpc.NewServer()
	peer2Server := grpc.NewServer()

	tracker.RegisterTrackerServiceServer(trackerServer, trackerService)
	//tracker server
	go func() {
		if err := trackerServer.Serve(trackerLis); err != nil {
			panic(err)
		}
	}()

	trackerClient := tracker.NewTrackerServiceClient(trackerConn)

	const defaultChunkSize = 10 * 1024
	//peer1 does not have the file so empty fn
	fs1 := fstest.MapFS{}
	peer1Service, err := service.New(context.Background(), &service.Config{
		Host:             peer1Lis.Addr().String(),
		Store:            store.New(),
		TrackerClient:    trackerClient,
		DefaultChunkSize: defaultChunkSize,
		Fs:               fs1,
	})
	if err != nil {
		t.Fatalf("failed to create peer1 service: %s", err)
	}

	//register peer1
	peer.RegisterPeerServiceServer(peer1Server, peer1Service)

	go func() {
		t.Log("peer1 server is running...")
		if err := peer1Server.Serve(peer1Lis); err != nil {
			log.Fatalln("peer1 server is dead:", err)
			panic(err)
		}
	}()

	//peer1 client
	peer1Client := peer.NewPeerServiceClient(peer1Conn)

	//peer2 setups

	//fs needs to have the file
	file := fstest.MapFile{
		Data: []byte("this is some test data to download"),
	}

	fs2 := fstest.MapFS{
		filename: &file,
	}
	//store needs to have the file
	s := store.New()
	s.AddFileMetadata(store.FileMetadata{
		Name:     filename,
		Size:     int64(len(file.Data)),
		Checksum: "checksum",
	})

	fmt.Println("Host1:", peer1Lis.Addr())
	fmt.Println("Host2:", peer2Lis.Addr())

	peer2Service, err := service.New(context.Background(), &service.Config{
		Host:             peer2Lis.Addr().String(),
		Store:            s,
		TrackerClient:    trackerClient,
		DefaultChunkSize: defaultChunkSize,
		Fs:               fs2,
	})
	if err != nil {
		t.Fatalf("failed to create peer2 service: %s", err)
	}
	//register peer 2 server
	peer.RegisterPeerServiceServer(peer2Server, peer2Service)
	go func() {
		t.Log("peer2 server is running...")
		if err := peer2Server.Serve(peer2Lis); err != nil {
			log.Fatalln("peer2 server is dead:", err)
		}
	}()

	//path for the downloaded file from peer2 to be store on peer1
	path := "peer/static"
	if err := os.MkdirAll(path, 0755); err != nil {
		if !os.IsExist(err) {
			t.Fatalf("failed to create path: %s", err)
		}
	}

	in := peer.DownloadFileRequest{
		FileName: filename,
	}
	stream, err := peer1Client.DownloadFile(context.Background(), &in)
	if err != nil {
		t.Fatalf("failed to create download stream: %s", err)
	}

	var buff bytes.Buffer
	for {
		chunk, err := stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("failed to receive chunk: %s", err)
		}
		buff.Write(chunk.Data)
		fmt.Printf("successfully received: chunk[%d/%d]\n", chunk.ChunkNumber, chunk.TotalChunks)
	}
	fmt.Println("WOW")

	t.Cleanup(func() {
		//remove file
		path := "peer"
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("expected to remove the file[%s]: %s", filename, err)
		}
	})
}

func TestUploadFile(t *testing.T) {
	staticDir := "peer/static"
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
		path := "peer"
		if err := os.RemoveAll(path); err != nil {
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

	tracker.RegisterTrackerServiceServer(server, trackerServiceMock{})

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
	peer *tracker.Peer
}

func (ts trackerServiceMock) RegisterPeer(ctx context.Context, in *tracker.RegisterPeerRequest) (*tracker.RegisterPeerResponse, error) {
	return &tracker.RegisterPeerResponse{StatusCode: int64(codes.OK), Message: codes.OK.String()}, nil
}

func (ts trackerServiceMock) UnRegisterPeer(ctx context.Context, in *tracker.UnRegisterPeerRequest) (*tracker.UnRegisterPeerResponse, error) {
	return &tracker.UnRegisterPeerResponse{StatusCode: int64(codes.OK), Message: codes.OK.String()}, nil
}
func (ts trackerServiceMock) GetPeers(ctx context.Context, in *tracker.GetPeersRequest) (*tracker.GetPeersResponse, error) {
	return &tracker.GetPeersResponse{Peers: []*tracker.Peer{ts.peer}}, nil
}
func (ts trackerServiceMock) GetPeersForFile(ctx context.Context, in *tracker.GetPeersForFileRequest) (*tracker.GetPeersResponse, error) {
	return &tracker.GetPeersResponse{Peers: []*tracker.Peer{ts.peer}}, nil
}
func (ts trackerServiceMock) UpdatePeer(ctx context.Context, in *tracker.UpdatePeerRequest) (*tracker.UpdatePeerResponse, error) {
	return &tracker.UpdatePeerResponse{StatusCode: int64(codes.OK), Message: codes.OK.String()}, nil
}

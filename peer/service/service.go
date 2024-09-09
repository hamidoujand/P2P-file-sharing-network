package service

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/peer"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/tracker"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service represents all of the rpc service calls.
type Service struct {
	peer.UnimplementedPeerServiceServer
	store            *store.Store
	defaultChunkSize int64
	trackerClient    tracker.TrackerServiceClient
	fs               fs.FS
}

type Config struct {
	Host             string
	Store            *store.Store
	TrackerClient    tracker.TrackerServiceClient
	DefaultChunkSize int64
	Fs               fs.FS
}

// New creates a new rpc service.
func New(ctx context.Context, conf *Config) (*Service, error) {
	//register peer into tracker
	if conf.Fs == nil {
		return nil, errors.New("FS can not be nil")
	}

	var fileMetadata []store.FileMetadata

	//walk the fs
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		//files
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("info: %w", err)
		}

		file, err := conf.Fs.Open(path)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		hasher := sha256.New()
		chunk := make([]byte, 4*1024)

		for {
			n, err := reader.Read(chunk)
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("read: %w", err)
			}

			hasher.Write(chunk[:n])
		}

		rawHash := hasher.Sum(nil)
		checksum := fmt.Sprintf("%X", rawHash)

		fm := store.FileMetadata{
			Name:     info.Name(),
			Size:     info.Size(),
			Checksum: checksum,
		}
		//add it into store
		conf.Store.AddFileMetadata(fm)

		fileMetadata = append(fileMetadata, fm)
		return nil
	}

	if err := fs.WalkDir(conf.Fs, ".", walk); err != nil {
		return nil, fmt.Errorf("walk fs: %w", err)
	}

	files := make([]*tracker.File, len(fileMetadata))
	for i, fm := range fileMetadata {
		files[i] = &tracker.File{
			Name:     fm.Name,
			Size:     fm.Size,
			Checksum: fm.Checksum,
		}
	}

	//register peer with tracker
	in := &tracker.RegisterPeerRequest{
		Host:  conf.Host,
		Files: files,
	}
	resp, err := conf.TrackerClient.RegisterPeer(ctx, in)

	if err != nil {
		return nil, fmt.Errorf("rpc:tracker: %w", err)
	}

	if resp.StatusCode != int64(codes.OK) {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	return &Service{
		store:            conf.Store,
		defaultChunkSize: conf.DefaultChunkSize,
		trackerClient:    conf.TrackerClient,
		fs:               conf.Fs,
	}, nil
}

// Ping is used to check the health of the server.
func (s *Service) Ping(ctx context.Context, in *peer.PingRequest) (*peer.PingResponse, error) {
	message := in.GetMessage()
	now := timestamppb.Now()
	out := &peer.PingResponse{
		Status:    codes.OK.String(),
		Message:   message,
		Timestamp: now,
	}

	return out, nil
}

// CheckFileExistence check whether the peer has the file or not.
func (s *Service) CheckFileExistence(ctx context.Context, in *peer.CheckFileExistenceRequest) (*peer.CheckFileExistenceResponse, error) {
	filename := in.GetName()
	meta, err := s.store.GetFileMetadata(filename)
	if err != nil {
		if errors.Is(err, store.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "file %s, not found", filename)
		}
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	resp := peer.CheckFileExistenceResponse{
		Exists:   true,
		Metadata: meta.ToProtoBuff(),
	}
	return &resp, nil
}

// GetFileMetadata returns the metadata related to a file or possible error.
func (s *Service) GetFileMetadata(ctx context.Context, in *peer.GetFileMetadataRequest) (*peer.GetFileMetadataResponse, error) {
	filename := in.GetName()
	meta, err := s.store.GetFileMetadata(filename)
	if err != nil {
		if errors.Is(err, store.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "file %s, not found", filename)
		}
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	resp := peer.GetFileMetadataResponse{
		Metadata: meta.ToProtoBuff(),
	}

	return &resp, nil
}

// DownloadFile handles downloading files chunk by chunk to the client.
func (s *Service) DownloadFile(in *peer.DownloadFileRequest, stream grpc.ServerStreamingServer[peer.FileChunk]) error {
	filename := in.GetFileName()

	//check the store for metadata
	_, err := s.store.GetFileMetadata(filename)
	if err != nil {
		if errors.Is(err, store.ErrFileNotFound) {
			//try tracker service
			in := tracker.GetPeersForFileRequest{
				FileName: filename,
			}

			peers, err := s.trackerClient.GetPeersForFile(context.Background(), &in)
			if err != nil {
				return fmt.Errorf("get peers for file: %w", err)
			}

			//not found inside of the network
			if len(peers.Peers) == 0 {
				return status.Errorf(codes.NotFound, "file %s, not found", filename)
			}

			for _, p := range peers.Peers {
				//need a peer client
				peerConn, err := grpc.NewClient(p.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				defer peerConn.Close()
				peerClient := peer.NewPeerServiceClient(peerConn)

				//ping peer
				in := peer.PingRequest{
					Message: "Hi",
				}
				resp, err := peerClient.Ping(context.Background(), &in)
				if err != nil {
					fmt.Printf("ping[%s] failed: %s", p.Host, err.Error())
					continue //continue to next peer that has the file
				}
				if resp.Status != codes.OK.String() {
					fmt.Printf("status[%s] not ok: %s", p.Host, resp.Status)
					continue
				}
				//TODO do the rest
				//get file metadata

				//handle download

				//validate checksum
			}

		} else {
			return status.Error(codes.Internal, codes.Internal.String())
		}
	}

	file, err := s.fs.Open(in.GetFileName())
	if err != nil {
		if os.IsNotExist(err) {
			return status.Errorf(codes.NotFound, "file %s, not found", filename)
		}
		return status.Error(codes.Internal, codes.Internal.String())
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return status.Error(codes.Internal, codes.Internal.String())
	}

	totalChunks := int32((stats.Size() + s.defaultChunkSize - 1) / s.defaultChunkSize)
	buffer := make([]byte, s.defaultChunkSize)

	for chunkNumber := int32(1); chunkNumber <= totalChunks; chunkNumber++ {
		readBytes, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break //end of the file.
			}
			//something went wrong
			return status.Errorf(codes.Internal, "failed to read file: %s", err)
		}

		chunk := &peer.FileChunk{
			ChunkNumber: chunkNumber,
			Data:        buffer[:readBytes],
			TotalChunks: totalChunks,
		}

		if err := stream.Send(chunk); err != nil {
			return status.Errorf(codes.Internal, "failed to send chunk[%d]: %s", chunkNumber, err)
		}
	}
	return nil
}

// UploadFile uploads the file chunk by chunk from client.
func (s *Service) UploadFile(stream grpc.ClientStreamingServer[peer.UploadFileChunk, peer.UploadFileResponse]) error {

	var filename string
	var file *os.File
	var bufWriter *bufio.Writer
	hash := sha256.New()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			//check the file
			if file != nil {
				if err := bufWriter.Flush(); err != nil {
					return fmt.Errorf("flush: %w", err)
				}

				//add metadata store
				stats, err := file.Stat()
				if err != nil {
					return fmt.Errorf("stat: %w", err)
				}

				fm := store.FileMetadata{
					Name:     filename,
					Size:     stats.Size(),
					Checksum: fmt.Sprintf("%X", hash.Sum(nil)),
				}

				//save metadata for this file
				s.store.AddFileMetadata(fm)
				//close it
				file.Close()
			}

			return stream.SendAndClose(&peer.UploadFileResponse{
				Success: true,
				Message: codes.OK.String(),
			})
		}

		if err != nil {
			return fmt.Errorf("revc: %w", err)
		}

		//create file one time
		if filename == "" {
			var err error
			filename = chunk.GetFileName()
			filepath := filepath.Join("peer", "static", filename)
			file, err = os.Create(filepath)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			bufWriter = bufio.NewWriter(file)
		}
		target := make([]byte, len(chunk.Data))
		copy(target, chunk.Data)

		_, err = hash.Write(target)
		if err != nil {
			return fmt.Errorf("writing hash: %w", err)
		}
		//writing chunks, into buffer
		_, err = bufWriter.Write(chunk.GetData())
		if err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}

}

//==============================================================================
// helpers

func download() {

}

package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/peer"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service represents all of the rpc service calls.
type Service struct {
	peer.UnimplementedPeerServiceServer
	store            *store.Store
	defaultChunkSize int64
}

// New creates a new rpc service.
func New(store *store.Store, defaultChunkSize int64) *Service {
	return &Service{
		store:            store,
		defaultChunkSize: defaultChunkSize,
	}
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
			return status.Errorf(codes.NotFound, "file %s, not found", filename)
		}
		return status.Error(codes.Internal, codes.Internal.String())
	}

	path := filepath.Join("peer/static", filename)
	stats, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return status.Errorf(codes.NotFound, "file %s, not found", filename)
		}
		return status.Error(codes.Internal, codes.Internal.String())
	}
	//open the file
	file, err := os.Open(path)
	if err != nil {
		return status.Errorf(codes.Internal, "unable to open file %s: %s", filename, err.Error())
	}
	defer file.Close()

	totalChunks := int32((stats.Size() + s.defaultChunkSize - 1) / s.defaultChunkSize)
	buffer := make([]byte, s.defaultChunkSize)

	for chunkNumber := int32(0); chunkNumber < totalChunks; chunkNumber++ {
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
	hash := sha256.New()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			//check the file
			if file != nil {
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
			filepath := path.Join("..", "static", filename)
			file, err = os.Create(filepath)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
		}
		target := make([]byte, len(chunk.Data))
		copy(target, chunk.Data)

		_, err = hash.Write(target)
		if err != nil {
			return fmt.Errorf("writing hash: %w", err)
		}
		//writing chunks
		_, err = file.Write(chunk.GetData())
		if err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}
}

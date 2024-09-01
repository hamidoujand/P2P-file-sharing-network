package service

import (
	"context"
	"errors"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service represents all of the rpc service calls.
type Service struct {
	pb.UnimplementedPeerServiceServer
	store *store.Store
}

// New creates a new rpc service.
func New(store *store.Store) *Service {
	return &Service{
		store: store,
	}
}

// Ping is used to check the health of the server.
func (s *Service) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	message := in.GetMessage()
	now := timestamppb.Now()
	out := &pb.PingResponse{
		Status:    codes.OK.String(),
		Message:   message,
		Timestamp: now,
	}

	return out, nil
}

// CheckFileExistence check whether the peer has the file or not.
func (s *Service) CheckFileExistence(ctx context.Context, in *pb.CheckFileExistenceRequest) (*pb.CheckFileExistenceResponse, error) {
	filename := in.GetName()
	meta, err := s.store.GetFileMetadata(filename)
	if err != nil {
		if errors.Is(err, store.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "file %s, not found", filename)
		}
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	resp := pb.CheckFileExistenceResponse{
		Exists:   true,
		Metadata: meta.ToProtoBuff(),
	}
	return &resp, nil
}

// GetFileMetadata returns the metadata related to a file or possible error.
func (s *Service) GetFileMetadata(ctx context.Context, in *pb.GetFileMetadataRequest) (*pb.GetFileMetadataResponse, error) {
	filename := in.GetName()
	fileType := in.GetFileType()

	meta, err := s.store.GetFileMetadata(filename)
	if err != nil {
		if errors.Is(err, store.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "file %s, not found", filename)
		}
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	//check the filetype
	if fileType != meta.FileType {
		return nil, status.Errorf(codes.NotFound, "file %s, not found", filename)
	}

	resp := pb.GetFileMetadataResponse{
		Metadata: meta.ToProtoBuff(),
	}

	return &resp, nil
}

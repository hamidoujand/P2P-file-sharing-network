package service

import (
	"context"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/pb"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
	"google.golang.org/grpc/codes"
)

// Service represents set of rpc calls related to tracker.
type Service struct {
	pb.UnimplementedTrackerServiceServer
	store *peerstore.Store
}

func New(store *peerstore.Store) *Service {
	return &Service{
		store: store,
	}
}

// RegisterPeer will add a new peer into store.
func (s *Service) RegisterPeer(ctx context.Context, in *pb.RegisterPeerRequest) (*pb.RegisterPeerResponse, error) {
	files := make([]peerstore.FileMetadata, len(in.Files))

	for i, file := range in.Files {
		fileMeta := peerstore.FileMetadata{
			Name:         file.GetName(),
			Size:         file.GetSize(),
			Mime:         file.GetFileType(),
			Checksum:     file.GetChecksum(),
			LastModified: file.GetLastModified().AsTime(),
		}
		files[i] = fileMeta
	}

	s.store.RegisterPeer(in.Ip, in.Port, files)

	return &pb.RegisterPeerResponse{StatusCode: uint32(codes.OK), Message: codes.OK.String()}, nil
}

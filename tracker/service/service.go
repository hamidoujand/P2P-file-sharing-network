package service

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/pb"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	s.store.RegisterPeer(in.Host, files)

	return &pb.RegisterPeerResponse{StatusCode: uint32(codes.OK), Message: codes.OK.String()}, nil
}

// UnRegisterPeer willl remove a peer from network.
func (s *Service) UnRegisterPeer(ctx context.Context, in *pb.UnRegisterPeerRequest) (*pb.UnRegisterPeerResponse, error) {
	err := s.store.RemovePeerByHost(in.Host)
	if err != nil {
		if errors.Is(err, peerstore.ErrPeerNotFound) {
			return &pb.UnRegisterPeerResponse{
				StatusCode: uint32(codes.NotFound),
				Message:    codes.NotFound.String(),
			}, status.Errorf(codes.NotFound, "peer %s not found", in.Host)
		} else {
			return &pb.UnRegisterPeerResponse{
				StatusCode: uint32(codes.Internal),
				Message:    codes.Internal.String(),
			}, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.UnRegisterPeerResponse{
		StatusCode: uint32(codes.OK),
		Message:    codes.OK.String(),
	}, nil
}

// GetPeers returns all peers with their file metadata.
func (s *Service) GetPeers(ctx context.Context, _ *empty.Empty) (*pb.GetPeersResponse, error) {
	peers := s.store.GetAllPeers()

	buffPeers := make([]*pb.Peer, len(peers))
	for i, peer := range peers {
		buffPeer := pb.Peer{}
		buffPeer.Host = peer.Host

		buffFiles := make([]*pb.File, len(peer.Files))
		for i, file := range peer.Files {
			buffFiles[i] = &pb.File{
				Name:         file.Name,
				Size:         file.Size,
				Checksum:     file.Checksum,
				LastModified: timestamppb.New(file.LastModified),
				FileType:     file.Mime,
			}
		}
		buffPeer.Files = buffFiles
		buffPeers[i] = &buffPeer
	}

	return &pb.GetPeersResponse{
		Peers: buffPeers,
	}, nil
}

package service

import (
	"context"
	"errors"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/pb/tracker"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service represents set of rpc calls related to tracker.
type Service struct {
	tracker.UnimplementedTrackerServiceServer
	store *peerstore.Store
}

func New(store *peerstore.Store) *Service {
	return &Service{
		store: store,
	}
}

// RegisterPeer will add a new peer into store.
func (s *Service) RegisterPeer(ctx context.Context, in *tracker.RegisterPeerRequest) (*tracker.RegisterPeerResponse, error) {
	files := make([]peerstore.FileMetadata, len(in.Files))

	for i, file := range in.Files {
		fileMeta := peerstore.FileMetadata{
			Name:     file.GetName(),
			Size:     file.GetSize(),
			Checksum: file.GetChecksum(),
		}
		files[i] = fileMeta
	}

	s.store.RegisterPeer(in.Host, files)

	return &tracker.RegisterPeerResponse{StatusCode: int64(codes.OK), Message: codes.OK.String()}, nil
}

// UnRegisterPeer willl remove a peer from network.
func (s *Service) UnRegisterPeer(ctx context.Context, in *tracker.UnRegisterPeerRequest) (*tracker.UnRegisterPeerResponse, error) {
	err := s.store.RemovePeerByHost(in.Host)
	if err != nil {
		if errors.Is(err, peerstore.ErrPeerNotFound) {
			return &tracker.UnRegisterPeerResponse{
				StatusCode: int64(codes.NotFound),
				Message:    codes.NotFound.String(),
			}, status.Errorf(codes.NotFound, "peer %s not found", in.Host)
		} else {
			return &tracker.UnRegisterPeerResponse{
				StatusCode: int64(codes.Internal),
				Message:    codes.Internal.String(),
			}, status.Error(codes.Internal, err.Error())
		}
	}

	return &tracker.UnRegisterPeerResponse{
		StatusCode: int64(codes.OK),
		Message:    codes.OK.String(),
	}, nil
}

// GetPeers returns all peers with their file metadata.
func (s *Service) GetPeers(ctx context.Context, _ *tracker.GetPeersRequest) (*tracker.GetPeersResponse, error) {
	peers := s.store.GetAllPeers()

	buffPeers := toBufferPeers(peers)
	return &tracker.GetPeersResponse{
		Peers: buffPeers,
	}, nil
}

// GetPeersForFile will returns all peers that contain the requested file.
func (s *Service) GetPeersForFile(ctx context.Context, in *tracker.GetPeersForFileRequest) (*tracker.GetPeersResponse, error) {

	peers := s.store.GetPeersForFile(in.GetFileName())
	buffPeers := toBufferPeers(peers)

	resp := tracker.GetPeersResponse{
		Peers: buffPeers,
	}
	return &resp, nil
}

// UpdatePeer will update the files related to a peer.
func (s *Service) UpdatePeer(ctx context.Context, in *tracker.UpdatePeerRequest) (*tracker.UpdatePeerResponse, error) {
	//get the peer
	_, err := s.store.GetPeerByHost(in.GetHost())
	if err != nil {
		if errors.Is(err, peerstore.ErrPeerNotFound) {
			return nil, status.Errorf(codes.NotFound, "peer %s, not found", in.GetHost())
		}
		//internal
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	files := make([]peerstore.FileMetadata, len(in.GetFiles()))
	for i, f := range in.GetFiles() {
		files[i] = peerstore.FileMetadata{
			Name:     f.GetName(),
			Size:     f.GetSize(),
			Checksum: f.GetChecksum(),
		}
	}
	s.store.UpdatePeer(in.GetHost(), files)
	return &tracker.UpdatePeerResponse{StatusCode: int64(codes.OK), Message: codes.OK.String()}, nil
}

func toBufferPeers(peers []peerstore.Peer) []*tracker.Peer {
	buffPeers := make([]*tracker.Peer, len(peers))
	for i, peer := range peers {
		buffPeer := tracker.Peer{}
		buffPeer.Host = peer.Host

		buffFiles := make([]*tracker.File, len(peer.Files))
		for i, file := range peer.Files {
			buffFiles[i] = &tracker.File{
				Name:     file.Name,
				Size:     file.Size,
				Checksum: file.Checksum,
			}
		}
		buffPeer.Files = buffFiles
		buffPeers[i] = &buffPeer
	}
	return buffPeers
}

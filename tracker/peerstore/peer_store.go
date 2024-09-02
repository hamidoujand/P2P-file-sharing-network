package peerstore

import (
	"errors"
	"sync"
	"time"
)

var ErrPeerNotFound = errors.New("peer not found")

// FileMetadata represents all required info related to a file on the network.
type FileMetadata struct {
	Name         string
	Size         int64
	Mime         string
	Checksum     string
	LastModified time.Time
}

// Files is a collection of file metadata.
type Files map[string]FileMetadata

type Peer struct {
	Host  string
	Files []FileMetadata
}

// Store represents the in memory storage used to store peers and their files.
type Store struct {
	mu    sync.RWMutex
	store map[string]Files
}

// New creates a new store.
func New() *Store {
	return &Store{
		store: map[string]Files{},
	}
}

// RegisterPeer adds a new peer with its files into the store.
func (p *Store) RegisterPeer(host string, files []FileMetadata) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.store[host] = make(Files, len(files))
	for _, file := range files {
		p.store[host][file.Name] = file
	}
}

// GetPeerByHost will return the peer files in case peer found, otherwise returns
// ErrPeerNotFound.
func (s *Store) GetPeerByHost(host string) (Files, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peer, ok := s.store[host]
	if !ok {
		return nil, ErrPeerNotFound
	}
	return peer, nil
}

// RemovePeerByHost will remove a peer from store, or return ErrPeerNotFound.
func (s *Store) RemovePeerByHost(host string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.store[host]; !ok {
		return ErrPeerNotFound
	}

	delete(s.store, host)

	return nil
}

// GetAllPeers collects all peers that store has and returns them.
func (s *Store) GetAllPeers() []Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers := make([]Peer, 0, len(s.store))

	for k, v := range s.store {
		files := make([]FileMetadata, 0, len(v))

		for _, file := range v {
			files = append(files, file)
		}
		peer := Peer{
			Host:  k,
			Files: files,
		}
		peers = append(peers, peer)
	}
	return peers
}

// GetPeersForFile will return the list of peers that have the requested file.
func (s *Store) GetPeersForFile(file FileMetadata) []Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var peers []Peer

	for host, files := range s.store {
		if meta, ok := files[file.Name]; ok {
			//additional checks
			if meta.Mime == file.Mime && meta.Size == file.Size && meta.Checksum == file.Checksum {
				var p Peer
				p.Host = host
				p.Files = append(p.Files, meta)
				peers = append(peers, p)
			}
		}
	}

	return peers
}

// UpdatePeer will update files for a peer.
func (s *Store) UpdatePeer(host string, updates []FileMetadata) {
	s.mu.Lock()
	defer s.mu.Unlock()
	updatedFiles := make(Files, len(updates))

	for _, update := range updates {
		updatedFiles[update.Name] = update
	}
	s.store[host] = updatedFiles
}

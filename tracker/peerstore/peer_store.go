package peerstore

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"time"
)

var ErrPeerNotFound = errors.New("peer not found")

// FileMetadata represents all required info related to a file on the network.
type FileMetadata struct {
	Name         string
	Size         uint64
	Mime         string
	Checksum     string
	LastModified time.Time
}

// Files is a collection of file metadata.
type Files map[string]FileMetadata

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
func (p *Store) RegisterPeer(ip string, port uint32, files []FileMetadata) {
	p.mu.Lock()
	defer p.mu.Unlock()
	portString := strconv.Itoa(int(port))
	peerName := net.JoinHostPort(ip, portString)

	p.store[peerName] = make(Files, len(files))
	for _, file := range files {
		p.store[peerName][file.Name] = file
	}
}

// GetPeerByHost will return the peer files in case peer found, otherwise returns
// ErrPeerNotFound.
func (s *Store) GetPeerByHost(ip string, port uint32) (Files, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := net.JoinHostPort(ip, strconv.Itoa(int(port)))
	peer, ok := s.store[key]
	if !ok {
		return nil, ErrPeerNotFound
	}
	return peer, nil
}

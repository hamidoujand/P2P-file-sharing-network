package store

import (
	"errors"
	"sync"
)

var ErrFileNotFound = errors.New("file not found")

// Store represents an in-memory storage for filemetadata
type Store struct {
	mu    sync.RWMutex
	store map[string]FileMetadata
}

// New returns a new store.
func New() *Store {
	return &Store{
		store: map[string]FileMetadata{},
	}
}

// GetFile returns the file matched by the name or ErrFileNotFound.
func (s *Store) GetFileMetadata(name string) (FileMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, ok := s.store[name]
	if !ok {
		return FileMetadata{}, ErrFileNotFound
	}
	return file, nil
}

// AddFileMetadata adds the new file metadata.
func (s *Store) AddFileMetadata(fm FileMetadata) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[fm.Name] = fm
}

// RemoveFileMetadata removes the file matched the given name or return ErrFileNotFound.
func (s *Store) RemoveFileMetadata(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, ok := s.store[filename]
	if !ok {
		return ErrFileNotFound
	}
	delete(s.store, file.Name)
	return nil
}

// UpdateFile updates the metad
func (s *Store) UpdateFileMetadata(updates FileMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, ok := s.store[updates.Name]
	if !ok {
		return ErrFileNotFound
	}
	s.store[file.Name] = updates
	return nil
}

// ListFileMetadats returns a slice of all file metadatas that it has.
func (s *Store) ListFileMetadatas() []FileMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	files := make([]FileMetadata, 0, len(s.store))

	for _, file := range s.store {
		files = append(files, file)
	}
	return files
}

package store

import (
	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/peer"
)

// FileMetadata represents the metadata related to file that each peer has.
type FileMetadata struct {
	Name             string
	Size             int64
	FileType         string
	Checksum         string
	ChunkNumbers     int64
	DefaultChunkSize int64
}

// NewFileMetadata creates a metadata from a protobuff file.
func NewFileMetadata(file *peer.FileMetadata) FileMetadata {
	return FileMetadata{
		Name:     file.GetName(),
		Size:     file.GetSize(),
		Checksum: file.GetChecksum(),
	}
}

// ToProtoBuff creates a *pb.FileMetadata.
func (fm FileMetadata) ToProtoBuff() *peer.FileMetadata {
	return &peer.FileMetadata{
		Name:     fm.Name,
		Size:     fm.Size,
		Checksum: fm.Checksum,
	}
}

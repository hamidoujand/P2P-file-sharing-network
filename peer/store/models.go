package store

import (
	"time"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FileMetadata represents the metadata related to file that each peer has.
type FileMetadata struct {
	Name         string
	Size         uint64
	FileType     string
	Checksum     string
	LastModified time.Time
}

// NewFileMetadata creates a metadata from a protobuff file.
func NewFileMetadata(file *pb.FileMetadata) FileMetadata {
	return FileMetadata{
		Name:         file.GetName(),
		Size:         file.GetSize(),
		FileType:     file.GetFileType(),
		Checksum:     file.GetChecksum(),
		LastModified: file.LastModified.AsTime(),
	}
}

// ToProtoBuff creates a *pb.FileMetadata.
func (fm FileMetadata) ToProtoBuff() *pb.FileMetadata {
	return &pb.FileMetadata{
		Name:         fm.Name,
		Size:         fm.Size,
		Checksum:     fm.Checksum,
		LastModified: timestamppb.New(fm.LastModified),
		FileType:     fm.FileType,
	}
}

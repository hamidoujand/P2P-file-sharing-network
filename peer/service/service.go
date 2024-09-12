package service

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/peer"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/tracker"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service represents all of the rpc service calls.
type Service struct {
	peer.UnimplementedPeerServiceServer
	store            *store.Store
	defaultChunkSize int64
	trackerClient    tracker.TrackerServiceClient
	fs               fs.FS
	dialer           grpc.DialOption
	host             string
}

type Config struct {
	Host             string
	Store            *store.Store
	TrackerClient    tracker.TrackerServiceClient
	DefaultChunkSize int64
	Fs               fs.FS
	Dialer           grpc.DialOption
}

// New creates a new rpc service.
func New(ctx context.Context, conf *Config) (*Service, error) {
	//register peer into tracker
	if conf.Fs == nil {
		return nil, errors.New("FS can not be nil")
	}

	var fileMetadata []store.FileMetadata

	//walk the fs
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		//files
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("info: %w", err)
		}

		file, err := conf.Fs.Open(path)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		hasher := sha256.New()
		chunk := make([]byte, 4*1024)

		for {
			n, err := reader.Read(chunk)
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("read: %w", err)
			}

			hasher.Write(chunk[:n])
		}

		rawHash := hasher.Sum(nil)
		checksum := fmt.Sprintf("%X", rawHash)

		fm := store.FileMetadata{
			Name:     info.Name(),
			Size:     info.Size(),
			Checksum: checksum,
		}
		//add it into store
		conf.Store.AddFileMetadata(fm)

		fileMetadata = append(fileMetadata, fm)
		return nil
	}

	if err := fs.WalkDir(conf.Fs, ".", walk); err != nil {
		return nil, fmt.Errorf("walk fs: %w", err)
	}

	files := make([]*tracker.File, len(fileMetadata))
	for i, fm := range fileMetadata {
		files[i] = &tracker.File{
			Name:     fm.Name,
			Size:     fm.Size,
			Checksum: fm.Checksum,
		}
	}

	//register peer with tracker
	in := &tracker.RegisterPeerRequest{
		Host:  conf.Host,
		Files: files,
	}
	resp, err := conf.TrackerClient.RegisterPeer(ctx, in)

	if err != nil {
		return nil, fmt.Errorf("rpc:tracker: %w", err)
	}

	if resp.StatusCode != int64(codes.OK) {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	return &Service{
		store:            conf.Store,
		defaultChunkSize: conf.DefaultChunkSize,
		trackerClient:    conf.TrackerClient,
		fs:               conf.Fs,
		dialer:           conf.Dialer,
		host:             conf.Host,
	}, nil
}

// Ping is used to check the health of the server.
func (s *Service) Ping(ctx context.Context, in *peer.PingRequest) (*peer.PingResponse, error) {
	message := fmt.Sprintf("peer[%s]: %s", s.host, in.GetMessage())
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
			//when the file is not on this peer.
			fmt.Printf("file [%s] not found on peer[%s]\n", filename, s.host)
			//try tracker service
			in := tracker.GetPeersForFileRequest{
				FileName: filename,
			}

			peers, err := s.trackerClient.GetPeersForFile(context.Background(), &in)
			if err != nil {
				return status.Errorf(codes.Internal, "get peers for file: %s", err)
			}
			if len(peers.Peers) == 0 {
				//not found in entire network
				fmt.Printf("file [%s] not found in entire network\n", filename)
				return status.Errorf(codes.NotFound, "file [%s] not found on entire network", filename)
			}

			fmt.Printf("total number of peers that have file[%s]: %d\n", filename, len(peers.Peers))

			for _, p := range peers.Peers {
				fmt.Printf("trying to ping peer [%s]\n", p.Host)
				//need a peer client
				peerConn, err := grpc.NewClient(p.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					fmt.Printf("failed to dial peer [%s]: %s", p.Host, err)
					continue
				}
				defer peerConn.Close()
				peerClient := peer.NewPeerServiceClient(peerConn)

				//ping peer
				in := peer.PingRequest{
					Message: "Hi",
				}

				resp, err := peerClient.Ping(context.Background(), &in)
				if err != nil {
					fmt.Printf("ping [%s] failed: %s\n", p.Host, err.Error())
					continue //continue to next peer that has the file
				}
				if resp.Status != codes.OK.String() {
					fmt.Printf("status [%s] not ok: %s\n", p.Host, resp.Status)
					continue
				}
				//get file metadata
				getIn := peer.GetFileMetadataRequest{
					Name: filename,
				}
				fm, err := peerClient.GetFileMetadata(context.Background(), &getIn)
				if err != nil {
					fmt.Printf("failed to fetch metadata from peer [%s]: %s\n", p.Host, err)
					continue
				}
				//handle download from another peer
				downloadIn := peer.DownloadFileRequest{
					FileName: fm.Metadata.GetName(),
				}

				anotherPeerStream, err := peerClient.DownloadFile(context.Background(), &downloadIn)
				if err != nil {
					fmt.Printf("failed to init download from peer [%s]: %s\n", p.Host, err)
					continue
				}
				path := "static" + "/" + fm.Metadata.GetName()

				file, err := os.Create(path)
				if err != nil {
					if !os.IsExist(err) {
						return status.Errorf(codes.Internal, "create: %s", err)
					}
					//reset the seeker, since we redownloadin from another peer
					file.Seek(0, 0)
				}

				bufWriter := bufio.NewWriter(file)

				for {
					otherPeerChunk, err := anotherPeerStream.Recv()
					if err == io.EOF {
						//flush
						if err := bufWriter.Flush(); err != nil {
							return status.Errorf(codes.Internal, "flush: %s", err)
						}
						//close
						if err := file.Close(); err != nil {
							return status.Errorf(codes.Internal, "close: %s", err)
						}

						sfm := store.FileMetadata{
							Name:     fm.Metadata.GetName(),
							Size:     fm.Metadata.GetSize(),
							Checksum: fm.Metadata.GetChecksum(),
						}
						//save metadata for this file into the store.
						s.store.AddFileMetadata(sfm)

						//update this peer on tracker
						fileMetas := s.store.ListFileMetadatas()
						trackerFiles := make([]*tracker.File, len(fileMetas))
						for i, fm := range fileMetas {
							trackerFiles[i] = &tracker.File{
								Name:     fm.Name,
								Size:     fm.Size,
								Checksum: fm.Checksum,
							}
						}

						updatePeerReq := &tracker.UpdatePeerRequest{
							Host:  s.host, //the current peer
							Files: trackerFiles,
						}
						_, err := s.trackerClient.UpdatePeer(context.Background(), updatePeerReq)
						if err != nil {
							//log the error since that does not concerns the end user
							fmt.Printf("peer[%s] failed to update it's state in tracker: %s", s.host, err.Error())
						}

						//close the client stream as well
						return nil
					}

					if err != nil {
						fmt.Printf("failed to receive chunk [%d]: %s", otherPeerChunk.ChunkNumber, err)
						continue
					}

					if _, err := bufWriter.Write(otherPeerChunk.Data); err != nil {
						return status.Errorf(codes.Internal, "write: %s", err)
					}

					//also we need to send this chunk to the cli that asked for it
					chunk := &peer.FileChunk{
						ChunkNumber: otherPeerChunk.ChunkNumber,
						Data:        otherPeerChunk.Data,
						TotalChunks: otherPeerChunk.TotalChunks,
					}

					if err := stream.Send(chunk); err != nil {
						return status.Errorf(codes.Internal, "failed to send chunk[%d]: %s", chunk.ChunkNumber, err)
					}
				}
			}
		} else {
			return status.Error(codes.Internal, codes.Internal.String())
		}
	}

	//when the file is on this peer
	fmt.Printf("file [%s] is on this peer\n", filename)
	file, err := s.fs.Open(in.GetFileName())
	if err != nil {
		if os.IsNotExist(err) {
			return status.Errorf(codes.NotFound, "file %s, not found", filename)
		}
		return status.Error(codes.Internal, codes.Internal.String())
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return status.Error(codes.Internal, codes.Internal.String())
	}

	totalChunks := int32((stats.Size() + s.defaultChunkSize - 1) / s.defaultChunkSize)
	buffer := make([]byte, s.defaultChunkSize)
	buffReader := bufio.NewReader(file)

	for chunkNumber := int32(1); chunkNumber <= totalChunks; chunkNumber++ {
		readBytes, err := buffReader.Read(buffer)
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
	var bufWriter *bufio.Writer
	hash := sha256.New()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			//check the file
			if file != nil {
				if err := bufWriter.Flush(); err != nil {
					return fmt.Errorf("flush: %w", err)
				}

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

				//update this peer on tracker
				fileMetas := s.store.ListFileMetadatas()
				trackerFiles := make([]*tracker.File, len(fileMetas))
				for i, fm := range fileMetas {
					trackerFiles[i] = &tracker.File{
						Name:     fm.Name,
						Size:     fm.Size,
						Checksum: fm.Checksum,
					}
				}

				updatePeerReq := &tracker.UpdatePeerRequest{
					Host:  s.host, //the current peer
					Files: trackerFiles,
				}
				_, err = s.trackerClient.UpdatePeer(context.Background(), updatePeerReq)
				if err != nil {
					//log the error since that does not concerns the end user
					fmt.Printf("peer[%s] failed to update it's state in tracker service: %s", s.host, err.Error())
				}

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
			filepath := filepath.Join("static", filename)
			file, err = os.Create(filepath)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			bufWriter = bufio.NewWriter(file)
		}
		target := make([]byte, len(chunk.Data))
		copy(target, chunk.Data)

		_, err = hash.Write(target)
		if err != nil {
			return fmt.Errorf("writing hash: %w", err)
		}
		//writing chunks, into buffer
		_, err = bufWriter.Write(chunk.GetData())
		if err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}

}

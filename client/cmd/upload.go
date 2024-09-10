package cmd

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hamidoujand/P2P-file-sharing-network/client/pb/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DownloadFileCommand represents a command and all required info to download a file from a peer.
type UploadFileCommand struct {
	fs       *flag.FlagSet
	peer     string
	filename string
}

func NewUploadFileCommand() *UploadFileCommand {
	c := UploadFileCommand{
		fs: flag.NewFlagSet("upload", flag.ContinueOnError),
	}

	//set all args
	c.fs.StringVar(&c.peer, "peer", "", "peer is the host:ip to upload file into.")
	c.fs.StringVar(&c.filename, "filename", "", "filename is the name of the file you want to upload.")
	return &c
}

func (uf *UploadFileCommand) Name() string {
	return uf.fs.Name()
}

func (uf *UploadFileCommand) Init(args []string) error {
	return uf.fs.Parse(args)
}

func (uf *UploadFileCommand) Run() error {
	if uf.filename == "" || uf.peer == "" {
		return errors.New("both 'filename' and 'peer' args are required")
	}

	//check the peer conn
	peerConn, err := grpc.NewClient(uf.peer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("new peer client: %w", err)
	}
	defer peerConn.Close()

	//peer client
	peerClient := peer.NewPeerServiceClient(peerConn)

	//ping
	pingInput := &peer.PingRequest{
		Message: "Hi",
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = peerClient.Ping(ctx, pingInput)
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	path := "client/static" + "/" + uf.filename
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file[%s] not found", uf.filename)
		}
		return fmt.Errorf("open: %w", err)
	}

	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	bufReader := bufio.NewReader(file)
	const chunkSize = 512 * 1024
	totalChunks := int32((stats.Size() + chunkSize - 1) / chunkSize)
	buff := make([]byte, chunkSize)

	stream, err := peerClient.UploadFile(context.Background())
	if err != nil {
		return fmt.Errorf("creating stream: %w", err)
	}

	for chunkNumber := int32(1); chunkNumber <= totalChunks; chunkNumber++ {
		n, err := bufReader.Read(buff)

		if err == io.EOF {
			//done
			break
		}

		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		//stream it
		chunk := peer.UploadFileChunk{
			ChunkNumber: int32(chunkNumber),
			TotalChunks: int32(totalChunks),
			FileName:    uf.filename,
			Data:        buff[:n],
		}
		if err := stream.Send(&chunk); err != nil {
			return fmt.Errorf("send chunk[%d]: %w", chunkNumber, err)
		}
		fmt.Printf("successfully sent: chunk[%d/%d]\n", chunkNumber, totalChunks)
	}
	//handle the response
	//close stream
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("close and receive: %w", err)
	}

	if !resp.Success {
		return errors.New("failed to complete the upload")
	}
	fmt.Println("uplaod successfully.")
	return nil
}

package cmd

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/hamidoujand/P2P-file-sharing-network/client/pb/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// DownloadFileCommand represents a command and all required info to download a file from a peer.
type DownlaodFileCommand struct {
	fs       *flag.FlagSet
	peer     string
	filename string
}

func NewDownloadFileCommand() *DownlaodFileCommand {
	c := DownlaodFileCommand{
		fs: flag.NewFlagSet("download", flag.ContinueOnError),
	}

	//set all args
	c.fs.StringVar(&c.peer, "peer", "", "peer is the host:ip used to connect to a peer")
	c.fs.StringVar(&c.filename, "filename", "", "filename is the name of the file you want to download")
	return &c
}

func (df *DownlaodFileCommand) Name() string {
	return df.fs.Name()
}

func (df *DownlaodFileCommand) Init(args []string) error {
	return df.fs.Parse(args)
}

func (df *DownlaodFileCommand) Run() error {
	//actual downlaod logic
	if df.filename == "" || df.peer == "" {
		return flag.ErrHelp
	}
	peerConn, err := grpc.NewClient(df.peer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer peerConn.Close()

	//peer client
	peerClient := peer.NewPeerServiceClient(peerConn)

	in := peer.DownloadFileRequest{
		FileName: df.filename,
	}

	stream, err := peerClient.DownloadFile(context.Background(), &in)
	if err != nil {
		status, ok := status.FromError(err)
		if !ok {
			return err
		}
		if status.Code() == codes.NotFound {
			return fmt.Errorf("file [%s] not found in the network", df.filename)
		}

		return fmt.Errorf("download file: %w", err)
	}

	//create static folders if not there already.
	static := "client/static"
	if err := os.MkdirAll(static, 0755); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("mkdir all: %w", err)
		}
	}

	//create a file to write into
	file, err := os.Create(static + "/" + df.filename)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	//do the download logic
	bufWriter := bufio.NewWriter(file)

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			//flush
			if err := bufWriter.Flush(); err != nil {
				return fmt.Errorf("flush: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close: %w", err)
			}
			break
		}

		if err != nil {
			//something went wrong
			return fmt.Errorf("recv: %w", err)
		}
		//write
		if _, err := bufWriter.Write(chunk.Data); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		fmt.Printf("downloaded[%d/%d]\n", chunk.ChunkNumber, chunk.TotalChunks)
	}

	return nil
}

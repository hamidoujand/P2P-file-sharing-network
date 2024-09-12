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
		return errors.New("both 'filename' and 'peer' args are required")
	}
	peerConn, err := grpc.NewClient(df.peer, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	in := peer.DownloadFileRequest{
		FileName: df.filename,
	}

	stream, err := peerClient.DownloadFile(context.Background(), &in)
	if err != nil {
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
			//remove the file
			if err := os.Remove("client/static/" + df.filename); err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("remove: %w", err)
				}
			}

			status, ok := status.FromError(err)
			if !ok {
				return fmt.Errorf("fromError: %w", err)
			}

			if status.Code() == codes.NotFound {
				return fmt.Errorf("file [%s] not found in network", df.filename)
			}
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

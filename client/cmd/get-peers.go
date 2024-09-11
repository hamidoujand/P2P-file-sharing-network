package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/hamidoujand/P2P-file-sharing-network/client/pb/tracker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GetPeersCommand struct {
	fs      *flag.FlagSet
	tracker string
}

func NewGetPeersCommand() *GetPeersCommand {
	c := GetPeersCommand{
		fs: flag.NewFlagSet("peers", flag.ContinueOnError),
	}

	//set all args
	c.fs.StringVar(&c.tracker, "tracker", "", "tracker is the control service.")
	return &c
}

func (gp *GetPeersCommand) Name() string {
	return gp.fs.Name()
}

func (gp *GetPeersCommand) Init(args []string) error {
	return gp.fs.Parse(args)
}

func (gp *GetPeersCommand) Run() error {
	if gp.tracker == "" {
		return errors.New("'tracker' is required arg")
	}

	//check the peer conn
	trackerConn, err := grpc.NewClient(gp.tracker, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("new peer client: %w", err)
	}
	defer trackerConn.Close()

	//peer client
	trackerClient := tracker.NewTrackerServiceClient(trackerConn)

	getPeersReq := &tracker.GetPeersRequest{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := trackerClient.GetPeers(ctx, getPeersReq)
	if err != nil {
		return fmt.Errorf("get peers: %w", err)
	}

	for _, peer := range resp.Peers {
		fmt.Println("============================================================")
		fmt.Printf("peer[%s]\n", peer.Host)
		if len(peer.Files) == 0 {
			continue
		}
		for _, file := range peer.Files {
			fmt.Printf("\tfile[%s]----> %s\n", file.GetName(), file.GetChecksum())
		}
		fmt.Println("============================================================")
	}

	return nil
}

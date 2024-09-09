package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/peer"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb/tracker"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/service"
	"github.com/hamidoujand/P2P-file-sharing-network/peer/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	host := "0.0.0.0:50052"
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	server := grpc.NewServer()
	store := store.New()
	const defaultChunk int64 = 1024 * 1024 //1MB

	//create static dir
	if err := os.MkdirAll("peer/static", 0755); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("mkidr: %w", err)
		}
	}

	trackerAddr := "127.0.0.1:50051"
	trackerConn, err := grpc.NewClient(trackerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("new tracker client: %w", err)
	}
	trackerClient := tracker.NewTrackerServiceClient(trackerConn)

	fsys := os.DirFS("peer/static")

	conf := service.Config{
		Host:             host,
		Store:            store,
		TrackerClient:    trackerClient,
		DefaultChunkSize: defaultChunk,
		Fs:               fsys,
	}

	service, err := service.New(context.Background(), &conf)
	if err != nil {
		return fmt.Errorf("new service: %w", err)
	}
	peer.RegisterPeerServiceServer(server, service)

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)

	go func() {
		log.Println("peer1 listening on:", listener.Addr())
		errCh <- server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-shutdownCh:
		server.GracefulStop()
		return nil
	}
}

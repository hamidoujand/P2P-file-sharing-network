package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/pb/tracker"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/peerstore"
	"github.com/hamidoujand/P2P-file-sharing-network/tracker/service"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	//==========================================================================
	//peers store
	store := peerstore.New()
	service := service.New(store)

	server := grpc.NewServer()
	tracker.RegisterTrackerServiceServer(server, service)

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)

	go func() {
		log.Println("tracker listening on:", listener.Addr())
		errCh <- server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-shutdownCh:
		//gracefull
		server.GracefulStop()
		return nil
	}
}

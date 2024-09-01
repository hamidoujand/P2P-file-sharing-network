package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hamidoujand/P2P-file-sharing-network/peer/pb"
	"google.golang.org/grpc"
)

type service struct {
	pb.UnimplementedPeerServiceServer
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	server := grpc.NewServer()
	pb.RegisterPeerServiceServer(server, &service{})

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

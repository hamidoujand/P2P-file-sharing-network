package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hamidoujand/P2P-file-sharing-network/tracker/pb"
	"google.golang.org/grpc"
)

type service struct {
	pb.UnimplementedTrackerServiceServer
}

func (s *service) HelloWorld(ctx context.Context, in *pb.RequestBody) (*pb.ResponseBody, error) {
	msg := fmt.Sprintf("hello, %s", in.Name)
	result := &pb.ResponseBody{
		Greet: msg,
	}
	return result, nil
}

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

	server := grpc.NewServer()
	pb.RegisterTrackerServiceServer(server, &service{})

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

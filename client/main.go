package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/hamidoujand/P2P-file-sharing-network/client/cmd"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		return errors.New("need to pass a subcommand: <cmd> <subcommand> args")
	}

	supportedCMDs := []cmd.Runner{
		cmd.NewDownloadFileCommand(),
		cmd.NewUploadFileCommand(),
	}

	subcommand := args[0]
	for _, command := range supportedCMDs {
		if command.Name() == subcommand {
			err := command.Init(args[1:])
			if err != nil {
				return fmt.Errorf("init: %w", err)
			}

			return command.Run()
		}
	}

	return fmt.Errorf("unknown command: %q", subcommand)
}

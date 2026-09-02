package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mashiike/stefunny"
)

func main() {
	os.Exit(run(stefunny.NewCLI(), os.Args[1:]))
}

// run executes the CLI and returns the process exit code.
//
// The CLI is taken as an argument so that tests can inject writers and an exit
// function instead of letting kong call os.Exit on the test binary. main() does
// nothing but exit with the returned code, because os.Exit skips deferred calls
// and would otherwise leak the signal context.
func run(cli *stefunny.CLI, args []string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, os.Interrupt)
	defer cancel()

	err := cli.Main(ctx, args)
	if err == nil {
		return 0
	}
	if errors.Is(err, stefunny.ErrHasDiff) {
		return 2
	}
	log.Printf("[error] %s", err)
	return 1
}

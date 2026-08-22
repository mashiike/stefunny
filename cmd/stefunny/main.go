package main

import (
	"context"
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

	if err := cli.Run(ctx, args); err != nil {
		log.Printf("[error] %s", err)
		return 1
	}
	return 0
}

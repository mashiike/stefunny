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
	os.Exit(run(os.Args[1:]))
}

// os.Exit skips deferred calls, so the CLI runs here and main() only exits with the returned code.
func run(args []string) int {
	cli := stefunny.NewCLI()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, os.Interrupt)
	defer cancel()

	if err := cli.Run(ctx, args); err != nil {
		log.Printf("[error] %s", err)
		return 1
	}
	return 0
}

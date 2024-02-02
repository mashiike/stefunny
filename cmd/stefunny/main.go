package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/logger"
)

func main() {
	cli := stefunny.NewCLI()
	cli.SetLogLevelFunc(func(level string) error {
		logger.Setup(os.Stderr, level)
		return nil
	})
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, os.Interrupt)
	defer cancel()

	if err := cli.Run(ctx, os.Args[1:]); err != nil {
		log.Printf("[error] %s", err)
	}
}

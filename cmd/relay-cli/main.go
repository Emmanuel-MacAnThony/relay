package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
)

func main() {
	slug := flag.String("slug", "", "endpoint slug to connect as")
	server := flag.String("server", "ws://localhost:8080", "relay server base URL")
	target := flag.String("target", "", "local service URL to forward requests to")
	flag.Parse()

	if *slug == "" || *target == "" {
		flag.Usage()
		os.Exit(1)
	}

	log := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	executor := NewExecutor(*target)
	client := NewClient(*server, *slug, executor, log)

	log.Info("relay cli starting", "slug", *slug, "server", *server, "target", *target)
	client.Run(ctx)
}

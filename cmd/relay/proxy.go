package main

import (
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/proxy"
)

// forwardTimeout is how long the Forwarder waits for a CLI response before
// treating the delivery as failed. 30 seconds covers slow local handlers
// without holding the goroutine open indefinitely.
const forwardTimeout = 30 * time.Second

func setupProxyLayer() (*proxy.Registry, *proxy.Forwarder, *proxy.Notifier) {
	registry := proxy.NewRegistry()
	forwarder := proxy.NewForwarder(registry, forwardTimeout)
	notifier := proxy.NewNotifier(registry)
	return registry, forwarder, notifier
}

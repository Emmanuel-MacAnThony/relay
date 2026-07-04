package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/proxy"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
	"nhooyr.io/websocket"
)

const (
	reconnectBaseDelay = 1 * time.Second
	reconnectMaxDelay  = 30 * time.Second
)

type Client struct {
	serverURL string
	slug      string
	executor  *Executor
	log       *logger.Logger
}

func NewClient(serverURL, slug string, executor *Executor, log *logger.Logger) *Client {
	return &Client{
		serverURL: serverURL,
		slug:      slug,
		executor:  executor,
		log:       log,
	}
}

// Run keeps the client connected to the relay server, reconnecting with
// exponential backoff whenever the connection drops. It exits when ctx
// is cancelled (process shutdown).
func (c *Client) Run(ctx context.Context) {
	delay := reconnectBaseDelay
	for {
		err := c.connect(ctx)
		if ctx.Err() != nil {
			return
		}
		c.log.Warn("disconnected, reconnecting", "delay", delay, "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		delay = min(delay*2, reconnectMaxDelay)
	}
}

// connect dials the relay server, registers this CLI for its slug, and
// processes forwarded requests until the connection drops or ctx is cancelled.
// It returns the error that caused the disconnect.
func (c *Client) connect(ctx context.Context) error {
	wsURL := c.serverURL + "/ws/cli/" + c.slug
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	c.log.Info("connected to relay", "slug", c.slug, "server", c.serverURL)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}

		var env proxy.ForwardEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			c.log.Warn("failed to parse envelope", "err", err)
			continue
		}
		if env.Type != proxy.TypeForwardRequest {
			continue
		}

		go func(env proxy.ForwardEnvelope) {
			attempt := c.executor.Execute(ctx, env.Data)

			result, err := json.Marshal(proxy.ResultEnvelope{
				Type:          proxy.TypeDeliveryResult,
				CorrelationID: env.CorrelationID,
				Data:          attempt,
			})
			if err != nil {
				c.log.Error("failed to marshal result", "err", err)
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, result); err != nil {
				c.log.Error("failed to send result", "correlation_id", env.CorrelationID, "err", err)
			}
		}(env)
	}
}

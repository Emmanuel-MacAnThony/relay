package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/api"
	"github.com/Emmanuel-MacAnThony/relay/internal/config"
	appdb "github.com/Emmanuel-MacAnThony/relay/internal/db"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
)

func main() {
	log := logger.New()
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool := appdb.Connect(ctx, cfg.DB.URL)
	defer pool.Close()

	registry, forwarder, notifier := setupProxyLayer()

	router := api.NewRouter(api.RouterDeps{
		Request:  setupRequestDomain(ctx, pool, forwarder, notifier, log),
		Endpoint: setupEndpointDomain(ctx, pool, cfg.Server.BaseURL, log),
		WS:       api.NewWSHandler(registry, forwarder),
	})


	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown error", "err", err)
		}
	}()

	log.Info("relay server starting", "port", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}
}

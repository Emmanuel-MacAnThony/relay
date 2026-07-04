package main

import (
	"context"

	"github.com/Emmanuel-MacAnThony/relay/internal/api"
	endpointinfra "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/infra"
	requestinfra "github.com/Emmanuel-MacAnThony/relay/internal/request/infra"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/create"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/get"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/list"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/replay"
	"github.com/Emmanuel-MacAnThony/relay/internal/proxy"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupRequestDomain(
	ctx context.Context,
	pool *pgxpool.Pool,
	forwarder *proxy.Forwarder,
	notifier *proxy.Notifier,
	log *logger.Logger,
) *api.RequestHandler {
	repo := requestinfra.NewRepo(ctx, pool)
	endpointRepo := endpointinfra.NewRepo(ctx, pool)

	return api.NewRequestHandler(api.RequestHandlerDeps{
		Create:   create.New(repo, endpointRepo, forwarder, notifier),
		Get:      get.New(repo),
		List:     list.New(repo),
		Replay:   replay.New(repo, forwarder, notifier),
		Attempts: repo,
	}, log)
}

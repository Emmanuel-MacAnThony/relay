package main

import (
	"context"

	"github.com/Emmanuel-MacAnThony/relay/internal/api"
	endpointinfra "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/infra"
	endpointcreate "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/usecases/create"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupEndpointDomain(ctx context.Context, pool *pgxpool.Pool, baseURL string, log *logger.Logger) *api.EndpointHandler {
	repo := endpointinfra.NewRepo(ctx, pool)

	return api.NewEndpointHandler(api.EndpointHandlerDeps{
		Create: endpointcreate.New(repo),
	}, baseURL, log)
}

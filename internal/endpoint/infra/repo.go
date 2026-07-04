package infra

import (
	"context"
	"errors"

	"github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"
	endpointcreate "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/usecases/create"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type Repo struct {
	ctx     context.Context
	queries *Queries
}

func NewRepo(ctx context.Context, pool *pgxpool.Pool) *Repo {
	return &Repo{ctx: ctx, queries: New(pool)}
}

func (r *Repo) Save(e domain.Endpoint) error {
	err := r.queries.SaveEndpoint(r.ctx, SaveEndpointParams{
		Slug:      e.Slug,
		CreatedAt: pgtype.Timestamptz{Time: e.CreatedAt, Valid: true},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return endpointcreate.ErrSlugConflict
		}
		return err
	}
	return nil
}

func (r *Repo) Get(slug string) (domain.Endpoint, error) {
	row, err := r.queries.GetEndpoint(r.ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Endpoint{}, domain.ErrNotFound
		}
		return domain.Endpoint{}, err
	}
	return domain.Endpoint{
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

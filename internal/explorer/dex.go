package dex

import (
	"arbitrage/internal/models"
	"context"
)

type DEX interface {
	FetchPools(context.Context) ([]models.Pool, error)
}

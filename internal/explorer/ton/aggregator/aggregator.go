package aggregator

import (
	dedustapi "arbitrage/internal/explorer/ton/dedust"
	stonfiapi "arbitrage/internal/explorer/ton/stonfi"
	"arbitrage/internal/models"
	"context"
	"fmt"
	"log/slog"
)

type AggregatorSettings struct {
	DedustExplorer *dedustapi.DedustExplorer
	UseDeDust      bool
	UseStonFi      bool
}

func FetchPools(ctx context.Context, settings AggregatorSettings) ([]models.Pool, error) {
	pools := make([]models.Pool, 0)

	if settings.UseDeDust {
		slog.Info("Fetching dedust pools")
		deDustPools, err := settings.DedustExplorer.FetchPools(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch dedust pools: %w", err)
		}
		pools = append(pools, deDustPools...)
	}

	if settings.UseStonFi {
		slog.Info("Fetching stonfi pools")
		stonFiPools, err := stonfiapi.FetchPools(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch stonfi pools: %w", err)
		}
		pools = append(pools, stonFiPools...)
	}

	return pools, nil
}

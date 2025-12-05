package aggregator

import (
	"arbitrage/internal/explorer/ton/chain"
	coffeeapi "arbitrage/internal/explorer/ton/coffee"
	dedustapi "arbitrage/internal/explorer/ton/dedust"
	stonfiapi "arbitrage/internal/explorer/ton/stonfi"
	"arbitrage/internal/models"
	"context"
	"fmt"
)

type AggregatorSettings struct {
	UseDeDust           bool
	DedustPoolsFilePath string

	UseStonFi           bool
	StonfiPoolsFilePath string

	UseCoffee bool
}

func FetchPools(ctx context.Context, client *chain.TonClient, settings AggregatorSettings) ([]models.Pool, error) {
	pools := make([]models.Pool, 0)

	if settings.UseDeDust {
		fmt.Println("Fetching dedust pools")
		explorer, err := dedustapi.NewDedustExplorer(client, settings.DedustPoolsFilePath)
		if err != nil {
			return nil, fmt.Errorf("create dedust explorer: %w", err)
		}
		deDustPools, err := explorer.FetchPools(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch dedust pools: %w", err)
		}
		pools = append(pools, deDustPools...)
	}

	if settings.UseStonFi {
		fmt.Println("Fetching stonfi pools")
		stonFiPools, err := stonfiapi.FetchPools(settings.StonfiPoolsFilePath)
		if err != nil {
			return nil, fmt.Errorf("fetch stonfi pools: %w", err)
		}
		pools = append(pools, stonFiPools...)
	}

	if settings.UseCoffee {
		fmt.Println("Fetching coffee pools")
		coffeePools, err := coffeeapi.FetchPools()
		if err != nil {
			return nil, fmt.Errorf("fetch coffee pools: %w", err)
		}
		pools = append(pools, coffeePools...)
	}

	return pools, nil
}

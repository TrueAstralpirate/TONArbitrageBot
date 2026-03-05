package dedustapi

import (
	"arbitrage/internal/explorer/ton/chain"
	"arbitrage/internal/explorer/ton/utils"
	models "arbitrage/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

type PoolAssets struct {
	Address0 string `json:"address0"`
	Address1 string `json:"address1"`
}

type DedustExplorer struct {
	Client               *chain.TonClient
	PoolToAssetAddresses map[string]PoolAssets
	PoolsMapFilePath     string
}

func validatePool(p *Pool) bool {
	for _, asset := range p.Assets {
		if asset.Name == "" {
			return false
		}
	}
	for _, reserve := range p.Reserves {
		if reserve == "0" {
			return false
		}
	}
	return p.Type == "volatile"
}

func ReadPoolAssetsFromFile(filePath string) (map[string]PoolAssets, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Declare the map
	var data map[string]PoolAssets

	// Unmarshal JSON
	err = json.Unmarshal(fileContent, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json: %v", err)
	}
	return data, nil
}

func WritePoolAssetsToFile(data map[string]PoolAssets, filePath string) error {
	// Marshal the map to pretty-printed JSON
	jsonBytes, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write the JSON bytes to the file
	err = os.WriteFile(filePath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func NewDedustExplorer(client *chain.TonClient, filePath string) (*DedustExplorer, error) {
	poolToAssets, err := ReadPoolAssetsFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool-assets map from file: %w", err)
	}
	return &DedustExplorer{
		Client:               client,
		PoolToAssetAddresses: poolToAssets,
		PoolsMapFilePath:     filePath,
	}, nil
}

func (e *DedustExplorer) FetchPools(ctx context.Context) ([]models.Pool, error) {
	pools, err := GetPools(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pools from API: %w", err)
	}
	resultPools := make([]models.Pool, 0)
	newPoolsFound := false
	for i := range pools {
		p := &pools[i]
		if validatePool(p) {
			fee, err := strconv.ParseFloat(p.TradeFee, 64)
			if err != nil {
				slog.Warn("failed to parse TradeFee", "pool", p.Address, "error", err)
				continue
			}

			reserve0, err := utils.ParseReserve(p.Reserves[0], p.Assets[0].Decimals)
			if err != nil {
				slog.Warn("failed to parse Reserve0", "pool", p.Address, "error", err)
				continue
			}
			reserve1, err := utils.ParseReserve(p.Reserves[1], p.Assets[1].Decimals)
			if err != nil {
				slog.Warn("failed to parse Reserve1", "pool", p.Address, "error", err)
				continue
			}

			assets, ok := e.PoolToAssetAddresses[p.Address]
			if !ok {
				assetsFromChain, err := e.Client.GetDedustAssets(ctx, p.Address)
				if err != nil {
					slog.Warn("failed to get pool assets from TON blockchain", "pool", p.Address, "error", err)
					continue
				}
				assets = PoolAssets{
					Address0: assetsFromChain.Asset0,
					Address1: assetsFromChain.Asset1,
				}
				e.PoolToAssetAddresses[p.Address] = assets
				newPoolsFound = true
			}

			resultPools = append(resultPools, models.Pool{
				DEX:      models.DEXNameDeDust,
				Address:  p.Address,
				TradeFee: fee,
				Type:     p.Type,
				TokenA: models.PoolToken{
					Metadata: models.TokenMetadata{
						Name:     p.Assets[0].Name,
						Symbol:   p.Assets[0].Symbol,
						Address:  assets.Address0,
						Decimals: p.Assets[0].Decimals,
					},
					Reserve: reserve0,
				},
				TokenB: models.PoolToken{
					Metadata: models.TokenMetadata{
						Name:     p.Assets[1].Name,
						Symbol:   p.Assets[1].Symbol,
						Address:  assets.Address1,
						Decimals: p.Assets[1].Decimals,
					},
					Reserve: reserve1,
				},
			})
		}
	}

	if newPoolsFound {
		if err := WritePoolAssetsToFile(e.PoolToAssetAddresses, e.PoolsMapFilePath); err != nil {
			slog.Error("failed to save pool-assets map", "error", err)
		}
	}

	return resultPools, nil
}

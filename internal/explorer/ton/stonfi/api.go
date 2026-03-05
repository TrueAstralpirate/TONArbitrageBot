package stonfiapi

import (
	"arbitrage/internal/explorer/ton/utils"
	"arbitrage/internal/httputil"
	"arbitrage/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

const (
	kTON            = "TON"
	kTONBurnAddress = "EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c"
)

type Asset struct {
	Blacklisted         bool   `json:"blacklisted"`
	Community           bool   `json:"community"`
	ContractAddress     string `json:"contract_address"`
	CustomPayloadApiUri string `json:"custom_payload_api_uri"`
	Decimals            int    `json:"decimals"`
	Deprecated          bool   `json:"deprecated"`
	DexPriceUsd         string `json:"dex_price_usd"`
	DexUsdPrice         string `json:"dex_usd_price"`
	DisplayName         string `json:"display_name"`
	Kind                string `json:"kind"`
	Symbol              string `json:"symbol"`
	ThirdPartyPriceUsd  string `json:"third_party_price_usd"`
	ThirdPartyUsdPrice  string `json:"third_party_usd_price"`
	WalletAddress       string `json:"wallet_address"`
}

// AssetListResponse is the response structure for /v1/assets
type AssetListResponse struct {
	AssetList []Asset `json:"asset_list"`
}

func GetAssets(ctx context.Context) ([]Asset, error) {
	url := "https://api.ston.fi/v1/assets"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := httputil.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result AssetListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return result.AssetList, nil
}

type Pool struct {
	Address                    string `json:"address"`
	Amp                        string `json:"amp"`
	CollectedToken0ProtocolFee string `json:"collected_token0_protocol_fee"`
	CollectedToken1ProtocolFee string `json:"collected_token1_protocol_fee"`
	LpAccountAddress           string `json:"lp_account_address"`
	LpBalance                  string `json:"lp_balance"`
	LpFee                      string `json:"lp_fee"`
	LpPriceUsd                 string `json:"lp_price_usd"`
	LpTotalSupply              string `json:"lp_total_supply"`
	LpTotalSupplyUsd           string `json:"lp_total_supply_usd"`
	LpWalletAddress            string `json:"lp_wallet_address"`
	ProtocolFee                string `json:"protocol_fee"`
	ProtocolFeeAddress         string `json:"protocol_fee_address"`
	RefFee                     string `json:"ref_fee"`
	Reserve0                   string `json:"reserve0"`
	Reserve1                   string `json:"reserve1"`
	RouterAddress              string `json:"router_address"`
	Token0Address              string `json:"token0_address"`
	Token0Balance              string `json:"token0_balance"`
	Token1Address              string `json:"token1_address"`
	Token1Balance              string `json:"token1_balance"`
}

// PoolListResponse represents the full response from /v1/pools
type PoolListResponse struct {
	PoolList []Pool `json:"pool_list"`
}

func GetPools(ctx context.Context) ([]Pool, error) {
	url := "https://api.ston.fi/v1/pools"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := httputil.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var poolData PoolListResponse
	if err := json.Unmarshal(body, &poolData); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}
	return poolData.PoolList, nil
}

func ReadPoolsInfoFromFile(filePath string) (map[string]string, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Declare the map
	var data map[string]string

	// Unmarshal JSON
	err = json.Unmarshal(fileContent, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json: %v", err)
	}
	return data, nil
}

func FetchPools(ctx context.Context, filePath string) ([]models.Pool, error) {
	assets, err := GetAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("get assets: %w", err)
	}
	pools, err := GetPools(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pools: %w", err)
	}

	assetsMap := make(map[string]Asset)
	for _, asset := range assets {
		assetsMap[asset.ContractAddress] = asset
	}

	poolsMap, err := ReadPoolsInfoFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read pools info from file: %w", err)
	}

	resultPools := make([]models.Pool, 0)
	for _, p := range pools {
		if _, ok := poolsMap[p.Address]; ok {
			slog.Warn("skip pool", "address", p.Address)
			continue
		}

		lpFee, err := strconv.ParseFloat(p.LpFee, 64)
		if err != nil {
			slog.Warn("failed to parse LpFee", "pool", p.Address, "error", err)
			continue
		}
		protocolFee, err := strconv.ParseFloat(p.ProtocolFee, 64)
		if err != nil {
			slog.Warn("failed to parse ProtocolFee", "pool", p.Address, "error", err)
			continue
		}

		finalFee := (lpFee + protocolFee) / 100.0

		asset0, ok := assetsMap[p.Token0Address]
		if !ok {
			continue
		}
		if asset0.ContractAddress == kTONBurnAddress {
			asset0.ContractAddress = kTON
		}
		asset1, ok := assetsMap[p.Token1Address]
		if !ok {
			continue
		}
		if asset1.ContractAddress == kTONBurnAddress {
			asset1.ContractAddress = kTON
		}

		reserve0, err := utils.ParseReserve(p.Reserve0, asset0.Decimals)
		if err != nil {
			slog.Warn("failed to parse Reserve0", "pool", p.Address, "error", err)
			continue
		}
		reserve1, err := utils.ParseReserve(p.Reserve1, asset1.Decimals)
		if err != nil {
			slog.Warn("failed to parse Reserve1", "pool", p.Address, "error", err)
			continue
		}
		if reserve0 == 0 && reserve1 == 0 {
			continue
		}

		resultPools = append(resultPools, models.Pool{
			DEX:           models.DEXNameStonFi,
			RouterAddress: p.RouterAddress,
			Address:       p.Address,
			TradeFee:      finalFee,
			Type:          "v1",
			TokenA: models.PoolToken{
				Metadata: models.TokenMetadata{
					Name:     asset0.DisplayName,
					Symbol:   asset0.Symbol,
					Address:  asset0.ContractAddress,
					Decimals: asset0.Decimals,
				},
				Reserve: reserve0,
			},
			TokenB: models.PoolToken{
				Metadata: models.TokenMetadata{
					Name:     asset1.DisplayName,
					Symbol:   asset1.Symbol,
					Address:  asset1.ContractAddress,
					Decimals: asset1.Decimals,
				},
				Reserve: reserve1,
			},
		})
	}

	return resultPools, nil
}

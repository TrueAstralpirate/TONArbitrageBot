package coffeeapi

import (
	"arbitrage/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	kNative = "native"
	kTON    = "TON"
)

// Coffee API response structures based on the swap.coffee documentation
type TokenAddress struct {
	Blockchain string `json:"blockchain"`
	Address    string `json:"address"`
}

type TokenMetadata struct {
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Decimals     int    `json:"decimals"`
	ImageURL     string `json:"image_url"`
	Listed       bool   `json:"listed"`
	Verification string `json:"verification"`
}

type PoolRestrictions struct {
	MinSwapAmount string `json:"min_swap_amount"`
	MaxSwapAmount string `json:"max_swap_amount"`
}

type PoolFees struct {
	AverageGas  float64 `json:"average_gas"`
	Divider     int     `json:"divider"`
	Input       int     `json:"input"`
	Output      int     `json:"output"`
	FirstToken  int     `json:"first_token"`
	SecondToken int     `json:"second_token"`
}

type CoffeePool struct {
	DEX         string                 `json:"dex"`
	Address     string                 `json:"address"`
	Type        string                 `json:"type"`
	AMMType     string                 `json:"amm_type"`
	AMMSettings map[string]interface{} `json:"amm_settings"`
	Tokens      []struct {
		Address  TokenAddress  `json:"address"`
		Metadata TokenMetadata `json:"metadata"`
	} `json:"tokens"`
	Reserves         []float64          `json:"reserves"`
	Restrictions     []PoolRestrictions `json:"restrictions"`
	Fees             PoolFees           `json:"fees"`
	UnavailableUntil *int64             `json:"unavailable_until"`
}

type PoolInfo struct {
	Address           string  `json:"address"`
	TVLUSD            float64 `json:"tvl_usd"`
	VolumeUSD         float64 `json:"volume_usd"`
	FeeUSD            float64 `json:"fee_usd"`
	APR               float64 `json:"apr"`
	LPAPR             float64 `json:"lp_apr"`
	BoostAPR          float64 `json:"boost_apr"`
	LockedAssetAmount string  `json:"locked_asset_amount"`
}

type CoffeePoolResponse struct {
	TotalCount int `json:"total_count"`
	Pools      []struct {
		Pool CoffeePool `json:"pool"`
		Info PoolInfo   `json:"info"`
	} `json:"pools"`
}

// GetPools fetches pools from Coffee DEX API
func GetPools() ([]CoffeePool, error) {
	url := "https://backend.swap.coffee/v1/pools"

	// Add query parameters to get only coffee-native pools
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("dexes", "coffee") // Only coffee DEX pools
	q.Add("trusted", "true") // Only trusted pools
	q.Add("size", "100")     // Get more pools
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result []CoffeePoolResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no pools found")
	}

	// Extract just the pools from the response
	pools := make([]CoffeePool, 0, len(result[0].Pools))
	for _, p := range result[0].Pools {
		pools = append(pools, p.Pool)
	}

	return pools, nil
}

// FetchPools fetches pools from Coffee DEX and converts them to the common Pool model
func FetchPools() ([]models.Pool, error) {
	pools, err := GetPools()
	if err != nil {
		return nil, fmt.Errorf("get pools: %w", err)
	}

	resultPools := make([]models.Pool, 0)
	for _, p := range pools {
		// Skip pools that don't have exactly 2 tokens
		if len(p.Tokens) != 2 {
			continue
		}

		// Skip pools that don't have exactly 2 reserves
		if len(p.Reserves) != 2 {
			continue
		}

		// Calculate trade fee as percentage (divider is typically 10000 for 0.1% fees)
		tradeFee := 0.0
		if p.Fees.Divider > 0 {
			tradeFee = (100.0 / float64(p.Fees.Divider)) * 100.0 // Convert to percentage
		}

		// Handle TON native token address
		token0Address := p.Tokens[0].Address.Address
		if token0Address == kNative {
			token0Address = kTON
		}
		token1Address := p.Tokens[1].Address.Address
		if token1Address == kNative {
			token1Address = kTON
		}

		// Skip pools with zero reserves
		if p.Reserves[0] == 0 && p.Reserves[1] == 0 {
			continue
		}

		// Skip pools that are not constant product AMM
		if p.AMMType != "constant_product" {
			continue
		}

		resultPools = append(resultPools, models.Pool{
			DEX:           models.DEXNameCoffee,
			RouterAddress: "", // Coffee DEX doesn't have a router address concept
			Address:       p.Address,
			TradeFee:      tradeFee,
			Type:          p.AMMType,
			TokenA: models.PoolToken{
				Metadata: models.TokenMetadata{
					Name:     p.Tokens[0].Metadata.Name,
					Symbol:   p.Tokens[0].Metadata.Symbol,
					Address:  token0Address,
					Decimals: p.Tokens[0].Metadata.Decimals,
				},
				Reserve: p.Reserves[0],
			},
			TokenB: models.PoolToken{
				Metadata: models.TokenMetadata{
					Name:     p.Tokens[1].Metadata.Name,
					Symbol:   p.Tokens[1].Metadata.Symbol,
					Address:  token1Address,
					Decimals: p.Tokens[1].Metadata.Decimals,
				},
				Reserve: p.Reserves[1],
			},
		})
	}

	return resultPools, nil
}

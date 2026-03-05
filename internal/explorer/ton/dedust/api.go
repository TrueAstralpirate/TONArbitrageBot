package dedustapi

import (
	"arbitrage/internal/httputil"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Metadata struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Image    string `json:"image"`
	Decimals int    `json:"decimals"`
	Address  string
}

type Asset struct {
	Type     string `json:"type"`
	Metadata `json:"metadata"`
}

type Stats struct {
	Fees    []string `json:"fees"`
	Volumes []string `json:"volume"`
}

// Define a struct to match the JSON structure.
// Update these fields based on the actual API response.
type Pool struct {
	Address     string   `json:"address"`
	LT          string   `json:"lt"`
	TotalSupply string   `json:"totalSupply"`
	Type        string   `json:"type"`
	TradeFee    string   `json:"tradeFee"`
	Assets      []Asset  `json:"assets"`
	LastPrice   string   `json:"lastPrice"`
	Reserves    []string `json:"reserves"`
	Stats       `json:"stats"`
}

type Token struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type Trade struct {
	Sender    string `json:"sender"`
	AssetIn   Token  `json:"assetIn"`
	AssetOut  Token  `json:"assetOut"`
	AmountIn  string `json:"amountIn"`
	AmountOut string `json:"amountOut"`
	LT        string `json:"lt"`
	CreatedAt string `json:"createdAt"`
}

// FetchTrades fetches trades for a given pool address
func FetchTrades(ctx context.Context, poolAddress string) ([]Trade, error) {
	url := fmt.Sprintf("https://api.dedust.io/v2/pools/%s/trades", poolAddress)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("accept", "application/json")

	resp, err := httputil.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	var trades []Trade
	if err := json.Unmarshal(body, &trades); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON: %w", err)
	}

	return trades, nil
}

func GetPools(ctx context.Context) ([]Pool, error) {
	url := "https://api.dedust.io/v2/pools"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := httputil.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching pools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var pools []Pool
	if err := json.Unmarshal(body, &pools); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return pools, nil
}

package dedustapi

import (
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

// fetchTrades fetches trades for a given pool address
func FetchTrades(poolAddress string) ([]Trade, error) {
	url := fmt.Sprintf("https://api.dedust.io/v2/pools/%s/trades", poolAddress)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
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

func GetPools() []Pool {
	url := "https://api.dedust.io/v2/pools"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status code:", resp.Status)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return nil
	}

	var pools []Pool
	if err := json.Unmarshal(body, &pools); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil
	}
	return pools
}

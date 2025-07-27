package models

import "fmt"

type TokenMetadata struct {
	Name     string
	Symbol   string
	Address  string
	Decimals int
}

type PoolToken struct {
	Metadata TokenMetadata
	Reserve  float64
}

type Pool struct {
	DEX           string
	RouterAddress string
	Address       string
	TradeFee      float64
	Type          string
	TokenA        PoolToken
	TokenB        PoolToken
}

func (p *Pool) EstimateSwap(tokenToSwap string, slippage, amount float64) (float64, *TokenMetadata, error) {
	fee := (1.0 - p.TradeFee/100.0)
	if tokenToSwap == p.TokenA.Metadata.Address {
		return (1.0 - slippage/100.0) * (p.TokenB.Reserve * fee * amount) / (p.TokenA.Reserve + fee*amount), &p.TokenB.Metadata, nil
	} else if tokenToSwap == p.TokenB.Metadata.Address {
		return (1.0 - slippage/100.0) * (p.TokenA.Reserve * fee * amount) / (p.TokenB.Reserve + fee*amount), &p.TokenA.Metadata, nil
	}
	return 0, nil, fmt.Errorf("pool doesn't contain token %s", tokenToSwap)
}

type ArbitrageCycle struct {
	TokensOrder  []TokenMetadata
	PoolsOrder   []Pool
	StartCapital float64
	Profit       float64
}

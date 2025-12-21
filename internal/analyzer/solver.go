package analyzer

import (
	"arbitrage/internal/models"
	"fmt"
)

const (
	BinSearchEps  = 0.0001
	DerivativeEps = 0.0001
	MaxValue      = 1000000000.0
)

func Calculate(pools []models.Pool, x float64, startAddress string, flag bool) (float64, string) {
	cur := x
	curAddress := startAddress
	var curName string
	if pools[0].TokenA.Metadata.Address == curAddress {
		curName = pools[0].TokenA.Metadata.Name
	} else {
		curName = pools[0].TokenB.Metadata.Name
	}
	for _, p := range pools {
		res, nextToken, err := p.EstimateSwap(curAddress, Slippage, cur)
		if err != nil {
			panic(err)
		}
		if flag {
			poolLink := p.Address
			switch p.DEX {
			case models.DEXNameStonFi:
				poolLink = fmt.Sprintf("https://app.ston.fi/pools/%s", p.Address)
			case models.DEXNameDeDust:
				poolLink = fmt.Sprintf("https://dedust.io/pools/%s", p.Address)
			}
			fmt.Println("Trade:", cur, curName, "--->", res, nextToken.Name, "|", p.DEX, poolLink)
		}
		cur = res
		curName = nextToken.Name
		curAddress = nextToken.Address
	}
	return cur, curAddress
}

func CalculateDerivative(pools []models.Pool, x float64, startAddress string) float64 {
	res0, _ := Calculate(pools, x+DerivativeEps, startAddress, false)
	res1, _ := Calculate(pools, x-DerivativeEps, startAddress, false)
	return (res0 - res1) / (2.0 * DerivativeEps)
}

func FindDerivativePoint(pools []models.Pool, k float64, startAddress string) (float64, error) {
	if k <= 0 {
		return 0, fmt.Errorf("k is less than zero")
	}
	var l, r float64
	l = 0
	r = MaxValue

	for r-l > BinSearchEps {
		m := (l + r) / 2.0
		res := CalculateDerivative(pools, m, startAddress)
		if res >= k {
			l = m
		} else {
			r = m
		}
	}
	return l, nil
}

package analyzer

import (
	"arbitrage/internal/models"
	"fmt"
	"log/slog"
	"math/big"
)

const bigFloatPrec = 256

// Calculate simulates swapping x of startAddress through the given pools sequentially.
// When log is true, it prints each swap step.
func Calculate(pools []models.Pool, x float64, startAddress string, log bool) (float64, string, error) {
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
			return 0, "", fmt.Errorf("estimate swap on pool %s: %w", p.Address, err)
		}
		if log {
			poolLink := p.Address
			switch p.DEX {
			case models.DEXNameStonFi:
				poolLink = fmt.Sprintf("https://app.ston.fi/pools/%s", p.Address)
			case models.DEXNameDeDust:
				poolLink = fmt.Sprintf("https://dedust.io/pools/%s", p.Address)
			}
			slog.Info(fmt.Sprintf("Trade %.4f %s -> %.4f %s", cur, curName, res, nextToken.Name), "dex", p.DEX, "pool", poolLink)
		}
		cur = res
		curName = nextToken.Name
		curAddress = nextToken.Address
	}
	return cur, curAddress, nil
}

// FindOptimalCapital computes the optimal input amount and expected profit for an
// arbitrage cycle using a closed-form solution derived from the constant-product AMM formula.
//
// For a chain of n pools, the output is y(x) = N·x / (D + E·x), where N, D, E are
// computed via recurrence. The optimal input is x* = (√(N·D) - D) / E.
func FindOptimalCapital(pools []models.Pool, startAddress string) (optimalX float64, profit float64, err error) {
	N := new(big.Float).SetPrec(bigFloatPrec).SetFloat64(1)
	D := new(big.Float).SetPrec(bigFloatPrec).SetFloat64(1)
	E := new(big.Float).SetPrec(bigFloatPrec).SetFloat64(0)

	s := new(big.Float).SetPrec(bigFloatPrec).SetFloat64(1.0 - Slippage/100.0)
	curAddress := startAddress

	for _, p := range pools {
		f := new(big.Float).SetPrec(bigFloatPrec).SetFloat64(1.0 - p.TradeFee/100.0)

		var X, Y float64
		var nextAddress string
		if curAddress == p.TokenA.Metadata.Address {
			X = p.TokenA.Reserve
			Y = p.TokenB.Reserve
			nextAddress = p.TokenB.Metadata.Address
		} else if curAddress == p.TokenB.Metadata.Address {
			X = p.TokenB.Reserve
			Y = p.TokenA.Reserve
			nextAddress = p.TokenA.Metadata.Address
		} else {
			return 0, 0, fmt.Errorf("pool %s doesn't contain token %s", p.Address, curAddress)
		}

		bigX := new(big.Float).SetPrec(bigFloatPrec).SetFloat64(X)
		bigY := new(big.Float).SetPrec(bigFloatPrec).SetFloat64(Y)

		// sfY = s * f * Y (numerator multiplier per pool)
		sfY := new(big.Float).SetPrec(bigFloatPrec).Mul(s, f)
		sfY.Mul(sfY, bigY)

		// fN = f * N (used in E recurrence: denominator uses f, not s*f)
		fN := new(big.Float).SetPrec(bigFloatPrec).Mul(f, N)

		// Recurrence: N' = sfY * N, D' = X * D, E' = X * E + f * N
		newN := new(big.Float).SetPrec(bigFloatPrec).Mul(sfY, N)
		newD := new(big.Float).SetPrec(bigFloatPrec).Mul(bigX, D)
		newE := new(big.Float).SetPrec(bigFloatPrec).Mul(bigX, E)
		newE.Add(newE, fN)

		N, D, E = newN, newD, newE
		curAddress = nextAddress
	}

	if curAddress != startAddress {
		return 0, 0, fmt.Errorf("cycle doesn't return to start token: ended at %s", curAddress)
	}

	// Not profitable if N <= D
	if N.Cmp(D) <= 0 {
		return 0, 0, nil
	}

	// x* = (√(N·D) - D) / E
	nd := new(big.Float).SetPrec(bigFloatPrec).Mul(N, D)
	sqrtND := new(big.Float).SetPrec(bigFloatPrec).Sqrt(nd)
	num := new(big.Float).SetPrec(bigFloatPrec).Sub(sqrtND, D)
	xStar := new(big.Float).SetPrec(bigFloatPrec).Quo(num, E)

	// profit = N·x/(D+E·x) - x
	ex := new(big.Float).SetPrec(bigFloatPrec).Mul(E, xStar)
	denom := new(big.Float).SetPrec(bigFloatPrec).Add(D, ex)
	nx := new(big.Float).SetPrec(bigFloatPrec).Mul(N, xStar)
	y := new(big.Float).SetPrec(bigFloatPrec).Quo(nx, denom)
	p := new(big.Float).SetPrec(bigFloatPrec).Sub(y, xStar)

	optimalX, _ = xStar.Float64()
	profit, _ = p.Float64()
	return optimalX, profit, nil
}

package models

import (
	"math"
	"testing"
)

func testPool() Pool {
	return Pool{
		DEX:      "TestDEX",
		Address:  "pool1",
		TradeFee: 0.3, // 0.3%
		TokenA: PoolToken{
			Metadata: TokenMetadata{Name: "TokenA", Symbol: "A", Address: "addrA", Decimals: 9},
			Reserve:  1000.0,
		},
		TokenB: PoolToken{
			Metadata: TokenMetadata{Name: "TokenB", Symbol: "B", Address: "addrB", Decimals: 9},
			Reserve:  2000.0,
		},
	}
}

func TestEstimateSwap_TokenA(t *testing.T) {
	pool := testPool()
	amount := 10.0
	slippage := 1.0

	result, meta, err := pool.EstimateSwap("addrA", slippage, amount)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Address != "addrB" {
		t.Fatalf("expected output token addrB, got %s", meta.Address)
	}

	// Manual AMM calculation:
	// fee = 1 - 0.3/100 = 0.997
	// output = (1 - 1/100) * (2000 * 0.997 * 10) / (1000 + 0.997 * 10)
	fee := 0.997
	expected := (1.0 - slippage/100.0) * (2000.0 * fee * amount) / (1000.0 + fee*amount)
	if math.Abs(result-expected) > 1e-9 {
		t.Fatalf("expected %f, got %f", expected, result)
	}
}

func TestEstimateSwap_TokenB(t *testing.T) {
	pool := testPool()
	amount := 5.0
	slippage := 1.0

	result, meta, err := pool.EstimateSwap("addrB", slippage, amount)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Address != "addrA" {
		t.Fatalf("expected output token addrA, got %s", meta.Address)
	}

	fee := 0.997
	expected := (1.0 - slippage/100.0) * (1000.0 * fee * amount) / (2000.0 + fee*amount)
	if math.Abs(result-expected) > 1e-9 {
		t.Fatalf("expected %f, got %f", expected, result)
	}
}

func TestEstimateSwap_UnknownToken(t *testing.T) {
	pool := testPool()
	_, _, err := pool.EstimateSwap("addrC", 1.0, 10.0)
	if err == nil {
		t.Fatal("expected error for unknown token, got nil")
	}
}

func TestEstimateSwap_ZeroReserve(t *testing.T) {
	pool := testPool()
	pool.TokenB.Reserve = 0

	result, _, err := pool.EstimateSwap("addrA", 1.0, 10.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With zero reserve in output token, result should be 0
	if result != 0 {
		t.Fatalf("expected 0 output with zero reserve, got %f", result)
	}
}

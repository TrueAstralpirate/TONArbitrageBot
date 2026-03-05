package analyzer

import (
	"arbitrage/internal/models"
	"math"
	"testing"
)

func makePool(addrA, addrB, poolAddr string, reserveA, reserveB, fee float64) models.Pool {
	return models.Pool{
		DEX:      "TestDEX",
		Address:  poolAddr,
		TradeFee: fee,
		TokenA: models.PoolToken{
			Metadata: models.TokenMetadata{Name: "A", Symbol: "A", Address: addrA, Decimals: 9},
			Reserve:  reserveA,
		},
		TokenB: models.PoolToken{
			Metadata: models.TokenMetadata{Name: "B", Symbol: "B", Address: addrB, Decimals: 9},
			Reserve:  reserveB,
		},
	}
}

func TestCalculate_SinglePool(t *testing.T) {
	pool := makePool("TON", "USDT", "pool1", 1000, 2000, 0.3)
	result, addr, err := Calculate([]models.Pool{pool}, 10.0, "TON", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "USDT" {
		t.Fatalf("expected output address USDT, got %s", addr)
	}

	// Verify against manual AMM calc with slippage=1.0 (Slippage var)
	fee := 1.0 - 0.3/100.0
	slip := 1.0 - Slippage/100.0
	expected := slip * (2000.0 * fee * 10.0) / (1000.0 + fee*10.0)
	if math.Abs(result-expected) > 1e-6 {
		t.Fatalf("expected %f, got %f", expected, result)
	}
}

func TestCalculate_MultiPool(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1000, 2000, 0.3)
	pool2 := makePool("USDT", "BTC", "pool2", 5000, 100, 0.3)

	result, addr, err := Calculate([]models.Pool{pool1, pool2}, 10.0, "TON", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "BTC" {
		t.Fatalf("expected output address BTC, got %s", addr)
	}
	if result <= 0 {
		t.Fatalf("expected positive result, got %f", result)
	}
}

func TestCalculate_ErrorOnBadToken(t *testing.T) {
	pool := makePool("TON", "USDT", "pool1", 1000, 2000, 0.3)
	_, _, err := Calculate([]models.Pool{pool}, 10.0, "INVALID", false)
	if err == nil {
		t.Fatal("expected error for invalid start token, got nil")
	}
}

func TestCalculateDerivative_Positive(t *testing.T) {
	// Two pools forming a cycle: TON->USDT->TON with favorable rates
	pool1 := makePool("TON", "USDT", "pool1", 1000, 3000, 0.3)
	pool2 := makePool("USDT", "TON", "pool2", 3000, 1100, 0.3)

	deriv, err := CalculateDerivative([]models.Pool{pool1, pool2}, 0.01, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// At very low capital, derivative should be > 1 for a profitable cycle
	if deriv <= 1.0 {
		t.Fatalf("expected derivative > 1 at low capital, got %f", deriv)
	}
}

func TestCalculateDerivative_LessThanOne(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1000, 3000, 0.3)
	pool2 := makePool("USDT", "TON", "pool2", 3000, 1100, 0.3)

	deriv, err := CalculateDerivative([]models.Pool{pool1, pool2}, 500.0, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// At high capital, derivative should be < 1 (past optimal)
	if deriv >= 1.0 {
		t.Fatalf("expected derivative < 1 at high capital, got %f", deriv)
	}
}

func TestFindDerivativePoint_ReturnsOptimal(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1000, 3000, 0.3)
	pool2 := makePool("USDT", "TON", "pool2", 3000, 1100, 0.3)

	optimal, err := FindDerivativePoint([]models.Pool{pool1, pool2}, 1.0, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if optimal <= 0 {
		t.Fatalf("expected positive optimal capital, got %f", optimal)
	}
	if optimal > MaxValue {
		t.Fatalf("optimal capital %f exceeds MaxValue", optimal)
	}
}

func TestFindDerivativePoint_InvalidK(t *testing.T) {
	pool := makePool("TON", "USDT", "pool1", 1000, 2000, 0.3)
	_, err := FindDerivativePoint([]models.Pool{pool}, 0, "TON")
	if err == nil {
		t.Fatal("expected error for k=0, got nil")
	}
	_, err = FindDerivativePoint([]models.Pool{pool}, -1, "TON")
	if err == nil {
		t.Fatal("expected error for k<0, got nil")
	}
}

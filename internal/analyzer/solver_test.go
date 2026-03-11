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

func TestFindOptimalCapital_2Pool(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1000, 3000, 0.3)
	pool2 := makePool("USDT", "TON", "pool2", 3000, 1100, 0.3)

	x, profit, err := FindOptimalCapital([]models.Pool{pool1, pool2}, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if x <= 0 {
		t.Fatalf("expected positive optimal capital, got %f", x)
	}
	if profit <= 0 {
		t.Fatalf("expected positive profit, got %f", profit)
	}

	revenue, _, err := Calculate([]models.Pool{pool1, pool2}, x, "TON", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs((revenue-x)-profit) > 0.01 {
		t.Fatalf("profit mismatch: FindOptimalCapital=%f, Calculate=%f", profit, revenue-x)
	}
}

func TestFindOptimalCapital_3Pool(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1000, 3000, 0.3)
	pool2 := makePool("USDT", "BTC", "pool2", 2000, 500, 0.3)
	pool3 := makePool("BTC", "TON", "pool3", 400, 1200, 0.3)

	x, profit, err := FindOptimalCapital([]models.Pool{pool1, pool2, pool3}, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if x <= 0 || profit <= 0 {
		t.Fatalf("expected profitable 3-pool cycle, got x=%f profit=%f", x, profit)
	}

	revenue, _, err := Calculate([]models.Pool{pool1, pool2, pool3}, x, "TON", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs((revenue-x)-profit) > 0.01 {
		t.Fatalf("profit mismatch: FindOptimalCapital=%f, Calculate=%f", profit, revenue-x)
	}
}

func TestFindOptimalCapital_4Pool(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1000, 3000, 0.3)
	pool2 := makePool("USDT", "BTC", "pool2", 2000, 500, 0.3)
	pool3 := makePool("BTC", "ETH", "pool3", 400, 800, 0.3)
	pool4 := makePool("ETH", "TON", "pool4", 600, 1500, 0.3)

	x, profit, err := FindOptimalCapital([]models.Pool{pool1, pool2, pool3, pool4}, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if x <= 0 || profit <= 0 {
		t.Fatalf("expected profitable 4-pool cycle, got x=%f profit=%f", x, profit)
	}

	revenue, _, err := Calculate([]models.Pool{pool1, pool2, pool3, pool4}, x, "TON", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs((revenue-x)-profit) > 0.01 {
		t.Fatalf("profit mismatch: FindOptimalCapital=%f, Calculate=%f", profit, revenue-x)
	}
}

func TestFindOptimalCapital_IsOptimal(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1000, 3000, 0.3)
	pool2 := makePool("USDT", "TON", "pool2", 3000, 1100, 0.3)

	x, profit, err := FindOptimalCapital([]models.Pool{pool1, pool2}, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	delta := 0.1
	revLess, _, _ := Calculate([]models.Pool{pool1, pool2}, x-delta, "TON", false)
	revMore, _, _ := Calculate([]models.Pool{pool1, pool2}, x+delta, "TON", false)
	profitLess := revLess - (x - delta)
	profitMore := revMore - (x + delta)

	if profitLess > profit+0.001 {
		t.Fatalf("x-delta has higher profit: %f > %f", profitLess, profit)
	}
	if profitMore > profit+0.001 {
		t.Fatalf("x+delta has higher profit: %f > %f", profitMore, profit)
	}
}

func TestFindOptimalCapital_NotProfitable(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1000, 1000, 0.3)
	pool2 := makePool("USDT", "TON", "pool2", 1000, 1000, 0.3)

	x, profit, err := FindOptimalCapital([]models.Pool{pool1, pool2}, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if x != 0 || profit != 0 {
		t.Fatalf("expected zero for unprofitable cycle, got x=%f profit=%f", x, profit)
	}
}

func TestFindOptimalCapital_InvalidToken(t *testing.T) {
	pool := makePool("TON", "USDT", "pool1", 1000, 2000, 0.3)
	pool2 := makePool("USDT", "TON", "pool2", 2000, 1000, 0.3)
	_, _, err := FindOptimalCapital([]models.Pool{pool, pool2}, "INVALID")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestFindOptimalCapital_LargeReserves(t *testing.T) {
	pool1 := makePool("TON", "USDT", "pool1", 1e9, 3e9, 0.3)
	pool2 := makePool("USDT", "BTC", "pool2", 2e9, 5e8, 0.3)
	pool3 := makePool("BTC", "ETH", "pool3", 4e8, 8e8, 0.3)
	pool4 := makePool("ETH", "TON", "pool4", 6e8, 1.5e9, 0.3)

	x, profit, err := FindOptimalCapital([]models.Pool{pool1, pool2, pool3, pool4}, "TON")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if x <= 0 || profit <= 0 {
		t.Fatalf("expected profitable cycle with large reserves, got x=%f profit=%f", x, profit)
	}

	revenue, _, err := Calculate([]models.Pool{pool1, pool2, pool3, pool4}, x, "TON", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs((revenue-x)-profit) > 0.01 {
		t.Fatalf("profit mismatch with large reserves: FindOptimalCapital=%f, Calculate=%f", profit, revenue-x)
	}
}

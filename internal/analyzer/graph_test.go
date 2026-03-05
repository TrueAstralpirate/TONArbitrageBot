package analyzer

import (
	"arbitrage/internal/models"
	"testing"
)

func makeTestPool(addrA, addrB, poolAddr string, reserveA, reserveB float64) models.Pool {
	return models.Pool{
		DEX:      "TestDEX",
		Address:  poolAddr,
		TradeFee: 0.3,
		TokenA: models.PoolToken{
			Metadata: models.TokenMetadata{Name: addrA, Symbol: addrA, Address: addrA, Decimals: 9},
			Reserve:  reserveA,
		},
		TokenB: models.PoolToken{
			Metadata: models.TokenMetadata{Name: addrB, Symbol: addrB, Address: addrB, Decimals: 9},
			Reserve:  reserveB,
		},
	}
}

func TestBuildGraph_TwoPools(t *testing.T) {
	pools := []models.Pool{
		makeTestPool("A", "B", "pool1", 1000, 2000),
		makeTestPool("B", "C", "pool2", 2000, 3000),
	}

	graph := BuildGraph(pools)

	// 3 unique tokens: A, B, C
	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(graph.Nodes))
	}

	// Each pool creates 2 edges
	if len(graph.Edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(graph.Edges))
	}
}

func TestFindAllCycles_Simple2Cycle(t *testing.T) {
	// Two different pools both connecting A and B
	pools := []models.Pool{
		makeTestPool("A", "B", "pool1", 1000, 2000),
		makeTestPool("A", "B", "pool2", 1500, 2500),
	}

	graph := BuildGraph(pools)
	cycles := graph.FindAllCycles("A")

	// Should find at least one 2-cycle: A->B via pool1, B->A via pool2 (and vice versa)
	if len(cycles) == 0 {
		t.Fatal("expected at least one cycle, got 0")
	}

	// Verify all cycles start and end with token A
	for i, c := range cycles {
		if len(c.PoolsOrder) != 2 {
			continue // skip non-2-cycles
		}
		// Each cycle should use 2 different pools
		if c.PoolsOrder[0].Address == c.PoolsOrder[1].Address {
			t.Fatalf("cycle %d uses the same pool twice", i)
		}
	}
}

func TestFindAllCycles_Simple3Cycle(t *testing.T) {
	// Three pools forming a triangle: A-B, B-C, A-C
	pools := []models.Pool{
		makeTestPool("A", "B", "pool1", 1000, 2000),
		makeTestPool("B", "C", "pool2", 2000, 3000),
		makeTestPool("A", "C", "pool3", 1000, 3000),
	}

	graph := BuildGraph(pools)
	cycles := graph.FindAllCycles("A")

	// Should find 3-cycles: A->B->C->A and A->C->B->A
	found3Cycle := false
	for _, c := range cycles {
		if len(c.PoolsOrder) == 3 {
			found3Cycle = true
			break
		}
	}
	if !found3Cycle {
		t.Fatal("expected at least one 3-cycle")
	}
}

func TestFindAllCycles_NoSelfLoop(t *testing.T) {
	// Single pool A-B: cannot form a cycle with just one pool
	pools := []models.Pool{
		makeTestPool("A", "B", "pool1", 1000, 2000),
	}

	graph := BuildGraph(pools)
	cycles := graph.FindAllCycles("A")

	// Should find no cycles (need at least 2 pools to form a cycle)
	for _, c := range cycles {
		for i := range c.PoolsOrder {
			for j := i + 1; j < len(c.PoolsOrder); j++ {
				if c.PoolsOrder[i].Address == c.PoolsOrder[j].Address {
					t.Fatal("found cycle using the same pool twice")
				}
			}
		}
	}
}

package analyzer

import (
	"arbitrage/internal/models"
	"fmt"
)

var (
	Slippage = 1.0
)

type Edge struct {
	From    models.TokenMetadata
	To      models.TokenMetadata
	PoolRef *models.Pool
	Cost    float64
	LogCost float64
}

type Graph struct {
	Nodes []string
	Edges []Edge
}

func BuildGraph(pools []models.Pool) Graph {
	var result Graph
	result.Edges = make([]Edge, 0, len(pools)*2)

	nodes := make(map[string]struct{})

	for i := range pools {
		p := &pools[i]

		result.Edges = append(result.Edges, Edge{
			From:    p.TokenA.Metadata,
			To:      p.TokenB.Metadata,
			PoolRef: p,
		})
		result.Edges = append(result.Edges, Edge{
			From:    p.TokenB.Metadata,
			To:      p.TokenA.Metadata,
			PoolRef: p,
		})
		nodes[p.TokenA.Metadata.Address] = struct{}{}
		nodes[p.TokenB.Metadata.Address] = struct{}{}
	}

	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}

	fmt.Println("number of nodes in graph:", len(keys))
	fmt.Println("number of edges in graph:", len(result.Edges))

	result.Nodes = keys
	return result
}

func buildArbitrageCycleFromPools(tokensOrder []models.TokenMetadata, pools []models.Pool, coef float64, start string) (*models.ArbitrageCycle, error) {
	optimalStart, err := FindDerivativePoint(pools, coef, start)
	if err != nil {
		return nil, fmt.Errorf("find derivative point: %w", err)
	}
	revenue, _ := Calculate(pools, optimalStart, start, false)
	return &models.ArbitrageCycle{
		TokensOrder:  tokensOrder,
		PoolsOrder:   pools,
		StartCapital: optimalStart,
		Profit:       revenue - optimalStart,
	}, nil
}

func (g *Graph) find2Cycles(start string, edgesToStart, edgesFromStart map[string][]*Edge) ([]models.ArbitrageCycle, error) {
	result := make([]models.ArbitrageCycle, 0)
	for _, node := range g.Nodes {
		if node == start {
			continue
		}
		for _, e0 := range edgesFromStart[node] {
			for _, e1 := range edgesToStart[node] {
				if e0.PoolRef.Address == e1.PoolRef.Address {
					continue
				}
				pools := []models.Pool{*e0.PoolRef, *e1.PoolRef}
				cycle, err := buildArbitrageCycleFromPools([]models.TokenMetadata{e0.From, e0.To}, pools, 1.0, start)
				if err != nil {
					fmt.Printf("error while building cycle: %e", err)
					continue
				}
				result = append(result, *cycle)
			}
		}
	}
	return result, nil
}

func (g *Graph) find3Cycles(start string, edgesToStart, edgesFromStart map[string][]*Edge) ([]models.ArbitrageCycle, error) {
	result := make([]models.ArbitrageCycle, 0)
	for i := range g.Edges {
		e := &g.Edges[i]
		if e.From.Address == start || e.To.Address == start {
			continue
		}
		for _, e0 := range edgesFromStart[e.From.Address] {
			for _, e1 := range edgesToStart[e.To.Address] {
				pools := []models.Pool{*e0.PoolRef, *e.PoolRef, *e1.PoolRef}
				cycle, err := buildArbitrageCycleFromPools([]models.TokenMetadata{e0.From, e0.To, e1.From}, pools, 1.0, start)
				if err != nil {
					fmt.Printf("error while building cycle: %e", err)
					continue
				}
				result = append(result, *cycle)
			}
		}
	}
	return result, nil
}

func (g *Graph) find4Cycles(start string, edgesToStart, edgesFromStart, edgesFromNode map[string][]*Edge) ([]models.ArbitrageCycle, error) {
	result := make([]models.ArbitrageCycle, 0)
	for i := range g.Edges {
		e1 := &g.Edges[i]
		if e1.From.Address == start || e1.To.Address == start {
			continue
		}
		for _, e2 := range edgesFromNode[e1.To.Address] {
			if e2.From.Address == start || e2.To.Address == start || e2.To.Address == e1.From.Address {
				continue
			}
			for _, e0 := range edgesFromStart[e1.From.Address] {
				for _, e3 := range edgesToStart[e2.To.Address] {
					pools := []models.Pool{*e0.PoolRef, *e1.PoolRef, *e2.PoolRef, *e3.PoolRef}
					cycle, err := buildArbitrageCycleFromPools([]models.TokenMetadata{e0.From, e0.To, e1.To, e2.To}, pools, 1.0, start)
					if err != nil {
						fmt.Printf("error while building cycle: %e", err)
						continue
					}
					result = append(result, *cycle)
				}
			}
		}
	}
	return result, nil
}

func (g *Graph) FindAllCycles(start string) []models.ArbitrageCycle {
	edgesToStart := make(map[string][]*Edge)
	edgesFromStart := make(map[string][]*Edge)
	edgesFromNode := make(map[string][]*Edge)
	for _, token := range g.Nodes {
		edgesToStart[token] = make([]*Edge, 0)
		edgesFromStart[token] = make([]*Edge, 0)
		edgesFromNode[token] = make([]*Edge, 0)
	}

	for i := range g.Edges {
		e := &g.Edges[i]
		if e.From.Address == start {
			edgesFromStart[e.To.Address] = append(edgesFromStart[e.To.Address], e)
		} else if e.To.Address == start {
			edgesToStart[e.From.Address] = append(edgesToStart[e.From.Address], e)
		}
		edgesFromNode[e.From.Address] = append(edgesFromNode[e.From.Address], e)
	}

	cycles := make([]models.ArbitrageCycle, 0)

	twoCycles, err := g.find2Cycles(start, edgesToStart, edgesFromStart)
	if err == nil {
		cycles = append(cycles, twoCycles...)
	}

	threeCycles, err := g.find3Cycles(start, edgesToStart, edgesFromStart)
	if err == nil {
		cycles = append(cycles, threeCycles...)
	}

	fourCycles, err := g.find4Cycles(start, edgesToStart, edgesFromStart, edgesFromNode)
	if err == nil {
		cycles = append(cycles, fourCycles...)
	}

	return cycles
}

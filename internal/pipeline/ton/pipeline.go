package tonpipeline

import (
	"arbitrage/internal/analyzer"
	tonexecutor "arbitrage/internal/execute/ton"
	"arbitrage/internal/explorer/ton/aggregator"
	"arbitrage/internal/explorer/ton/chain"
	"arbitrage/internal/models"
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

type PipelineSettings struct {
	SortCyclesByProfit bool
}

type Pipeline struct {
	Client *chain.TonClient
	Wallet *wallet.Wallet

	DedustPoolsFilePath string
	TONConfig           string
	UseDeDust           bool
	UseStonFi           bool
	UseCoffee           bool
	OnlyShowCycles      bool

	StartTokens     []string
	MinStartCapital []float64
	MaxStartCapital []float64
	StepFees        []float64
}

// DoForStartToken tries to find and execute the most profitable cycle for the given start token.
// Returns true if a profitable cycle was found (and executed, unless OnlyShowCycles is set).
func (p *Pipeline) DoForStartToken(ctx context.Context, cycles []models.ArbitrageCycle, startToken string, minStartCapital, maxStartCapital, stepFee float64) (bool, error) {
	sort.Slice(cycles, func(i, j int) bool {
		return cycles[i].Profit-stepFee*float64(len(cycles[i].PoolsOrder)) > cycles[j].Profit-stepFee*float64(len(cycles[j].PoolsOrder))
	})
	for _, c := range cycles {
		if c.StartCapital >= minStartCapital && c.StartCapital <= maxStartCapital && c.Profit-stepFee*float64(len(c.PoolsOrder)) > 0 {
			slog.Info("Executing cycle")
			_, _, err := analyzer.Calculate(c.PoolsOrder, c.StartCapital, startToken, true)
			if err != nil {
				slog.Error("failed to calculate cycle", "error", err)
				continue
			}

			if p.OnlyShowCycles {
				return true, nil
			}

			ce := tonexecutor.CycleExecutor{
				TONClient: p.Client,
				Wallet:    p.Wallet,
			}

			err = ce.ExecuteCycle(ctx, c)
			if err != nil {
				return false, fmt.Errorf("build messages from arbitrage cycle: %w", err)
			}
			return true, nil
		}
	}

	return false, nil
}

func (p *Pipeline) Do(ctx context.Context) error {
	for {
		slog.Info("Fetching pools")
		pools, err := aggregator.FetchPools(ctx, p.Client, aggregator.AggregatorSettings{
			UseDeDust:           p.UseDeDust,
			UseStonFi:           p.UseStonFi,
			UseCoffee:           p.UseCoffee,
			DedustPoolsFilePath: p.DedustPoolsFilePath,
		})
		slog.Info("Pools fetched")
		if err != nil {
			slog.Error("aggregator couldn't fetch pools", "error", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		graph := analyzer.BuildGraph(pools)

		for i, startToken := range p.StartTokens {
			cycles := graph.FindAllCycles(startToken)
			found, err := p.DoForStartToken(ctx, cycles, startToken, p.MinStartCapital[i], p.MaxStartCapital[i], p.StepFees[i])
			if err != nil {
				slog.Error("failed to execute cycle", "token", startToken, "error", err)
			}
			if found {
				break
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

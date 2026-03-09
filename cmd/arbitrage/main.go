package main

import (
	"arbitrage/internal/explorer/ton/chain"
	tonpipeline "arbitrage/internal/pipeline/ton"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	tonAddress  = "TON"
	usdtAddress = "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"
)

func main() {
	seedFile := flag.String("seed-file", "", "Path to wallet seed words file (required)")
	tonConfig := flag.String("ton-config", "https://ton-blockchain.github.io/global.config.json", "TON network config URL")
	dedustPoolsFile := flag.String("dedust-pools-file", "dedust_pool_assets_cache.json", "Path to DeDust pool-to-token-address cache file")
	useDedust := flag.Bool("use-dedust", true, "Enable DeDust DEX")
	useStonfi := flag.Bool("use-stonfi", true, "Enable StonFi DEX")
	onlyShowCycles := flag.Bool("only-show-cycles", false, "Only display cycles without executing")
	minTonCapital := flag.Float64("min-ton-capital", 0.1, "Minimum start capital for TON cycles")
	maxTonCapital := flag.Float64("max-ton-capital", 120, "Maximum start capital for TON cycles")
	stepTonFees := flag.Float64("step-ton-fees", 0.06, "Fee per swap step for TON cycles (in TON)")
	minUsdCapital := flag.Float64("min-usd-capital", 0.2, "Minimum start capital for USD cycles")
	maxUsdCapital := flag.Float64("max-usd-capital", 200, "Maximum start capital for USD cycles")
	stepUsdFees := flag.Float64("step-usd-fees", 0.1, "Fee per swap step for USD cycles")

	flag.Parse()

	if *seedFile == "" {
		fmt.Fprintln(os.Stderr, "error: --seed-file is required")
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	var err error
	var client *chain.TonClient
	for {
		client, err = chain.NewTonClient(ctx, *tonConfig)
		if err != nil {
			continue
		}
		break
	}

	data, err := os.ReadFile(*seedFile)
	if err != nil {
		slog.Error("failed to read seed file", "path", *seedFile, "error", err)
		os.Exit(1)
	}
	words := strings.Fields(strings.TrimSpace(string(data)))

	w, err := wallet.FromSeedWithOptions(client.Api, words, wallet.ConfigV5R1Final{
		NetworkGlobalID: wallet.MainnetGlobalID,
	})
	if err != nil {
		slog.Error("failed to get wallet from seed", "error", err)
		os.Exit(1)
	}

	pipeline := tonpipeline.Pipeline{
		Client:              client,
		Wallet:              w,
		DedustPoolsFilePath: *dedustPoolsFile,
		TONConfig:           *tonConfig,
		UseDeDust:      *useDedust,
		UseStonFi:      *useStonfi,
		StartTokens:         []string{tonAddress, usdtAddress},
		MinStartCapital:     []float64{*minTonCapital, *minUsdCapital},
		MaxStartCapital:     []float64{*maxTonCapital, *maxUsdCapital},
		StepFees:            []float64{*stepTonFees, *stepUsdFees},
		OnlyShowCycles:      *onlyShowCycles,
	}

	err = pipeline.Do(ctx)
	if err != nil {
		slog.Error("pipeline failed", "error", err)
		os.Exit(1)
	}
}

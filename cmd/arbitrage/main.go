package main

import (
	"arbitrage/internal/explorer/ton/chain"
	tonpipeline "arbitrage/internal/pipeline/ton"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

func parseFloats(s string) ([]float64, error) {
	parts := strings.Split(s, ",")
	result := make([]float64, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", p, err)
		}
		result = append(result, v)
	}
	return result, nil
}

func main() {
	seedFile := flag.String("seed-file", "", "Path to wallet seed words file (required)")
	tonConfig := flag.String("ton-config", "https://ton-blockchain.github.io/global.config.json", "TON network config URL")
	dedustPoolsFile := flag.String("dedust-pools-file", "dedust_pools.json", "Path to DeDust pools blocklist file")
	stonfiPoolsFile := flag.String("stonfi-pools-file", "stonfi_pools.json", "Path to StonFi pools blocklist file")
	useDedust := flag.Bool("use-dedust", true, "Enable DeDust DEX")
	useStonfi := flag.Bool("use-stonfi", true, "Enable StonFi DEX")
	useCoffee := flag.Bool("use-coffee", false, "Enable Coffee DEX")
	onlyShowCycles := flag.Bool("only-show-cycles", false, "Only display cycles without executing")
	startTokensStr := flag.String("start-tokens", "TON,EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs", "Comma-separated start token addresses")
	minCapitalStr := flag.String("min-capital", "0.1,0.2", "Comma-separated min start capital per token")
	maxCapitalStr := flag.String("max-capital", "120,200", "Comma-separated max start capital per token")
	stepFeesStr := flag.String("step-fees", "0.06,0.18", "Comma-separated step fees per token")

	flag.Parse()

	if *seedFile == "" {
		fmt.Fprintln(os.Stderr, "error: --seed-file is required")
		flag.Usage()
		os.Exit(1)
	}

	startTokens := strings.Split(*startTokensStr, ",")
	minCapital, err := parseFloats(*minCapitalStr)
	if err != nil {
		slog.Error("invalid --min-capital", "error", err)
		os.Exit(1)
	}
	maxCapital, err := parseFloats(*maxCapitalStr)
	if err != nil {
		slog.Error("invalid --max-capital", "error", err)
		os.Exit(1)
	}
	stepFees, err := parseFloats(*stepFeesStr)
	if err != nil {
		slog.Error("invalid --step-fees", "error", err)
		os.Exit(1)
	}

	if len(startTokens) != len(minCapital) || len(startTokens) != len(maxCapital) || len(startTokens) != len(stepFees) {
		slog.Error("--start-tokens, --min-capital, --max-capital, and --step-fees must have the same number of comma-separated values",
			"start-tokens", len(startTokens), "min-capital", len(minCapital), "max-capital", len(maxCapital), "step-fees", len(stepFees))
		os.Exit(1)
	}

	ctx := context.Background()

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
		StonfiPoolsFilePath: *stonfiPoolsFile,
		TONConfig:           *tonConfig,
		UseDeDust:           *useDedust,
		UseStonFi:           *useStonfi,
		UseCoffee:           *useCoffee,
		StartTokens:         startTokens,
		MinStartCapital:     minCapital,
		MaxStartCapital:     maxCapital,
		StepFees:            stepFees,
		OnlyShowCycles:      *onlyShowCycles,
	}

	err = pipeline.Do(ctx)
	if err != nil {
		slog.Error("pipeline failed", "error", err)
		os.Exit(1)
	}
}

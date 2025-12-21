package main

import (
	"arbitrage/internal/explorer/ton/chain"
	tonpipeline "arbitrage/internal/pipeline/ton"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	kDedustPoolsFilePath = "dedust_pools.json"
	kStonfiPoolsFilePath = "stonfi_pools.json"
	kWordsFilePath       = "words.txt"

	kTONMainNetConfig = "https://ton-blockchain.github.io/global.config.json"
	kTON              = "TON"
	kUSDT             = "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"
)

func main() {
	ctx := context.Background()

	var client *chain.TonClient
	var err error
	for {
		client, err = chain.NewTonClient(ctx, kTONMainNetConfig)
		if err != nil {
			continue
		}
		break
	}

	data, err := os.ReadFile(kWordsFilePath)
	if err != nil {
		panic(err)
	}
	words := strings.Split(string(data), "\n")

	w, err := wallet.FromSeedWithOptions(client.Api, words, wallet.ConfigV5R1Final{
		NetworkGlobalID: wallet.MainnetGlobalID,
	})
	if err != nil {
		panic(fmt.Errorf("get wallet from seed: %w", err))
	}

	pipeline := tonpipeline.Pipeline{
		Client:              client,
		Wallet:              w,
		DedustPoolsFilePath: kDedustPoolsFilePath,
		StonfiPoolsFilePath: kStonfiPoolsFilePath,
		TONConfig:           kTONMainNetConfig,
		UseDeDust:           true,
		UseStonFi:           true,
		UseCoffee:           false,
		StartTokens:         []string{kTON, kUSDT},
		MinStartCapital:     []float64{0.1, 0.2},
		MaxStartCapital:     []float64{15, 30},
		StepFees:            []float64{0.06, 0.18},
		OnlyShowCycles:      false,
	}

	err = pipeline.Do(ctx)
	if err != nil {
		panic(err)
	}
}

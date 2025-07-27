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

	kTONMainNetConfig = "https://ton.org/global.config.json"
	kTON              = "TON"
	kUSDT             = "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"
	kStart            = kTON
)

func main() {
	ctx := context.Background()

	var client *chain.TonClient
	var err error
	for {
		client, err = chain.NewTonClient(ctx, kTONMainNetConfig)
		if err != nil {
			continue
		} else {
			break
		}
	}

	data, err := os.ReadFile(kWordsFilePath)
	if err != nil {
		panic(err)
	}
	words := strings.Split(string(data), "\n")

	w, err := wallet.FromSeed(client.Api, words, wallet.ConfigV5R1Final{
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
		StartTokens:         []string{kTON, kUSDT},
		MaxStartCapital:     []float64{15, 30},
		StepFees:            []float64{0.07, 0.2},
	}

	err = pipeline.Do(ctx)
	if err != nil {
		panic(err)
	}
}

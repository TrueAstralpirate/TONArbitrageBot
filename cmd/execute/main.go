package main

import (
	tonexecutor "arbitrage/internal/execute/ton"
	"arbitrage/internal/explorer/ton/chain"
	"arbitrage/internal/models"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	kWordsFilePath = "words.txt"

	kTONMainNetConfig = "https://ton.org/global.config.json"
	kTON              = "TON"
)

func main() {
	ctx := context.Background()

	client, err := chain.NewTonClient(ctx, kTONMainNetConfig)
	if err != nil {
		panic(err)
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

	ce := tonexecutor.CycleExecutor{
		TONClient: client,
		Wallet:    w,
	}
	msg, err := ce.BuildStonFiSwapMessage(ctx, models.TokenMetadata{
		Address:  "EQAGSPUCAd6Yix92WQoxG5ibFHlR6x9i9EB-4T-EHu_STARS",
		Decimals: 9,
	}, models.TokenMetadata{
		Address:  "TON",
		Decimals: 9,
	}, models.Pool{
		DEX:     "StonFi",
		Address: "EQCxmvCOT7P6LshGpLxiNhXIp4wUjkKHytOOCGXAR_UzdGRg",
		TokenA: models.PoolToken{
			Metadata: models.TokenMetadata{
				Address:  "EQAGSPUCAd6Yix92WQoxG5ibFHlR6x9i9EB-4T-EHu_STARS",
				Decimals: 9,
			},
		},
	}, 12000, 0)
	if err != nil {
		panic(err)
	}
	fmt.Println(msg)
	tx, _, err := ce.Wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		panic(fmt.Errorf("send transaction: %w", err))
	}
	fmt.Println("transaction confirmed, hash:", base64.StdEncoding.EncodeToString(tx.Hash))
}

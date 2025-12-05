package main

import (
	coffeeapi "arbitrage/internal/explorer/ton/coffee"
	"arbitrage/internal/models"
	"fmt"
)

func main() {
	// Example usage of Coffee DEX connector
	fmt.Println("Fetching Coffee DEX pools...")

	// Import the coffee API package
	pools, err := coffeeapi.FetchPools()
	if err != nil {
		fmt.Printf("Error fetching Coffee DEX pools: %v\n", err)
		return
	}

	fmt.Printf("Found %d Coffee DEX pools\n", len(pools))

	// Display first few pools as examples
	for i, pool := range pools {
		if i >= 5 { // Show only first 5 pools
			break
		}

		fmt.Printf("\nPool %d:\n", i+1)
		fmt.Printf("  DEX: %s\n", pool.DEX)
		fmt.Printf("  Address: %s\n", pool.Address)
		fmt.Printf("  Type: %s\n", pool.Type)
		fmt.Printf("  Trade Fee: %.4f%%\n", pool.TradeFee)
		fmt.Printf("  Token A: %s (%s) - Reserve: %.6f\n",
			pool.TokenA.Metadata.Symbol,
			pool.TokenA.Metadata.Address,
			pool.TokenA.Reserve)
		fmt.Printf("  Token B: %s (%s) - Reserve: %.6f\n",
			pool.TokenB.Metadata.Symbol,
			pool.TokenB.Metadata.Address,
			pool.TokenB.Reserve)
	}

	// Filter for TON pairs
	var tonPairs []models.Pool
	for _, pool := range pools {
		if pool.TokenA.Metadata.Address == "TON" || pool.TokenB.Metadata.Address == "TON" {
			tonPairs = append(tonPairs, pool)
		}
	}

	fmt.Printf("\nFound %d TON pairs on Coffee DEX\n", len(tonPairs))
}

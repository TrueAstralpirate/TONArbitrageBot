# TON Arbitrage Bot

Finds and executes arbitrage cycles across DeDust and StonFi DEX on TON.

## Quick Start

```bash
git clone https://github.com/TrueAstralpirate/TONArbitrageBot.git
cd TONArbitrageBot
echo "word1 word2 ... word24" > words.txt
go run ./cmd/arbitrage --seed-file words.txt
```

## Requirements

- Go 1.24+
- A TON wallet seed phrase stored in a text file (words separated by spaces or newlines)

## Running

```bash
go run ./cmd/arbitrage --seed-file words.txt
```

To only display cycles without executing trades:

```bash
go run ./cmd/arbitrage --seed-file words.txt --only-show-cycles
```

## CLI Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--seed-file` | string | *(required)* | Path to wallet seed words file |
| `--ton-config` | string | `https://ton-blockchain.github.io/global.config.json` | TON network config URL |
| `--dedust-pools-file` | string | `dedust_pool_assets_cache.json` | Path to DeDust pool-to-token-address cache file |
| `--use-dedust` | bool | `true` | Enable DeDust DEX |
| `--use-stonfi` | bool | `true` | Enable StonFi DEX |
| `--only-show-cycles` | bool | `false` | Display cycles without executing trades |
| `--min-ton-capital` | float | `0.1` | Minimum start capital for TON cycles |
| `--max-ton-capital` | float | `120` | Maximum start capital for TON cycles |
| `--step-ton-fees` | float | `0.06` | Fee per swap step for TON cycles (in TON) |
| `--min-usd-capital` | float | `0.2` | Minimum start capital for USD cycles |
| `--max-usd-capital` | float | `200` | Maximum start capital for USD cycles |
| `--step-usd-fees` | float | `0.1` | Fee per swap step for USD cycles |

## Example Trades

#### Triangle Arbitrage (3-hop, cross-DEX)
```
Trade 0.1303 Toncoin -> 26.1917 STORM   dex=DeDust pool=https://dedust.io/pools/EQAm_QHFNFg5SM0i7tc2Jl9W1xVl0aA7e3lFvRAGixF8T4ig
Trade 26.1917 STORM -> 17959859.0355 Mir   dex=StonFi pool=https://app.ston.fi/pools/EQCBpwug8nrEAW8W-dk4d5IdEoA8OijKR5ZNhUCpNxCibQq5
Trade 17959859.0355 Mir -> 0.2577 TON   dex=StonFi pool=https://app.ston.fi/pools/EQAG4KuT5Fyv8-UlxnSUTQn_tM61teKKcfYoGFbb75SNAiXW
```

#### Quadrangle Arbitrage (4-hop, cross-DEX)
```
Trade 0.6583 Toncoin -> 0.5985 Tonstakers TON   dex=DeDust pool=https://dedust.io/pools/EQDsPeOuu66e7tWagXO6Tn8VFZMEV0ClQb7zOB6SEW_VKN3k
Trade 0.5985 Tonstakers TON -> 27884871.8142 Mir   dex=StonFi pool=https://app.ston.fi/pools/EQDgJ2RTKqLnI9fPoA2dCpnUkYbbvJ5Q9Au42dig9Ul-ZvBb
Trade 27884871.8142 Mir -> 1.2097 Tether USD   dex=StonFi pool=https://app.ston.fi/pools/EQAM2y3_PNqm0sU_shgS6f-TyVV-Ogj2ZwHj6RXEY1MWuhrP
Trade 1.2097 Tether USD -> 0.8996 Toncoin   dex=DeDust pool=https://dedust.io/pools/EQA-X_yo3fzzbDbJ_0bzFWKqtRuZFIRa1sJsveZJ1YpViO3r
```

## DeDust Pool Assets Cache

The file `dedust_pool_assets_cache.json` is a local cache that maps DeDust pool addresses to their token contract addresses. The DeDust API does not return token addresses directly — they must be queried from the TON blockchain. To avoid slow on-chain lookups on every startup, resolved addresses are stored in this cache and reused on subsequent runs.

The cache is updated automatically: when the bot encounters a pool not yet in the cache, it fetches the token addresses from the blockchain and writes them back to the file.

If the file does not exist or is empty, the bot will populate it from scratch on first run (slower).

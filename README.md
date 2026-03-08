# TON Arbitrage Bot

Finds and executes arbitrage cycles across DeDust, StonFi, and Coffee DEX on TON.

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
| `--use-coffee` | bool | `false` | Enable Coffee DEX |
| `--only-show-cycles` | bool | `false` | Display cycles without executing trades |
| `--min-ton-capital` | float | `0.1` | Minimum start capital for TON cycles |
| `--max-ton-capital` | float | `120` | Maximum start capital for TON cycles |
| `--step-ton-fees` | float | `0.06` | Fee per swap step for TON cycles (in TON) |
| `--min-usd-capital` | float | `0.2` | Minimum start capital for USD cycles |
| `--max-usd-capital` | float | `200` | Maximum start capital for USD cycles |
| `--step-usd-fees` | float | `0.18` | Fee per swap step for USD cycles |

## Example Trades

#### Triangle Arbitrage (3-hop, cross-DEX)
```
Trade 0.1774 TON -> 124972.0175 BLUE TRACTOR COIN   dex=StonFi pool=https://app.ston.fi/pools/EQA9kOZMqkvQGgS_sl_0VbleY6XAuC71wlfBTDGeSD-UNuxg
Trade 124972.0175 BLUE TRACTOR COIN -> 0.3950 Tether USD   dex=StonFi pool=https://app.ston.fi/pools/EQCWhEnyFdtNT16Jt-YDIoRVB9L93fjpE-I7Xprmwd6JdgIO
Trade 0.3950 Tether USD -> 0.2985 Toncoin   dex=DeDust pool=https://dedust.io/pools/EQA-X_yo3fzzbDbJ_0bzFWKqtRuZFIRa1sJsveZJ1YpViO3r
```

## DeDust Pool Assets Cache

The file `dedust_pool_assets_cache.json` is a local cache that maps DeDust pool addresses to their token contract addresses. The DeDust API does not return token addresses directly — they must be queried from the TON blockchain. To avoid slow on-chain lookups on every startup, resolved addresses are stored in this cache and reused on subsequent runs.

The cache is updated automatically: when the bot encounters a pool not yet in the cache, it fetches the token addresses from the blockchain and writes them back to the file.

If the file does not exist or is empty, the bot will populate it from scratch on first run (slower).

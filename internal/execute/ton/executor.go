package tonexecutor

import (
	"arbitrage/internal/explorer/ton/chain"
	"arbitrage/internal/models"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const (
	kTON               = "TON"
	kProxyTONV1        = "EQARULUYsmJq1RiZ-YiH-IJLcAZUVkVff-KBPwEmmaQGH6aC"
	kDeDustNativeVault = "EQDa4VOnTYlLvDJ0gZjNYm5PXfSmmtL6Vs6A_CZEtXCNICq_"
	// transactionConfirmationTimeout defines how long we wait for TON to confirm a tx before giving up.
	transactionConfirmationTimeout = 10 * time.Minute
)

var errBalanceDidNotChange = errors.New("balance did not change after timeout")

type CycleExecutor struct {
	TONClient *chain.TonClient
	Wallet    *wallet.Wallet
}

type TransferPayload struct {
	_                   tlb.Magic        `tlb:"#0f8a7ea5"`
	QueryID             uint64           `tlb:"## 64"`
	Amount              tlb.Coins        `tlb:"."`
	Destination         *address.Address `tlb:"addr"`
	ResponseDestination *address.Address `tlb:"addr"`
	CustomPayload       *cell.Cell       `tlb:"maybe ^"`
	ForwardTONAmount    tlb.Coins        `tlb:"."`
	ForwardPayload      *cell.Cell       `tlb:"either . ^"`
}

func (ce *CycleExecutor) GetJettonBalance(ctx context.Context, tokenAddress string) (*big.Int, error) {
	token := jetton.NewJettonMasterClient(ce.TONClient.Api, address.MustParseAddr(tokenAddress))

	tokenWallet, err := token.GetJettonWallet(ctx, ce.Wallet.WalletAddress())
	if err != nil {
		return nil, fmt.Errorf("get jetton wallet: %w", err)
	}

	tokenBalance, err := tokenWallet.GetBalance(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "contract is not initialized") {
			return big.NewInt(0), nil
		}
		return nil, fmt.Errorf("get jetton balance: %w", err)
	}
	return tokenBalance, nil
}

func (ce *CycleExecutor) GetTONBalance(ctx context.Context) (*tlb.Coins, error) {
	b, err := ce.TONClient.Api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	balance, err := ce.Wallet.GetBalance(ctx, b)
	return &balance, err
}

func (ce *CycleExecutor) GetTokenBalance(ctx context.Context, token models.TokenMetadata) (float64, error) {
	if token.Address == kTON {
		balance, err := ce.GetTONBalance(ctx)
		if err != nil {
			return 0, fmt.Errorf("get ton balance: %w", err)
		}
		return strconv.ParseFloat(balance.String(), 64)
	} else {
		balance, err := ce.GetJettonBalance(ctx, token.Address)
		if err != nil {
			return 0, fmt.Errorf("get jetton balance: %w", err)
		}
		value, err := strconv.ParseFloat(balance.String(), 64)
		if err != nil {
			return 0, fmt.Errorf("parse float: %w", err)
		}
		return value / math.Pow(float64(10.0), float64(token.Decimals)), nil
	}
}

func buildTransferPayloadV2(to, responseTo *address.Address, amountCoins, amountForwardTON tlb.Coins, payloadForward, customPayload *cell.Cell) (*cell.Cell, error) {
	if payloadForward == nil {
		payloadForward = cell.BeginCell().EndCell()
	}

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	rnd := binary.LittleEndian.Uint64(buf)

	body, err := tlb.ToCell(TransferPayload{
		QueryID:             rnd,
		Amount:              amountCoins,
		Destination:         to,
		ResponseDestination: responseTo,
		CustomPayload:       customPayload,
		ForwardTONAmount:    amountForwardTON,
		ForwardPayload:      payloadForward,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert TransferPayload to cell: %w", err)
	}

	return body, nil
}

func (ce *CycleExecutor) BuildStonFiSwapV1Message(ctx context.Context, token1RouterAddress, routerAddress *address.Address, amountCoins, amountForwardTON, minOut tlb.Coins) (*cell.Cell, error) {
	swapV1Message := StonFiSwapV1Message{
		TokenWalletAddr: token1RouterAddress,
		MinOut:          minOut,
		ToWalletAddr:    ce.Wallet.WalletAddress(),
		RefAddr:         nil,
	}
	payload, err := tlb.ToCell(swapV1Message)
	if err != nil {
		return nil, fmt.Errorf("to cell: %w", err)
	}

	return buildTransferPayloadV2(routerAddress, ce.Wallet.WalletAddress(), amountCoins, amountForwardTON, payload, nil)
}

func (ce *CycleExecutor) BuildStonFiSwapV2Payload(ctx context.Context, token1RouterAddress, routerAddress *address.Address, amountCoins, minOut tlb.Coins) (*cell.Cell, error) {
	crossSwapBody := CrossSwap{
		MinOut:        minOut,
		ReceiverAddr:  ce.Wallet.WalletAddress(),
		FwdGas:        tlb.MustFromTON("0.0"),
		CustomPayload: nil,
		RefundFwdGas:  tlb.MustFromTON("0.0"),
		RefundPayload: nil,
		RefFee:        0,
		RefAddr:       nil,
	}
	stonFiSwapV2Message := StonFiSwapV2Message{
		TokenWalletAddr: token1RouterAddress,
		RefundAddr:      ce.Wallet.WalletAddress(),
		ExcessesAddr:    ce.Wallet.WalletAddress(),
		Deadline:        uint64(time.Now().Add(24 * time.Hour).Unix()),
		CrossSwapBody:   &crossSwapBody,
	}
	return tlb.ToCell(stonFiSwapV2Message)
}

func (ce *CycleExecutor) BuildStonFiSwapMessage(ctx context.Context, tokenToSwap, tokenToGet models.TokenMetadata, pool models.Pool, amountToSwap, minOut float64) (*wallet.Message, error) {
	coinsAmountToSwap := tlb.MustFromDecimal(fmt.Sprintf("%.9f", amountToSwap), tokenToSwap.Decimals)
	coinsMinOut := tlb.MustFromDecimal(fmt.Sprintf("%.9f", minOut), tokenToGet.Decimals)

	poolInfo, err := ce.TONClient.GetStonFiPoolData(ctx, pool.Address)
	if err != nil {
		return nil, fmt.Errorf("get stonfi pool data: %w", err)
	}

	var currentTokenRouterWallet, nextTokenRouterWallet string

	slog.Debug("stonfi swap details",
		"tokenToSwap", tokenToSwap,
		"tokenA", pool.TokenA.Metadata,
		"tokenB", pool.TokenB.Metadata,
		"poolInfo", poolInfo)

	if tokenToSwap.Address == pool.TokenA.Metadata.Address {
		currentTokenRouterWallet = poolInfo.Token0WalletAddress
		nextTokenRouterWallet = poolInfo.Token1WalletAddress
	} else {
		currentTokenRouterWallet = poolInfo.Token1WalletAddress
		nextTokenRouterWallet = poolInfo.Token0WalletAddress
	}

	if poolInfo.Version == 1 {
		forwardTON1 := tlb.MustFromTON("0.24")
		forwardTON2 := tlb.MustFromTON("0.18")
		if tokenToSwap.Address == kTON {
			forwardTON1 = tlb.MustFromTON(fmt.Sprintf("%.9f", 0.24+amountToSwap))
			forwardTON2 = tlb.MustFromTON("0.24")
		}

		stonFiMessage, err := ce.BuildStonFiSwapV1Message(ctx, address.MustParseAddr(nextTokenRouterWallet), address.MustParseAddr(poolInfo.RouterAddress), coinsAmountToSwap, forwardTON2, coinsMinOut)
		if err != nil {
			return nil, fmt.Errorf("build ston fi swap v1 message: %w", err)
		}
		if tokenToSwap.Address == kTON {
			return ce.BuildTransferTONMessage(ctx, kProxyTONV1, forwardTON1, stonFiMessage)
		} else {
			return ce.BuildTransferJettonMessage(ctx, tokenToSwap.Address, forwardTON1, stonFiMessage)
		}
	} else {
		forwardTON1 := tlb.MustFromTON("0.3")
		forwardTON2 := tlb.MustFromTON("0.24")
		if tokenToSwap.Address == kTON {
			forwardTON1 = tlb.MustFromTON(fmt.Sprintf("%.9f", 0.3+amountToSwap))
		}

		payload, err := ce.BuildStonFiSwapV2Payload(ctx, address.MustParseAddr(nextTokenRouterWallet), address.MustParseAddr(poolInfo.RouterAddress), coinsAmountToSwap, coinsMinOut)
		if err != nil {
			return nil, fmt.Errorf("build ston fi swap v2 message: %w", err)
		}
		if tokenToSwap.Address == kTON {
			transferMessage := PTonTONTransferMessage{
				QueryID:        0,
				TonAmount:      coinsAmountToSwap,
				RefundAddr:     ce.Wallet.WalletAddress(),
				ForwardPayload: payload,
			}
			transferPayload, err := tlb.ToCell(transferMessage)
			if err != nil {
				return nil, fmt.Errorf("to cell: %w", err)
			}
			return ce.BuildTransferTONMessage(ctx, currentTokenRouterWallet, forwardTON1, transferPayload)
		} else {
			transferPayload, err := buildTransferPayloadV2(address.MustParseAddr(poolInfo.RouterAddress), ce.Wallet.WalletAddress(), coinsAmountToSwap, forwardTON2, payload, nil)
			if err != nil {
				return nil, fmt.Errorf("build transfer payload v2")
			}
			return ce.BuildTransferJettonMessage(ctx, tokenToSwap.Address, forwardTON1, transferPayload)
		}
	}
}

func (ce *CycleExecutor) BuildTransferTONMessage(ctx context.Context, proxyTONAddress string, fee tlb.Coins, payload *cell.Cell) (*wallet.Message, error) {
	return wallet.SimpleMessage(address.MustParseAddr(proxyTONAddress), fee, payload), nil
}

func (ce *CycleExecutor) BuildTransferJettonMessage(ctx context.Context, jetton0Address string, fee tlb.Coins, payload *cell.Cell) (*wallet.Message, error) {
	token0 := jetton.NewJettonMasterClient(ce.TONClient.Api, address.MustParseAddr(jetton0Address))
	token0Wallet, err := token0.GetJettonWallet(ctx, ce.Wallet.WalletAddress())
	if err != nil {
		return nil, fmt.Errorf("get jetton0 wallet: %w", err)
	}

	return wallet.SimpleMessage(token0Wallet.Address(), fee, payload), nil
}

func (ce *CycleExecutor) BuildDeDustSwapNativeMessage(ctx context.Context, poolAddress *address.Address, amountCoins, minOut tlb.Coins) (*cell.Cell, error) {
	nativeSwapMessage := DeDustNativeSwapMessage{
		QueryID: 0,
		Amount:  amountCoins,
		Step: SwapStep{
			PoolAddr: poolAddress,
			Params: SwapStepParams{
				Kind:   false,
				MinOut: minOut,
				Next:   nil,
			},
		},
		Params: &SwapParams{
			Deadline:        uint32(time.Now().Add(24 * time.Hour).Unix()),
			RecipientAddr:   ce.Wallet.WalletAddress(),
			ReferralAddr:    nil,
			FullfillPayload: nil,
			RejectPayload:   nil,
		},
	}
	return tlb.ToCell(nativeSwapMessage)
}

func (ce *CycleExecutor) BuildDeDustSwapJettonMessage(ctx context.Context, token0VaultAddress, poolAddress *address.Address, amountCoins, amountForwardTON, minOut tlb.Coins) (*cell.Cell, error) {
	jettonSwapMessage := DeDustJettonSwapMessage{
		Step: SwapStep{
			PoolAddr: poolAddress,
			Params: SwapStepParams{
				Kind:   false,
				MinOut: minOut,
				Next:   nil,
			},
		},
		Params: &SwapParams{
			Deadline:        uint32(time.Now().Add(24 * time.Hour).Unix()),
			RecipientAddr:   ce.Wallet.WalletAddress(),
			ReferralAddr:    nil,
			FullfillPayload: nil,
			RejectPayload:   nil,
		},
	}
	payload, err := tlb.ToCell(jettonSwapMessage)
	if err != nil {
		return nil, fmt.Errorf("build cell from swap message: %w", err)
	}

	transferPayload, err := buildTransferPayloadV2(token0VaultAddress, ce.Wallet.WalletAddress(), amountCoins, amountForwardTON, payload, nil)
	if err != nil {
		return nil, fmt.Errorf("build transfer payload v2: %w", err)
	}

	return transferPayload, nil
}

func (ce *CycleExecutor) BuildDeDustSwapMessage(ctx context.Context, tokenToSwap, tokenToGet models.TokenMetadata, pool models.Pool, amountToSwap, minOut float64) (*wallet.Message, error) {
	coinsAmountToSwap := tlb.MustFromDecimal(fmt.Sprintf("%.9f", amountToSwap), tokenToSwap.Decimals)
	coinsMinOut := tlb.MustFromDecimal(fmt.Sprintf("%.9f", minOut), tokenToGet.Decimals)

	forwardTON1 := tlb.MustFromTON("0.2")
	forwardTON2 := tlb.MustFromTON("0.15")

	if tokenToSwap.Address == kTON {
		forwardTON1 = tlb.MustFromTON(fmt.Sprintf("%.9f", 0.15+amountToSwap))
		payload, err := ce.BuildDeDustSwapNativeMessage(ctx, address.MustParseAddr(pool.Address), coinsAmountToSwap, coinsMinOut)
		if err != nil {
			return nil, fmt.Errorf("build dedust swap native message: %w", err)
		}
		return ce.BuildTransferTONMessage(ctx, kDeDustNativeVault, forwardTON1, payload)
	} else {
		vaultInfo, err := ce.TONClient.GetDedustVaultAddress(ctx, tokenToSwap.Address)
		if err != nil {
			return nil, fmt.Errorf("get dedust vault address: %w", err)
		}
		payload, err := ce.BuildDeDustSwapJettonMessage(ctx, vaultInfo, address.MustParseAddr(pool.Address), coinsAmountToSwap, forwardTON2, coinsMinOut)
		if err != nil {
			return nil, fmt.Errorf("build dedust swap jeeton message: %w", err)
		}
		return ce.BuildTransferJettonMessage(ctx, tokenToSwap.Address, forwardTON1, payload)
	}
}

func (ce *CycleExecutor) BuildMessageFromCycleStep(ctx context.Context, tokenToSwap, tokenToGet models.TokenMetadata, pool models.Pool, amountToSwap, minOut float64) (*wallet.Message, error) {
	if pool.DEX == models.DEXNameDeDust {
		return ce.BuildDeDustSwapMessage(ctx, tokenToSwap, tokenToGet, pool, amountToSwap, minOut)
	} else {
		return ce.BuildStonFiSwapMessage(ctx, tokenToSwap, tokenToGet, pool, amountToSwap, minOut)
	}
}

type executedStep struct {
	tokenFrom models.TokenMetadata
	tokenTo   models.TokenMetadata
	pool      models.Pool
	amountIn  float64
	amountOut float64
}

func (ce *CycleExecutor) sendWithSeqnoRetry(ctx context.Context, msg *wallet.Message) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 1 * time.Second
	b.MaxInterval = 10 * time.Second
	b.MaxElapsedTime = 60 * time.Second

	attempt := 0
	return backoff.Retry(func() error {
		err := ce.Wallet.Send(ctx, msg)
		if err == nil {
			return nil
		}
		attempt++
		slog.Warn("send failed, retrying", "attempt", attempt, "error", err)
		return err
	}, backoff.WithContext(b, ctx))
}

func (ce *CycleExecutor) waitForBalanceChange(ctx context.Context, token models.TokenMetadata, balanceBefore float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		balanceAfter, err := ce.GetTokenBalance(ctx, token)
		slog.Debug("polling balance", "balance", balanceAfter, "token", token.Symbol)
		time.Sleep(5 * time.Second)
		if time.Now().After(deadline) {
			return fmt.Errorf("%w for token %s", errBalanceDidNotChange, token.Symbol)
		}
		if err != nil {
			continue
		}
		if balanceAfter != balanceBefore {
			return nil
		}
	}
}

func (ce *CycleExecutor) rollbackCycle(ctx context.Context, steps []executedStep) error {
	if len(steps) == 0 {
		return nil
	}

	amountToSwap := steps[len(steps)-1].amountOut
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		tokenToSwap := step.tokenTo
		tokenToGet := step.tokenFrom

		res, nextToken, err := step.pool.EstimateSwap(tokenToSwap.Address, 1.0, amountToSwap)
		if err != nil {
			return fmt.Errorf("rollback estimate swap: %w", err)
		}
		if nextToken.Address != tokenToGet.Address {
			return fmt.Errorf("rollback unexpected token: got %s want %s", nextToken.Symbol, tokenToGet.Symbol)
		}

		balanceBefore, err := ce.GetTokenBalance(ctx, tokenToGet)
		if err != nil {
			return fmt.Errorf("rollback get balance before transaction: %w", err)
		}

		slog.Info("rollback swapping", "amount", amountToSwap, "from", tokenToSwap.Symbol, "to_amount", res, "to", tokenToGet.Symbol)
		msg, err := ce.BuildMessageFromCycleStep(ctx, tokenToSwap, tokenToGet, step.pool, amountToSwap, res)
		if err != nil {
			return fmt.Errorf("rollback build message from cycle step: %w", err)
		}
		err = ce.sendWithSeqnoRetry(ctx, msg)
		if err != nil {
			return fmt.Errorf("rollback send transaction: %w", err)
		}
		slog.Info("rollback transaction confirmed")

		err = ce.waitForBalanceChange(ctx, tokenToGet, balanceBefore, 2*time.Minute)
		if err != nil {
			return fmt.Errorf("rollback wait for balance change: %w", err)
		}

		amountToSwap = res
	}

	return nil
}

func (ce *CycleExecutor) ExecuteCycle(ctx context.Context, cycle models.ArbitrageCycle) error {
	amountToSwap := cycle.StartCapital
	executedSteps := make([]executedStep, 0, len(cycle.PoolsOrder))

	for i := range cycle.PoolsOrder {
		token := cycle.TokensOrder[i]
		p := cycle.PoolsOrder[i]

		res, nextToken, err := p.EstimateSwap(token.Address, 1.0, amountToSwap)
		if err != nil {
			return fmt.Errorf("estimate swap: %w", err)
		}

		balanceBeforeTransaction, err := ce.GetTokenBalance(ctx, *nextToken)
		if err != nil {
			return fmt.Errorf("get balance before transaction: %w", err)
		}

		slog.Info("Processing pool", "step", i)
		msg, err := ce.BuildMessageFromCycleStep(ctx, token, *nextToken, p, amountToSwap, res)
		if err != nil {
			return fmt.Errorf("build message from cycle step: %w", err)
		}

		slog.Info("swapping", "amount", amountToSwap, "from", token.Symbol, "to_amount", res, "to", nextToken.Symbol)
		err = ce.sendWithSeqnoRetry(ctx, msg)
		if err != nil {
			return fmt.Errorf("send transaction: %w", err)
		}
		slog.Info("transaction confirmed")
		err = ce.waitForBalanceChange(ctx, *nextToken, balanceBeforeTransaction, 2*time.Minute)
		if err != nil {
			if errors.Is(err, errBalanceDidNotChange) {
				rollbackErr := ce.rollbackCycle(ctx, executedSteps)
				if rollbackErr != nil {
					return fmt.Errorf("balance did not change; rollback failed: %w", rollbackErr)
				}
			}
			return fmt.Errorf("wait for balance change: %w", err)
		}

		executedSteps = append(executedSteps, executedStep{
			tokenFrom: token,
			tokenTo:   *nextToken,
			pool:      p,
			amountIn:  amountToSwap,
			amountOut: res,
		})

		amountToSwap = res
	}
	return nil
}

package chain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const (
	kAssetVariantNative        = "native"
	kAssetVariantJetton        = "jetton"
	kAssetVariantExtraCurrency = "extra_currency"
	kTON                       = "TON"
	kDedustVaultAddress        = "EQBfBWT7X2BHg9tXAxzhz2aKiNTU1tpt5NsiK0uSDW_YAJ67"
	kStonfiRouterV1            = "EQB3ncyBUTjZUA5EnFKR5_EnOMI9V1tTEAAPaiU71gc4TiUt"
)

type Asset struct {
	Variant       string
	Jetton        *JettonAsset        // present if Variant == "jetton"
	ExtraCurrency *ExtraCurrencyAsset // present if Variant == "extra_currency"
}

type JettonAsset struct {
	Workchain int8
	Address   *big.Int // 256-bit address
}

type ExtraCurrencyAsset struct {
	CurrencyID int32
}

func BuildSliceFromAsset(asset *Asset) (*cell.Slice, error) {
	builder := cell.BeginCell()

	switch asset.Variant {
	case kAssetVariantNative:
		builder.StoreUInt(0b00, 4)
	case kAssetVariantJetton:
		builder.StoreUInt(0b01, 4)
		builder.StoreInt(int64(asset.Jetton.Workchain), 8)
		builder.StoreBigUInt(asset.Jetton.Address, 256)
	default:
		return nil, fmt.Errorf("unknown asset variant: %s", asset.Variant)
	}

	return builder.ToSlice(), nil
}

func BuildAssetFromSlice(slice *cell.Slice) (*Asset, error) {
	tag, err := slice.LoadUInt(4)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset tag: %w", err)
	}

	var result Asset

	switch tag {
	case 0b00:
		result.Variant = kAssetVariantNative

	case 0b01:
		result.Variant = kAssetVariantJetton

		workchain, err := slice.LoadInt(8)
		if err != nil {
			return nil, fmt.Errorf("failed to load workchain_id: %w", err)
		}
		addrBits, err := slice.LoadBigUInt(256)
		if err != nil {
			return nil, fmt.Errorf("failed to load address: %w", err)
		}
		result.Jetton = &JettonAsset{
			Workchain: int8(workchain),
			Address:   addrBits,
		}

	case 0b10:
		result.Variant = kAssetVariantExtraCurrency

		currencyID, err := slice.LoadInt(32)
		if err != nil {
			return nil, fmt.Errorf("failed to load currency_id: %w", err)
		}
		id := int32(currencyID)
		result.ExtraCurrency = &ExtraCurrencyAsset{
			CurrencyID: id,
		}

	default:
		return nil, fmt.Errorf("unknown asset tag: %02b", tag)
	}

	return &result, nil
}

func BuildJettonAddress(a JettonAsset) *address.Address {
	return address.NewAddress(0, byte(a.Workchain), a.Address.Bytes())
}

func BuildAddress(a Asset) string {
	if a.Variant == kAssetVariantNative {
		return "TON"
	} else if a.Variant == kAssetVariantJetton {
		return BuildJettonAddress(*a.Jetton).String()
	}
	return ""
}

type TonClient struct {
	Api ton.APIClientWrapped
}

func NewTonClient(ctx context.Context, configUrl string) (*TonClient, error) {
	client := liteclient.NewConnectionPool()

	err := client.AddConnectionsFromConfigUrl(ctx, configUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to add connections from config url: %w", err)
	}

	api := ton.NewAPIClient(client).WithRetry() // Enables automatic retries with failover to another node

	return &TonClient{
		Api: api,
	}, nil
}

type GetDedustAssetsResult struct {
	Asset0 string
	Asset1 string
}

func (tc *TonClient) GetDedustAssets(ctx context.Context, addr string) (*GetDedustAssetsResult, error) {
	block, err := tc.Api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the last block: %w", err)
	}

	chainAddr := address.MustParseAddr(addr)

	res, err := tc.Api.RunGetMethod(ctx, block, chainAddr, "get_assets")
	if err != nil {
		return nil, fmt.Errorf("failed to call get_assets: %w", err)
	}

	asset0, err := BuildAssetFromSlice(res.MustSlice(0))
	if err != nil {
		return nil, fmt.Errorf("failed to parse asset0: %w", err)
	}
	asset1, err := BuildAssetFromSlice(res.MustSlice(1))
	if err != nil {
		return nil, fmt.Errorf("failed to parse asset1: %w", err)
	}

	return &GetDedustAssetsResult{
		Asset0: BuildAddress(*asset0),
		Asset1: BuildAddress(*asset1),
	}, nil
}

func (tc *TonClient) GetDedustVaultAddress(ctx context.Context, addr string) (*address.Address, error) {
	block, err := tc.Api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the last block: %w", err)
	}

	asset := Asset{}
	if addr == kTON {
		asset.Variant = kAssetVariantNative
	} else {
		asset.Variant = kAssetVariantJetton
		tlbAddress := address.MustParseAddr(addr)
		asset.Jetton = &JettonAsset{
			Workchain: int8(tlbAddress.Workchain()),
			Address:   new(big.Int).SetBytes(tlbAddress.Data()),
		}
	}

	slice, err := BuildSliceFromAsset(&asset)
	if err != nil {
		return nil, fmt.Errorf("failed to build slice from asset: %w", err)
	}
	res, err := tc.Api.RunGetMethod(ctx, block, address.MustParseAddr(kDedustVaultAddress), "get_vault_address", slice)
	if err != nil {
		return nil, fmt.Errorf("failed to call get_assets: %w", err)
	}
	vaultAddr, err := res.MustSlice(0).LoadAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to load address: %w", err)
	}
	return vaultAddr, nil
}

type GetDedustReservesResult struct {
	Reserve0 int
	Reserve1 int
}

func (tc *TonClient) GetDedustReserves(ctx context.Context, addr string) (*GetDedustReservesResult, error) {
	block, err := tc.Api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the last block: %w", err)
	}

	chainAddr := address.MustParseAddr(addr)

	res, err := tc.Api.RunGetMethod(ctx, block, chainAddr, "get_reserves")
	if err != nil {
		return nil, fmt.Errorf("failed to call get_assets: %w", err)
	}

	return &GetDedustReservesResult{
		Reserve0: int(res.MustInt(0).Int64()),
		Reserve1: int(res.MustInt(1).Int64()),
	}, nil
}

type GetStonfiPoolData struct {
	Version             int
	Weight0             float64
	Reserve0            int
	Reserve1            int
	RouterAddress       string
	Token0WalletAddress string
	Token1WalletAddress string
}

func (tc *TonClient) GetStonFiV1PoolData(ctx context.Context, addr string) (*GetStonfiPoolData, error) {
	block, err := tc.Api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the last block: %w", err)
	}

	chainAddr := address.MustParseAddr(addr)

	res, err := tc.Api.RunGetMethod(ctx, block, chainAddr, "get_pool_data")
	if err != nil {
		return nil, fmt.Errorf("failed to call get_pool_data: %w", err)
	}
	_, err = res.Int(1)
	if err != nil {
		return nil, fmt.Errorf("failed to parse int(0): %w", err)
	}
	return &GetStonfiPoolData{
		Version:             1,
		Reserve0:            int(res.MustInt(0).Int64()),
		Reserve1:            int(res.MustInt(1).Int64()),
		RouterAddress:       kStonfiRouterV1,
		Token0WalletAddress: res.MustSlice(2).MustLoadAddr().String(),
		Token1WalletAddress: res.MustSlice(3).MustLoadAddr().String(),
	}, nil
}

func (tc *TonClient) GetStonFiV2PoolData(ctx context.Context, addr string) (*GetStonfiPoolData, error) {
	block, err := tc.Api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the last block: %w", err)
	}

	chainAddr := address.MustParseAddr(addr)

	res, err := tc.Api.RunGetMethod(ctx, block, chainAddr, "get_pool_data")
	if err != nil {
		return nil, fmt.Errorf("failed to call get_pool_data: %w", err)
	}
	return &GetStonfiPoolData{
		Version:             2,
		Reserve0:            int(res.MustInt(3).Int64()),
		Reserve1:            int(res.MustInt(4).Int64()),
		RouterAddress:       res.MustSlice(1).MustLoadAddr().String(),
		Token0WalletAddress: res.MustSlice(5).MustLoadAddr().String(),
		Token1WalletAddress: res.MustSlice(6).MustLoadAddr().String(),
	}, nil
}

func (tc *TonClient) GetStonFiPoolData(ctx context.Context, addr string) (*GetStonfiPoolData, error) {
	res, err := tc.GetStonFiV1PoolData(ctx, addr)
	if err == nil {
		return res, nil
	}

	res, err = tc.GetStonFiV2PoolData(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("unknown pool: %w", err)
	}
	return res, nil
}

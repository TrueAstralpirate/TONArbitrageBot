package tonexecutor

import (
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type SwapParams struct {
	Deadline        uint32           `tlb:"## 32"`
	RecipientAddr   *address.Address `tlb:"addr"`
	ReferralAddr    *address.Address `tlb:"addr"`
	FullfillPayload *cell.Cell       `tlb:"maybe ^"`
	RejectPayload   *cell.Cell       `tlb:"maybe ^"`
}

type SwapStep struct {
	PoolAddr *address.Address `tlb:"addr"`
	Params   SwapStepParams   `tlb:"."`
}

type SwapStepParams struct {
	Kind   bool      `tlb:"bool"`
	MinOut tlb.Coins `tlb:"."`
	Next   *SwapStep `tlb:"maybe ^"`
}

type DeDustNativeSwapMessage struct {
	_ tlb.Magic `tlb:"#ea06185d"`

	QueryID uint64      `tlb:"## 64"`
	Amount  tlb.Coins   `tlb:"."`
	Step    SwapStep    `tlb:"."`
	Params  *SwapParams `tlb:"^"`
}

type DeDustJettonSwapMessage struct {
	_      tlb.Magic   `tlb:"#e3a0d482"`
	Step   SwapStep    `tlb:"."`
	Params *SwapParams `tlb:"^"`
}

type StonFiSwapV1Message struct {
	_ tlb.Magic `tlb:"#25938561"`

	TokenWalletAddr *address.Address `tlb:"addr"`
	MinOut          tlb.Coins        `tlb:"."`
	ToWalletAddr    *address.Address `tlb:"addr"`
	RefAddr         *address.Address `tlb:"maybe addr"`
}

type CrossSwap struct {
	MinOut        tlb.Coins        `tlb:"."`
	ReceiverAddr  *address.Address `tlb:"addr"`
	FwdGas        tlb.Coins        `tlb:"."`
	CustomPayload *cell.Cell       `tlb:"maybe ^"`
	RefundFwdGas  tlb.Coins        `tlb:"."`
	RefundPayload *cell.Cell       `tlb:"maybe ^"`
	RefFee        uint16           `tlb:"## 16"`
	RefAddr       *address.Address `tlb:"addr"`
}

type StonFiSwapV2Message struct {
	_ tlb.Magic `tlb:"#6664de2a"`

	TokenWalletAddr *address.Address `tlb:"addr"`
	RefundAddr      *address.Address `tlb:"addr"`
	ExcessesAddr    *address.Address `tlb:"addr"`
	Deadline        uint64           `tlb:"## 64"`
	CrossSwapBody   *CrossSwap       `tlb:"^"`
}

type PTonTONTransferMessage struct {
	_ tlb.Magic `tlb:"#01f3835d"`

	QueryID        uint64           `tlb:"## 64"`
	TonAmount      tlb.Coins        `tlb:"."`
	RefundAddr     *address.Address `tlb:"addr"`
	ForwardPayload *cell.Cell       `tlb:"either . ^"`
}

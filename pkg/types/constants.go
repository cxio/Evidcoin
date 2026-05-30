package types

import "time"

// 全协议核心常量集中登记（第 03 章）。禁止在上层散落魔法数字。

const (
	// BlockInterval is the fixed block production interval.
	BlockInterval = 6 * time.Minute
	// BlocksPerYear is the number of blocks in one sidereal year; year
	// boundaries are computed against this value.
	BlocksPerYear = 87661
)

// 链标识符（第 03 章 §1）。Genesis-ID 等创世具体值未冻结（C-9），不在此伪造。
const (
	// ProtocolID distinguishes this chain from others (blockchain.md).
	ProtocolID = "Evidcoin@v1"
)

// 共识与端点约定相关常量（第 03 章 §3）。
const (
	// RedundantBroadcastInterval is the ordered redundant publish gap.
	RedundantBroadcastInterval = 15 * time.Second
	// FirstBlockDelay is the extra delay before publishing the first block.
	FirstBlockDelay = 30 * time.Second
	// MintWindowStart is the earliest block-height offset for a mint proof tx.
	MintWindowStart = -80000
	// MintWindowEnd is the latest block-height offset (anti-forgery boundary).
	MintWindowEnd = -240
	// ExpiryWindow is the number of blocks after which an uncollected tx expires.
	ExpiryWindow = 240
	// StakeEvalOffset is the block offset whose total stake feeds the mint hash.
	StakeEvalOffset = -32
	// ForkEvalLength is the fork growth length at which evaluation ends.
	ForkEvalLength = 31
	// ForkDecisiveThreshold is the majority share that triggers an early switch.
	ForkDecisiveThreshold = 16
	// ForkAcceptLimit is the maximum acceptable length of a freshly seen fork.
	ForkAcceptLimit = 20
	// MinFeeWindow is the block window used to derive the minimum fee average.
	MinFeeWindow = 6000
)

// 脚本与交易限额常量（第 06、10 章承载语义，此处集中登记数值）。
const (
	// MaxStackHeight is the maximum script stack height.
	MaxStackHeight = 255
	// MaxStackItem is the maximum byte length of a single stack item.
	MaxStackItem = 4095
	// MaxLockScript is the maximum lock script byte length.
	MaxLockScript = 8191
	// MaxUnlockScript is the maximum unlock script byte length.
	MaxUnlockScript = 8191
	// MaxTxSize is the maximum transaction byte length.
	MaxTxSize = 65535
)

// 货币单位（C-8 裁决，第 01 章承载口径）。chx 为最小承载单位。
const (
	// ChxPerBi is the number of chx (smallest unit) in one Bi (display coin):
	// 1 Bi = 10^8 chx.
	ChxPerBi = 100_000_000
)

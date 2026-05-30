package types

import "time"

// 全协议核心常量集中登记（第 03 章）。禁止在上层散落魔法数字。

const (
	// BlockInterval 是固定出块间隔。
	BlockInterval = 6 * time.Minute
	// BlocksPerYear 是一个恒星年对应的区块数量；
	// 年边界计算以该值为准。
	BlocksPerYear = 87661
)

// 链标识符（第 03 章 §1）。Genesis-ID 等创世具体值未冻结（C-9），不在此伪造。
const (
	// ProtocolID 用于区分本链与其他链（blockchain.md）。
	ProtocolID = "Evidcoin@v1"
)

// 共识与端点约定相关常量（第 03 章 §3）。
const (
	// RedundantBroadcastInterval 是有序冗余发布间隔。
	RedundantBroadcastInterval = 15 * time.Second
	// FirstBlockDelay 是发布首个区块前的额外延迟。
	FirstBlockDelay = 30 * time.Second
	// MintWindowStart 是铸凭证明交易允许的最早区块高度偏移。
	MintWindowStart = -80000
	// MintWindowEnd 是铸凭证明交易允许的最晚区块高度偏移（防伪边界）。
	MintWindowEnd = -240
	// ExpiryWindow 是未收录交易过期所需经过的区块数。
	ExpiryWindow = 240
	// StakeEvalOffset 是用于铸凭哈希总质押评估的区块偏移。
	StakeEvalOffset = -32
	// ForkEvalLength 是分叉增长达到后结束评估的长度阈值。
	ForkEvalLength = 31
	// ForkDecisiveThreshold 是触发提前切换的多数占比阈值。
	ForkDecisiveThreshold = 16
	// ForkAcceptLimit 是新观察到分叉可接受的最大长度。
	ForkAcceptLimit = 20
	// MinFeeWindow 是推导最低手续费均值使用的区块窗口。
	MinFeeWindow = 6000
)

// 脚本与交易限额常量（第 06、10 章承载语义，此处集中登记数值）。
const (
	// MaxStackHeight 是脚本栈最大高度。
	MaxStackHeight = 255
	// MaxStackItem 是单个栈项允许的最大字节长度。
	MaxStackItem = 4095
	// MaxLockScript 是锁定脚本最大字节长度。
	MaxLockScript = 8191
	// MaxUnlockScript 是解锁脚本最大字节长度。
	MaxUnlockScript = 8191
	// MaxTxSize 是交易最大字节长度。
	MaxTxSize = 65535
)

// 货币单位（C-8 裁决，第 01 章承载口径）。chx 为最小承载单位。
const (
	// ChxPerBi 是 1 Bi（展示单位）对应的 chx（最小单位）数量：
	// 1 Bi = 10^8 chx。
	ChxPerBi = 100_000_000
)

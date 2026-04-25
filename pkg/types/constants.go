// Package types 定义 Evidcoin 系统的核心类型与全局常量。
package types

import "time"

// 哈希长度常量
const (
	// Hash384Len SHA3-384 / SHA3-384 哈希字节长度（区块 ID、交易 ID、CheckRoot 等）
	Hash384Len = 48
	// Hash512Len SHA3-512 / BLAKE3-512 哈希字节长度（铸凭哈希）
	Hash512Len = 64
	// Hash256Len BLAKE3-256 / SHA3-256 哈希字节长度（UTXO/UTCO 树内部节点）
	Hash256Len = 32
	// PubKeyHashLen 公钥哈希字节长度：SHA3-256(BLAKE2b-512)
	PubKeyHashLen = 32
)

// 区块链参数
const (
	// BlockInterval 固定出块间隔（6 分钟）
	BlockInterval = 6 * time.Minute
	// BlocksPerYear 每年区块数（基于恒星年 365.25636 天）
	BlocksPerYear = 87661
	// GenesisHeight 创世区块高度
	GenesisHeight = 0
)

// 交易尺寸限制
const (
	// MaxTxSize 单笔交易最大字节数
	MaxTxSize = 65535
	// MaxLockScript 锁定脚本最大字节数
	MaxLockScript = 1024
	// MaxUnlockScript 解锁脚本最大字节数
	MaxUnlockScript = 4096
	// MaxMemo 币金附言最大字节数
	MaxMemo = 255
	// MaxTitle 凭信/存证标题最大字节数
	MaxTitle = 255
	// MaxCredDesc 凭信描述最大字节数（低 10 位编码长度）
	MaxCredDesc = 1023
	// MaxAttestContent 存证内容最大字节数（低 12 位编码长度）
	MaxAttestContent = 4095
	// MaxSelfData Coinbase 自由数据最大字节数
	MaxSelfData = 255
)

// 脚本执行限制
const (
	// MaxStackHeight 脚本执行栈最大深度
	MaxStackHeight = 256
	// MaxStackItem 栈项最大字节数
	MaxStackItem = 1024
)

// 择优池参数
const (
	// BestPoolCapacity 择优池容量（最多 20 名候选者）
	BestPoolCapacity = 20
	// ForkWindowSize 分叉竞争窗口（29 个区块）
	ForkWindowSize = 29
)

// 交易过期
const (
	// TxExpiryBlocks 未确认交易过期区块数（240 个区块，约 24 小时）
	TxExpiryBlocks = 240
	// FeeRecalcPeriod 最低手续费重算周期（6000 个区块，约 25 天）
	FeeRecalcPeriod = 6000
)

// 凭信生命周期约束
const (
	// MaxCreditExpiryBlocks 凭信最大有效期（100 年对应区块数）
	MaxCreditExpiryBlocks = 100 * BlocksPerYear
	// CreditActivityBlocks 无期限凭信最长不活跃区块数（11 年）
	CreditActivityBlocks = 11 * BlocksPerYear
)

// 信元类型
const (
	// OutTypeCoin 输出类型：币金
	OutTypeCoin = byte(1)
	// OutTypeCredit 输出类型：凭信
	OutTypeCredit = byte(2)
	// OutTypeProof 输出类型：存证
	OutTypeProof = byte(3)
	// OutTypeMediator 输出类型：介管脚本
	OutTypeMediator = byte(4)
)

// MintHashMix 铸币哈希混合常数
const MintHashMix = uint64(0x517cc1b727220a95)

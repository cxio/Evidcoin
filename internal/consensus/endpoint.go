package consensus

import "github.com/cxio/evidcoin/pkg/types"

// 端点约定（第 12 章 §8，proposal 12 §8）。
//
// 协议（Protocol）：必须严格验证，违背即非法；有最终合法性验证手段。
// 共约（Convention）：靠自觉遵守，无需也无法验证；不得作为拒绝合法区块的依据。
//
// 本文件实现：
//   - 交易过期 240 块检测（协议）
//   - 区块不收录未来交易检测（协议）
//   - 最低交易费阈值计算（共约：仅供本地策略参考）
//   - 共约说明类型（EndpointKind）
//
// 错时延迟、新币 31 确认后使用：属共约，无需也无法在协议层验证，本文件以
// 常量注释形式标注，不产生验证函数（避免误作为合法性条件）。

// EndpointKind 区分协议项与共约项（第 12 章概述）。
type EndpointKind uint8

const (
	// EndpointProtocol 表示此约定为协议级，必须严格验证。
	EndpointProtocol EndpointKind = 1
	// EndpointConvention 表示此约定为共约级，靠自觉遵守，不得拒绝合法区块。
	EndpointConvention EndpointKind = 2
)

// ---- 协议级验证函数 ----

// IsTxExpired 判断交易是否超过 240 块（24h）未收录而过期（协议，第 12 章 §8）。
// 按交易时间戳与区块时间戳推算已经过的区块数（6min/块）；不足或等于 240 块视为未过期。
//
// txTimestamp：交易时间戳（Unix 秒）。
// blockTimestamp：所在区块时间戳（Unix 秒）。
//
// 返回 true 表示已过期，铸造者不得收录，节点拒绝视为非法。
func IsTxExpired(txTimestamp, blockTimestamp int64) bool {
	elapsed := blockTimestamp - txTimestamp
	if elapsed <= 0 {
		return false
	}
	// 经过的区块数 ≈ elapsed / BlockInterval（秒）
	blockInterval := int64(types.BlockInterval.Seconds())
	blocksElapsed := elapsed / blockInterval
	return blocksElapsed > int64(types.ExpiryWindow) // > 240
}

// IsFutureTx 判断交易是否为未来交易（时间戳晚于区块时间戳）（协议，第 12 章 §8）。
// 区块不得收录未来交易。txTimestamp > blockTimestamp 时返回 true。
func IsFutureTx(txTimestamp, blockTimestamp int64) bool {
	return txTimestamp > blockTimestamp
}

// ---- 共约级辅助函数（不得作为拒绝合法区块的依据） ----

// MinFeeThreshold 根据最近 N 块的平均交易费计算推荐最低手续费（共约，第 12 章 §8）。
// 阈值 = avg(recentFees) / 4，其中 recentFees 取前 6000 块（由调用方传入）。
// 若 recentFees 为空，返回 0（不限制）。
// 调用方须注意：此值仅为本地策略参考，不得用于拒绝合法区块或阻止低费交易转播。
func MinFeeThreshold(recentFees []uint64) uint64 {
	if len(recentFees) == 0 {
		return 0
	}
	var sum uint64
	for _, f := range recentFees {
		sum += f
	}
	avg := sum / uint64(len(recentFees))
	return avg / 4
}

// MinFeeWindowSize 是推导最低手续费均值使用的区块窗口大小（共约，第 12 章 §8）。
// 值与 types.MinFeeWindow 一致，集中登记。
const MinFeeWindowSize = types.MinFeeWindow // 6000

// NewCoinConfirmations 是新币使用前建议等待的确认数（共约，第 12 章 §8）。
// 值为 31，规避分叉风险；此为共约，不强制验证。
const NewCoinConfirmations = 31

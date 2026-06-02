package consensus

import (
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// 出块时序（第 12 章 §1，proposal 12 §1）。
//
// 出块间隔固定 6 分钟，区块时间戳由创世时间戳与高度精确推导，
// 不依赖铸造者本机实际时间。铸造冗余 15s 间隔与首块 30s 延后为共约，
// 不得作为区块头合法性必要条件。

// BlockTimeAt 返回高度 h 的区块规范时间戳（由创世时间戳精确推导）。
// 创世时间戳（C-9）以占位常量注入；C-9 裁决前返回值不代表 mainnet 实际时间。
func BlockTimeAt(h types.BlockHeight) time.Time {
	genesis := time.Unix(GenesisTimestamp(), 0).UTC()
	return types.BlockTime(genesis, h)
}

// BlockTimeAtUnix 返回高度 h 的区块规范时间戳（Unix 秒数）。
// 内部调用 BlockTimeAt，便于与协议时间戳字段（int64）直接比较。
func BlockTimeAtUnix(h types.BlockHeight) int64 {
	return BlockTimeAt(h).Unix()
}

// RedundantBroadcastDelay 返回铸造冗余广播间隔（共约：不作为拒绝合法区块依据）。
// 值固定为 15 秒（types.RedundantBroadcastInterval）。
func RedundantBroadcastDelay() time.Duration {
	return types.RedundantBroadcastInterval
}

// FirstBlockExtraDelay 返回发布首个区块的额外延迟（共约：不作为拒绝合法区块依据）。
// 值固定为 30 秒（types.FirstBlockDelay）。
func FirstBlockExtraDelay() time.Duration {
	return types.FirstBlockDelay
}

// RedundantPublishTime 返回择优池第 rank 位候选者（0 起）的目标广播时刻。
// 第 0 位（rank=0）在 BlockTimeAt(h) + FirstBlockExtraDelay() 时广播；
// 后续每位延迟 RedundantBroadcastDelay()。
// 这些时刻均为共约，不作为区块头合法性验证依据。
func RedundantPublishTime(h types.BlockHeight, rank int) time.Time {
	base := BlockTimeAt(h).Add(FirstBlockExtraDelay())
	if rank <= 0 {
		return base
	}
	return base.Add(time.Duration(rank) * RedundantBroadcastDelay())
}

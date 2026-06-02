package consensus

import "github.com/cxio/evidcoin/pkg/types"

// 失败分叉交易回收（第 12 章 §7，proposal 12 §7）。
//
// 分叉竞争结束后，失败分叉上的交易应合并回主链交易池，以免除用户担忧。
// 失败分叉上「过早使用新币」的交易会因新币输入源失效而无效，应排除。
//
// 本文件只定义回收策略接口与过滤逻辑，不实现交易池存储与网络传播。

// RecycleCandidate 是失败分叉上一笔待回收交易的描述。
type RecycleCandidate struct {
	// TxID 是交易标识。
	TxID types.TxID
	// HasInvalidNewCoin 为 true 表示该交易引用了失败分叉产生的新币（输入源失效），
	// 不可回收到主链；false 表示无此问题，可安全回收。
	HasInvalidNewCoin bool
}

// FilterRecyclable 从失败分叉交易集合中过滤出可安全回收到主链的交易 ID 集合。
// 规则（第 12 章 §7）：
//   - HasInvalidNewCoin == true 的交易排除（其新币输入源在主链上无效）。
//   - 其余交易返回，由调用方合并到主链交易池（重复检测、重放保护由调用方完成）。
func FilterRecyclable(candidates []RecycleCandidate) []types.TxID {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]types.TxID, 0, len(candidates))
	for _, c := range candidates {
		if !c.HasInvalidNewCoin {
			out = append(out, c.TxID)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

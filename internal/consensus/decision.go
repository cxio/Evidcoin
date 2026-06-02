package consensus

import "github.com/cxio/evidcoin/pkg/types"

// 长度 20 临界分叉裁决（第 12 章 §5，proposal 12 §5）。
//
// 发现长度恰为 20 的新分叉时，若临近铸造时间点才被发现，可能造成网络割离。
// 规定：本链当前区块（#21）择优池前 5 名成员自由决定是否接纳，签名并广播。
//   - 收集广播签名，按择优池成员顺序，最靠前成员的裁决有效。
//   - 5 名成员都未签名 → 默认否决。
//
// 本文件只实现裁决逻辑与消息结构，不实现 P2P 线格式与广播。

// CriticalDecisionMsgTag 是临界裁决消息使用的域标签前缀，
// 用于与其它签名消息区分（防重放）。
const CriticalDecisionMsgTag = "evidcoin.critical_fork_decision.v1"

// CriticalDecision 表示前 5 名成员对某长度 20 分叉的一次裁决签名记录。
type CriticalDecision struct {
	// PoolRank 是裁决者在择优池中的排名（0 起；范围 0-4，即前 5 名）。
	PoolRank int
	// Accept 为 true 表示接纳，false 表示拒绝。
	Accept bool
}

// CriticalForkMsg 是绑定防重放信息的临界裁决消息体（待签名）。
type CriticalForkMsg struct {
	// ForkPoint 是分叉点区块 ID（48 字节）。
	ForkPoint types.BlockID
	// LocalTip 是本链当前末端区块 ID（48 字节）。
	LocalTip types.BlockID
	// ForkTip 是支链末端区块 ID（48 字节）。
	ForkTip types.BlockID
	// CurrentHeight 是签名时的本链当前高度（防重放时效锚）。
	CurrentHeight uint32
	// Accept 表示签名者是否接纳该分叉。
	Accept bool
}

// ResolveCriticalFork 根据收集到的裁决列表决定是否接纳长度 20 的分叉。
//
// decisions 是来自择优池前 5 名的裁决集合（可为子集；乱序无影响，函数内排序）。
// 规则（第 12 章 §5）：
//   - 按 PoolRank 升序取最靠前有效裁决（PoolRank 范围 0-4）。
//   - 最靠前者的 Accept 值即为最终裁决结果。
//   - 若 decisions 为空或无 PoolRank 在 [0,4] 范围内的成员 → 默认否决（返回 false, ErrDecisionNoQuorum）。
func ResolveCriticalFork(decisions []CriticalDecision) (accept bool, err error) {
	const topN = 5

	// 找 PoolRank 最小（最靠前）的有效裁决。
	bestRank := topN // 超出范围视为无效
	bestAccept := false

	for _, d := range decisions {
		if d.PoolRank < 0 || d.PoolRank >= topN {
			// 不在前 5 名范围，忽略
			continue
		}
		if d.PoolRank < bestRank {
			bestRank = d.PoolRank
			bestAccept = d.Accept
		}
	}

	if bestRank >= topN {
		// 无有效裁决 → 默认否决
		return false, ErrDecisionNoQuorum
	}
	return bestAccept, nil
}

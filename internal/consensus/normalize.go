package consensus

import (
	"bytes"
	"sort"

	"github.com/cxio/evidcoin/pkg/types"
)

// 区块候选归一化（第 12 章 §3，DEC-0303）。
//
// 分叉链段逐高度比较前，每高度先经过两步归一化选出有效候选块：
//   1. 同铸造者多签归一化：按铸造者公钥哈希分组，每组保留低收益代表块。
//   2. 交易量约束归一化：按择优池排名升序，后位候选满足 Stakes>3x 或 TxCount>2x
//      （任一条件成立）即可替换当前最优者。

// ForkBlock 是参与分叉比较的单高度候选块关键字段。
// 全部数值由上层在接收/验证区块时注入，本包不重复执行交易/状态计算。
type ForkBlock struct {
	// BlockID 是区块哈希（48 字节，用于多签归一化三级比较末项）。
	BlockID types.BlockID
	// MintHash 是铸凭哈希（32 字节，用于分叉链段逐高度计分）。
	MintHash types.MintHash
	// MintPKHash 是铸造者公钥哈希（32 字节，用于同铸造者分组）。
	MintPKHash [32]byte
	// MinterReward 是 Coinbase 中直接分配给铸造者身份的金额（聪），
	// 不含校验组报酬/公共服务奖励/其它第三方收益（DEC-0303）。
	MinterReward uint64
	// TotalFee 是区块所有交易费总额（聪，含 Coinbase 交易费）。
	TotalFee uint64
	// Stakes 是该区块对应的候选块币权销毁值（聪时）（DEC-0303 B-5 第三义）。
	Stakes uint64
	// TxCount 是区块内交易数量（含 Coinbase；DEC-0303）。
	TxCount uint64
	// PoolRank 是铸造者在择优池中的排名（0 起；排名越小越优先；DEC-0303）。
	PoolRank int
}

// NormalizeMultiSig 执行同铸造者多签归一化（第 12 章 §3.1，DEC-0303）：
// 按 MintPKHash 分组，每组仅保留优先级最高者（低收益 → 低总费 → 小 BlockID）。
// 入参 candidates 不会被修改；返回去重后的候选集（每个铸造者最多一块）。
func NormalizeMultiSig(candidates []ForkBlock) []ForkBlock {
	if len(candidates) == 0 {
		return nil
	}
	// 按 MintPKHash 分组，保留每组中"最优"块。
	best := make(map[[32]byte]ForkBlock, len(candidates))
	for _, c := range candidates {
		prev, ok := best[c.MintPKHash]
		if !ok || multiSigLess(c, prev) {
			best[c.MintPKHash] = c
		}
	}
	out := make([]ForkBlock, 0, len(best))
	for _, b := range best {
		out = append(out, b)
	}
	return out
}

// multiSigLess 判断 a 是否优于 b（在同组多签归一化中应保留 a 而非 b）。
// 优先级：MinterReward 低 → TotalFee 低 → BlockID 字节序小（DEC-0303）。
func multiSigLess(a, b ForkBlock) bool {
	if a.MinterReward != b.MinterReward {
		return a.MinterReward < b.MinterReward
	}
	if a.TotalFee != b.TotalFee {
		return a.TotalFee < b.TotalFee
	}
	return bytes.Compare(a.BlockID[:], b.BlockID[:]) < 0
}

// NormalizeTxVolume 执行交易量约束归一化（第 12 章 §3.2，DEC-0303）：
// 入参 candidates 已完成多签归一化，各块具有不同铸造者。
// 函数按择优池排名升序排列后从最优候选出发，逐个检查后位候选是否满足：
//
//	challenger.Stakes > winner.Stakes * 3  OR  challenger.TxCount > winner.TxCount * 2
//
// 满足任一则以 challenger 替换 winner 并继续；否则停止。
// winner.Stakes==0 或 winner.TxCount==0 时，后位对应指标 >0 即满足对应条件（DEC-0303）。
// 返回最终 winner。缺位（nil/空）时返回零值 ForkBlock 与 false。
func NormalizeTxVolume(candidates []ForkBlock) (winner ForkBlock, ok bool) {
	if len(candidates) == 0 {
		return ForkBlock{}, false
	}
	// 按 PoolRank 升序排列（排名小者靠前）。
	sorted := make([]ForkBlock, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PoolRank < sorted[j].PoolRank
	})

	winner = sorted[0]
	for _, challenger := range sorted[1:] {
		if txVolumeExceedsThreshold(challenger, winner) {
			winner = challenger
		} else {
			// 一旦后位未满足，停止（DEC-0303）。
			break
		}
	}
	return winner, true
}

// txVolumeExceedsThreshold 判断 challenger 是否满足交易量约束（DEC-0303）：
//
//	challenger.Stakes > winner.Stakes * 3  OR  challenger.TxCount > winner.TxCount * 2
//
// winner.Stakes==0 时 challenger.Stakes>0 即满足 Stakes 条件；
// winner.TxCount==0 时 challenger.TxCount>0 即满足 TxCount 条件。
func txVolumeExceedsThreshold(challenger, winner ForkBlock) bool {
	stakesOK := challenger.Stakes > winner.Stakes*3
	txOK := challenger.TxCount > winner.TxCount*2
	return stakesOK || txOK
}

// SelectCandidate 对同一高度的候选块集合执行完整两步归一化，返回最终选出的有效候选块。
// 若 candidates 为空，返回零值与 false。
func SelectCandidate(candidates []ForkBlock) (ForkBlock, bool) {
	step1 := NormalizeMultiSig(candidates)
	return NormalizeTxVolume(step1)
}

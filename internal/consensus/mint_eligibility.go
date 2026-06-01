package consensus

import "github.com/cxio/evidcoin/pkg/types"

// 铸凭交易资格窗口与评参/币权评估高度（第 11 章 §1，DEC-0302）。

// evalRefOffset 是评参区块相对链末端的偏移：取 -8 号区块铸凭哈希（第 11 章 §1）。
const evalRefOffset = 8

// mintWindowLowerExclusive 是正常期铸凭交易高度差的下限（独占）：
// h 必须 > 239，即至少早于当前 240 个区块（约 1 天，由 MintWindowEnd=-240 推导）。
const mintWindowLowerExclusive = -types.MintWindowEnd - 1 // 240 - 1 = 239

// mintWindowUpperInclusive 是正常期铸凭交易高度差的上限（包含）：
// h 必须 <= 80000，即不早于当前 80000 个区块（由 MintWindowStart=-80000 推导）。
const mintWindowUpperInclusive = -types.MintWindowStart // 80000

// MintTxEligibleNormal 判定正常期铸凭交易资格（第 11 章 §1，DEC-0302）：
//
//	h := currentHeight - txHeight; 资格为 h > 239 && h <= 80000
//
// 资格只依赖交易所在区块高度，与交易自身 Timestamp 无关（故此函数不接受时间戳）。
// 调用方须另行保证铸凭交易为已确认交易，且非当前待铸区块内交易（防自引用）。
func MintTxEligibleNormal(currentHeight, txHeight uint32) bool {
	// 未来交易（txHeight >= currentHeight）一律不合格；区块不收录未来交易。
	if txHeight >= currentHeight {
		return false
	}
	h := currentHeight - txHeight
	return h > mintWindowLowerExclusive && h <= mintWindowUpperInclusive
}

// EvalRefHeight 返回正常期评参区块高度：链末端 -8 号区块（第 11 章 §1）。
// currentHeight < 8 时无足够前驱区块，返回 ok=false（初段评参规则见 genesis.go）。
func EvalRefHeight(currentHeight uint32) (height uint32, ok bool) {
	if currentHeight < evalRefOffset {
		return 0, false
	}
	return currentHeight - evalRefOffset, true
}

// StakeEvalHeight 返回币权销毁评估区块高度：链末端 -32 号区块（第 11 章 §1）。
// 取该区块头 Stakes（聪时）作为铸凭哈希的币权销毁因子。
// currentHeight < 32 时无足够前驱区块，返回 ok=false。
func StakeEvalHeight(currentHeight uint32) (height uint32, ok bool) {
	const stakeOffset = -types.StakeEvalOffset // 32
	if currentHeight < stakeOffset {
		return 0, false
	}
	return currentHeight - stakeOffset, true
}

package validation

import "math/bits"

// TxPriority 是交易纳入优先级等级（第 13 章 §6，共约非强制）。
// 优先级是校验组与铸造者之间的共约（convention），不属于共识规则。
// 管理层可据此排序待纳入交易，但不得以优先级不符为由拒绝合法区块。
type TxPriority uint8

const (
	// PriorityHighBurnStake 是最高优先级：高燃烧量 + 高币权的输入（第 13 章 §6）。
	PriorityHighBurnStake TxPriority = 3
	// PriorityHighFee 是次级优先级：高手续费（第 13 章 §6）。
	PriorityHighFee TxPriority = 2
	// PriorityWithCredit 是第三优先级：包含信用（凭信）输入（第 13 章 §6）。
	PriorityWithCredit TxPriority = 1
)

// CheckStakesConstraint 检验挑战者区块的 Stakes 是否满足替换约束（DEC-0303）。
// 约束：challengerStakes 须严格大于 baseStakes 的 3 倍（> 3 * base）。
// 使用 math/bits.Mul64 避免 3*base 溢出导致的误判。
//
// 返回 nil 表示约束满足（挑战者可替换基准区块）；返回 ErrStakesConstraintFail 表示不满足。
func CheckStakesConstraint(baseStakes, challengerStakes uint64) error {
	// hi 为 3*baseStakes 的高 64 位；若溢出（hi != 0），则 3*base > MaxUint64，
	// 任何 challengerStakes 均不可能严格超越，直接拒绝。
	hi, tripled := bits.Mul64(baseStakes, 3)
	if hi != 0 || challengerStakes <= tripled {
		return ErrStakesConstraintFail
	}
	return nil
}

package rewards

import "github.com/cxio/evidcoin/pkg/types"

// hundredDayBoundary 是百日前/后分界高度（含此值时为百日前，DEC-0401）。
const hundredDayBoundary uint32 = 24000

// RewardOutput 是一个奖励输出条目，携带配置值与计算后的金额。
// 配置值固定为升序排列（1=铸凭者、2=校验组、3=Blockqs、4=Depots、5=STUN），
// 顺序影响 TxID，不得调换（DEC-0401）。
type RewardOutput struct {
	// Config 是输出配置值（1~5，见 DEC-0401）。
	Config uint8
	// Amount 是奖励金额（单位 chx）。
	Amount types.Amount
}

// DistributeReward 按 DEC-0401 规则计算各输出奖励金额。
//
// base 是 RewardBase（发行量 + 未销毁交易费 + reclaimed_award，单位 chx）。
// height 是当前区块高度。
//
// 百日前（height <= 24000）：返回 2 项，铸凭者（Config=1）20%、校验组（Config=2）承接余数（80%）。
// 百日后（height >= 24001）：返回 5 项，铸凭者 10%、校验组 40%、Blockqs 20%、Depots 20%、STUN 承接余数（10%）。
// 所有项按配置值升序排列；前 N-1 项向下取整，最后一项承接全部余数。
// 计算禁止浮点中间值。
func DistributeReward(base types.Amount, height uint32) []RewardOutput {
	if height <= hundredDayBoundary {
		// 百日前：铸凭者 20%（向下取整），校验组承接余数。
		minter := base * 20 / 100
		validator := base - minter
		return []RewardOutput{
			{Config: 1, Amount: minter},
			{Config: 2, Amount: validator},
		}
	}
	// 百日后：5 输出，前 4 项向下取整，STUN 承接余数。
	c1 := base * 10 / 100          // 铸凭者 10%
	c2 := base * 40 / 100          // 校验组 40%
	c3 := base * 20 / 100          // Blockqs 20%
	c4 := base * 20 / 100          // Depots 20%
	c5 := base - c1 - c2 - c3 - c4 // STUN 承接余数
	return []RewardOutput{
		{Config: 1, Amount: c1},
		{Config: 2, Amount: c2},
		{Config: 3, Amount: c3},
		{Config: 4, Amount: c4},
		{Config: 5, Amount: c5},
	}
}

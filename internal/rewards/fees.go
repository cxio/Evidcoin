package rewards

import "github.com/cxio/evidcoin/pkg/types"

// SplitFee 将总交易费拆分为已销毁与未销毁两部分（第 14 章 §2、DEC-0401）：
//
//	burned   = total / 2        // 整数除法向下取整
//	unburned = total - burned   // 奇数时余 1 chx 归未销毁部分
//
// BurnCoin 恒为非负（chx）。销毁单点化于 Coinbase，普通交易不可销毁币金。
func SplitFee(total types.Amount) (burned, unburned types.Amount) {
	burned = total / 2
	unburned = total - burned
	return
}

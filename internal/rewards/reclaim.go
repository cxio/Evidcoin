package rewards

import "github.com/cxio/evidcoin/pkg/types"

// reclaimConfirmWindow 是兑奖确认窗口的块数（DEC-0401）。
const reclaimConfirmWindow = 48

// ReclaimAmount 根据确认次数计算服务奖励的回收额（第 14 章 §6~7、DEC-0401）。
//
// 兑奖规则：
//   - 0 次确认：全额回收（serviceAward 全部进入 reclaimed_award）。
//   - 1 次确认：兑 50%（floor），剩余 50% 回收。
//   - 2+ 次确认：全额兑奖，回收 0。
//
// 计算禁止浮点；1 次确认时 redeemed = serviceAward * 50 / 100（整数除法向下取整），
// 回收 = serviceAward - redeemed。
func ReclaimAmount(serviceAward types.Amount, confirmCount int) types.Amount {
	if confirmCount >= 2 {
		return 0
	}
	if confirmCount == 1 {
		redeemed := uint64(serviceAward) * 50 / 100
		return serviceAward - types.Amount(redeemed)
	}
	// 0 次确认：全额回收。
	return serviceAward
}

// CountConfirmations 从 48 个连续兑奖槽中统计对指定服务的确认次数（DEC-0401）。
//
// window 是块 K+1..K+48 的兑奖槽，window[0]=块 K+1，window[47]=块 K+48（共 48 项）。
// 服务 svc 对块 K 的确认：块 K+j（j=1..48）在其 AwardSlots 中通过 bit 偏移 j 确认块 K。
// 具体地，window[j-1].IsSet(svc, j) 为 true 即表示块 K+j 确认了块 K 的服务 svc。
// 返回确认次数（0~48）；window 长度不足 48 时仅统计已有项。
func CountConfirmations(window []AwardSlots, svc ServiceIndex) int {
	count := 0
	for j := 1; j <= len(window) && j <= reclaimConfirmWindow; j++ {
		// window[j-1] 是块 K+j；bit 偏移 j 确认块 K（因为 K+j - 1 - (j-1) = K）。
		if window[j-1].IsSet(svc, j) {
			count++
		}
	}
	return count
}

// ComputeReclaimedAward 计算指定块公共服务奖励的总回收额（第 14 章 §7、DEC-0401）。
//
// serviceAmounts[0..2] 是块 K（Blockqs/Depots/STUN）各服务的奖励金额（chx）。
// window 是块 K+1..K+48 的兑奖槽（共 48 项），window[0]=块 K+1，window[47]=块 K+48。
// 按 DEC-0401：到块 K+49 时，未被充分确认的部分进入当前块 Coinbase 的 reclaimed_award。
// 每项服务独立计算确认次数与回收额，三项之和为总回收额。
func ComputeReclaimedAward(serviceAmounts [3]types.Amount, window [reclaimConfirmWindow]AwardSlots) types.Amount {
	var total types.Amount
	for svc := ServiceIndex(0); svc <= 2; svc++ {
		// 将固定大小数组的切片传给 CountConfirmations。
		confirmCount := CountConfirmations(window[:], svc)
		total += ReclaimAmount(serviceAmounts[svc], confirmCount)
	}
	return total
}

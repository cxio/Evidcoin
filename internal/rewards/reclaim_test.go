package rewards

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestReclaim(t *testing.T) {
	t.Run("ReclaimAmount", func(t *testing.T) {
		cases := []struct {
			name         string
			award        types.Amount
			confirmCount int
			want         types.Amount
		}{
			// 0 次确认：全额回收。
			{"confirm0_100", 100, 0, 100},
			{"confirm0_1", 1, 0, 1},
			{"confirm0_0", 0, 0, 0},
			// 1 次确认：兑 50%（floor），回收剩余。
			// redeemed = 100*50/100 = 50，回收 = 100-50 = 50。
			{"confirm1_100_even", 100, 1, 50},
			// redeemed = 101*50/100 = 50（floor），回收 = 101-50 = 51。
			{"confirm1_101_odd", 101, 1, 51},
			// redeemed = 1*50/100 = 0，回收 = 1-0 = 1。
			{"confirm1_1", 1, 1, 1},
			// redeemed = 0*50/100 = 0，回收 = 0。
			{"confirm1_0", 0, 1, 0},
			// 2 次确认：全额兑奖，回收 0。
			{"confirm2", 999, 2, 0},
			{"confirm3", 500, 3, 0},
			{"confirm_many", 1000, 48, 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got := ReclaimAmount(tc.award, tc.confirmCount)
				if got != tc.want {
					t.Errorf("ReclaimAmount(%d, %d) = %d, want %d",
						tc.award, tc.confirmCount, got, tc.want)
				}
			})
		}
	})

	t.Run("CountConfirmations", func(t *testing.T) {
		// 构造 48 个兑奖槽，验证对块 K 的确认计数。
		// window[j-1] 是块 K+j，bit 偏移 j 确认块 K。
		var window [48]AwardSlots

		// 块 K+1 确认（window[0], bit offset=1）。
		window[0].SetBit(ServiceBlockqs, 1)
		// 块 K+48 确认（window[47], bit offset=48）。
		window[47].SetBit(ServiceBlockqs, 48)

		got := CountConfirmations(window[:], ServiceBlockqs)
		if got != 2 {
			t.Errorf("CountConfirmations: got %d, want 2", got)
		}
		// Depots 无确认。
		if got := CountConfirmations(window[:], ServiceDepots); got != 0 {
			t.Errorf("CountConfirmations Depots: got %d, want 0", got)
		}
	})

	t.Run("CountConfirmations_all_set", func(t *testing.T) {
		// 全部 48 块都确认 Blockqs。
		var window [48]AwardSlots
		for j := 1; j <= 48; j++ {
			window[j-1].SetBit(ServiceBlockqs, j)
		}
		if got := CountConfirmations(window[:], ServiceBlockqs); got != 48 {
			t.Errorf("all confirmed: got %d, want 48", got)
		}
	})

	t.Run("ComputeReclaimedAward", func(t *testing.T) {
		// 三个服务各有奖励，分别测试不同确认情形。
		// Blockqs: 1000 chx，0 确认 → 全额回收 1000
		// Depots: 500 chx，1 确认 → 回收 250（500*50/100=250）
		// STUN: 300 chx，2 确认 → 回收 0
		// 预期总回收 = 1000 + 250 + 0 = 1250
		amounts := [3]types.Amount{1000, 500, 300}
		var window [48]AwardSlots

		// Blockqs：0 确认（不设置任何 bit）。
		// Depots：1 确认，设置 window[0] 的 Depots bit offset=1。
		window[0].SetBit(ServiceDepots, 1)
		// STUN：2 确认，设置 window[0] 和 window[1] 的 STUN bit。
		window[0].SetBit(ServiceSTUN, 1)
		window[1].SetBit(ServiceSTUN, 2)

		got := ComputeReclaimedAward(amounts, window)
		const want = types.Amount(1000 + 250 + 0) // 1250
		if got != want {
			t.Errorf("ComputeReclaimedAward = %d, want %d", got, want)
		}
	})

	t.Run("ComputeReclaimedAward_fully_confirmed", func(t *testing.T) {
		// 所有服务都被 2+ 次确认，回收额为 0。
		amounts := [3]types.Amount{1000, 2000, 3000}
		var window [48]AwardSlots
		for svc := ServiceIndex(0); svc <= 2; svc++ {
			window[0].SetBit(svc, 1)
			window[1].SetBit(svc, 2)
		}
		got := ComputeReclaimedAward(amounts, window)
		if got != 0 {
			t.Errorf("fully confirmed: ComputeReclaimedAward = %d, want 0", got)
		}
	})

	t.Run("ComputeReclaimedAward_none_confirmed", func(t *testing.T) {
		// 无任何确认，全额回收。
		amounts := [3]types.Amount{100, 200, 300}
		var window [48]AwardSlots
		got := ComputeReclaimedAward(amounts, window)
		want := types.Amount(600)
		if got != want {
			t.Errorf("none confirmed: ComputeReclaimedAward = %d, want %d", got, want)
		}
	})

	t.Run("block_K49_reclaim_scenario", func(t *testing.T) {
		// 模拟规范场景：块 K 有公共服务奖励，在 K+49 块计算回收额。
		// K 的奖励：Blockqs=600, Depots=600, STUN=300（共 1500 chx）。
		// window 代表 K+1..K+48 的 AwardSlots。
		amounts := [3]types.Amount{600, 600, 300}
		var window [48]AwardSlots

		// Blockqs：仅 1 次确认（K+5，即 window[4], bit=5）。
		window[4].SetBit(ServiceBlockqs, 5)
		// Depots：0 次确认。
		// STUN：2 次确认（K+10 和 K+20）。
		window[9].SetBit(ServiceSTUN, 10)
		window[19].SetBit(ServiceSTUN, 20)

		got := ComputeReclaimedAward(amounts, window)
		// Blockqs 回收：600 - 600*50/100 = 600 - 300 = 300
		// Depots 回收：600（全额）
		// STUN 回收：0（全额兑）
		// 合计：300 + 600 + 0 = 900
		want := types.Amount(900)
		if got != want {
			t.Errorf("K+49 scenario: ComputeReclaimedAward = %d, want %d", got, want)
		}
	})
}

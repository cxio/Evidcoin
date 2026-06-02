package rewards

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestDistribution(t *testing.T) {
	t.Run("pre_hundred_day", func(t *testing.T) {
		// 百日前 profile：铸凭者 20%，校验组承接余数。
		cases := []struct {
			name   string
			height uint32
			base   types.Amount
			want   [2]types.Amount
		}{
			// 高度 0（创世）：base=1000，铸凭者=200，校验组=800。
			{"height0", 0, 1000, [2]types.Amount{200, 800}},
			// 高度 24000（分界值，仍为百日前）。
			{"height24000", 24000, 1000, [2]types.Amount{200, 800}},
			// RewardBase 含余数：base=1001，铸凭者=200（floor），校验组=801（余数归最后项）。
			{"remainder", 0, 1001, [2]types.Amount{200, 801}},
			// base=99：铸凭者=19（floor(99*20/100)=19），校验组=80。
			{"floor_19", 0, 99, [2]types.Amount{19, 80}},
			// base=0：两项均为 0。
			{"zero_base", 0, 0, [2]types.Amount{0, 0}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				outputs := DistributeReward(tc.base, tc.height)
				if len(outputs) != 2 {
					t.Fatalf("DistributeReward height=%d: got %d outputs, want 2", tc.height, len(outputs))
				}
				// Config 顺序必须为升序（1, 2）。
				if outputs[0].Config != 1 || outputs[1].Config != 2 {
					t.Errorf("configs = [%d,%d], want [1,2]", outputs[0].Config, outputs[1].Config)
				}
				if outputs[0].Amount != tc.want[0] {
					t.Errorf("minter amount = %d, want %d", outputs[0].Amount, tc.want[0])
				}
				if outputs[1].Amount != tc.want[1] {
					t.Errorf("validator amount = %d, want %d", outputs[1].Amount, tc.want[1])
				}
				// 总和守恒。
				if outputs[0].Amount+outputs[1].Amount != tc.base {
					t.Errorf("sum %d+%d != base %d", outputs[0].Amount, outputs[1].Amount, tc.base)
				}
			})
		}
	})

	t.Run("post_hundred_day", func(t *testing.T) {
		// 百日后 profile：5 输出，配置值升序，STUN 承接余数。
		cases := []struct {
			name   string
			height uint32
			base   types.Amount
			want   [5]types.Amount // [c1,c2,c3,c4,c5]
		}{
			// 高度 24001（第一个百日后块）：base=1000。
			// c1=100, c2=400, c3=200, c4=200, c5=100。
			{"height24001_clean", 24001, 1000, [5]types.Amount{100, 400, 200, 200, 100}},
			// base=1001：c1=100(floor), c2=400(floor), c3=200(floor), c4=200(floor), c5=101（余数）。
			{"height24001_remainder", 24001, 1001, [5]types.Amount{100, 400, 200, 200, 101}},
			// base=99：c1=9(floor), c2=39(floor), c3=19(floor), c4=19(floor), c5=13(余数)。
			// 验算：9+39+19+19=86，c5=99-86=13。
			{"base99", 24001, 99, [5]types.Amount{9, 39, 19, 19, 13}},
			// base=0：全零。
			{"base0", 24001, 0, [5]types.Amount{0, 0, 0, 0, 0}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				outputs := DistributeReward(tc.base, tc.height)
				if len(outputs) != 5 {
					t.Fatalf("DistributeReward height=%d: got %d outputs, want 5", tc.height, len(outputs))
				}
				// Config 顺序必须为升序（1, 2, 3, 4, 5）。
				for i, o := range outputs {
					if o.Config != uint8(i+1) {
						t.Errorf("outputs[%d].Config = %d, want %d", i, o.Config, i+1)
					}
				}
				for i, wantAmt := range tc.want {
					if outputs[i].Amount != wantAmt {
						t.Errorf("outputs[%d].Amount = %d, want %d", i, outputs[i].Amount, wantAmt)
					}
				}
				// 总和守恒。
				var sum types.Amount
				for _, o := range outputs {
					sum += o.Amount
				}
				if sum != tc.base {
					t.Errorf("sum=%d != base=%d", sum, tc.base)
				}
			})
		}
	})

	t.Run("boundary", func(t *testing.T) {
		// 高度 24000：百日前，2 输出。
		if n := len(DistributeReward(1000, 24000)); n != 2 {
			t.Errorf("height=24000: got %d outputs, want 2", n)
		}
		// 高度 24001：百日后，5 输出。
		if n := len(DistributeReward(1000, 24001)); n != 5 {
			t.Errorf("height=24001: got %d outputs, want 5", n)
		}
	})
}

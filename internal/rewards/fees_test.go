package rewards

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestFees(t *testing.T) {
	tests := []struct {
		name         string
		total        types.Amount
		wantBurned   types.Amount
		wantUnburned types.Amount
	}{
		// 零总额：两部分均为 0。
		{"zero", 0, 0, 0},
		// 偶数：各占一半。
		{"even_100", 100, 50, 50},
		{"even_200", 200, 100, 100},
		// 奇数：余 1 chx 归未销毁部分。
		{"odd_1", 1, 0, 1},
		{"odd_101", 101, 50, 51},
		{"odd_999", 999, 499, 500},
		// 大额偶数（实际场景规模）。
		{"large_even", 8 * types.Amount(types.ChxPerBi), 4 * types.Amount(types.ChxPerBi), 4 * types.Amount(types.ChxPerBi)},
		// 大额奇数。
		{"large_odd", 8*types.Amount(types.ChxPerBi) + 1, 4 * types.Amount(types.ChxPerBi), 4*types.Amount(types.ChxPerBi) + 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotBurned, gotUnburned := SplitFee(tc.total)
			if gotBurned != tc.wantBurned {
				t.Errorf("SplitFee(%d).burned = %d, want %d", tc.total, gotBurned, tc.wantBurned)
			}
			if gotUnburned != tc.wantUnburned {
				t.Errorf("SplitFee(%d).unburned = %d, want %d", tc.total, gotUnburned, tc.wantUnburned)
			}
			// burned + unburned 必须等于 total（守恒性）。
			if gotBurned+gotUnburned != tc.total {
				t.Errorf("SplitFee(%d): burned+unburned = %d+%d != total", tc.total, gotBurned, gotUnburned)
			}
			// BurnCoin 恒为非负（types.Amount 为 uint64，不存在负值；此处显式防御）。
			if gotBurned > tc.total {
				t.Errorf("SplitFee(%d): burned %d exceeds total", tc.total, gotBurned)
			}
		})
	}
}

// TestFeesNoFloat 确认 SplitFee 不依赖浮点，整数除法向下取整语义一致。
func TestFeesNoFloat(t *testing.T) {
	// 奇数 chx 的销毁额必须是向下取整：floor(1/2)=0，floor(101/2)=50，floor(999/2)=499。
	cases := [][3]types.Amount{
		{1, 0, 1},
		{101, 50, 51},
		{999, 499, 500},
	}
	for _, c := range cases {
		b, u := SplitFee(c[0])
		if b != c[1] || u != c[2] {
			t.Errorf("SplitFee(%d) = (%d,%d), want (%d,%d)", c[0], b, u, c[1], c[2])
		}
	}
}

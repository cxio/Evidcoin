package rewards

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestIssuance(t *testing.T) {
	// 逐期计算（与 Issuance 内部一致，避免公共因子累积误差）。
	p := [13]uint64{}
	p[0] = 4_000_000_000
	for i := 1; i <= 12; i++ {
		p[i] = p[i-1] * 80 / 100
	}
	// 验证 period 11 ≥ 3 币，period 12 < 3 币（触发长期微通胀）。
	const longTerm = uint64(3) * types.ChxPerBi // 300_000_000
	if p[11] < longTerm {
		t.Fatalf("period 11 subsidy %d should be >= longTerm %d", p[11], longTerm)
	}
	if p[12] >= longTerm {
		t.Fatalf("period 12 subsidy %d should be < longTerm %d", p[12], longTerm)
	}

	const bpy = uint32(types.BlocksPerYear)

	tests := []struct {
		name        string
		blockHeight uint32
		want        types.Amount
	}{
		// 第一阶段：年份 1~3，各年第一块与最后一块。
		{"year1_first", 0, types.Amount(10 * types.ChxPerBi)},
		{"year1_last", bpy - 1, types.Amount(10 * types.ChxPerBi)},
		{"year2_first", bpy, types.Amount(20 * types.ChxPerBi)},
		{"year2_last", 2*bpy - 1, types.Amount(20 * types.ChxPerBi)},
		{"year3_first", 2 * bpy, types.Amount(30 * types.ChxPerBi)},
		{"year3_last", 3*bpy - 1, types.Amount(30 * types.ChxPerBi)},

		// 第二阶段：year 4-5 为 period 0（40 币/块）。
		{"year4_first", 3 * bpy, types.Amount(p[0])},
		{"year5_first", 4 * bpy, types.Amount(p[0])},
		{"year5_last", 5*bpy - 1, types.Amount(p[0])},

		// Period 1（years 6-7）。
		{"year6_first", 5 * bpy, types.Amount(p[1])},
		{"year7_first", 6 * bpy, types.Amount(p[1])},

		// Period 11（years 26-27）：最后一个 ≥ longTerm 的 period。
		{"year26", 25 * bpy, types.Amount(p[11])},
		{"year27", 26 * bpy, types.Amount(p[11])},

		// Period 12（years 28-29）：subsidy < longTerm，钳制为 longTerm。
		{"year28", 27 * bpy, types.Amount(longTerm)},
		{"year29", 28 * bpy, types.Amount(longTerm)},

		// 远期（100 年）：稳定于长期微通胀。
		{"year100", 99 * bpy, types.Amount(longTerm)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Issuance(tc.blockHeight)
			if got != tc.want {
				t.Errorf("Issuance(%d) = %d, want %d", tc.blockHeight, got, tc.want)
			}
		})
	}
}

// TestIssuanceNoFloat 确认发行量计算结果与 float64 路径不同，验证整数除法精度。
// period 10: 4_000_000_000 * 0.8^10 = 4_000_000_000 * 0.1073741824 ≈ 429_496_729.6（小数被截断）
func TestIssuanceNoFloat(t *testing.T) {
	const bpy = uint32(types.BlocksPerYear)
	// year 24 (block 23*bpy): period = (24-4)/2 = 10
	got := Issuance(23 * bpy)
	// 预期精确整数除法结果（非 float 四舍五入）
	// p[10] = 536_870_912 * 80 / 100 = 429_496_729（向下取整）
	p := uint64(4_000_000_000)
	for i := 0; i < 10; i++ {
		p = p * 80 / 100
	}
	want := types.Amount(p)
	if got != want {
		t.Errorf("Issuance(year24) = %d, want %d (integer division)", got, want)
	}
}

package blockchain

import "testing"

// 区块尺寸限额曲线测试（proposal 05 §8 / conception #区块限额）。
// 限额仅约束数据量，含解锁脚本、不含见证。单位 1MB = 1 MiB = 1<<20 字节。

func TestBlockSizeLimit(t *testing.T) {
	const mib = 1 << 20

	tests := []struct {
		name    string
		height  uint32
		wantMB  int
		comment string
	}{
		{"genesis", 0, 1, "第1月起，固定 1MB"},
		{"month1_mid", 1, 1, "第1月内"},
		{"month1_end", 7304, 1, "第1月末（块 0..7304）"},
		{"month2_start", 7305, 1, "第2月始，仍属前3月固定 1MB"},
		{"month3_mid", 14610, 1, "第3月内"},
		{"month3_end", 21914, 1, "第3月末，固定段最后一块"},
		{"month4_start", 21915, 2, "第4月始，开始逐月递增"},
		{"month4_end", 29219, 2, "第4月末（块 21915..29219）"},
		{"month5_start", 29220, 3, "第5月始"},
		{"month11_end", 80354, 9, "第11月末"},
		{"month12_start", 80355, 10, "第12月（末月 7306 块）始，达 10MB"},
		{"year1_end", 87660, 10, "第一年最后一块，封顶 10MB"},
		{"year2_start", 87661, 11, "第2年始，逐年递增至 11MB"},
		{"year2_end", 175321, 11, "第2年末（块 87661..175321）"},
		{"year3_start", 175322, 12, "第3年始，12MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlockSizeLimit(tt.height)
			want := tt.wantMB * mib
			if got != want {
				t.Errorf("BlockSizeLimit(%d) = %d 字节 (%d MB), 期望 %d 字节 (%d MB): %s",
					tt.height, got, got/mib, want, tt.wantMB, tt.comment)
			}
		})
	}
}

// TestBlockSizeLimitMonotonic 验证限额随高度单调不减（逐月/逐年递增，无回退）。
func TestBlockSizeLimitMonotonic(t *testing.T) {
	// 抽样覆盖前 3 年的关键过渡点。
	prev := BlockSizeLimit(0)
	for h := uint32(0); h <= 3*87661; h += 1217 { // 步进非月/年整除，扫描各段
		cur := BlockSizeLimit(h)
		if cur < prev {
			t.Fatalf("限额在 height=%d 处回退：%d < %d", h, cur, prev)
		}
		prev = cur
	}
}

// TestBlockSizeLimitUnit 验证单位口径为 1 MiB = 1<<20 字节。
func TestBlockSizeLimitUnit(t *testing.T) {
	if got := BlockSizeLimit(0); got != 1<<20 {
		t.Fatalf("创世限额 = %d, 期望 %d (1 MiB)", got, 1<<20)
	}
}

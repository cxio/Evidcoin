package consensus

import "testing"

// TestMintTxEligibleNormal 断言正常期资格窗口：
// h := currentHeight - txHeight；资格为 h > 239 && h <= 80000。
// 边界点：239（不合格）、240（合格）、80000（合格）、80001（不合格）。
func TestMintTxEligibleNormal(t *testing.T) {
	const current = uint32(200000)
	tests := []struct {
		name     string
		h        uint32 // current - txHeight
		eligible bool
	}{
		{"diff 1 too recent", 1, false},
		{"diff 239 boundary lower fail", 239, false},
		{"diff 240 boundary lower pass", 240, true},
		{"diff 1000 mid", 1000, true},
		{"diff 80000 boundary upper pass", 80000, true},
		{"diff 80001 boundary upper fail", 80001, false},
		{"diff zero same height", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txHeight := current - tt.h
			got := MintTxEligibleNormal(current, txHeight)
			if got != tt.eligible {
				t.Fatalf("MintTxEligibleNormal(%d,%d) [h=%d] = %v, want %v",
					current, txHeight, tt.h, got, tt.eligible)
			}
		})
	}
}

// TestMintTxEligibleNormalIgnoresTimestamp 断言资格只依赖区块高度差，
// 与交易自身 Timestamp 无关（API 不接受 timestamp 参数即可证明此约束）。
func TestMintTxEligibleNormalIgnoresTimestamp(t *testing.T) {
	const current = uint32(100000)
	txHeight := current - 1000 // h=1000，合格
	if !MintTxEligibleNormal(current, txHeight) {
		t.Fatal("expected eligible regardless of any timestamp")
	}
}

// TestEvalRefHeight 断言评参区块取链末端 -8 号区块高度。
func TestEvalRefHeight(t *testing.T) {
	tests := []struct {
		name    string
		current uint32
		want    uint32
		ok      bool
	}{
		{"current 8 -> 0", 8, 0, true},
		{"current 100 -> 92", 100, 92, true},
		{"current 7 underflow", 7, 0, false},
		{"current 0 underflow", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := EvalRefHeight(tt.current)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Fatalf("EvalRefHeight(%d) = (%d,%v), want (%d,%v)",
					tt.current, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestStakeEvalHeight 断言币权销毁取链末端 -32 号区块高度。
func TestStakeEvalHeight(t *testing.T) {
	tests := []struct {
		name    string
		current uint32
		want    uint32
		ok      bool
	}{
		{"current 32 -> 0", 32, 0, true},
		{"current 1000 -> 968", 1000, 968, true},
		{"current 31 underflow", 31, 0, false},
		{"current 0 underflow", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := StakeEvalHeight(tt.current)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Fatalf("StakeEvalHeight(%d) = (%d,%v), want (%d,%v)",
					tt.current, got, ok, tt.want, tt.ok)
			}
		})
	}
}

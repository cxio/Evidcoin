package validation

import (
	"math"
	"testing"
)

func TestTxPriorityConstants(t *testing.T) {
	if PriorityHighBurnStake != 3 {
		t.Errorf("PriorityHighBurnStake: got %d, want 3", PriorityHighBurnStake)
	}
	if PriorityHighFee != 2 {
		t.Errorf("PriorityHighFee: got %d, want 2", PriorityHighFee)
	}
	if PriorityWithCredit != 1 {
		t.Errorf("PriorityWithCredit: got %d, want 1", PriorityWithCredit)
	}
}

// TestCheckStakesConstraint 校验 Stakes 约束场景（须严格大于 3×base）。
func TestCheckStakesConstraint(t *testing.T) {
	tests := []struct {
		name       string
		base       uint64
		challenger uint64
		wantErr    bool
	}{
		{
			name:       "严格大于 3x，满足约束",
			base:       100,
			challenger: 301,
			wantErr:    false,
		},
		{
			name:       "恰好等于 3x，不满足（需严格大于）",
			base:       100,
			challenger: 300,
			wantErr:    true,
		},
		{
			name:       "小于 3x，不满足",
			base:       100,
			challenger: 299,
			wantErr:    true,
		},
		{
			name:       "base=0，任何正数满足",
			base:       0,
			challenger: 1,
			wantErr:    false,
		},
		{
			name:       "base=0，challenger=0，不满足（0 不严格大于 0）",
			base:       0,
			challenger: 0,
			wantErr:    true,
		},
		{
			name:       "base 极大（接近 MaxUint64/3），3x 溢出 → 拒绝",
			base:       math.MaxUint64/3 + 1,
			challenger: math.MaxUint64,
			wantErr:    true,
		},
		{
			name:       "base=MaxUint64，3x 严重溢出 → 拒绝",
			base:       math.MaxUint64,
			challenger: math.MaxUint64,
			wantErr:    true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckStakesConstraint(tc.base, tc.challenger)
			if (err != nil) != tc.wantErr {
				t.Errorf("CheckStakesConstraint(%d, %d): err=%v, wantErr=%v",
					tc.base, tc.challenger, err, tc.wantErr)
			}
		})
	}
}

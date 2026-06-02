package consensus

import (
	"testing"
)

// ---- 临界裁决测试（TestDecision*） ----

func TestDecisionNoQuorum(t *testing.T) {
	// 无任何有效裁决 → 默认否决
	_, err := ResolveCriticalFork(nil)
	if err != ErrDecisionNoQuorum {
		t.Errorf("expected ErrDecisionNoQuorum, got %v", err)
	}

	// 无效 rank（>=5）的裁决仍视为无效
	decisions := []CriticalDecision{
		{PoolRank: 5, Accept: true},
		{PoolRank: 10, Accept: true},
	}
	_, err = ResolveCriticalFork(decisions)
	if err != ErrDecisionNoQuorum {
		t.Errorf("expected ErrDecisionNoQuorum with out-of-range ranks, got %v", err)
	}
}

func TestDecisionEarliestMemberWins(t *testing.T) {
	// 最靠前成员（PoolRank 最小）的裁决有效
	cases := []struct {
		name       string
		decisions  []CriticalDecision
		wantAccept bool
	}{
		{
			name: "rank0 接纳优于 rank3 拒绝",
			decisions: []CriticalDecision{
				{PoolRank: 0, Accept: true},
				{PoolRank: 3, Accept: false},
			},
			wantAccept: true,
		},
		{
			name: "rank1 拒绝优于 rank4 接纳",
			decisions: []CriticalDecision{
				{PoolRank: 4, Accept: true},
				{PoolRank: 1, Accept: false},
			},
			wantAccept: false,
		},
		{
			name: "只有 rank4 签名",
			decisions: []CriticalDecision{
				{PoolRank: 4, Accept: true},
			},
			wantAccept: true,
		},
		{
			name: "乱序提交，仍取最小 rank",
			decisions: []CriticalDecision{
				{PoolRank: 3, Accept: false},
				{PoolRank: 1, Accept: true},
				{PoolRank: 4, Accept: false},
				{PoolRank: 2, Accept: false},
			},
			wantAccept: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			accept, err := ResolveCriticalFork(tc.decisions)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if accept != tc.wantAccept {
				t.Errorf("accept: got %v, want %v", accept, tc.wantAccept)
			}
		})
	}
}

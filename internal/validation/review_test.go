package validation

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// makeResults 按给定 Verdict 序列构造 TaskResult 切片。
func makeResults(verdicts ...Verdict) []TaskResult {
	results := make([]TaskResult, len(verdicts))
	for i, v := range verdicts {
		results[i] = TaskResult{
			TxID:        types.TxID{byte(i)},
			ValidatorID: "v",
			Verdict:     v,
		}
	}
	return results
}

// TestEvalRedundancy 基本场景。
func TestEvalRedundancy(t *testing.T) {
	tests := []struct {
		name        string
		verdicts    []Verdict
		wantOutcome ReviewOutcome
		wantErr     error
	}{
		{
			name:        "结果不足（空）",
			verdicts:    nil,
			wantOutcome: OutcomePending,
			wantErr:     ErrRedundancyTooLow,
		},
		{
			name:        "结果不足（1 条）",
			verdicts:    []Verdict{VerdictLegal},
			wantOutcome: OutcomePending,
			wantErr:     ErrRedundancyTooLow,
		},
		{
			name:        "全部合法",
			verdicts:    []Verdict{VerdictLegal, VerdictLegal},
			wantOutcome: OutcomeLegal,
			wantErr:     nil,
		},
		{
			name:        "含拒绝但无非法",
			verdicts:    []Verdict{VerdictLegal, VerdictRejected},
			wantOutcome: OutcomeLegal,
			wantErr:     nil,
		},
		{
			name:        "至少一非法→进入一级复核",
			verdicts:    []Verdict{VerdictLegal, VerdictIllegal},
			wantOutcome: OutcomeNeedsLevel2,
			wantErr:     nil,
		},
		{
			name:        "三条全部非法",
			verdicts:    []Verdict{VerdictIllegal, VerdictIllegal, VerdictIllegal},
			wantOutcome: OutcomeNeedsLevel2,
			wantErr:     nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome, err := EvalRedundancy(makeResults(tc.verdicts...))
			if outcome != tc.wantOutcome {
				t.Errorf("outcome: got %d, want %d", outcome, tc.wantOutcome)
			}
			if err != tc.wantErr {
				t.Errorf("err: got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// TestEvalLevel1Review 一级扩展复核场景。
func TestEvalLevel1Review(t *testing.T) {
	tests := []struct {
		name        string
		verdicts    []Verdict
		wantOutcome ReviewOutcome
	}{
		{
			name:        "空结果→合法",
			verdicts:    nil,
			wantOutcome: OutcomeLegal,
		},
		{
			name:        "全部合法",
			verdicts:    []Verdict{VerdictLegal, VerdictLegal, VerdictLegal},
			wantOutcome: OutcomeLegal,
		},
		{
			name:        "超半数非法→非法",
			// 3 条，2 非法 > 3/2 → OutcomeIllegal
			verdicts:    []Verdict{VerdictIllegal, VerdictIllegal, VerdictLegal},
			wantOutcome: OutcomeIllegal,
		},
		{
			name:        "恰好半数非法→需二级",
			// 4 条，2 非法 = 4/2 → 不满足"严格超半数"→ NeedsLevel2
			verdicts:    []Verdict{VerdictIllegal, VerdictIllegal, VerdictLegal, VerdictLegal},
			wantOutcome: OutcomeNeedsLevel2,
		},
		{
			name:        "少于半数非法→需二级",
			// 3 条，1 非法 < 3/2 → NeedsLevel2
			verdicts:    []Verdict{VerdictIllegal, VerdictLegal, VerdictLegal},
			wantOutcome: OutcomeNeedsLevel2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome := EvalLevel1Review(makeResults(tc.verdicts...))
			if outcome != tc.wantOutcome {
				t.Errorf("EvalLevel1Review: got %d, want %d", outcome, tc.wantOutcome)
			}
		})
	}
}

// TestEvalLevel2Review 二级扩展复核场景。
func TestEvalLevel2Review(t *testing.T) {
	tests := []struct {
		name        string
		verdicts    []Verdict
		wantOutcome ReviewOutcome
	}{
		{
			name:        "空结果→合法",
			verdicts:    nil,
			wantOutcome: OutcomeLegal,
		},
		{
			name:        "全部合法",
			verdicts:    []Verdict{VerdictLegal, VerdictLegal},
			wantOutcome: OutcomeLegal,
		},
		{
			name:        "一名报错→非法",
			verdicts:    []Verdict{VerdictLegal, VerdictIllegal, VerdictLegal},
			wantOutcome: OutcomeIllegal,
		},
		{
			name:        "全部非法",
			verdicts:    []Verdict{VerdictIllegal, VerdictIllegal},
			wantOutcome: OutcomeIllegal,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome := EvalLevel2Review(makeResults(tc.verdicts...))
			if outcome != tc.wantOutcome {
				t.Errorf("EvalLevel2Review: got %d, want %d", outcome, tc.wantOutcome)
			}
		})
	}
}

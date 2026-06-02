package validation

// MinRedundancy 是每笔交易校验的最低冗余度（第 13 章 §3）。
// 管理层在派发任务时须确保至少 MinRedundancy 名校验员接受同一任务。
const MinRedundancy = 2

// ReviewOutcome 是某一复核阶段的结论。
type ReviewOutcome uint8

const (
	// OutcomeLegal 表示交易合法（全部通过，或复核清除）。
	OutcomeLegal ReviewOutcome = 1
	// OutcomeIllegal 表示交易非法。
	OutcomeIllegal ReviewOutcome = 2
	// OutcomeNeedsLevel2 表示一级复核未决，需进入二级复核。
	OutcomeNeedsLevel2 ReviewOutcome = 3
	// OutcomePending 表示结果数量不足，尚无法裁决。
	OutcomePending ReviewOutcome = 4
)

// EvalRedundancy 评估初始冗余校验结果（第 13 章 §3）。
//
// 规则：
//   - 结果数量 < MinRedundancy（2）→ 返回 OutcomePending, ErrRedundancyTooLow。
//   - 全部为合法 → OutcomeLegal。
//   - 至少一名判非法 → OutcomeNeedsLevel2（进入一级扩展复核）。
//
// 拒绝/错误结果不视为合法，亦不视为非法；含拒绝/错误但无非法时仍返回 OutcomeLegal。
// 如需按业务需求处理拒绝/错误，由调用方在分发阶段补偿。
func EvalRedundancy(results []TaskResult) (ReviewOutcome, error) {
	if len(results) < MinRedundancy {
		return OutcomePending, ErrRedundancyTooLow
	}
	for _, r := range results {
		if r.Verdict == VerdictIllegal {
			return OutcomeNeedsLevel2, nil
		}
	}
	return OutcomeLegal, nil
}

// EvalLevel1Review 评估一级扩展复核结果（第 13 章 §3）。
//
// 规则（results 为一级复核校验员的反馈集）：
//   - 零报错（无 VerdictIllegal）→ OutcomeLegal。
//   - 超半数报错（illegalCount > len/2）→ OutcomeIllegal。
//   - 低于半数报错 → OutcomeNeedsLevel2（进入二级复核）。
//
// results 为空时视为零报错，返回 OutcomeLegal。
func EvalLevel1Review(results []TaskResult) ReviewOutcome {
	if len(results) == 0 {
		return OutcomeLegal
	}
	illegalCount := 0
	for _, r := range results {
		if r.Verdict == VerdictIllegal {
			illegalCount++
		}
	}
	if illegalCount == 0 {
		return OutcomeLegal
	}
	// 严格超半数：illegalCount * 2 > len(results)
	if illegalCount*2 > len(results) {
		return OutcomeIllegal
	}
	return OutcomeNeedsLevel2
}

// EvalLevel2Review 评估二级扩展复核结果（第 13 章 §3）。
//
// 规则（results 为二级复核校验员的反馈集）：
//   - 有任意一名报错（VerdictIllegal）→ OutcomeIllegal。
//   - 无报错 → OutcomeLegal。
func EvalLevel2Review(results []TaskResult) ReviewOutcome {
	for _, r := range results {
		if r.Verdict == VerdictIllegal {
			return OutcomeIllegal
		}
	}
	return OutcomeLegal
}

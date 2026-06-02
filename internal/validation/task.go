package validation

import (
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// TaskKind 标识校验任务类型（第 13 章 §1）。
type TaskKind uint8

const (
	// TaskFullValidation 是完整交易校验任务，分配给校验员执行。
	TaskFullValidation TaskKind = 1
	// TaskReview1 是一级扩展复核任务（由管理层在至少一名校验员判非法时发起）。
	TaskReview1 TaskKind = 2
	// TaskReview2 是二级扩展复核任务（由管理层在一级复核低于半数报错时发起）。
	TaskReview2 TaskKind = 3
)

// Task 是分配给校验员的校验任务（第 13 章 §1）。
type Task struct {
	// TxID 是待校验交易的唯一标识。
	TxID types.TxID
	// Kind 是任务类型。
	Kind TaskKind
	// AssignedAt 是任务分配时间（用于超时与业绩记录）。
	AssignedAt time.Time
	// TxData 是待校验交易的完整原始字节，供校验员执行脚本与状态验证。
	TxData []byte
}

// Verdict 标识校验员对单笔交易的校验结论（第 13 章 §1）。
type Verdict uint8

const (
	// VerdictLegal 表示交易合法（通过完整校验）。
	VerdictLegal Verdict = 1
	// VerdictIllegal 表示交易非法（见 TaskResult.Reason）。
	VerdictIllegal Verdict = 2
	// VerdictRejected 表示校验员拒绝任务（超时、资源不足等，非交易本身问题）。
	VerdictRejected Verdict = 3
	// VerdictError 表示校验执行过程中发生环境错误（非协议违规）。
	VerdictError Verdict = 4
)

// TaskResult 是校验员对单笔交易校验任务的无条件反馈（第 13 章 §1）。
type TaskResult struct {
	// TxID 是被校验交易的标识（与 Task.TxID 对应）。
	TxID types.TxID
	// ValidatorID 是提交结果的校验员标识（不透明字符串，用于业绩记录）。
	// 本层不定义信誉系统，ValidatorID 仅供管理层内部业绩统计使用。
	ValidatorID string
	// Verdict 是校验结论。
	Verdict Verdict
	// Reason 是非法或错误时的可选英文说明（仅用于调试，不进入协议）。
	Reason string
}

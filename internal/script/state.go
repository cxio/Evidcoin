package script

// ExecState 表示脚本 VM 的执行状态（DEC-0505 六态）。
type ExecState uint8

const (
	// StateRunning 脚本正在执行中（初始态）。
	StateRunning ExecState = 0
	// StatePassStop 执行结束（END 或公共路径无数据 INPUT），以通关状态 passState 为最终结果。
	StatePassStop ExecState = 1
	// StateVerifyFail PASS false 导致验证失败，立即停止执行。
	StateVerifyFail ExecState = 2
	// StateScriptError 脚本执行遇到不可恢复的错误（如触达禁用指令、GOTO 目标缺失）。
	StateScriptError ExecState = 3
	// StateCostFail 成本预算耗尽（CostFail，DEC-0504）。
	StateCostFail ExecState = 4
	// StatePrivateStop 公共路径以 PrivateStop 结束（私有路径不执行，仅用于占位标识）。
	StatePrivateStop ExecState = 5
)

// IsDone 返回执行状态是否已终止（不再继续执行）。
func (s ExecState) IsDone() bool { return s != StateRunning }

// Passed 返回脚本最终是否通过。
// 只有 StatePassStop 且通关状态 passState 为 true 时才视为通过。
func (s ExecState) Passed(passState bool) bool {
	return s == StatePassStop && passState
}

// String 返回执行状态的文字描述。
func (s ExecState) String() string {
	switch s {
	case StateRunning:
		return "Running"
	case StatePassStop:
		return "PassStop"
	case StateVerifyFail:
		return "VerifyFail"
	case StateScriptError:
		return "ScriptError"
	case StateCostFail:
		return "CostFail"
	case StatePrivateStop:
		return "PrivateStop"
	default:
		return "Unknown"
	}
}

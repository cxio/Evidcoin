package script

// ExecMode 表示脚本 VM 的执行模式。
// 公共验证模式要求确定性，私有路径在 END/INPUT 后不再执行（DEC-0503/0505）。
type ExecMode uint8

const (
	// ModePublic 公共验证路径：所有节点均可验证，要求指令确定性。
	ModePublic ExecMode = 0
	// ModePrivate 私有验证路径：END/INPUT 后的剩余路径，公共节点不执行。
	ModePrivate ExecMode = 1
)

// IsPublic 返回当前是否为公共验证路径。
func (m ExecMode) IsPublic() bool { return m == ModePublic }

// checkPublicSafety 在公共模式下检查当前 opcode 是否合法执行。
// 返回 non-nil error 表示公共路径违规，调用方应将状态设为 StateScriptError。
// 参考：DEC-0505
func checkPublicSafety(mode ExecMode, op Opcode) error {
	if mode != ModePublic {
		return nil
	}
	// 禁用指令（SCRIPT/VALUE/EVAL/INOUT）在公共路径触达即 ScriptError
	if op.IsDisabled() {
		return ErrDisabledInPublic
	}
	// SYS_TIME 在公共路径 END 之前触达即 ScriptError
	if op == SYS_TIME {
		return ErrSysTimeInPublic
	}
	// EXT_PRIV 在公共路径触达即 ScriptError
	if op == EXT_PRIV {
		return ErrExtPrivInPublic
	}
	return nil
}

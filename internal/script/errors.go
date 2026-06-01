package script

import "errors"

// 脚本解析阶段错误。
var (
	// ErrTruncatedAttrParam 附参截断（字节不足）。
	ErrTruncatedAttrParam = errors.New("script: truncated attached parameter")
	// ErrTruncatedAssocData 关联数据截断（字节不足）。
	ErrTruncatedAssocData = errors.New("script: truncated associated data")
	// ErrTrailingBytes 脚本末尾存在未被任何指令消费的残余字节。
	ErrTrailingBytes = errors.New("script: trailing unconsumed bytes")
	// ErrUnknownOpcode 未知 opcode（系统保留或尚未注册）。
	ErrUnknownOpcode = errors.New("script: unknown opcode")
	// ErrScriptTooLong 脚本超过允许的最大字节长度。
	ErrScriptTooLong = errors.New("script: script exceeds maximum allowed length")
	// ErrInvalidUnlockOpcode 解锁脚本中出现不允许的 opcode。
	ErrInvalidUnlockOpcode = errors.New("script: opcode not allowed in unlock script")
)

// 脚本执行阶段错误（返回给 VM 状态机使用）。
var (
	// ErrStackOverflow 数据栈高度超过 MaxStackHeight（255）。
	ErrStackOverflow = errors.New("script: data stack overflow (MaxStackHeight exceeded)")
	// ErrStackUnderflow 数据栈弹出操作时栈为空或元素不足。
	ErrStackUnderflow = errors.New("script: data stack underflow")
	// ErrStackItemTooLarge 单个栈项字节数超过 MaxStackItem（4095）。
	ErrStackItemTooLarge = errors.New("script: stack item exceeds MaxStackItem (4095 bytes)")
	// ErrArgCountMismatch 实参区实参数量与指令要求不符。
	ErrArgCountMismatch = errors.New("script: arg count mismatch")
	// ErrTypeMismatch 类型错误（操作数类型与指令期望不符）。
	ErrTypeMismatch = errors.New("script: type mismatch")
	// ErrIndexOutOfRange 访问切片/字典时索引越界。
	ErrIndexOutOfRange = errors.New("script: index out of range")
	// ErrCostExceeded 成本预算耗尽（CostFail）。
	ErrCostExceeded = errors.New("script: cost budget exceeded")
	// ErrDisabledInPublic 公共验证路径触达禁用指令（ScriptError）。
	ErrDisabledInPublic = errors.New("script: disabled opcode reached in public verification path")
	// ErrSysTimeInPublic SYS_TIME 在 END 前的公共验证路径（ScriptError）。
	ErrSysTimeInPublic = errors.New("script: SYS_TIME reached before END in public path")
	// ErrExtPrivInPublic EXT_PRIV 在公共验证路径（ScriptError）。
	ErrExtPrivInPublic = errors.New("script: EXT_PRIV reached in public path before END")
	// ErrGotoTargetMissing GOTO/EMBED 目标缺失或不可验证（ScriptError）。
	ErrGotoTargetMissing = errors.New("script: GOTO/EMBED target missing or unverifiable")
	// ErrGotoDepthExceeded GOTO 跳转深度超过限制（<= 3）。
	ErrGotoDepthExceeded = errors.New("script: GOTO jump depth exceeded (max 3)")
	// ErrGotoCountExceeded GOTO 跳转次数超过限制（<= 2）。
	ErrGotoCountExceeded = errors.New("script: GOTO jump count exceeded (max 2)")
	// ErrEmbedDepthExceeded EMBED 嵌入深度超过限制（== 0）。
	ErrEmbedDepthExceeded = errors.New("script: EMBED depth exceeded (must be 0)")
	// ErrEmbedCountExceeded EMBED 嵌入次数超过限制（<= 4）。
	ErrEmbedCountExceeded = errors.New("script: EMBED count exceeded (max 4)")
	// ErrInputEmpty INPUT 缓存区无值或数量不足（中断退出，不影响 PASS 状态）。
	ErrInputEmpty = errors.New("script: INPUT buffer empty or insufficient")
	// ErrInvalidFloat 浮点字面量为 NaN/Inf（不允许的字面量值）。
	ErrInvalidFloat = errors.New("script: float literal must not be NaN or Inf")
	// ErrModelOutside MODEL 专用指令在 MODEL 块外执行。
	ErrModelOutside = errors.New("script: model instruction used outside MODEL block")
	// ErrNotInCoinbase SYS_AWARD 在非 Coinbase 交易输出脚本中使用。
	ErrNotInCoinbase = errors.New("script: SYS_AWARD only allowed in Coinbase output script")
	// ErrWitnessMissing SYS_CHKPASS 缺少 Witness 数据。
	ErrWitnessMissing = errors.New("script: SYS_CHKPASS requires witness data from environment")
)

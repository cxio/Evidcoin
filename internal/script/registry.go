package script

import "fmt"

// registry 是全局指令元数据注册表，将 Opcode 映射到 Metadata。
// 注册顺序不影响查询结果；同一 opcode 不可重复注册。
var registry = make(map[Opcode]*Metadata, 256)

// Register 向注册表中注册一条指令的元数据。
// 同一 opcode 重复注册会 panic，以在初始化阶段尽早暴露错误。
func Register(m Metadata) {
	if _, exists := registry[m.Opcode]; exists {
		panic(fmt.Sprintf("script: opcode %d (%s) already registered", m.Opcode, m.Mnemonic))
	}
	// opcode 254/255 永久保留，禁止注册。
	if m.Opcode.IsReserved() {
		panic(fmt.Sprintf("script: opcode %d is system reserved and cannot be registered", m.Opcode))
	}
	cp := m
	registry[m.Opcode] = &cp
}

// Lookup 返回指定 opcode 的元数据。若未注册，返回 nil。
func Lookup(op Opcode) *Metadata {
	return registry[op]
}

// MustLookup 返回指定 opcode 的元数据。若未注册，panic。
func MustLookup(op Opcode) *Metadata {
	m := registry[op]
	if m == nil {
		panic(fmt.Sprintf("script: opcode %d not registered", op))
	}
	return m
}

// AllRegistered 返回注册表中所有 Metadata 的快照（按 opcode 排序）。
func AllRegistered() []*Metadata {
	result := make([]*Metadata, 0, len(registry))
	for i := 0; i < 256; i++ {
		if m, ok := registry[Opcode(i)]; ok {
			result = append(result, m)
		}
	}
	return result
}

// RegistrySize 返回当前注册表中的指令数量。
func RegistrySize() int { return len(registry) }

// init 注册所有指令元数据。
// 按 opcode 顺序注册，方便对照 Instruction/ 规格文档校验。
func init() {
	registerValueInstructions()
	registerCaptureInstructions()
	registerStackInstructions()
	registerCollectionInstructions()
	registerInteractionInstructions()
	registerResultInstructions()
	registerFlowInstructions()
	registerConversionInstructions()
	registerArithmeticInstructions()
	registerComparisonInstructions()
	registerLogicInstructions()
	registerPatternInstructions()
	registerEnvironmentInstructions()
	registerToolInstructions()
	registerSystemInstructions()
	registerFunctionInstructions()
	registerModuleInstructions()
	registerExtensionInstructions()
}

// 可用性组合常量，便于注册表使用。
const (
	// availAll 所有路径均可用（公共+私有+解锁）。
	availAll = AvailUnlock | AvailPublic | AvailPrivate
	// availPublicPrivate 公共+私有可用（不含解锁）。
	availPublicPrivate = AvailPublic | AvailPrivate
	// availPrivateOnly 仅私有路径可用。
	availPrivateOnly = AvailPrivate
	// availDisabledPublic 禁用（主网公共路径 ScriptError）+ 私有可用。
	availDisabledPublic = AvailDisabled | AvailPrivate
)

// ─── 值指令 [0-18] 注册 ──────────────────────────────────────────────────────

func registerValueInstructions() {
	// 公共+私有+解锁；确定性；低/免成本
	unlockDet := availAll
	pubOnly := availPublicPrivate

	Register(Metadata{Opcode: NIL, Mnemonic: "NIL", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1, AssocDataParam: -1,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description:    "push nil onto stack",
		ErrorScenarios: []string{"stack full (MaxStackHeight exceeded)"},
	})
	Register(Metadata{Opcode: TRUE, Mnemonic: "TRUE", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1, AssocDataParam: -1,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push bool true onto stack",
	})
	Register(Metadata{Opcode: FALSE, Mnemonic: "FALSE", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1, AssocDataParam: -1,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push bool false onto stack",
	})
	Register(Metadata{Opcode: BYTE_LIT, Mnemonic: "BYTE", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push byte literal [0,255] onto stack; 1-byte fixed-width attr param",
	})
	Register(Metadata{Opcode: RUNE_LIT, Mnemonic: "RUNE", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{4}, AssocDataParam: -1,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push rune (Unicode code point) literal [0,0x10FFFF]; 4-byte big-endian",
	})
	Register(Metadata{Opcode: INT_LIT, Mnemonic: "INT", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		// 附参为 ULEB128 变长，用 0 表示变长
		AttrParamSizes: []int{0}, AssocDataParam: -1,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push int64 literal; followed by ULEB128 (DEC-0001)",
	})
	Register(Metadata{Opcode: BIGINT_LIT, Mnemonic: "BIGINT", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		// 附参 1 字节：bit7=符号，低 7 位=字节数；关联数据 = magnitude
		AttrParamSizes: []int{1}, AssocDataParam: 0,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description:    "push BigInt literal; attr=slen byte, assoc=magnitude (DEC-0001)",
		ErrorScenarios: []string{"leading zeros in magnitude", "negative zero"},
	})
	Register(Metadata{Opcode: FLOAT_LIT, Mnemonic: "FLOAT", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{8}, AssocDataParam: -1,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description:    "push float64 literal; 8-byte big-endian IEEE 754 bit pattern (DEC-0502); literal must not be NaN/Inf",
		ErrorScenarios: []string{"NaN literal", "+Inf literal", "-Inf literal"},
	})
	Register(Metadata{Opcode: STRING_LIT, Mnemonic: "STRING", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push string literal; attr=ULEB128 byte count, assoc=UTF-8 bytes",
	})
	Register(Metadata{Opcode: VALUES_LIT, Mnemonic: "VALUES", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push value set (slice) literal; attr=total bytes of member sequence",
	})
	Register(Metadata{Opcode: DATA_LIT, Mnemonic: "DATA", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push byte sequence literal; attr=ULEB128 byte count",
	})
	Register(Metadata{Opcode: REGEXP_LIT, Mnemonic: "REGEXP", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: 0,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push RegExp literal; attr=1 byte content length (<256); RE2 semantics",
	})
	Register(Metadata{Opcode: DATE_LIT, Mnemonic: "DATE", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		// 关联数据为 UNIX 毫秒有符号变长，用附参 0 表达变长
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierFree,
		Description: "push Time literal; assoc=UNIX ms signed varint",
	})
	Register(Metadata{Opcode: DICT_LIT, Mnemonic: "DICT", Category: 1,
		ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
		Availability: unlockDet, Deterministic: true, CostTier: CostTierLow,
		Description:    "push Dict; arg1=key slice, arg2=value slice; ordered map[string]any",
		ErrorScenarios: []string{"arg1 not []string"},
	})
	Register(Metadata{Opcode: CODE_LIT, Mnemonic: "CODE", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: pubOnly, Deterministic: true, CostTier: CostTierLow,
		Description: "push compiled instruction sequence (Script value, not executed); attr=sequence length",
	})
	// SCRIPT 和 VALUE 前期禁用，主网公共路径触达即 ScriptError
	Register(Metadata{Opcode: SCRIPT, Mnemonic: "SCRIPT", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0, 48, 0}, AssocDataParam: -1,
		Availability: availDisabledPublic, Deterministic: true, CostTier: CostTierExternal,
		Description:    "[DISABLED mainnet] load script from historical tx output; attr: year(varint)+TxID(48B)+outIdx(varint)",
		ErrorScenarios: []string{"public path: ScriptError (disabled)", "target missing: ScriptError"},
	})
	Register(Metadata{Opcode: VALUE, Mnemonic: "VALUE", Category: 1,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availDisabledPublic, Deterministic: true, CostTier: CostTierUnknown,
		Description: "[DISABLED mainnet] general value accessor; attr=value index (1 byte)",
	})
}

// ─── 截取指令 [19-23] 注册 ───────────────────────────────────────────────────

func registerCaptureInstructions() {
	Register(Metadata{Opcode: AT, Mnemonic: "@", Category: 2,
		ArgCount: ArgNone, ReturnCount: 0, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "redirect next instruction's return value to arg area",
	})
	Register(Metadata{Opcode: SAVE, Mnemonic: "$", Category: 2,
		ArgCount: ArgNone, ReturnCount: 0, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "capture next instruction's return value into local scope (append)",
	})
	Register(Metadata{Opcode: LOCAL, Mnemonic: "$[]", Category: 2,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "load local scope member into arg area; attr=signed index [-128,127]",
		ErrorScenarios: []string{"index out of range"},
	})
	Register(Metadata{Opcode: LOOPVAR, Mnemonic: "$X", Category: 2,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "load loop variable into arg area; attr=identifier [0-3]",
	})
	Register(Metadata{Opcode: DIRECT, Mnemonic: "~", Category: 2,
		ArgCount: ArgNone, ReturnCount: 0, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "force next instruction to take args from data stack (bypass arg area)",
	})
}

// ─── 栈操作指令 [24-34] 注册 ─────────────────────────────────────────────────

func registerStackInstructions() {
	Register(Metadata{Opcode: NOP, Mnemonic: "NOP", Category: 3,
		ArgCount: ArgVariadic, ReturnCount: 0, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "no-op; consumes and discards arg area",
	})
	Register(Metadata{Opcode: PUSH, Mnemonic: "PUSH", Category: 3,
		ArgCount: ArgVariadic, ReturnCount: 0, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "push all arg area items onto data stack in order",
		ErrorScenarios: []string{"stack overflow (MaxStackHeight)"},
	})
	Register(Metadata{Opcode: POP, Mnemonic: "POP", Category: 3,
		ArgCount: ArgNone, ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "pop top of stack",
		ErrorScenarios: []string{"stack underflow"},
	})
	Register(Metadata{Opcode: POPS, Mnemonic: "POPS", Category: 3,
		ArgCount: ArgNone, ReturnCount: -1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "pop multiple items from top; attr=count (0=all); returns sequence (auto-spread)",
		ErrorScenarios: []string{"count exceeds stack height"},
	})
	Register(Metadata{Opcode: TOP, Mnemonic: "TOP", Category: 3,
		ArgCount: ArgNone, ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "reference (copy) top of stack without removing",
		ErrorScenarios: []string{"empty stack"},
	})
	Register(Metadata{Opcode: TOPS, Mnemonic: "TOPS", Category: 3,
		ArgCount: ArgNone, ReturnCount: -1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "reference multiple top-of-stack items; attr=count (0=all); auto-spread",
		ErrorScenarios: []string{"count exceeds stack height"},
	})
	Register(Metadata{Opcode: PEEK, Mnemonic: "PEEK", Category: 3,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "reference item at position; arg=position (0=bottom, negative from top)",
		ErrorScenarios: []string{"position out of range"},
	})
	Register(Metadata{Opcode: PEEKS, Mnemonic: "PEEKS", Category: 3,
		ArgCount: FixedArgs(1), ReturnCount: -1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "reference items in range; attr=count (0=rest after start), arg=start position",
		ErrorScenarios: []string{"position out of range", "insufficient items"},
	})
	Register(Metadata{Opcode: SHIFT, Mnemonic: "SHIFT", Category: 3,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "remove top items and pack into slice (no auto-spread); attr=count (0=all)",
		ErrorScenarios: []string{"count exceeds stack height"},
	})
	Register(Metadata{Opcode: CLONE, Mnemonic: "CLONE", Category: 3,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "shallow-copy top items into slice; attr=count (0=all)",
		ErrorScenarios: []string{"count exceeds stack height"},
	})
	Register(Metadata{Opcode: VIEW, Mnemonic: "VIEW", Category: 3,
		ArgCount: FixedArgs(1), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "reference items in range as slice; attr=count, arg=start position",
		ErrorScenarios: []string{"position/count out of range"},
	})
}

// ─── 集合指令 [35-45] 注册 ───────────────────────────────────────────────────

func registerCollectionInstructions() {
	Register(Metadata{Opcode: SLICE, Mnemonic: "SLICE", Category: 4,
		ArgCount: FixedArgs(2), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierLow,
		Description:    "sub-slice reference; attr=size (0=rest), arg1=target slice, arg2=start index",
		ErrorScenarios: []string{"index out of range"},
	})
	Register(Metadata{Opcode: REVERSE, Mnemonic: "REVERSE", Category: 4,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierLow,
		Description: "reverse slice members into new slice",
	})
	Register(Metadata{Opcode: MERGE, Mnemonic: "MERGE", Category: 4,
		ArgCount: ArgVariadic, ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierLow,
		Description: "merge multiple slices into one new slice",
	})
	Register(Metadata{Opcode: EXTEND, Mnemonic: "EXTEND", Category: 4,
		ArgCount: ArgVariadic, ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierLow,
		Description: "extend target slice with extra members; first arg=target slice",
	})
	Register(Metadata{Opcode: PACK, Mnemonic: "PACK", Category: 4,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierMedium,
		Description: "pack slice members into concatenated byte sequence",
	})
	Register(Metadata{Opcode: SPREAD, Mnemonic: "SPREAD", Category: 4,
		ArgCount: FixedArgs(1), ReturnCount: -1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "spread slice members (auto-spread)",
	})
	Register(Metadata{Opcode: INDEX, Mnemonic: "INDEX", Category: 4,
		ArgCount: FixedArgs(2), ReturnCount: -1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "get slice member(s); arg1=slice, arg2=index or index set",
		ErrorScenarios: []string{"index out of range"},
	})
	Register(Metadata{Opcode: ITEM, Mnemonic: "ITEM", Category: 4,
		ArgCount: FixedArgs(2), ReturnCount: -1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "get dict/object/module member(s); arg1=collection, arg2=key or key set",
	})
	Register(Metadata{Opcode: SET, Mnemonic: "SET", Category: 4,
		ArgCount: FixedArgs(3), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierLow,
		Description:    "set dict key(s)/value(s); returns original dict",
		ErrorScenarios: []string{"key set and value set length mismatch"},
	})
	Register(Metadata{Opcode: CALL, Mnemonic: "CALL", Category: 4,
		// ArgSemiFixed: count = 2 + attr param value
		ArgCount: ArgSemiFixed, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierUnknown,
		Description:    "call object/module method; attr=arg count for method; arg1=target, arg2=method name, ...=method args",
		ErrorScenarios: []string{"method not found", "public: target not statically resolvable"},
	})
	Register(Metadata{Opcode: SIZE, Mnemonic: "SIZE", Category: 4,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "return collection size as Int",
	})
}

// ─── 交互指令 [46-50] 注册 ───────────────────────────────────────────────────

func registerInteractionInstructions() {
	Register(Metadata{Opcode: INPUT, Mnemonic: "INPUT", Category: 5,
		ArgCount: ArgNone, ReturnCount: -1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		// 公共路径视为隐式 END，不导入数据；私有路径正常导入
		Availability: availPublicPrivate, Deterministic: false, CostTier: CostTierFree,
		Description: "blocking import from INPUT buffer; attr=count (0=all); public: implicit PassStop",
	})
	Register(Metadata{Opcode: OUTPUT, Mnemonic: "OUTPUT", Category: 5,
		ArgCount: ArgVariadic, ReturnCount: 0, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: false, CostTier: CostTierFree,
		Description: "non-blocking export to OUTPUT buffer; accumulates until BUFDUMP",
	})
	Register(Metadata{Opcode: BUFDUMP, Mnemonic: "BUFDUMP", Category: 5,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPrivateOnly, Deterministic: false, CostTier: CostTierFree,
		Description: "flush OUTPUT buffer to external environment; attr=identifier; private only",
	})
	Register(Metadata{Opcode: PRINT, Mnemonic: "PRINT", Category: 5,
		ArgCount: ArgVariadic, ReturnCount: 0, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: false, CostTier: CostTierFree,
		Description: "print arg area to console (fmt.Println format)",
	})
}

// ─── 结果指令 [51-57] 注册 ───────────────────────────────────────────────────

func registerResultInstructions() {
	Register(Metadata{Opcode: PASS, Mnemonic: "PASS", Category: 6,
		ArgCount: FixedArgs(1), ReturnCount: 0, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "write pass state; false=immediate VerifyFail, true=continue (DEC-0505)",
		ErrorScenarios: []string{"non-bool arg"},
	})
	Register(Metadata{Opcode: CHECK, Mnemonic: "CHECK", Category: 6,
		ArgCount: FixedArgs(1), ReturnCount: 0, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "write pass state and continue; last write wins (DEC-0505)",
		ErrorScenarios: []string{"non-bool arg"},
	})
	Register(Metadata{Opcode: GOTO, Mnemonic: "GOTO", Category: 6,
		ArgCount: ArgVariadic, ReturnCount: 0,
		AttrParamSizes: []int{0, 48, 0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierExternal,
		Description:    "jump to target output script; attr=year(varint)+TxID(48B)+outIdx(varint); args passed as initial stack",
		ErrorScenarios: []string{"target missing/unverifiable: ScriptError"},
	})
	Register(Metadata{Opcode: EMBED, Mnemonic: "EMBED", Category: 6,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{0, 48, 0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierExternal,
		Description:    "embed target output script sharing current scope; attr same as GOTO",
		ErrorScenarios: []string{"target missing/unverifiable: ScriptError"},
	})
	Register(Metadata{Opcode: EXIT, Mnemonic: "EXIT", Category: 6,
		ArgCount: ArgVariadic, ReturnCount: 0, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "terminate script; check pass state; optional return value",
	})
	Register(Metadata{Opcode: RETURN, Mnemonic: "RETURN", Category: 6,
		ArgCount: ArgVariadic, ReturnCount: 0, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "return from MAP/FILTER iteration; no arg=nil",
	})
	Register(Metadata{Opcode: END, Mnemonic: "END", Category: 6,
		ArgCount: ArgNone, ReturnCount: 0, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "end public verification; public=PassStop, private=ignored and continue (DEC-0505)",
	})
}

// ─── 流程控制 [58-66] 注册 ───────────────────────────────────────────────────

func registerFlowInstructions() {
	Register(Metadata{Opcode: IF, Mnemonic: "IF", Category: 7,
		ArgCount: FixedArgs(1), ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "conditional block; attr=sub-block length; arg=bool",
	})
	Register(Metadata{Opcode: ELSE, Mnemonic: "ELSE", Category: 7,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "else block for preceding IF at same level; attr=sub-block length",
	})
	Register(Metadata{Opcode: SWITCH, Mnemonic: "SWITCH", Category: 7,
		ArgCount: FixedArgs(2), ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "multi-way branch; attr=block length; arg1=subject, arg2=case value list",
	})
	Register(Metadata{Opcode: CASE, Mnemonic: "CASE", Category: 7,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "case sub-block; attr=sub-block length; execute if matching SWITCH subject",
	})
	Register(Metadata{Opcode: DEFAULT, Mnemonic: "DEFAULT", Category: 7,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "default sub-block for SWITCH; execute if no CASE matched",
	})
	Register(Metadata{Opcode: EACH, Mnemonic: "EACH", Category: 7,
		ArgCount: FixedArgs(1), ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierMedium,
		Description: "finite iteration; attr=sub-block length; arg=target collection (slice/dict)",
	})
	Register(Metadata{Opcode: CONTINUE, Mnemonic: "CONTINUE", Category: 7,
		ArgCount: ArgVariadic, ReturnCount: 0, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "skip to next EACH iteration; optional bool arg (execute if true)",
	})
	Register(Metadata{Opcode: BREAK, Mnemonic: "BREAK", Category: 7,
		ArgCount: ArgVariadic, ReturnCount: 0, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "exit EACH/SWITCH block; optional bool arg (execute if true)",
	})
	Register(Metadata{Opcode: BLOCK, Mnemonic: "BLOCK", Category: 7,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "create sub-block for local scope or SOURCE extraction; attr=sub-block length",
	})
}

// ─── 转换指令 [67-79] 注册 ───────────────────────────────────────────────────

func registerConversionInstructions() {
	Register(Metadata{Opcode: BOOL, Mnemonic: "BOOL", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "convert to bool: nil/0/empty=false, else=true",
	})
	Register(Metadata{Opcode: BYTE_CONV, Mnemonic: "BYTE_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "convert to byte [0,255]",
		ErrorScenarios: []string{"value out of [0,255]"},
	})
	Register(Metadata{Opcode: RUNE_CONV, Mnemonic: "RUNE_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "convert to rune [0,0x10FFFF]",
		ErrorScenarios: []string{"value out of [0,0x10FFFF]"},
	})
	Register(Metadata{Opcode: INT_CONV, Mnemonic: "INT_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "convert to int64; Float=truncate toward zero; String=Go literal parse",
	})
	Register(Metadata{Opcode: BIGINT_CONV, Mnemonic: "BIGINT_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "convert to BigInt; Bytes=big-endian (math/big)",
	})
	Register(Metadata{Opcode: FLOAT_CONV, Mnemonic: "FLOAT_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description:    "convert to float64; result must comply DEC-0502 profile",
		ErrorScenarios: []string{"result is NaN/Inf (invalid input)"},
	})
	Register(Metadata{Opcode: STRING_CONV, Mnemonic: "STRING_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "convert to string; attr=radix [2-36] or float format (b/e/E/f/g/G/x/X)",
	})
	Register(Metadata{Opcode: BYTES_CONV, Mnemonic: "BYTES_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "convert to byte sequence; Int=big-endian 8B; does NOT support bytes->Script",
	})
	Register(Metadata{Opcode: RUNES_CONV, Mnemonic: "RUNES_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "convert to []Rune; String/Bytes=UTF-8 decode",
	})
	Register(Metadata{Opcode: TIME_CONV, Mnemonic: "TIME_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "convert to Time; Int=Unix ms; String=RFC3339",
	})
	Register(Metadata{Opcode: REGEXP_CONV, Mnemonic: "REGEXP_CONV", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierMedium,
		Description:    "construct RegExp from string (RE2 only)",
		ErrorScenarios: []string{"invalid RE2 pattern"},
	})
	Register(Metadata{Opcode: ANYS, Mnemonic: "ANYS", Category: 8,
		ArgCount: FixedArgs(1), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "slice type conversion; attr=target type identifier (0=[]any, else=value opcode code)",
	})
}

// ─── 运算指令 [80-103] 注册 ──────────────────────────────────────────────────

func registerArithmeticInstructions() {
	Register(Metadata{Opcode: EXPR, Mnemonic: "EXPR", Category: 9,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierMedium,
		Description: "expression block; internal values unified as float64; attr=expr length (1 byte, max 255)",
	})
	// 符号指令仅表达式内
	for _, op := range []struct {
		op Opcode
		mn string
	}{
		{MUL_OP, "*"}, {DIV_OP, "/"}, {MOD_OP, "%"}, {ADD_OP, "+"}, {SUB_OP, "-"},
	} {
		Register(Metadata{Opcode: op.op, Mnemonic: op.mn, Category: 9,
			ArgCount: ArgNone, ReturnCount: 0, AssocDataParam: -1,
			Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
			Description: "symbol operator (inside EXPR only): " + op.mn,
		})
	}
	// 双实参命名指令
	for _, op := range []struct {
		op   Opcode
		mn   string
		desc string
	}{
		{MUL, "MUL", "multiply Int/Float"},
		{DIV, "DIV", "divide Int/Float"},
		{ADD, "ADD", "add Int/Float; also String/Bytes concat, Dict merge"},
		{SUB, "SUB", "subtract Int/Float"},
		{MOD, "MOD", "modulo Int/Float (math.Mod)"},
		{POW, "POW", "power Int/Float (math.Pow)"},
		{LMOV, "LMOV", "left shift Int"},
		{RMOV, "RMOV", "right shift Int"},
		{AND, "AND", "bitwise AND Int"},
		{ANDX, "ANDX", "bit-clear (&^) Int"},
		{OR, "OR", "bitwise OR Int"},
		{XOR, "XOR", "bitwise XOR Int"},
	} {
		Register(Metadata{Opcode: op.op, Mnemonic: op.mn, Category: 9,
			ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
			Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierMedium,
			Description: op.desc,
		})
	}
	// 单实参指令
	Register(Metadata{Opcode: NEG, Mnemonic: "NEG", Category: 9,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "negate Int/Float",
	})
	Register(Metadata{Opcode: NOT, Mnemonic: "NOT", Category: 9,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "logical NOT Bool",
	})
	Register(Metadata{Opcode: DIVMOD, Mnemonic: "DIVMOD", Category: 9,
		ArgCount: FixedArgs(2), ReturnCount: 2, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierMedium,
		Description: "quotient and remainder; returns two values (auto-spread)",
	})
	Register(Metadata{Opcode: REP, Mnemonic: "REP", Category: 9,
		ArgCount: FixedArgs(1), ReturnCount: -1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "replicate item; attr=copies (0=remove top-of-stack); shallow copy",
	})
	Register(Metadata{Opcode: DEL, Mnemonic: "DEL", Category: 9,
		ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "delete dict entry/entries; arg1=dict, arg2=key or key set",
	})
	Register(Metadata{Opcode: CLEAR, Mnemonic: "CLEAR", Category: 9,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "clear all dict entries",
	})
}

// ─── 比较指令 [104-111] 注册 ─────────────────────────────────────────────────

func registerComparisonInstructions() {
	for _, op := range []struct {
		op   Opcode
		mn   string
		desc string
	}{
		{EQUAL, "EQUAL", "a==b; float: NaN returns false, +0.0==-0.0"},
		{NEQUAL, "NEQUAL", "a!=b"},
		{LT, "LT", "a<b"},
		{LTE, "LTE", "a<=b"},
		{GT, "GT", "a>b"},
		{GTE, "GTE", "a>=b"},
	} {
		Register(Metadata{Opcode: op.op, Mnemonic: op.mn, Category: 10,
			ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
			Availability: availAll, Deterministic: true, CostTier: CostTierFree,
			Description:    op.desc,
			ErrorScenarios: []string{"incompatible types", "NaN in sort comparison"},
		})
	}
	Register(Metadata{Opcode: ISEFV, Mnemonic: "ISEFV", Category: 10,
		ArgCount: FixedArgs(1), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "check exceptional float value; attr=0(NaN),1(+Inf),2(-Inf)",
	})
	Register(Metadata{Opcode: WITHIN, Mnemonic: "WITHIN", Category: 10,
		ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "half-open range [min,max); arg1=value, arg2=[min,max] slice",
	})
}

// ─── 逻辑指令 [112-115] 注册 ─────────────────────────────────────────────────

func registerLogicInstructions() {
	Register(Metadata{Opcode: BOTH, Mnemonic: "BOTH", Category: 11,
		ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "logical AND of two bool args",
	})
	Register(Metadata{Opcode: EITHER, Mnemonic: "EITHER", Category: 11,
		ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "logical OR of two bool args",
	})
	Register(Metadata{Opcode: EVERY, Mnemonic: "EVERY", Category: 11,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierMedium,
		Description: "logical AND of all collection members; empty=true",
	})
	Register(Metadata{Opcode: SOME, Mnemonic: "SOME", Category: 11,
		ArgCount: FixedArgs(1), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierMedium,
		Description: "logical OR: at least n members true; attr=n; empty with n>0=false",
	})
}

// ─── 模式指令 [116-127] 注册 ─────────────────────────────────────────────────

func registerPatternInstructions() {
	Register(Metadata{Opcode: MODEL, Mnemonic: "MODEL", Category: 12,
		ArgCount: FixedArgs(1), ReturnCount: -1,
		// 附参：变长整数包含取值标记高位（最高位）+ 子块代码长度（15 位，max 32KB）
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierHigh,
		Description: "pattern match; arg=target Script/Bytes; returns Bool or ([]extracted, Bool)",
	})
	// MODEL 内专用指令：在 MODEL 外执行产生 ScriptError
	for _, op := range []struct {
		op   Opcode
		mn   string
		desc string
	}{
		{WILDCARD, "_", "single instruction wildcard (MODEL only)"},
		{ELLIPSIS, "...", "any instruction sequence wildcard, non-greedy (MODEL only)"},
	} {
		Register(Metadata{Opcode: op.op, Mnemonic: op.mn, Category: 12,
			ArgCount: ArgNone, ReturnCount: 0, AssocDataParam: -1,
			Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
			Description: op.desc,
		})
	}
	Register(Metadata{Opcode: WILDCARDS, Mnemonic: "_[n]", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "n-instruction wildcard (MODEL only); attr=count",
	})
	Register(Metadata{Opcode: OPTIONAL, Mnemonic: "?{}", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "optional instruction sequence (MODEL only, non-backtracking); attr=seq length",
	})
	Register(Metadata{Opcode: MATCHMOD, Mnemonic: "^?", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "wildcard modifier (MODEL only); attr=bit flags",
	})
	Register(Metadata{Opcode: EXTRACT, Mnemonic: "#", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "extract matched instruction parts (MODEL only); attr=bit flags",
	})
	Register(Metadata{Opcode: TYPEMATCH, Mnemonic: "!?", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "type match/optional (MODEL only); attr=type id with match/optional high bits",
	})
	Register(Metadata{Opcode: INTRANGE, Mnemonic: ">{int}", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		// 两个变长整数附参：下界+上界
		AttrParamSizes: []int{0, 0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "integer range match [lo,hi) (MODEL only); attrs=lo+hi (varint)",
	})
	Register(Metadata{Opcode: FLORANGE, Mnemonic: ">{float}", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		// 附参：下界(8B)+上界(8B)+误差(4B)
		AttrParamSizes: []int{8, 8, 4}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "float range match [lo,hi) with epsilon (MODEL only); attrs=lo(8B)+hi(8B)+eps(4B)",
	})
	Register(Metadata{Opcode: REMATCH, Mnemonic: "RE{}", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{1, 1, 1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierHigh,
		Description: "regex match find (MODEL only); attr1=instr name/code, attr2=find mode, attr3=regex length",
	})
	Register(Metadata{Opcode: REGEXTRACT, Mnemonic: "&", Category: 12,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "extract regex match result (MODEL only, after RE{}); attr=result index",
	})
}

// ─── 环境指令 [128-137] 注册 ─────────────────────────────────────────────────

func registerEnvironmentInstructions() {
	Register(Metadata{Opcode: ENV, Mnemonic: "ENV", Category: 13,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "get environment variable (system/tx/validation scope); attr=identifier",
	})
	Register(Metadata{Opcode: IN, Mnemonic: "IN", Category: 13,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "get current input item data (validation scope); attr=identifier",
	})
	Register(Metadata{Opcode: OUT, Mnemonic: "OUT", Category: 13,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0, 0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "get output item data (tx scope); attr1=output index, attr2=identifier",
	})
	Register(Metadata{Opcode: INOUT, Mnemonic: "INOUT", Category: 13,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0, 0}, AssocDataParam: -1,
		Availability: availDisabledPublic, Deterministic: true, CostTier: CostTierExternal,
		Description: "[DISABLED mainnet] get sibling output of current input's source tx; attr1=out idx, attr2=identifier",
	})
	Register(Metadata{Opcode: XFROM, Mnemonic: "XFROM", Category: 13,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "get source tx info (only in GOTO/EMBED target scripts); attr=identifier",
	})
	Register(Metadata{Opcode: SIGNED, Mnemonic: "SIGNED", Category: 13,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "check if pubkey at index signed and was verified; attr=index",
	})
	Register(Metadata{Opcode: VAR, Mnemonic: "VAR", Category: 13,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "get global variable (script scope); attr=position [0-255]",
	})
	Register(Metadata{Opcode: SETVAR, Mnemonic: "SETVAR", Category: 13,
		ArgCount: FixedArgs(1), ReturnCount: 0,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "set global variable (script scope); attr=position [0-255]",
	})
	Register(Metadata{Opcode: SOURCE, Mnemonic: "SOURCE", Category: 13,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierMedium,
		Description: "extract compiled bytecode segment as Bytes; attr=identifier (high bit=cross-block)",
	})
}

// ─── 工具指令 [138-163] 注册 ─────────────────────────────────────────────────

func registerToolInstructions() {
	Register(Metadata{Opcode: EVAL, Mnemonic: "EVAL", Category: 14,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availDisabledPublic, Deterministic: true, CostTier: CostTierHigh,
		Description: "[DISABLED mainnet] execute script in isolated scope; returns stack residue as slice",
	})
	Register(Metadata{Opcode: COPY, Mnemonic: "COPY", Category: 14,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "shallow copy slice to new slice",
	})
	Register(Metadata{Opcode: DCOPY, Mnemonic: "DCOPY", Category: 14,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierMedium,
		Description: "deep copy slice (recursive)",
	})
	Register(Metadata{Opcode: KEYVAL, Mnemonic: "KEYVAL", Category: 14,
		ArgCount: FixedArgs(1), ReturnCount: -1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "extract dict keys/values; attr=0(both),1(keys),2(values); auto-spread",
	})
	Register(Metadata{Opcode: MATCH, Mnemonic: "MATCH", Category: 14,
		ArgCount: FixedArgs(2), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierHigh,
		Description: "regex match; attr=mode(_/g/G/f/S); arg1=target String/Bytes, arg2=*RegExp",
	})
	Register(Metadata{Opcode: SUBSTR, Mnemonic: "SUBSTR", Category: 14,
		ArgCount: FixedArgs(2), ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "substring by rune count; attr=rune count; arg1=string, arg2=start rune pos",
	})
	Register(Metadata{Opcode: REPLACE, Mnemonic: "REPLACE", Category: 14,
		ArgCount: FixedArgs(3), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierHigh,
		Description: "string replace; attr=count(0=all); arg1=target, arg2=pattern, arg3=replacement",
	})
	Register(Metadata{Opcode: RANDOM, Mnemonic: "RANDOM", Category: 14,
		ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierMedium,
		Description: "deterministic random (ChaCha8); arg1=seed(32B), arg2=upper bound (exclusive)",
	})
	Register(Metadata{Opcode: SLRAND, Mnemonic: "SLRAND", Category: 14,
		ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierMedium,
		Description: "deterministic slice shuffle (ChaCha8); arg1=seed(32B), arg2=slice",
	})
	Register(Metadata{Opcode: CMPFLO, Mnemonic: "CMPFLO", Category: 14,
		ArgCount: FixedArgs(3), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "float comparison with epsilon; attr=0(==),-1(<=),1(>=); arg3=epsilon",
	})
	Register(Metadata{Opcode: RANGE, Mnemonic: "RANGE", Category: 14,
		ArgCount: FixedArgs(2), ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "create numeric sequence; attr=count (1 byte, max 255); arg1=start, arg2=step",
	})
	Register(Metadata{Opcode: MAP, Mnemonic: "MAP", Category: 14,
		ArgCount: ArgVariadic, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierHigh,
		Description: "map iteration; independent script scope per sub-block; nil return ignored",
	})
	Register(Metadata{Opcode: FILTER, Mnemonic: "FILTER", Category: 14,
		ArgCount: ArgVariadic, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierHigh,
		Description: "filter iteration; true=keep; returns same type as target",
	})
	Register(Metadata{Opcode: SHELL, Mnemonic: "SHELL", Category: 14,
		ArgCount: ArgVariadic, ReturnCount: 0,
		AttrParamSizes: []int{0}, AssocDataParam: 0,
		// 公共路径忽略执行但消费实参；私有路径执行
		Availability: availPublicPrivate, Deterministic: false, CostTier: CostTierUnknown,
		Description: "execute shell command; public: ignore execution but consume args; NOT disabled",
	})
}

// ─── 系统指令 [164-169] 注册 ─────────────────────────────────────────────────

func registerSystemInstructions() {
	Register(Metadata{Opcode: SYS_TIME, Mnemonic: "SYS_TIME", Category: 15,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{0}, AssocDataParam: -1,
		// 公共路径触达即 ScriptError（特例，非禁用但不确定性）
		Availability: availPrivateOnly, Deterministic: false, CostTier: CostTierFree,
		Description:    "get current time (client real time); public path: ScriptError; only after END",
		ErrorScenarios: []string{"public path before END: ScriptError"},
	})
	Register(Metadata{Opcode: SYS_AWARD, Mnemonic: "SYS_AWARD", Category: 15,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1, 4}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierMedium,
		Description: "award slot verification; only in Coinbase output scripts; attr1=out idx, attr2=block height (4B)",
	})
	Register(Metadata{Opcode: SYS_CHKPASS, Mnemonic: "SYS_CHKPASS", Category: 15,
		// 特例：实参从环境（Witness）提取
		ArgCount: ArgNone, ReturnCount: 0, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierHigh,
		Description:    "address+signature verification (pass instruction); args from witness environment only",
		ErrorScenarios: []string{"witness missing: fail", "address mismatch: VerifyFail"},
	})
	Register(Metadata{Opcode: SYS_NULL, Mnemonic: "SYS_NULL", Category: 15,
		ArgCount: ArgNone, ReturnCount: 0, AssocDataParam: -1,
		// 特例：可在解锁脚本中使用
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "NULL point marker for SOURCE extraction; allowed in unlock script (special case)",
	})
}

// ─── 函数段 [170-224] 注册 ───────────────────────────────────────────────────

func registerFunctionInstructions() {
	// 编码/解码函数
	for _, op := range []struct {
		op   Opcode
		mn   string
		desc string
	}{
		{FN_BASE58, "FN_BASE58", "Base58 encode/decode (Bytes->String or String->Bytes)"},
		{FN_BASE32, "FN_BASE32", "Base32 encode/decode (RFC4648 uppercase, no padding)"},
		{FN_BASE64, "FN_BASE64", "Base64 encode/decode (URL-safe, no padding)"},
	} {
		Register(Metadata{Opcode: op.op, Mnemonic: op.mn, Category: 16,
			ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
			Availability: availAll, Deterministic: true, CostTier: CostTierMedium,
			Description: op.desc,
		})
	}
	Register(Metadata{Opcode: FN_ADDRESS, Mnemonic: "FN_ADDRESS", Category: 16,
		ArgCount: FixedArgs(2), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierMedium,
		Description: "pubkey hash <-> account address (Base58); arg1=data, arg2=prefix",
	})
	Register(Metadata{Opcode: FN_PUBHASH, Mnemonic: "FN_PUBHASH", Category: 16,
		ArgCount: FixedArgs(1), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierHigh,
		Description: "compute single-sig public key hash from raw public key",
	})
	Register(Metadata{Opcode: FN_MPUBHASH, Mnemonic: "FN_MPUBHASH", Category: 16,
		ArgCount: FixedArgs(3), ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierHigh,
		Description: "create multi-sig composite pubkey hash; arg1=[]baseHash, arg2=m, arg3=n",
	})
	Register(Metadata{Opcode: FN_CHECKSIG, Mnemonic: "FN_CHECKSIG", Category: 16,
		ArgCount: FixedArgs(4), ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierHigh,
		Description: "single-sig verification (custom path); marks system layer on success",
	})
	Register(Metadata{Opcode: FN_MCHECKSIG, Mnemonic: "FN_MCHECKSIG", Category: 16,
		ArgCount: FixedArgs(5), ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierHigh,
		Description: "multi-sig verification (custom path); marks system layer on success",
	})
	// 哈希函数（多算法）
	for _, op := range []struct {
		op   Opcode
		mn   string
		bits int
	}{
		{FN_HASH224, "FN_HASH224", 224},
		{FN_HASH256, "FN_HASH256", 256},
		{FN_HASH384, "FN_HASH384", 384},
		{FN_HASH512, "FN_HASH512", 512},
	} {
		Register(Metadata{Opcode: op.op, Mnemonic: op.mn, Category: 16,
			ArgCount: FixedArgs(1), ReturnCount: 1,
			AttrParamSizes: []int{1}, AssocDataParam: -1,
			Availability: availAll, Deterministic: true, CostTier: CostTierHigh,
			Description: fmt.Sprintf("%d-bit hash; attr=algorithm (0=BLAKE3,1=BLAKE2,2=SHA2,3=SHA3)", op.bits),
		})
	}
	Register(Metadata{Opcode: FN_PRINTF, Mnemonic: "FN_PRINTF", Category: 16,
		ArgCount: ArgVariadic, ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: false, CostTier: CostTierFree,
		Description: "formatted print (fmt.Printf); returns nil",
	})
	Register(Metadata{Opcode: FN_X, Mnemonic: "FN_X", Category: 16,
		ArgCount: ArgVariadic, ReturnCount: -1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierUnknown,
		Description: "standard function reference; attr=target index (0-255); officially defined only",
	})
}

// ─── 模块段 [225-250] 注册 ───────────────────────────────────────────────────

func registerModuleInstructions() {
	Register(Metadata{Opcode: MO_MATH, Mnemonic: "MO_MATH", Category: 17,
		ArgCount: ArgNone, ReturnCount: 1, AssocDataParam: -1,
		Availability: availAll, Deterministic: true, CostTier: CostTierFree,
		Description: "create math module object (Abs/Ceil/Floor/Pow/Max/Min/Mod/...); call via CALL",
	})
	Register(Metadata{Opcode: MO_FMT, Mnemonic: "MO_FMT", Category: 17,
		ArgCount: ArgNone, ReturnCount: 1, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierFree,
		Description: "create fmt module object (Sprint/Sprintf/Sprintln); call via CALL",
	})
	Register(Metadata{Opcode: MO_XX, Mnemonic: "MO_XX", Category: 17,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{1}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierUnknown,
		Description: "standard module reference; attr=target index (0-255); officially defined only",
	})
}

// ─── 扩展段 [251-253] 注册 ───────────────────────────────────────────────────

func registerExtensionInstructions() {
	Register(Metadata{Opcode: EXT_MO, Mnemonic: "EXT_MO", Category: 18,
		ArgCount: ArgNone, ReturnCount: 1,
		AttrParamSizes: []int{2}, AssocDataParam: -1,
		Availability: availPublicPrivate, Deterministic: true, CostTier: CostTierUnknown,
		Description: "extended module reference; attr=target index (2 bytes, 64K space)",
	})
	Register(Metadata{Opcode: EXT_PRIV, Mnemonic: "EXT_PRIV", Category: 18,
		ArgCount: ArgNone, ReturnCount: 0,
		AttrParamSizes: []int{2}, AssocDataParam: -1,
		// 公共路径触达即 ScriptError
		Availability: availPrivateOnly, Deterministic: false, CostTier: CostTierUnknown,
		Description:    "private extension; public path before END/INPUT: ScriptError (DEC-0505)",
		ErrorScenarios: []string{"public path: ScriptError"},
	})
}

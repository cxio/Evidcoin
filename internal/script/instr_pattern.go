package script

// instr_pattern.go 实现模式指令 [116-127] 的执行函数。
// 参考：docs/proposal/Instruction/12.Pattern-Instructions.md。
// 当前阶段所有模式指令统一占位返回 ErrModelOutside，后续专门任务实现。

func init() {
	registerExec(MODEL, execMODEL)
	registerExec(WILDCARD, execWILDCARD)
	registerExec(WILDCARDS, execWILDCARDS)
	registerExec(OPTIONAL, execOPTIONAL)
	registerExec(MATCHMOD, execMATCHMOD)
	registerExec(EXTRACT, execEXTRACT)
	registerExec(TYPEMATCH, execTYPEMATCH)
	registerExec(INTRANGE, execINTRANGE)
	registerExec(FLORANGE, execFLORANGE)
	registerExec(REMATCH, execREMATCH)
	registerExec(REGEXTRACT, execREGEXTRACT)
	registerExec(ELLIPSIS, execELLIPSIS)
}

// execMODEL 开启模式匹配子环境（MODEL，opcode 116）。占位：返回 ErrModelOutside。
func execMODEL(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execWILDCARD 单指令通配 _（WILDCARD，opcode 117，仅 MODEL 内）。
func execWILDCARD(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execWILDCARDS 指令段通配 _[n]（WILDCARDS，opcode 118，仅 MODEL 内）。
func execWILDCARDS(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execOPTIONAL 序列可选 ?{}（OPTIONAL，opcode 119，仅 MODEL 内）。
func execOPTIONAL(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execMATCHMOD 通配/可选指示 ^?（MATCHMOD，opcode 120，仅 MODEL 内）。
func execMATCHMOD(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execEXTRACT 取值指令 #（EXTRACT，opcode 121，仅 MODEL 内）。
func execEXTRACT(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execTYPEMATCH 类型匹配 !?（TYPEMATCH，opcode 122，仅 MODEL 内）。
func execTYPEMATCH(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execINTRANGE 整数范围匹配 >{}（INTRANGE，opcode 123，仅 MODEL 内）。
func execINTRANGE(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execFLORANGE 浮点范围匹配（FLORANGE，opcode 124，仅 MODEL 内）。
func execFLORANGE(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execREMATCH 正则匹配查找 RE{}（REMATCH，opcode 125，仅 MODEL 内）。
func execREMATCH(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execREGEXTRACT 正则取值 &（REGEXTRACT，opcode 126，仅 MODEL 内）。
func execREGEXTRACT(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

// execELLIPSIS 任意连续指令段通配 ...（ELLIPSIS，opcode 127，仅 MODEL 内）。
func execELLIPSIS(vm *VM, _ *InstrFrame) error { return ErrModelOutside }

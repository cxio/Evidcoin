package script

// instr_result.go 实现结果指令 [51-57] 的执行函数。
// 参考：docs/proposal/Instruction/6.Result-Instructions.md，DEC-0505。

func init() {
	registerExec(PASS, execPASS)
	registerExec(CHECK, execCHECK)
	// GOTO(53) 和 EMBED(54) 涉及跨脚本跳转（外部引用），在 Task 6 完整实现；
	// 此处注册占位函数，确保执行时返回 ErrGotoTargetMissing（ScriptError）。
	registerExec(GOTO, execGOTO)
	registerExec(EMBED, execEMBED)
	registerExec(EXIT, execEXIT)
	registerExec(RETURN, execRETURN)
	registerExec(END, execEND)
}

// execPASS 写入通关状态：PASS true 继续；PASS false 立即 VerifyFail（DEC-0505）。
func execPASS(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	b, err := v.AsBool()
	if err != nil {
		return err
	}
	if !b {
		vm.state = StateVerifyFail
		return nil
	}
	// PASS true：更新通关状态为 true，继续执行
	vm.passState = true
	return nil
}

// execCHECK 写入通关状态但不终止；后写覆盖前值（DEC-0505）。
// CHECK true/false 均继续执行，最终由 END 或脚本结尾读取 passState。
func execCHECK(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	b, err := v.AsBool()
	if err != nil {
		return err
	}
	vm.passState = b
	return nil
}

// execGOTO 跳转到目标输出脚本（跨脚本，附参=年度+TxID+输出序位）。
// 完整实现需要外部脚本解析器；当前占位返回 ErrGotoTargetMissing。
func execGOTO(vm *VM, _ *InstrFrame) error {
	return ErrGotoTargetMissing
}

// execEMBED 嵌入目标输出脚本（共享当前域，附参同 GOTO）。
// 完整实现需要外部脚本解析器；当前占位返回 ErrGotoTargetMissing。
func execEMBED(vm *VM, _ *InstrFrame) error {
	return ErrGotoTargetMissing
}

// execEXIT 终止脚本，检查通关状态（实参可选返回值，当前忽略返回值）。
func execEXIT(vm *VM, _ *InstrFrame) error {
	// 可选：读取实参区的返回值（若有）
	vm.args.Clear()
	vm.state = StatePassStop
	return nil
}

// execRETURN 从函数子块退出并返回值（仅用于 MAP/FILTER 子块）。
// 当前占位为 PassStop；完整实现在 Task 7（集合指令）中扩展。
func execRETURN(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	vm.state = StatePassStop
	return nil
}

// execEND 公共验证逻辑结束（DEC-0505）。
// 以当前通关状态产生 PassStop；之后的帧为私有路径，不在公共路径执行。
func execEND(vm *VM, _ *InstrFrame) error {
	vm.MarkPublicEnd()
	vm.state = StatePassStop
	return nil
}

package script

// instr_interaction.go 实现交互指令 [46-50] 的执行函数。
// 参考：docs/proposal/Instruction/5.Interaction-Instructions.md，DEC-0503/0505。

func init() {
	registerExec(INPUT, execINPUT)
	registerExec(OUTPUT, execOUTPUT)
	registerExec(BUFDUMP, execBUFDUMP)
	// opcode 49 保留，不注册
	registerExec(PRINT, execPRINT)
}

// execINPUT 从 INPUT 缓冲区导入数据。
// 公共验证节点视为隐式 END（以当前通关状态产生 PassStop，DEC-0503/0505）。
// 私有路径：从 vm.inputBuf 取一个值压栈；缓冲区为空时返回 ErrInputEmpty。
func execINPUT(vm *VM, _ *InstrFrame) error {
	if vm.mode == ModePublic {
		// 公共路径：视为 END，产生 PassStop
		vm.state = StatePassStop
		return nil
	}
	// 私有路径：从缓冲区取值
	if len(vm.inputBuf) == 0 {
		return ErrInputEmpty
	}
	v := vm.inputBuf[0]
	vm.inputBuf = vm.inputBuf[1:]
	return vm.stack.Push(v)
}

// execOUTPUT 把实参区数据导出到 OUTPUT 缓冲区（非阻塞）。
// executor.go 会在指令后清空实参区，此处先读取再导出。
func execOUTPUT(vm *VM, _ *InstrFrame) error {
	items := vm.args.Items()
	vm.args.Clear()
	vm.outputBuf = append(vm.outputBuf, items...)
	return nil
}

// execBUFDUMP 转出 OUTPUT 缓冲区到外部环境（附参=标识值）。
// 当前占位实现：清空 OUTPUT 缓冲区并记录标识。
func execBUFDUMP(vm *VM, _ *InstrFrame) error {
	// 占位：清空缓冲区（完整实现在 Task 6 注入外部监听器后扩展）
	vm.outputBuf = vm.outputBuf[:0]
	return nil
}

// execPRINT 打印实参区内容到控制台（开发/调试用）。
// 公共路径同样执行（消费实参，不产生副作用到共识数据），私有路径正常打印。
func execPRINT(vm *VM, _ *InstrFrame) error {
	// 仅消费实参区（不实际输出到 stdout，避免测试噪声；完整实现可注入 io.Writer）
	vm.args.Clear()
	return nil
}

package script

// instr_capture.go 实现截取指令 [19-23] 的执行函数。
// 参考：docs/proposal/Instruction/2.Capture-Instructions.md，DEC-0501。

func init() {
	registerExec(AT, execAT)
	registerExec(SAVE, execSAVE)
	registerExec(LOCAL, execLOCAL)
	registerExec(LOOPVAR, execLOOPVAR)
	registerExec(DIRECT, execDIRECT)
}

// execAT 拦截后一条指令的返回值放入实参区（AT/@，opcode 19）。
// 占位：从栈顶取值放入实参区（完整实现需要 Task 9 执行循环改造）。
func execAT(vm *VM, _ *InstrFrame) error {
	v, err := vm.stack.Pop()
	if err != nil {
		return err
	}
	vm.args.Enqueue(v)
	return nil
}

// execSAVE 截取后一条指令返回值存入局部域（SAVE/$，opcode 20）。
// 占位：弹出栈顶值（局部域堆叠在 Task 9 中实现）。
func execSAVE(vm *VM, _ *InstrFrame) error {
	_, err := vm.stack.Pop()
	return err
}

// execLOCAL 从局部域取值放入实参区（LOCAL/$[n]，opcode 21）。
// 附参=下标。占位：压入 NilValue（局部域未完整实现）。
func execLOCAL(vm *VM, _ *InstrFrame) error {
	vm.args.Enqueue(NilValue())
	return nil
}

// execLOOPVAR 循环变量引用（LOOPVAR/$X，opcode 22）。
// 附参=变量标识[0-3]。占位：压入 NilValue。
func execLOOPVAR(vm *VM, _ *InstrFrame) error {
	vm.args.Enqueue(NilValue())
	return nil
}

// execDIRECT 指示后一指令从数据栈直接取参，跳过实参区（DIRECT/~，opcode 23）。
// 占位：无操作（需要 Task 9 执行循环改造）。
func execDIRECT(vm *VM, _ *InstrFrame) error {
	return nil
}

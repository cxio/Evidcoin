package script

// instr_module.go 实现模块指令 [225-250] 的执行函数。
// 参考：docs/proposal/Instruction/17.Module-Instructions.md。
// 当前阶段模块对象压入 NilValue 占位；完整模块系统在后续任务实现。

func init() {
	registerExec(MO_MATH, execMO_MATH)
	registerExec(MO_FMT, execMO_FMT)
	// opcode 227-249 不注册（保留未分配）
	registerExec(MO_XX, execMO_XX)
}

// execMO_MATH 数学运算模块对象（MO_MATH，opcode 225）。
// 占位：压入 NilValue（完整实现压入 TypeObject 标识符）。
func execMO_MATH(vm *VM, _ *InstrFrame) error {
	return vm.stack.Push(NilValue())
}

// execMO_FMT 数据格式化模块（MO_FMT，opcode 226）。
// 占位：压入 NilValue。
func execMO_FMT(vm *VM, _ *InstrFrame) error {
	return vm.stack.Push(NilValue())
}

// execMO_XX 标准模块引用（MO_XX，opcode 250）。
// 附参=目标索引(1字节)。占位：压入 NilValue。
func execMO_XX(vm *VM, _ *InstrFrame) error {
	return vm.stack.Push(NilValue())
}

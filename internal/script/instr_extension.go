package script

// instr_extension.go 实现扩展指令 [251-253] 的执行函数。
// 参考：docs/proposal/Instruction/18.Extension-Instructions.md。
// opcode 252 保留不注册。

func init() {
	registerExec(EXT_MO, execEXT_MO)
	// opcode 252 不注册（保留）
	registerExec(EXT_PRIV, execEXT_PRIV)
}

// execEXT_MO 扩展模块包（EXT_MO，opcode 251）。
// 附参=目标索引(2字节)。占位：压入 NilValue。
func execEXT_MO(vm *VM, _ *InstrFrame) error {
	return vm.stack.Push(NilValue())
}

// execEXT_PRIV 私有扩展包（EXT_PRIV，opcode 253）。
// 公共路径已由 checkPublicSafety 拦截返回 ErrExtPrivInPublic（ScriptError）。
// 私有路径：占位，压入 NilValue。
func execEXT_PRIV(vm *VM, _ *InstrFrame) error {
	return vm.stack.Push(NilValue())
}

package script

import "fmt"

// instr_system.go 实现系统指令 [164-169] 的执行函数。
// 参考：docs/proposal/Instruction/15.System-Instructions.md，DEC-0503/0505。

func init() {
	registerExec(SYS_TIME, execSYS_TIME)
	registerExec(SYS_AWARD, execSYS_AWARD)
	registerExec(SYS_CHKPASS, execSYS_CHKPASS)
	// opcode 167、168 保留，不注册
	registerExec(SYS_NULL, execSYS_NULL)
}

// execSYS_TIME 取全局时间（SYS_TIME，opcode 164）。
// 禁止在公共验证路径（END 之前）使用（DEC-0505）。
func execSYS_TIME(vm *VM, f *InstrFrame) error {
	// END 之前的公共路径不允许使用 SYS_TIME
	if !vm.PassedPublicEnd() {
		return ErrSysTimeInPublic
	}
	if vm.env == nil {
		return vm.stack.Push(NilValue())
	}
	// 从环境取时间值；附参标识用于未来扩展
	var attrParam uint64
	if len(f.AttrParams) > 0 {
		attrParam = readULEB128Param(f.AttrParams[0])
	}
	key := fmt.Sprintf("sys.time.%d", attrParam)
	v, err := vm.env.Lookup(key)
	if err != nil {
		return vm.stack.Push(NilValue())
	}
	return vm.stack.Push(v)
}

// execSYS_AWARD 兑奖验算（SYS_AWARD，opcode 165）。
// 仅 Coinbase 交易输出脚本可用（DEC-0503）。
// 当前占位实现：压入 BoolValue(true)；完整逻辑在 Task 8 或更高阶段实现。
func execSYS_AWARD(vm *VM, _ *InstrFrame) error {
	if vm.witness == nil || !vm.witness.IsCoinbase() {
		return ErrNotInCoinbase
	}
	// 占位：兑奖验算通过
	return vm.stack.Push(BoolValue(true))
}

// execSYS_CHKPASS 系统通关验证（SYS_CHKPASS，opcode 166）。
// 地址核实+签名验证，也是通关指令（DEC-0505）。
// 当前占位实现：有见证数据则通关，无则失败。
func execSYS_CHKPASS(vm *VM, _ *InstrFrame) error {
	if vm.witness == nil {
		return ErrWitnessMissing
	}
	data := vm.witness.GetWitness()
	if len(data) == 0 {
		// 空见证数据：验证失败
		vm.state = StateVerifyFail
		return nil
	}
	// 占位：见证数据存在，标注通关
	vm.SetSigned(0)
	vm.passState = true
	return nil
}

// execSYS_NULL 源码零点标识（SYS_NULL，opcode 169）。
// 仅标记 NULL 点，当前占位为无操作。
func execSYS_NULL(vm *VM, _ *InstrFrame) error {
	// 占位：无操作（NULL 点游标在 SOURCE 功能实现时扩展）
	return nil
}

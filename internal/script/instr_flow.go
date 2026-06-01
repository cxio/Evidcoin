package script

// instr_flow.go 实现流程控制指令 [58-66] 的执行函数。
// 参考：docs/proposal/Instruction/7.Flow-Control.md。
// 注意：GOTO(53) 和 EMBED(54) 已在 instr_result.go 中注册，此处不重复注册。

func init() {
	registerExec(IF, execIF)
	registerExec(ELSE, execELSE)
	registerExec(SWITCH, execSWITCH)
	registerExec(CASE, execCASE)
	registerExec(DEFAULT, execDEFAULT)
	registerExec(EACH, execEACH)
	registerExec(CONTINUE, execCONTINUE)
	registerExec(BREAK, execBREAK)
	registerExec(BLOCK, execBLOCK)
}

// execIF 条件为真时执行关联子块（IF，opcode 58）。
func execIF(vm *VM, f *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	b, err := v.AsBool()
	if err != nil {
		return err
	}
	if b && len(f.AssocData) > 0 {
		subFrames, err := DecodeScript(f.AssocData, 0)
		if err != nil {
			return err
		}
		for i := range subFrames {
			if vm.state.IsDone() {
				break
			}
			vm.executeFrame(&subFrames[i])
		}
	}
	return nil
}

// execELSE 若最近 IF 为假则执行关联子块（ELSE，opcode 59）。
// 占位：无 IF 状态追踪，直接跳过。
func execELSE(vm *VM, _ *InstrFrame) error {
	return nil
}

// execSWITCH 多路选择（SWITCH，opcode 60）。
// 占位：消费实参，子块分发逻辑待 Task 9 实现。
func execSWITCH(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return nil
}

// execCASE 匹配分支（CASE，opcode 61）。
// 占位：消费实参。
func execCASE(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return nil
}

// execDEFAULT 默认分支（DEFAULT，opcode 62）。
// 占位：消费实参。
func execDEFAULT(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return nil
}

// execEACH 有限迭代，对 Slice 每个成员执行子块（EACH，opcode 63）。
func execEACH(vm *VM, f *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	for _, item := range sl {
		if vm.state.IsDone() {
			break
		}
		_ = item // 循环变量绑定由 LOOPVAR 实现，当前占位
		if len(f.AssocData) > 0 {
			subFrames, err2 := DecodeScript(f.AssocData, 0)
			if err2 != nil {
				return err2
			}
			for i := range subFrames {
				if vm.state.IsDone() {
					break
				}
				vm.executeFrame(&subFrames[i])
			}
		}
	}
	return nil
}

// execCONTINUE 终止当前 EACH 迭代（CONTINUE，opcode 64）。
// 占位：消费实参（需执行循环异常控制）。
func execCONTINUE(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return nil
}

// execBREAK 退出 EACH/SWITCH 块（BREAK，opcode 65）。
// 占位：消费实参（需执行循环异常控制）。
func execBREAK(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return nil
}

// execBLOCK 创建子块并执行（BLOCK，opcode 66），关联数据为子块字节码。
func execBLOCK(vm *VM, f *InstrFrame) error {
	if len(f.AssocData) > 0 {
		subFrames, err := DecodeScript(f.AssocData, 0)
		if err != nil {
			return err
		}
		for i := range subFrames {
			if vm.state.IsDone() {
				break
			}
			vm.executeFrame(&subFrames[i])
		}
	}
	return nil
}

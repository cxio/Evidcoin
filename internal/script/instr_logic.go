package script

import "fmt"

// instr_logic.go 实现逻辑指令 [112-115] 的执行函数。
// 参考：docs/proposal/Instruction/11.Logic-Instructions.md

func init() {
	registerExec(BOTH, execBOTH)
	registerExec(EITHER, execEITHER)
	registerExec(EVERY, execEVERY)
	registerExec(SOME, execSOME)
}

// execBOTH 逻辑 AND，取2 Bool 实参（BOTH，opcode 112）。
func execBOTH(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, err := args[0].AsBool()
	if err != nil {
		return err
	}
	b, err := args[1].AsBool()
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(a && b))
}

// execEITHER 逻辑 OR，取2 Bool 实参（EITHER，opcode 113）。
func execEITHER(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, err := args[0].AsBool()
	if err != nil {
		return err
	}
	b, err := args[1].AsBool()
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(a || b))
}

// execEVERY 逻辑 AND 集合（EVERY，opcode 114）。
// 取1 Slice 实参，所有成员为 Bool true → true；空Slice→true；有非Bool→ErrTypeMismatch。
func execEVERY(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	for _, item := range sl {
		b, err := item.AsBool()
		if err != nil {
			return fmt.Errorf("%w: EVERY requires all Bool elements", ErrTypeMismatch)
		}
		if !b {
			return vm.stack.Push(BoolValue(false))
		}
	}
	return vm.stack.Push(BoolValue(true))
}

// execSOME 逻辑 OR 集合（SOME，opcode 115）。
// 附参=n（ULEB128），取1 Slice 实参，至少n个为true → true；n=0恒true；空Slice且n>0→false。
func execSOME(vm *VM, f *InstrFrame) error {
	var n uint64
	if len(f.AttrParams) > 0 {
		n = readULEB128Param(f.AttrParams[0])
	}
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	if n == 0 {
		return vm.stack.Push(BoolValue(true))
	}
	var count uint64
	for _, item := range sl {
		b, err := item.AsBool()
		if err != nil {
			return fmt.Errorf("%w: SOME requires all Bool elements", ErrTypeMismatch)
		}
		if b {
			count++
			if count >= n {
				return vm.stack.Push(BoolValue(true))
			}
		}
	}
	return vm.stack.Push(BoolValue(false))
}

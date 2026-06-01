package script

// instr_stack.go 实现栈操作指令 [24-34] 的执行函数。
// 参考：docs/proposal/Instruction/3.Stack-Operations.md

func init() {
	registerExec(NOP, execNOP)
	registerExec(PUSH, execPUSH)
	registerExec(POP, execPOP)
	registerExec(POPS, execPOPS)
	registerExec(TOP, execTOP)
	registerExec(TOPS, execTOPS)
	registerExec(PEEK, execPEEK)
	registerExec(PEEKS, execPEEKS)
	registerExec(SHIFT, execSHIFT)
	registerExec(CLONE, execCLONE)
	registerExec(VIEW, execVIEW)
}

// execNOP 无操作，读取（清空）实参区。
// executor.go 的 executeFrame 已在指令执行后清空实参区，故此处只作占位。
func execNOP(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return nil
}

// execPUSH 将实参区中的所有值按 FIFO 顺序压入数据栈。
func execPUSH(vm *VM, _ *InstrFrame) error {
	items := vm.args.Items()
	vm.args.Clear()
	for _, v := range items {
		if err := vm.stack.Push(v); err != nil {
			return err
		}
	}
	return nil
}

// execPOP 弹出栈顶项（丢弃）。
func execPOP(vm *VM, _ *InstrFrame) error {
	_, err := vm.stack.Pop()
	return err
}

// execPOPS 弹出栈顶 n 个项（附参=n，0=全部）。
func execPOPS(vm *VM, f *InstrFrame) error {
	n := int(f.AttrParams[0][0])
	if n == 0 {
		vm.stack.Clear()
		return nil
	}
	for i := 0; i < n; i++ {
		if _, err := vm.stack.Pop(); err != nil {
			return err
		}
	}
	return nil
}

// execTOP 引用（复制）栈顶项并压入栈顶（等效于 DUP）。
func execTOP(vm *VM, _ *InstrFrame) error {
	v, err := vm.stack.Top()
	if err != nil {
		return err
	}
	return vm.stack.Push(v)
}

// execTOPS 引用栈顶 n 个项（附参=n，0=全部），打包为切片压栈。
func execTOPS(vm *VM, f *InstrFrame) error {
	n := int(f.AttrParams[0][0])
	total := vm.stack.Len()
	if n == 0 {
		n = total
	}
	if n > total {
		return ErrStackUnderflow
	}
	items := vm.stack.Items()
	slice := make([]Value, n)
	copy(slice, items[total-n:])
	return vm.stack.Push(SliceValue(slice))
}

// execPEEK 引用栈内任意位置的值并压栈（实参=位置，0=栈底，负数从栈顶）。
func execPEEK(vm *VM, _ *InstrFrame) error {
	posVal, err := vm.getOneArg()
	if err != nil {
		return err
	}
	pos, err := posVal.AsInt()
	if err != nil {
		return err
	}
	v, err := vm.stack.Peek(int(pos))
	if err != nil {
		return err
	}
	return vm.stack.Push(v)
}

// execPEEKS 引用栈内任意位置段（附参=条目数，实参=起始位置），打包为切片压栈。
func execPEEKS(vm *VM, f *InstrFrame) error {
	n := int(f.AttrParams[0][0])
	posVal, err := vm.getOneArg()
	if err != nil {
		return err
	}
	pos, err := posVal.AsInt()
	if err != nil {
		return err
	}
	slice := make([]Value, n)
	for i := 0; i < n; i++ {
		v, err := vm.stack.Peek(int(pos) + i)
		if err != nil {
			return err
		}
		slice[i] = v
	}
	return vm.stack.Push(SliceValue(slice))
}

// execSHIFT 移出栈顶 n 个条目（附参=n，0=全部）打包为切片压栈。
func execSHIFT(vm *VM, f *InstrFrame) error {
	n := int(f.AttrParams[0][0])
	total := vm.stack.Len()
	if n == 0 {
		n = total
	}
	if n > total {
		return ErrStackUnderflow
	}
	slice := make([]Value, n)
	for i := n - 1; i >= 0; i-- {
		v, _ := vm.stack.Pop()
		slice[i] = v
	}
	return vm.stack.Push(SliceValue(slice))
}

// execCLONE 克隆栈顶 n 个条目（附参=n，0=全部）打包为切片压栈（不移出原值）。
func execCLONE(vm *VM, f *InstrFrame) error {
	n := int(f.AttrParams[0][0])
	total := vm.stack.Len()
	if n == 0 {
		n = total
	}
	if n > total {
		return ErrStackUnderflow
	}
	items := vm.stack.Items()
	slice := make([]Value, n)
	copy(slice, items[total-n:])
	return vm.stack.Push(SliceValue(slice))
}

// execVIEW 引用栈条目打包为切片（附参=条目数，实参=起始位置），压栈（不移出原值）。
func execVIEW(vm *VM, f *InstrFrame) error {
	n := int(f.AttrParams[0][0])
	posVal, err := vm.getOneArg()
	if err != nil {
		return err
	}
	pos, err := posVal.AsInt()
	if err != nil {
		return err
	}
	slice := make([]Value, n)
	for i := 0; i < n; i++ {
		v, err := vm.stack.Peek(int(pos) + i)
		if err != nil {
			return err
		}
		slice[i] = v
	}
	return vm.stack.Push(SliceValue(slice))
}

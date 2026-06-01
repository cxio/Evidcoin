package script

import "math"

// instr_arithmetic.go 实现运算指令 [80-103] 的执行函数。
// 参考：docs/proposal/Instruction/9.Arithmetic-Instructions.md。

func init() {
	// EXPR 和表达式内符号指令（81-85）占位压入 NilValue
	registerExec(EXPR, execEXPR)
	registerExec(MUL_OP, execMUL_OP)
	registerExec(DIV_OP, execDIV_OP)
	registerExec(MOD_OP, execMOD_OP)
	registerExec(ADD_OP, execADD_OP)
	registerExec(SUB_OP, execSUB_OP)
	// 完整实现的运算指令
	registerExec(MUL, execMUL)
	registerExec(DIV, execDIV)
	registerExec(ADD, execADD)
	registerExec(SUB, execSUB)
	registerExec(MOD, execMOD)
	registerExec(POW, execPOW)
	registerExec(LMOV, execLMOV)
	registerExec(RMOV, execRMOV)
	registerExec(AND, execAND)
	registerExec(ANDX, execANDX)
	registerExec(OR, execOR)
	registerExec(XOR, execXOR)
	registerExec(NEG, execNEG)
	registerExec(NOT, execNOT)
	registerExec(DIVMOD, execDIVMOD)
	registerExec(REP, execREP)
	registerExec(DEL, execDEL)
	registerExec(CLEAR, execCLEAR)
}

// ─── 占位指令（表达式内使用）────────────────────────────────────────────────

// execEXPR 独立运算表达式块（EXPR，opcode 80）。占位：压入 NilValue。
func execEXPR(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return vm.stack.Push(NilValue())
}

// execMUL_OP 表达式内乘号（MUL_OP，opcode 81）。占位。
func execMUL_OP(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return vm.stack.Push(NilValue())
}

// execDIV_OP 表达式内除号（DIV_OP，opcode 82）。占位。
func execDIV_OP(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return vm.stack.Push(NilValue())
}

// execMOD_OP 表达式内模号（MOD_OP，opcode 83）。占位。
func execMOD_OP(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return vm.stack.Push(NilValue())
}

// execADD_OP 表达式内加号（ADD_OP，opcode 84）。占位。
func execADD_OP(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return vm.stack.Push(NilValue())
}

// execSUB_OP 表达式内减号（SUB_OP，opcode 85）。占位。
func execSUB_OP(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return vm.stack.Push(NilValue())
}

// ─── 完整实现的运算指令 ────────────────────────────────────────────────────

// execMUL 乘法（MUL，opcode 86）：双实参 Int/Float。
func execMUL(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, b := args[0], args[1]
	if a.typ == TypeInt && b.typ == TypeInt {
		x, _ := a.AsInt()
		y, _ := b.AsInt()
		return vm.stack.Push(IntValue(x * y))
	}
	af, err := toFloat(a)
	if err != nil {
		return err
	}
	bf, err := toFloat(b)
	if err != nil {
		return err
	}
	return vm.stack.Push(FloatValue(af * bf))
}

// execDIV 除法（DIV，opcode 87）：双实参 Int/Float；Int 除零返回 ErrTypeMismatch。
func execDIV(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, b := args[0], args[1]
	if a.typ == TypeInt && b.typ == TypeInt {
		x, _ := a.AsInt()
		y, _ := b.AsInt()
		if y == 0 {
			return ErrTypeMismatch
		}
		return vm.stack.Push(IntValue(x / y))
	}
	af, err := toFloat(a)
	if err != nil {
		return err
	}
	bf, err := toFloat(b)
	if err != nil {
		return err
	}
	// Float 除零保留 ±Inf（DEC-0502）
	return vm.stack.Push(FloatValue(af / bf))
}

// execADD 加法（ADD，opcode 88）：Int/Float/String/Bytes；Dict 占位 NilValue。
func execADD(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, b := args[0], args[1]
	switch {
	case a.typ == TypeInt && b.typ == TypeInt:
		x, _ := a.AsInt()
		y, _ := b.AsInt()
		return vm.stack.Push(IntValue(x + y))
	case a.typ == TypeString && b.typ == TypeString:
		s1, _ := a.AsString()
		s2, _ := b.AsString()
		return vm.stack.Push(StringValue(s1 + s2))
	case a.typ == TypeBytes && b.typ == TypeBytes:
		b1, _ := a.AsBytes()
		b2, _ := b.AsBytes()
		merged := make([]byte, len(b1)+len(b2))
		copy(merged, b1)
		copy(merged[len(b1):], b2)
		return vm.stack.Push(BytesValue(merged))
	case a.typ == TypeDict || b.typ == TypeDict:
		// Dict 合并占位
		return vm.stack.Push(NilValue())
	default:
		af, err := toFloat(a)
		if err != nil {
			return err
		}
		bf, err := toFloat(b)
		if err != nil {
			return err
		}
		return vm.stack.Push(FloatValue(af + bf))
	}
}

// execSUB 减法（SUB，opcode 89）：双实参 Int/Float。
func execSUB(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, b := args[0], args[1]
	if a.typ == TypeInt && b.typ == TypeInt {
		x, _ := a.AsInt()
		y, _ := b.AsInt()
		return vm.stack.Push(IntValue(x - y))
	}
	af, err := toFloat(a)
	if err != nil {
		return err
	}
	bf, err := toFloat(b)
	if err != nil {
		return err
	}
	return vm.stack.Push(FloatValue(af - bf))
}

// execMOD 取模（MOD，opcode 90）：双实参 Int/Float（math.Mod）。
func execMOD(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, b := args[0], args[1]
	if a.typ == TypeInt && b.typ == TypeInt {
		x, _ := a.AsInt()
		y, _ := b.AsInt()
		if y == 0 {
			return ErrTypeMismatch
		}
		return vm.stack.Push(IntValue(x % y))
	}
	af, err := toFloat(a)
	if err != nil {
		return err
	}
	bf, err := toFloat(b)
	if err != nil {
		return err
	}
	return vm.stack.Push(FloatValue(math.Mod(af, bf)))
}

// execPOW 幂运算（POW，opcode 91）：双实参 Int/Float。
func execPOW(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, b := args[0], args[1]
	if a.typ == TypeInt && b.typ == TypeInt {
		base, _ := a.AsInt()
		exp, _ := b.AsInt()
		result, err := intPow(base, exp)
		if err != nil {
			return err
		}
		return vm.stack.Push(IntValue(result))
	}
	af, err := toFloat(a)
	if err != nil {
		return err
	}
	bf, err := toFloat(b)
	if err != nil {
		return err
	}
	return vm.stack.Push(FloatValue(math.Pow(af, bf)))
}

// execLMOV 左移（LMOV，opcode 92）：双 Int 实参，y 须 >=0 && <64。
func execLMOV(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	x, err := args[0].AsInt()
	if err != nil {
		return err
	}
	y, err := args[1].AsInt()
	if err != nil {
		return err
	}
	if y < 0 || y >= 64 {
		return ErrTypeMismatch
	}
	return vm.stack.Push(IntValue(x << uint(y)))
}

// execRMOV 右移（RMOV，opcode 93）：双 Int 实参，y 须 >=0 && <64。
func execRMOV(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	x, err := args[0].AsInt()
	if err != nil {
		return err
	}
	y, err := args[1].AsInt()
	if err != nil {
		return err
	}
	if y < 0 || y >= 64 {
		return ErrTypeMismatch
	}
	return vm.stack.Push(IntValue(x >> uint(y)))
}

// execAND 位与（AND，opcode 94）：双 Int 实参。
func execAND(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	x, err := args[0].AsInt()
	if err != nil {
		return err
	}
	y, err := args[1].AsInt()
	if err != nil {
		return err
	}
	return vm.stack.Push(IntValue(x & y))
}

// execANDX 位清空 &^（ANDX，opcode 95）：双 Int 实参。
func execANDX(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	x, err := args[0].AsInt()
	if err != nil {
		return err
	}
	y, err := args[1].AsInt()
	if err != nil {
		return err
	}
	return vm.stack.Push(IntValue(x &^ y))
}

// execOR 位或（OR，opcode 96）：双 Int 实参。
func execOR(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	x, err := args[0].AsInt()
	if err != nil {
		return err
	}
	y, err := args[1].AsInt()
	if err != nil {
		return err
	}
	return vm.stack.Push(IntValue(x | y))
}

// execXOR 位异或（XOR，opcode 97）：双 Int 实参。
func execXOR(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	x, err := args[0].AsInt()
	if err != nil {
		return err
	}
	y, err := args[1].AsInt()
	if err != nil {
		return err
	}
	return vm.stack.Push(IntValue(x ^ y))
}

// execNEG 取反（NEG，opcode 98）：单实参 Int/Float。
func execNEG(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.typ {
	case TypeInt:
		x, _ := v.AsInt()
		return vm.stack.Push(IntValue(-x))
	case TypeFloat:
		f, _ := v.AsFloat()
		return vm.stack.Push(FloatValue(-f))
	default:
		return ErrTypeMismatch
	}
}

// execNOT 逻辑非（NOT，opcode 99）：单 Bool 实参。
func execNOT(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	b, err := v.AsBool()
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(!b))
}

// execDIVMOD 整除+模（DIVMOD，opcode 100）：双实参 Int/Float，先压商再压余数。
func execDIVMOD(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	a, b := args[0], args[1]
	if a.typ == TypeInt && b.typ == TypeInt {
		x, _ := a.AsInt()
		y, _ := b.AsInt()
		if y == 0 {
			return ErrTypeMismatch
		}
		if err := vm.stack.Push(IntValue(x / y)); err != nil {
			return err
		}
		return vm.stack.Push(IntValue(x % y))
	}
	af, err := toFloat(a)
	if err != nil {
		return err
	}
	bf, err := toFloat(b)
	if err != nil {
		return err
	}
	q := math.Trunc(af / bf)
	r := math.Mod(af, bf)
	if err := vm.stack.Push(FloatValue(q)); err != nil {
		return err
	}
	return vm.stack.Push(FloatValue(r))
}

// execREP 重复条目（REP，opcode 101）：附参=份数(ULEB128)，取1实参，压栈 n 次。
// n=0 相当于弹出栈顶并丢弃。
func execREP(vm *VM, f *InstrFrame) error {
	var n uint64
	if len(f.AttrParams) > 0 {
		n = readULEB128Param(f.AttrParams[0])
	}
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	// n=0：丢弃
	for i := uint64(0); i < n; i++ {
		if err := vm.stack.Push(v); err != nil {
			return err
		}
	}
	return nil
}

// execDEL 删除字典条目（DEL，opcode 102）。占位：返回 NilValue。
func execDEL(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return vm.stack.Push(NilValue())
}

// execCLEAR 清空字典（CLEAR，opcode 103）。占位：返回 NilValue。
func execCLEAR(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return vm.stack.Push(NilValue())
}

// ─── 辅助函数 ────────────────────────────────────────────────────────────────

// toFloat 将 Int 或 Float 值转为 float64，其他类型返回 ErrTypeMismatch。
func toFloat(v Value) (float64, error) {
	switch v.typ {
	case TypeInt:
		n, _ := v.AsInt()
		return float64(n), nil
	case TypeFloat:
		f, _ := v.AsFloat()
		return f, nil
	default:
		return 0, ErrTypeMismatch
	}
}

// intPow 计算整数幂（base^exp），exp<0 返回 ErrTypeMismatch。
func intPow(base, exp int64) (int64, error) {
	if exp < 0 {
		return 0, ErrTypeMismatch
	}
	result := int64(1)
	for exp > 0 {
		if exp%2 == 1 {
			result *= base
		}
		base *= base
		exp /= 2
	}
	return result, nil
}

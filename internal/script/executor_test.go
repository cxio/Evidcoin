package script

import "testing"

// TestExecutorEmptyScript 测试空脚本以 true 通关产生 PassStop。
func TestExecutorEmptyScript(t *testing.T) {
	vm := NewVM()
	state := vm.Run(nil)
	if state != StatePassStop {
		t.Errorf("empty script state = %v, want PassStop", state)
	}
	if !vm.PassState() {
		t.Error("empty script passState should be true")
	}
}

// TestExecutorPublicDisabledOpcode 测试禁用指令在公共路径触达即 ScriptError。
func TestExecutorPublicDisabledOpcode(t *testing.T) {
	disabledOps := []Opcode{SCRIPT, VALUE, EVAL, INOUT}
	for _, op := range disabledOps {
		t.Run(Lookup(op).Mnemonic, func(t *testing.T) {
			vm := NewVM(WithMode(ModePublic))
			// 构造只含禁用指令的帧序列（不通过 bytecode 解码，直接注入帧）
			frames := []InstrFrame{{Op: op}}
			state := vm.Run(frames)
			if state != StateScriptError {
				t.Errorf("disabled opcode %v in public path: state = %v, want ScriptError", op, state)
			}
		})
	}
}

// TestExecutorSysTimeInPublic 测试 SYS_TIME 在公共路径触达即 ScriptError。
func TestExecutorSysTimeInPublic(t *testing.T) {
	vm := NewVM(WithMode(ModePublic))
	frames := []InstrFrame{{Op: SYS_TIME}}
	state := vm.Run(frames)
	if state != StateScriptError {
		t.Errorf("SYS_TIME in public path: state = %v, want ScriptError", state)
	}
}

// TestExecutorExtPrivInPublic 测试 EXT_PRIV 在公共路径触达即 ScriptError。
func TestExecutorExtPrivInPublic(t *testing.T) {
	vm := NewVM(WithMode(ModePublic))
	frames := []InstrFrame{{Op: EXT_PRIV}}
	state := vm.Run(frames)
	if state != StateScriptError {
		t.Errorf("EXT_PRIV in public path: state = %v, want ScriptError", state)
	}
}

// TestExecutorPrivateModeAllowsDisabled 测试私有路径中禁用指令不拒绝（但可能无执行函数）。
// 注意：私有路径不做公共安全检查；若无执行函数会返回 ScriptError（实现未注册），
// 但错误原因是"未注册"而非"公共路径禁用"。
func TestExecutorPrivateModeAllowsDisabled(t *testing.T) {
	// 验证：私有路径不因禁用指令触发 ErrDisabledInPublic
	// checkPublicSafety 在 ModePrivate 下直接返回 nil
	for _, op := range []Opcode{SCRIPT, VALUE, EVAL, INOUT} {
		err := checkPublicSafety(ModePrivate, op)
		if err != nil {
			t.Errorf("checkPublicSafety(ModePrivate, %v) = %v, want nil", op, err)
		}
	}
}

// TestExecutorCostFail 测试成本耗尽产生 CostFail。
func TestExecutorCostFail(t *testing.T) {
	// DICT_LIT 指令 CostTierLow（占位 cost=1）。
	// limit=1 时：第 1 次 DICT_LIT total=1（1>1=false，OK），
	//             第 2 次 DICT_LIT total=2（2>1=true，CostFail）。
	vm := NewVM(WithBudget(NewBudget(1)))
	// 预先向栈压入 4 个 Slice 值，DICT_LIT 每次从栈取 2 个
	for i := 0; i < 4; i++ {
		_ = vm.stack.Push(SliceValue(nil))
	}
	frames := []InstrFrame{
		{Op: DICT_LIT}, // cost=1, total=1, 1>1=false → OK
		{Op: DICT_LIT}, // cost=1, total=2, 2>1=true → CostFail
	}
	state := vm.Run(frames)
	if state != StateCostFail {
		t.Errorf("expected CostFail, got %v", state)
	}
}

// TestExecutorInitialPassState 测试初始通关状态为 true（DEC-0505）。
func TestExecutorInitialPassState(t *testing.T) {
	vm := NewVM()
	if !vm.PassState() {
		t.Error("initial passState should be true")
	}
}

// TestExecutorGetOneArg 测试 getOneArg 先用实参区再用数据栈。
func TestExecutorGetOneArg(t *testing.T) {
	vm := NewVM()

	// 实参区有值时从实参区取
	vm.args.Enqueue(IntValue(42))
	v, err := vm.getOneArg()
	if err != nil {
		t.Fatalf("getOneArg with args: %v", err)
	}
	if n, _ := v.AsInt(); n != 42 {
		t.Errorf("getOneArg from args = %d, want 42", n)
	}

	// 实参区为空时从栈取
	_ = vm.stack.Push(IntValue(99))
	v, err = vm.getOneArg()
	if err != nil {
		t.Fatalf("getOneArg from stack: %v", err)
	}
	if n, _ := v.AsInt(); n != 99 {
		t.Errorf("getOneArg from stack = %d, want 99", n)
	}

	// 实参区和栈均空时报错
	if _, err = vm.getOneArg(); err == nil {
		t.Error("getOneArg on empty stack/args should return error")
	}
}

// TestExecutorGetArgs 测试 getArgs 固定数量逻辑。
func TestExecutorGetArgs(t *testing.T) {
	vm := NewVM()

	// 从实参区取 2 个（数量恰好匹配）
	vm.args.Enqueue(IntValue(1))
	vm.args.Enqueue(IntValue(2))
	vals, err := vm.getArgs(2)
	if err != nil {
		t.Fatalf("getArgs(2) from args: %v", err)
	}
	n0, _ := vals[0].AsInt()
	n1, _ := vals[1].AsInt()
	if n0 != 1 || n1 != 2 {
		t.Errorf("getArgs(2) = [%d, %d], want [1, 2]", n0, n1)
	}

	// 实参区数量不符时报错
	vm.args.Enqueue(IntValue(5))
	if _, err = vm.getArgs(2); err == nil {
		t.Error("getArgs(2) with 1 item in args should return ErrArgCountMismatch")
	}
	vm.args.Clear()

	// 从栈取 3 个（LIFO -> 顺序反转补正）
	_ = vm.stack.Push(IntValue(10))
	_ = vm.stack.Push(IntValue(20))
	_ = vm.stack.Push(IntValue(30))
	vals, err = vm.getArgs(3)
	if err != nil {
		t.Fatalf("getArgs(3) from stack: %v", err)
	}
	for i, want := range []int64{10, 20, 30} {
		n, _ := vals[i].AsInt()
		if n != want {
			t.Errorf("getArgs(3)[%d] = %d, want %d", i, n, want)
		}
	}
}

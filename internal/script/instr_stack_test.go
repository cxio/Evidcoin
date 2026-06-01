package script

import "testing"

// TestInstrStackNOP 测试 NOP 清空实参区。
func TestInstrStackNOP(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(IntValue(1))
	vm.args.Enqueue(IntValue(2))
	vm.Run([]InstrFrame{{Op: NOP}})
	if vm.args.Len() != 0 {
		t.Errorf("NOP: args.Len() = %d, want 0", vm.args.Len())
	}
}

// TestInstrStackPUSH 测试 PUSH 将实参区内容顺序压栈。
func TestInstrStackPUSH(t *testing.T) {
	vm := NewVM()
	// 先把值放入实参区（由 AT/@ 截取机制，此处直接注入）
	vm.args.Enqueue(IntValue(10))
	vm.args.Enqueue(IntValue(20))
	vm.args.Enqueue(IntValue(30))
	vm.Run([]InstrFrame{{Op: PUSH}})
	if vm.stack.Len() != 3 {
		t.Fatalf("PUSH: stack len = %d, want 3", vm.stack.Len())
	}
	// 顺序：10 在底，30 在顶
	for i, want := range []int64{10, 20, 30} {
		v, _ := vm.stack.Peek(i)
		n, _ := v.AsInt()
		if n != want {
			t.Errorf("PUSH: stack[%d] = %d, want %d", i, n, want)
		}
	}
}

// TestInstrStackPOP 测试 POP 弹出栈顶。
func TestInstrStackPOP(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(42))
	vm.Run([]InstrFrame{{Op: POP}})
	if vm.stack.Len() != 0 {
		t.Error("POP: stack should be empty")
	}
}

// TestInstrStackPOPUnderflow 测试 POP 空栈产生 ScriptError。
func TestInstrStackPOPUnderflow(t *testing.T) {
	vm := NewVM()
	vm.Run([]InstrFrame{{Op: POP}})
	if vm.State() != StateScriptError {
		t.Errorf("POP on empty stack: state = %v, want ScriptError", vm.State())
	}
}

// TestInstrStackTOP 测试 TOP 复制栈顶（DUP）。
func TestInstrStackTOP(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(99))
	vm.Run([]InstrFrame{{Op: TOP}})
	if vm.stack.Len() != 2 {
		t.Fatalf("TOP: stack len = %d, want 2", vm.stack.Len())
	}
	v, _ := vm.stack.Top()
	n, _ := v.AsInt()
	if n != 99 {
		t.Errorf("TOP: top = %d, want 99", n)
	}
}

// TestInstrStackTOPUnderflow 测试 TOP 空栈产生 ScriptError。
func TestInstrStackTOPUnderflow(t *testing.T) {
	vm := NewVM()
	vm.Run([]InstrFrame{{Op: TOP}})
	if vm.State() != StateScriptError {
		t.Errorf("TOP on empty stack: state = %v, want ScriptError", vm.State())
	}
}

// TestInstrStackPEEK 测试 PEEK 按位置引用栈值。
func TestInstrStackPEEK(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(1))
	_ = vm.stack.Push(IntValue(2))
	_ = vm.stack.Push(IntValue(3))

	// PEEK(-1) = 栈顶
	vm.args.Enqueue(IntValue(-1))
	vm.Run([]InstrFrame{{Op: PEEK}})
	if vm.State() == StateScriptError {
		t.Fatal("PEEK(-1) unexpected ScriptError")
	}
	v, _ := vm.stack.Top()
	n, _ := v.AsInt()
	if n != 3 {
		t.Errorf("PEEK(-1) = %d, want 3", n)
	}
}

// TestInstrStackSHIFT 测试 SHIFT 移出栈顶打包为切片。
func TestInstrStackSHIFT(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(1))
	_ = vm.stack.Push(IntValue(2))
	_ = vm.stack.Push(IntValue(3))

	// SHIFT 2：移出栈顶 2 个
	vm.Run([]InstrFrame{{Op: SHIFT, AttrParams: [][]byte{{2}}}})
	if vm.stack.Len() != 2 { // 1 原始 + 1 切片
		t.Fatalf("SHIFT 2: stack len = %d, want 2", vm.stack.Len())
	}
	// 栈顶应为包含 [2,3] 的切片
	v, _ := vm.stack.Top()
	if v.Typ() != TypeSlice {
		t.Errorf("SHIFT 2: top type = %v, want Slice", v.Typ())
	}
	s, _ := v.AsSlice()
	if len(s) != 2 {
		t.Fatalf("SHIFT 2: slice len = %d, want 2", len(s))
	}
	n0, _ := s[0].AsInt()
	n1, _ := s[1].AsInt()
	if n0 != 2 || n1 != 3 {
		t.Errorf("SHIFT 2: slice = [%d,%d], want [2,3]", n0, n1)
	}
}

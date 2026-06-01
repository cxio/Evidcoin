package script

import "testing"

// instr_capture_test.go 测试截取指令 [19-23]。

func TestExecAT(t *testing.T) {
	// AT：从栈弹出值放入实参区
	vm := NewVM()
	_ = vm.stack.Push(IntValue(42))
	f := &InstrFrame{Op: AT}
	if err := execAT(vm, f); err != nil {
		t.Fatalf("execAT 返回错误: %v", err)
	}
	if vm.args.Len() != 1 {
		t.Fatalf("实参区应有1个元素，实际 %d", vm.args.Len())
	}
	v, _ := vm.args.Dequeue()
	n, err := v.AsInt()
	if err != nil || n != 42 {
		t.Errorf("实参区值应为 42，实际 %v", v)
	}
}

func TestExecAT_EmptyStack(t *testing.T) {
	// AT：栈为空时应返回下溢错误
	vm := NewVM()
	f := &InstrFrame{Op: AT}
	if err := execAT(vm, f); err == nil {
		t.Fatal("栈为空时 AT 应返回错误")
	}
}

func TestExecLOCAL(t *testing.T) {
	// LOCAL：压入 NilValue 到实参区
	vm := NewVM()
	f := &InstrFrame{Op: LOCAL}
	if err := execLOCAL(vm, f); err != nil {
		t.Fatalf("execLOCAL 返回错误: %v", err)
	}
	if vm.args.Len() != 1 {
		t.Fatalf("实参区应有1个元素，实际 %d", vm.args.Len())
	}
	v, _ := vm.args.Dequeue()
	if !v.IsNil() {
		t.Errorf("LOCAL 应压入 NilValue，实际 %v", v)
	}
}

func TestExecLOOPVAR(t *testing.T) {
	// LOOPVAR：压入 NilValue 到实参区
	vm := NewVM()
	f := &InstrFrame{Op: LOOPVAR}
	if err := execLOOPVAR(vm, f); err != nil {
		t.Fatalf("execLOOPVAR 返回错误: %v", err)
	}
	v, _ := vm.args.Dequeue()
	if !v.IsNil() {
		t.Errorf("LOOPVAR 应压入 NilValue，实际 %v", v)
	}
}

func TestExecDIRECT(t *testing.T) {
	// DIRECT：无操作，不改变栈状态
	vm := NewVM()
	_ = vm.stack.Push(IntValue(1))
	f := &InstrFrame{Op: DIRECT}
	if err := execDIRECT(vm, f); err != nil {
		t.Fatalf("execDIRECT 返回错误: %v", err)
	}
	// 栈应保持不变
	if vm.stack.Len() != 1 {
		t.Errorf("DIRECT 不应改变栈，高度应为1，实际 %d", vm.stack.Len())
	}
}

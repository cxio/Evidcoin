package script

import "testing"

// instr_module_test.go 测试模块指令 [225-250]。

func TestExecMO_MATH(t *testing.T) {
	// MO_MATH：执行后栈顶为 NilValue（占位）
	vm := NewVM()
	if err := execMO_MATH(vm, nil); err != nil {
		t.Fatalf("MO_MATH 返回错误: %v", err)
	}
	top, err := vm.stack.Pop()
	if err != nil {
		t.Fatalf("无法弹出栈顶: %v", err)
	}
	if !top.IsNil() {
		t.Errorf("MO_MATH 占位应压入 NilValue，实际 %v", top)
	}
}

func TestExecMO_FMT(t *testing.T) {
	// MO_FMT：执行后栈顶为 NilValue（占位）
	vm := NewVM()
	if err := execMO_FMT(vm, nil); err != nil {
		t.Fatalf("MO_FMT 返回错误: %v", err)
	}
	top, err := vm.stack.Pop()
	if err != nil {
		t.Fatalf("无法弹出栈顶: %v", err)
	}
	if !top.IsNil() {
		t.Errorf("MO_FMT 占位应压入 NilValue，实际 %v", top)
	}
}

func TestExecMO_XX(t *testing.T) {
	vm := NewVM()
	if err := execMO_XX(vm, nil); err != nil {
		t.Fatalf("MO_XX 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	if !top.IsNil() {
		t.Errorf("MO_XX 占位应压入 NilValue，实际 %v", top)
	}
}

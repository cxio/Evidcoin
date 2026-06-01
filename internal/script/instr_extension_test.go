package script

import "testing"

// instr_extension_test.go 测试扩展指令 [251-253]。

func TestExecEXT_MO(t *testing.T) {
	// EXT_MO：私有模式执行后栈顶为 NilValue
	vm := NewVM(WithMode(ModePrivate))
	if err := execEXT_MO(vm, nil); err != nil {
		t.Fatalf("EXT_MO 返回错误: %v", err)
	}
	top, err := vm.stack.Pop()
	if err != nil {
		t.Fatalf("无法弹出栈顶: %v", err)
	}
	if !top.IsNil() {
		t.Errorf("EXT_MO 占位应压入 NilValue，实际 %v", top)
	}
}

func TestExecEXT_PRIV_Public(t *testing.T) {
	// EXT_PRIV：公共模式执行 → StateScriptError（由 checkPublicSafety 拦截）
	vm := NewVM() // 默认公共模式
	state := vm.Run([]InstrFrame{{Op: EXT_PRIV}})
	if state != StateScriptError {
		t.Errorf("EXT_PRIV 在公共模式应产生 StateScriptError，实际 %v", state)
	}
}

func TestExecEXT_PRIV_Private(t *testing.T) {
	// EXT_PRIV：私有模式执行后栈顶为 NilValue
	vm := NewVM(WithMode(ModePrivate))
	if err := execEXT_PRIV(vm, nil); err != nil {
		t.Fatalf("EXT_PRIV 私有模式返回错误: %v", err)
	}
	top, err := vm.stack.Pop()
	if err != nil {
		t.Fatalf("无法弹出栈顶: %v", err)
	}
	if !top.IsNil() {
		t.Errorf("EXT_PRIV 私有模式占位应压入 NilValue，实际 %v", top)
	}
}

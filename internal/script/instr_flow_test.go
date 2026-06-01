package script

import "testing"

// instr_flow_test.go 测试流程控制指令 [58-66]。

// buildSubBlock 构建仅含单条无附参/无关联数据 opcode 的子块字节码。
func buildSubBlock(op Opcode) []byte {
	return []byte{byte(op)}
}

func TestExecIF_True(t *testing.T) {
	// IF 条件为真时执行子块（子块含 TRUE 指令）
	vm := NewVM()
	subBlock := buildSubBlock(TRUE) // TRUE 指令压入 BoolValue(true)
	f := &InstrFrame{
		Op:        IF,
		AssocData: subBlock,
	}
	// 压入条件 true
	_ = vm.stack.Push(BoolValue(true))

	if err := execIF(vm, f); err != nil {
		t.Fatalf("execIF(true) 返回错误: %v", err)
	}
	// 子块执行后栈顶应为 BoolValue(true)
	top, err := vm.stack.Pop()
	if err != nil {
		t.Fatalf("无法弹出栈顶: %v", err)
	}
	b, err := top.AsBool()
	if err != nil || !b {
		t.Errorf("IF 子块执行后栈顶应为 true，实际 %v", top)
	}
}

func TestExecIF_False(t *testing.T) {
	// IF 条件为假时跳过子块，栈保持空
	vm := NewVM()
	subBlock := buildSubBlock(TRUE)
	f := &InstrFrame{
		Op:        IF,
		AssocData: subBlock,
	}
	_ = vm.stack.Push(BoolValue(false))

	if err := execIF(vm, f); err != nil {
		t.Fatalf("execIF(false) 返回错误: %v", err)
	}
	// 子块未执行，栈应为空
	if vm.stack.Len() != 0 {
		t.Errorf("IF 条件为假时不应执行子块，栈高度应为0，实际 %d", vm.stack.Len())
	}
}

func TestExecEACH(t *testing.T) {
	// EACH：遍历 Slice，对每个元素执行子块（子块含 TRUE 指令）
	vm := NewVM()
	items := []Value{IntValue(1), IntValue(2), IntValue(3)}
	subBlock := buildSubBlock(TRUE)
	f := &InstrFrame{
		Op:        EACH,
		AssocData: subBlock,
	}
	_ = vm.stack.Push(SliceValue(items))

	if err := execEACH(vm, f); err != nil {
		t.Fatalf("execEACH 返回错误: %v", err)
	}
	// 3 个元素，每次执行 TRUE 压栈，共应有3个 true
	if vm.stack.Len() != 3 {
		t.Errorf("EACH 应执行3次子块，栈高度应为3，实际 %d", vm.stack.Len())
	}
}

func TestExecBLOCK(t *testing.T) {
	// BLOCK：执行子块（子块含 NIL 指令）
	vm := NewVM()
	subBlock := buildSubBlock(NIL)
	f := &InstrFrame{
		Op:        BLOCK,
		AssocData: subBlock,
	}
	if err := execBLOCK(vm, f); err != nil {
		t.Fatalf("execBLOCK 返回错误: %v", err)
	}
	top, err := vm.stack.Pop()
	if err != nil {
		t.Fatalf("无法弹出栈顶: %v", err)
	}
	if !top.IsNil() {
		t.Errorf("BLOCK 子块执行 NIL 后栈顶应为 NilValue，实际 %v", top)
	}
}

func TestExecBLOCK_EmptyAssocData(t *testing.T) {
	// BLOCK：关联数据为空时无操作
	vm := NewVM()
	f := &InstrFrame{Op: BLOCK}
	if err := execBLOCK(vm, f); err != nil {
		t.Fatalf("execBLOCK(空子块) 返回错误: %v", err)
	}
	if vm.stack.Len() != 0 {
		t.Errorf("BLOCK 空子块后栈应为空，实际高度 %d", vm.stack.Len())
	}
}

package script

import (
	"math"
	"testing"
)

// instr_arithmetic_test.go 测试运算指令 [80-103]。

func TestExecMUL(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(3))
	_ = vm.stack.Push(IntValue(4))
	if err := execMUL(vm, nil); err != nil {
		t.Fatalf("MUL 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 12 {
		t.Errorf("3*4 应为12，实际 %d", n)
	}
}

func TestExecDIV(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(10))
	_ = vm.stack.Push(IntValue(2))
	if err := execDIV(vm, nil); err != nil {
		t.Fatalf("DIV 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 5 {
		t.Errorf("10/2 应为5，实际 %d", n)
	}
}

func TestExecDIV_ZeroInt(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(10))
	_ = vm.stack.Push(IntValue(0))
	if err := execDIV(vm, nil); err == nil {
		t.Fatal("Int 除零应返回错误")
	}
}

func TestExecADD_Int(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(3))
	_ = vm.stack.Push(IntValue(4))
	if err := execADD(vm, nil); err != nil {
		t.Fatalf("ADD Int 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 7 {
		t.Errorf("3+4 应为7，实际 %d", n)
	}
}

func TestExecADD_String(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(StringValue("hello"))
	_ = vm.stack.Push(StringValue("world"))
	if err := execADD(vm, nil); err != nil {
		t.Fatalf("ADD String 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	s, _ := top.AsString()
	if s != "helloworld" {
		t.Errorf("期望 helloworld，实际 %q", s)
	}
}

func TestExecADD_Bytes(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(BytesValue([]byte{1, 2}))
	_ = vm.stack.Push(BytesValue([]byte{3, 4}))
	if err := execADD(vm, nil); err != nil {
		t.Fatalf("ADD Bytes 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	b, _ := top.AsBytes()
	if len(b) != 4 || b[0] != 1 || b[1] != 2 || b[2] != 3 || b[3] != 4 {
		t.Errorf("ADD Bytes 结果错误: %v", b)
	}
}

func TestExecSUB(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(10))
	_ = vm.stack.Push(IntValue(3))
	if err := execSUB(vm, nil); err != nil {
		t.Fatalf("SUB 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 7 {
		t.Errorf("10-3 应为7，实际 %d", n)
	}
}

func TestExecNEG_Int(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(5))
	if err := execNEG(vm, nil); err != nil {
		t.Fatalf("NEG Int 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != -5 {
		t.Errorf("NEG(5) 应为-5，实际 %d", n)
	}
}

func TestExecNOT(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(BoolValue(true))
	if err := execNOT(vm, nil); err != nil {
		t.Fatalf("NOT 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	b, _ := top.AsBool()
	if b {
		t.Error("NOT(true) 应为 false")
	}
}

func TestExecPOW(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(2))
	_ = vm.stack.Push(IntValue(8))
	if err := execPOW(vm, nil); err != nil {
		t.Fatalf("POW 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 256 {
		t.Errorf("2^8 应为256，实际 %d", n)
	}
}

func TestExecLMOV(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(1))
	_ = vm.stack.Push(IntValue(3))
	if err := execLMOV(vm, nil); err != nil {
		t.Fatalf("LMOV 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 8 {
		t.Errorf("1<<3 应为8，实际 %d", n)
	}
}

func TestExecDIVMOD_Int(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(10))
	_ = vm.stack.Push(IntValue(3))
	if err := execDIVMOD(vm, nil); err != nil {
		t.Fatalf("DIVMOD 返回错误: %v", err)
	}
	// 栈顶是余数
	rem, _ := vm.stack.Pop()
	r, _ := rem.AsInt()
	// 第二个是商
	quot, _ := vm.stack.Pop()
	q, _ := quot.AsInt()
	if q != 3 {
		t.Errorf("10/3 商应为3，实际 %d", q)
	}
	if r != 1 {
		t.Errorf("10%%3 余应为1，实际 %d", r)
	}
}

func TestExecREP_N(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(99))
	f := &InstrFrame{
		Op:         REP,
		AttrParams: [][]byte{{2}}, // 份数=2
	}
	if err := execREP(vm, f); err != nil {
		t.Fatalf("REP 2 返回错误: %v", err)
	}
	if vm.stack.Len() != 2 {
		t.Errorf("REP 2 后栈高度应为2，实际 %d", vm.stack.Len())
	}
}

func TestExecREP_Zero(t *testing.T) {
	// REP 0：弹出后不压入（丢弃）
	vm := NewVM()
	_ = vm.stack.Push(IntValue(99))
	f := &InstrFrame{
		Op:         REP,
		AttrParams: [][]byte{{0}}, // 份数=0
	}
	if err := execREP(vm, f); err != nil {
		t.Fatalf("REP 0 返回错误: %v", err)
	}
	if vm.stack.Len() != 0 {
		t.Errorf("REP 0 后栈应为空，实际高度 %d", vm.stack.Len())
	}
}

func TestExecAND(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(0b1010))
	_ = vm.stack.Push(IntValue(0b1100))
	if err := execAND(vm, nil); err != nil {
		t.Fatalf("AND 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 0b1000 {
		t.Errorf("AND 结果错误，期望 8，实际 %d", n)
	}
}

func TestExecOR(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(0b1010))
	_ = vm.stack.Push(IntValue(0b0101))
	if err := execOR(vm, nil); err != nil {
		t.Fatalf("OR 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 0b1111 {
		t.Errorf("OR 结果错误，期望 15，实际 %d", n)
	}
}

func TestExecMOD_Int(t *testing.T) {
	vm := NewVM()
	_ = vm.stack.Push(IntValue(10))
	_ = vm.stack.Push(IntValue(3))
	if err := execMOD(vm, nil); err != nil {
		t.Fatalf("MOD 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	n, _ := top.AsInt()
	if n != 1 {
		t.Errorf("10%%3 应为1，实际 %d", n)
	}
}

var _ = math.Abs // 确保 math 导入被使用

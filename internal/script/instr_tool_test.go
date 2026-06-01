package script

import (
	"regexp"
	"testing"
)

// instr_tool_test.go 测试工具指令 [138-163]。

func TestExecCOPY(t *testing.T) {
	vm := NewVM()
	items := []Value{IntValue(1), IntValue(2), IntValue(3)}
	_ = vm.stack.Push(SliceValue(items))
	if err := execCOPY(vm, nil); err != nil {
		t.Fatalf("COPY 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	sl, err := top.AsSlice()
	if err != nil || len(sl) != 3 {
		t.Errorf("COPY 结果错误: %v", sl)
	}
}

func TestExecSUBSTR(t *testing.T) {
	// "hello" 从位置1取3字符 → "ell"
	vm := NewVM()
	_ = vm.stack.Push(StringValue("hello"))
	_ = vm.stack.Push(IntValue(1))
	f := &InstrFrame{
		Op:         SUBSTR,
		AttrParams: [][]byte{{3}}, // 取3字符
	}
	if err := execSUBSTR(vm, f); err != nil {
		t.Fatalf("SUBSTR 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	s, _ := top.AsString()
	if s != "ell" {
		t.Errorf("SUBSTR('hello',1,3) 应为 'ell'，实际 %q", s)
	}
}

func TestExecCMPFLO_Equal(t *testing.T) {
	// 1.0 ≈ 1.001 误差0.01 → true
	vm := NewVM()
	_ = vm.stack.Push(FloatValue(1.0))
	_ = vm.stack.Push(FloatValue(1.001))
	_ = vm.stack.Push(FloatValue(0.01))
	f := &InstrFrame{
		Op:         CMPFLO,
		AttrParams: [][]byte{{0}}, // 模式0=相等
	}
	if err := execCMPFLO(vm, f); err != nil {
		t.Fatalf("CMPFLO 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	b, _ := top.AsBool()
	if !b {
		t.Error("CMPFLO(1.0, 1.001, 0.01) 应为 true")
	}
}

func TestExecRANGE_Int(t *testing.T) {
	// 起始10步进1长度3 → [10,11,12]
	vm := NewVM()
	_ = vm.stack.Push(IntValue(10))
	_ = vm.stack.Push(IntValue(1))
	f := &InstrFrame{
		Op:         RANGE,
		AttrParams: [][]byte{{3}}, // 长度3
	}
	if err := execRANGE(vm, f); err != nil {
		t.Fatalf("RANGE 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	sl, err := top.AsSlice()
	if err != nil || len(sl) != 3 {
		t.Fatalf("RANGE 应返回3个元素，实际 %d", len(sl))
	}
	expected := []int64{10, 11, 12}
	for i, v := range sl {
		n, _ := v.AsInt()
		if n != expected[i] {
			t.Errorf("RANGE[%d] 应为 %d，实际 %d", i, expected[i], n)
		}
	}
}

func TestExecSHELL_NoError(t *testing.T) {
	// SHELL：消费实参，不报错
	vm := NewVM()
	vm.args.Enqueue(StringValue("echo test"))
	if err := execSHELL(vm, nil); err != nil {
		t.Fatalf("SHELL 应无错误，实际 %v", err)
	}
	if vm.args.Len() != 0 {
		t.Error("SHELL 应清空实参区")
	}
}

func TestExecRANDOM_Error(t *testing.T) {
	// RANDOM：当前占位，应返回 ScriptError
	vm := NewVM()
	state := vm.Run([]InstrFrame{{Op: RANDOM}})
	if state != StateScriptError {
		t.Errorf("RANDOM 应产生 StateScriptError，实际 %v", state)
	}
}

func TestExecSLRAND_Error(t *testing.T) {
	// SLRAND：当前占位，应返回 ScriptError
	vm := NewVM()
	state := vm.Run([]InstrFrame{{Op: SLRAND}})
	if state != StateScriptError {
		t.Errorf("SLRAND 应产生 StateScriptError，实际 %v", state)
	}
}

func TestExecMATCH_Regexp(t *testing.T) {
	// MATCH：字符串匹配正则
	vm := NewVM()
	re := regexp.MustCompile(`\d+`)
	_ = vm.stack.Push(StringValue("abc123"))
	_ = vm.stack.Push(Value{typ: TypeRegExp, data: re})
	f := &InstrFrame{Op: MATCH, AttrParams: [][]byte{{0}}}
	if err := execMATCH(vm, f); err != nil {
		t.Fatalf("MATCH 返回错误: %v", err)
	}
	top, _ := vm.stack.Pop()
	b, _ := top.AsBool()
	if !b {
		t.Error("MATCH('abc123', /\\d+/) 应为 true")
	}
}

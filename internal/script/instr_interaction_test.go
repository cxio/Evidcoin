package script

import "testing"

// TestInstrInteractionINPUT_Public 测试 INPUT 在公共路径视为 END，产生 PassStop。
func TestInstrInteractionINPUT_Public(t *testing.T) {
	vm := NewVM(WithMode(ModePublic))
	vm.Run([]InstrFrame{{Op: INPUT}})
	if vm.State() != StatePassStop {
		t.Errorf("INPUT in public path: state = %v, want PassStop", vm.State())
	}
}

// TestInstrInteractionINPUT_Private 测试私有路径 INPUT 从缓冲区取值。
func TestInstrInteractionINPUT_Private(t *testing.T) {
	vm := NewVM(
		WithMode(ModePrivate),
		WithInputBuffer([]Value{IntValue(77)}),
	)
	vm.Run([]InstrFrame{{Op: INPUT}})
	if vm.stack.Len() != 1 {
		t.Fatalf("INPUT (private): stack len = %d, want 1", vm.stack.Len())
	}
	v, _ := vm.stack.Top()
	n, _ := v.AsInt()
	if n != 77 {
		t.Errorf("INPUT (private): got %d, want 77", n)
	}
}

// TestInstrInteractionINPUT_PrivateEmpty 测试私有路径 INPUT 缓冲区为空返回 ScriptError。
func TestInstrInteractionINPUT_PrivateEmpty(t *testing.T) {
	vm := NewVM(WithMode(ModePrivate))
	vm.Run([]InstrFrame{{Op: INPUT}})
	if vm.State() != StateScriptError {
		t.Errorf("INPUT (private, empty): state = %v, want ScriptError", vm.State())
	}
}

// TestInstrInteractionOUTPUT 测试 OUTPUT 把实参区内容导出到 OUTPUT 缓冲区。
func TestInstrInteractionOUTPUT(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(StringValue("exported"))
	vm.Run([]InstrFrame{{Op: OUTPUT}})
	out := vm.OutputBuffer()
	if len(out) != 1 {
		t.Fatalf("OUTPUT buffer len = %d, want 1", len(out))
	}
	s, _ := out[0].AsString()
	if s != "exported" {
		t.Errorf("OUTPUT buffer[0] = %q, want 'exported'", s)
	}
}

// TestInstrInteractionMultipleOUTPUT 测试多次 OUTPUT 追加到缓冲区。
func TestInstrInteractionMultipleOUTPUT(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(IntValue(1))
	execOUTPUT(vm, nil)
	vm.args.Enqueue(IntValue(2))
	execOUTPUT(vm, nil)
	out := vm.OutputBuffer()
	if len(out) != 2 {
		t.Fatalf("OUTPUT buffer len = %d, want 2", len(out))
	}
}

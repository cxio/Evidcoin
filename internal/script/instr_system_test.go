package script

import "testing"

// instr_system_test.go 测试系统指令 [164-169]。

// ─── SYS_TIME 测试 ────────────────────────────────────────────────────────────

func TestExecSYS_TIME(t *testing.T) {
	t.Run("passedPublicEnd=false 时返回 ScriptError", func(t *testing.T) {
		vm := NewVM()
		// passedPublicEnd 初始为 false
		frames := []InstrFrame{{Op: SYS_TIME}}
		state := vm.Run(frames)
		if state != StateScriptError {
			t.Fatalf("expected ScriptError, got %s", state)
		}
	})
	t.Run("passedPublicEnd=true 时正常执行", func(t *testing.T) {
		env := newMockEnv("sys.time.0", IntValue(999))
		vm := NewVM(WithEnvironment(env))
		vm.MarkPublicEnd()
		f := &InstrFrame{Op: SYS_TIME}
		if err := execSYS_TIME(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v, err := vm.stack.Pop()
		if err != nil {
			t.Fatalf("stack pop error: %v", err)
		}
		n, _ := v.AsInt()
		if n != 999 {
			t.Fatalf("expected 999, got %d", n)
		}
	})
}

// ─── SYS_AWARD 测试 ───────────────────────────────────────────────────────────

func TestExecSYS_AWARD(t *testing.T) {
	t.Run("无 witness 时返回 ScriptError", func(t *testing.T) {
		vm := NewVM()
		frames := []InstrFrame{{Op: SYS_AWARD}}
		state := vm.Run(frames)
		if state != StateScriptError {
			t.Fatalf("expected ScriptError, got %s", state)
		}
	})
	t.Run("非 coinbase witness 返回 ScriptError", func(t *testing.T) {
		vm := NewVM(WithWitnessProvider(&mockWitness{data: []byte{1}, coinbase: false}))
		frames := []InstrFrame{{Op: SYS_AWARD}}
		state := vm.Run(frames)
		if state != StateScriptError {
			t.Fatalf("expected ScriptError, got %s", state)
		}
	})
	t.Run("coinbase witness 正常执行", func(t *testing.T) {
		vm := NewVM(WithWitnessProvider(&mockWitness{data: []byte{1}, coinbase: true}))
		f := &InstrFrame{Op: SYS_AWARD}
		if err := execSYS_AWARD(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v, _ := vm.stack.Pop()
		b, _ := v.AsBool()
		if !b {
			t.Fatal("expected true from SYS_AWARD")
		}
	})
}

// ─── SYS_CHKPASS 测试 ────────────────────────────────────────────────────────

func TestExecSYS_CHKPASS(t *testing.T) {
	t.Run("无 witness 返回 ScriptError", func(t *testing.T) {
		vm := NewVM()
		frames := []InstrFrame{{Op: SYS_CHKPASS}}
		state := vm.Run(frames)
		if state != StateScriptError {
			t.Fatalf("expected ScriptError, got %s", state)
		}
	})
	t.Run("空见证数据返回 VerifyFail", func(t *testing.T) {
		vm := NewVM(WithWitnessProvider(&mockWitness{data: nil}))
		f := &InstrFrame{Op: SYS_CHKPASS}
		if err := execSYS_CHKPASS(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if vm.state != StateVerifyFail {
			t.Fatalf("expected VerifyFail, got %s", vm.state)
		}
	})
	t.Run("有见证数据时 passState=true", func(t *testing.T) {
		vm := NewVM(WithWitnessProvider(&mockWitness{data: []byte{0xde, 0xad, 0xbe}}))
		f := &InstrFrame{Op: SYS_CHKPASS}
		if err := execSYS_CHKPASS(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !vm.passState {
			t.Fatal("expected passState=true")
		}
		if !vm.GetSigned(0) {
			t.Fatal("expected signed[0]=true")
		}
	})
}

// ─── SYS_NULL 测试 ────────────────────────────────────────────────────────────

func TestExecSYS_NULL(t *testing.T) {
	t.Run("执行后状态仍 Running，无副作用", func(t *testing.T) {
		vm := NewVM()
		f := &InstrFrame{Op: SYS_NULL}
		if err := execSYS_NULL(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if vm.state != StateRunning {
			t.Fatalf("expected StateRunning, got %s", vm.state)
		}
		if vm.stack.Len() != 0 {
			t.Fatal("expected empty stack after SYS_NULL")
		}
	})
}

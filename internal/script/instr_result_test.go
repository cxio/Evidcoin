package script

import "testing"

// TestInstrResultEND 测试 END 产生 PassStop（以当前通关状态）。
func TestInstrResultEND(t *testing.T) {
	vm := NewVM()
	vm.Run([]InstrFrame{{Op: END}})
	if vm.State() != StatePassStop {
		t.Errorf("END: state = %v, want PassStop", vm.State())
	}
	if !vm.PassState() {
		t.Error("END: passState should be true (initial default)")
	}
}

// TestInstrResultEND_AfterCheckFalse 测试 CHECK(false) 后 END 产生 PassStop(false)。
func TestInstrResultEND_AfterCheckFalse(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(BoolValue(false))
	execCHECK(vm, nil) // 将 passState 设为 false
	execEND(vm, nil)
	if vm.State() != StatePassStop {
		t.Errorf("state = %v, want PassStop", vm.State())
	}
	if vm.PassState() {
		t.Error("passState should be false after CHECK(false)")
	}
}

// TestInstrResultPASS_True 测试 PASS true 继续执行。
func TestInstrResultPASS_True(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(BoolValue(true))
	execPASS(vm, nil)
	if vm.State() != StateRunning {
		t.Errorf("PASS true: state = %v, want Running", vm.State())
	}
}

// TestInstrResultPASS_False 测试 PASS false 立即 VerifyFail。
func TestInstrResultPASS_False(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(BoolValue(false))
	execPASS(vm, nil)
	if vm.State() != StateVerifyFail {
		t.Errorf("PASS false: state = %v, want VerifyFail", vm.State())
	}
}

// TestInstrResultPASS_TypeMismatch 测试 PASS 非 Bool 实参产生 ScriptError。
func TestInstrResultPASS_TypeMismatch(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(IntValue(1))
	vm.Run([]InstrFrame{{Op: PASS}})
	if vm.State() != StateScriptError {
		t.Errorf("PASS(Int): state = %v, want ScriptError", vm.State())
	}
}

// TestInstrResultCHECK_Coverage 测试 CHECK 后写覆盖通关状态。
func TestInstrResultCHECK_Coverage(t *testing.T) {
	// CHECK(true) 后 CHECK(false) -> passState = false
	vm := NewVM()
	vm.args.Enqueue(BoolValue(true))
	execCHECK(vm, nil)
	if !vm.PassState() {
		t.Error("CHECK(true): passState should be true")
	}
	vm.args.Enqueue(BoolValue(false))
	execCHECK(vm, nil)
	if vm.PassState() {
		t.Error("CHECK(false) after CHECK(true): passState should be false (覆盖)")
	}

	// CHECK(false) 后 CHECK(true) -> passState = true
	vm2 := NewVM()
	vm2.args.Enqueue(BoolValue(false))
	execCHECK(vm2, nil)
	vm2.args.Enqueue(BoolValue(true))
	execCHECK(vm2, nil)
	if !vm2.PassState() {
		t.Error("CHECK(true) after CHECK(false): passState should be true (覆盖)")
	}
}

// TestInstrResultCHECK_DoesNotTerminate 测试 CHECK 不终止执行。
func TestInstrResultCHECK_DoesNotTerminate(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(BoolValue(false))
	execCHECK(vm, nil)
	if vm.State() != StateRunning {
		t.Errorf("CHECK: state = %v, want Running (not terminated)", vm.State())
	}
}

// TestInstrResultCHECK_ThenEND 测试 CHECK(true)+CHECK(false)+END -> PassStop(false)。
func TestInstrResultCHECK_ThenEND(t *testing.T) {
	vm := NewVM()
	vm.args.Enqueue(BoolValue(true))
	execCHECK(vm, nil)
	vm.args.Enqueue(BoolValue(false))
	execCHECK(vm, nil)
	execEND(vm, nil)
	if vm.State() != StatePassStop {
		t.Errorf("state = %v, want PassStop", vm.State())
	}
	if vm.PassState() {
		t.Error("passState should be false (last CHECK was false)")
	}
}

// TestInstrResultEXIT 测试 EXIT 产生 PassStop。
func TestInstrResultEXIT(t *testing.T) {
	vm := NewVM()
	vm.Run([]InstrFrame{{Op: EXIT}})
	if vm.State() != StatePassStop {
		t.Errorf("EXIT: state = %v, want PassStop", vm.State())
	}
}

// TestInstrResultGOTO_ScriptError 测试 GOTO 占位实现产生 ScriptError。
func TestInstrResultGOTO_ScriptError(t *testing.T) {
	// GOTO 需要 53 字节附参（年度+TxID(48B)+序位），
	// 此处以空附参测试占位行为
	vm := NewVM()
	vm.Run([]InstrFrame{{Op: GOTO}})
	if vm.State() != StateScriptError {
		t.Errorf("GOTO (placeholder): state = %v, want ScriptError", vm.State())
	}
}

// TestInstrResultPublicAfterEND 测试 END 之后的帧不执行（公共路径私有保护）。
func TestInstrResultPublicAfterEND(t *testing.T) {
	vm := NewVM(WithMode(ModePublic))
	// END 后有 SYS_TIME（公共路径禁用），但由于 END 已 PassStop，后续帧不执行
	frames := []InstrFrame{
		{Op: END},
		{Op: SYS_TIME}, // 如果执行了会触发 ScriptError
	}
	vm.Run(frames)
	if vm.State() != StatePassStop {
		t.Errorf("after END: state = %v, want PassStop (SYS_TIME not executed)", vm.State())
	}
}

// TestInstrResultPASS_NoArg_ScriptError 测试 PASS 无实参时产生 ScriptError。
func TestInstrResultPASS_NoArg_ScriptError(t *testing.T) {
	vm := NewVM()
	vm.Run([]InstrFrame{{Op: PASS}})
	if vm.State() != StateScriptError {
		t.Errorf("PASS (no arg): state = %v, want ScriptError", vm.State())
	}
}

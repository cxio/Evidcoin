package script

import "testing"

// instr_pattern_test.go 测试模式指令 [116-127]。

func TestExecMODEL_ScriptError(t *testing.T) {
	// MODEL 当前占位返回 ErrModelOutside（ScriptError）
	vm := NewVM()
	state := vm.Run([]InstrFrame{{Op: MODEL}})
	if state != StateScriptError {
		t.Errorf("MODEL 应产生 StateScriptError，实际 %v", state)
	}
}

func TestExecWILDCARD_ScriptError(t *testing.T) {
	vm := NewVM()
	state := vm.Run([]InstrFrame{{Op: WILDCARD}})
	if state != StateScriptError {
		t.Errorf("WILDCARD 应产生 StateScriptError，实际 %v", state)
	}
}

func TestExecELLIPSIS_ScriptError(t *testing.T) {
	vm := NewVM()
	state := vm.Run([]InstrFrame{{Op: ELLIPSIS}})
	if state != StateScriptError {
		t.Errorf("ELLIPSIS 应产生 StateScriptError，实际 %v", state)
	}
}

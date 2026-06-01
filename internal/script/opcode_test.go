package script

import "testing"

// TestOpcodeRanges 验证指令段范围划分正确。
func TestOpcodeRanges(t *testing.T) {
	tests := []struct {
		name     string
		op       Opcode
		isBasic  bool
		isFunc   bool
		isMod    bool
		isExt    bool
		isReserv bool
	}{
		{"NIL(0) is basic", NIL, true, false, false, false, false},
		{"BasicSegEnd(169) is basic", BasicSegEnd, true, false, false, false, false},
		{"FunctionSegStart(170) is function", FunctionSegStart, false, true, false, false, false},
		{"FunctionSegEnd(224) is function", FunctionSegEnd, false, true, false, false, false},
		{"ModuleSegStart(225) is module", ModuleSegStart, false, false, true, false, false},
		{"ModuleSegEnd(250) is module", ModuleSegEnd, false, false, true, false, false},
		{"ExtensionSegStart(251) is extension", ExtensionSegStart, false, false, false, true, false},
		{"ExtensionSegEnd(253) is extension", ExtensionSegEnd, false, false, false, true, false},
		{"254 is reserved", Opcode(254), false, false, false, false, true},
		{"255 is reserved", Opcode(255), false, false, false, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.op.IsBasic(); got != tc.isBasic {
				t.Errorf("IsBasic() = %v, want %v", got, tc.isBasic)
			}
			if got := tc.op.IsFunction(); got != tc.isFunc {
				t.Errorf("IsFunction() = %v, want %v", got, tc.isFunc)
			}
			if got := tc.op.IsModule(); got != tc.isMod {
				t.Errorf("IsModule() = %v, want %v", got, tc.isMod)
			}
			if got := tc.op.IsExtension(); got != tc.isExt {
				t.Errorf("IsExtension() = %v, want %v", got, tc.isExt)
			}
			if got := tc.op.IsReserved(); got != tc.isReserv {
				t.Errorf("IsReserved() = %v, want %v", got, tc.isReserv)
			}
		})
	}
}

// TestOpcodeDisabledList 验证前期禁用指令清单完整（DEC-0505）。
func TestOpcodeDisabledList(t *testing.T) {
	// 4 项前期禁用：SCRIPT(17)、VALUE(18)、EVAL(138)、INOUT(131)
	tests := []struct {
		op       Opcode
		disabled bool
	}{
		{SCRIPT, true},
		{VALUE, true},
		{EVAL, true},
		{INOUT, true},
		// CALL 和 SHELL 不属于禁用指令（DEC-0505 确认）
		{CALL, false},
		{SHELL, false},
		// 普通指令
		{NIL, false},
		{PASS, false},
		{SYS_TIME, false},
	}
	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			if got := tc.op.IsDisabled(); got != tc.disabled {
				t.Errorf("opcode %d IsDisabled() = %v, want %v", tc.op, got, tc.disabled)
			}
		})
	}
}

// TestOpcodeUnlockAllowed 验证解锁脚本 opcode 限制（[0-50] + SYS_NULL 特例）。
func TestOpcodeUnlockAllowed(t *testing.T) {
	tests := []struct {
		op      Opcode
		allowed bool
	}{
		{NIL, true},         // 0
		{Opcode(50), true},  // PRINT = 50，最大允许
		{SYS_NULL, true},    // 169，特例
		{Opcode(51), false}, // PASS = 51，超出
		{Opcode(169), true}, // SYS_NULL 再确认
		// SCRIPT(17)/VALUE(18) 在 [0-50] 内，但被禁用，不允许进入主网解锁脚本（DEC-0505）
		{SCRIPT, false},
		{VALUE, false},
		{Opcode(170), false},
		{Opcode(253), false},
		{Opcode(254), false},
		{Opcode(255), false},
	}
	for _, tc := range tests {
		if got := tc.op.IsAllowedInUnlock(); got != tc.allowed {
			t.Errorf("opcode %d IsAllowedInUnlock() = %v, want %v", tc.op, got, tc.allowed)
		}
	}
}

// TestOpcodeIsAssigned 验证 254/255 不可分配。
func TestOpcodeIsAssigned(t *testing.T) {
	if !Opcode(0).IsAssigned() {
		t.Error("opcode 0 should be assignable")
	}
	if !Opcode(253).IsAssigned() {
		t.Error("opcode 253 should be assignable")
	}
	if Opcode(254).IsAssigned() {
		t.Error("opcode 254 should NOT be assignable (system reserved)")
	}
	if Opcode(255).IsAssigned() {
		t.Error("opcode 255 should NOT be assignable (system reserved)")
	}
}

package script

import "testing"

// TestRegistrySize 验证注册表中注册了合理数量的指令。
func TestRegistrySize(t *testing.T) {
	size := RegistrySize()
	// 注册表应有数十条指令（已定义指令约 100+）
	if size < 80 {
		t.Errorf("registry too small: got %d, want at least 80", size)
	}
	if size > 254 {
		t.Errorf("registry too large: got %d (max 254 assignable opcodes)", size)
	}
}

// TestRegistryLookup 验证基本查询功能。
func TestRegistryLookup(t *testing.T) {
	tests := []struct {
		op       Opcode
		mnemonic string
	}{
		{NIL, "NIL"},
		{TRUE, "TRUE"},
		{FALSE, "FALSE"},
		{PASS, "PASS"},
		{CHECK, "CHECK"},
		{END, "END"},
		{SCRIPT, "SCRIPT"},
		{VALUE, "VALUE"},
		{EVAL, "EVAL"},
		{INOUT, "INOUT"},
		{SYS_NULL, "SYS_NULL"},
		{SYS_TIME, "SYS_TIME"},
		{CALL, "CALL"},
		{SHELL, "SHELL"},
	}
	for _, tc := range tests {
		m := Lookup(tc.op)
		if m == nil {
			t.Errorf("opcode %d (%s) not found in registry", tc.op, tc.mnemonic)
			continue
		}
		if m.Mnemonic != tc.mnemonic {
			t.Errorf("opcode %d: mnemonic = %q, want %q", tc.op, m.Mnemonic, tc.mnemonic)
		}
	}
}

// TestRegistryUnassignedOpcodes 验证系统保留 opcode 未注册。
func TestRegistryUnassignedOpcodes(t *testing.T) {
	for _, op := range []Opcode{254, 255} {
		if m := Lookup(op); m != nil {
			t.Errorf("opcode %d is system reserved but was found in registry: %s", op, m.Mnemonic)
		}
	}
}

// TestRegistryDisabledOpcodes 验证禁用指令已正确标记。
func TestRegistryDisabledOpcodes(t *testing.T) {
	disabled := []Opcode{SCRIPT, VALUE, EVAL, INOUT}
	for _, op := range disabled {
		m := Lookup(op)
		if m == nil {
			t.Errorf("disabled opcode %d not registered", op)
			continue
		}
		if !m.IsDisabled() {
			t.Errorf("opcode %d (%s) should be marked disabled", op, m.Mnemonic)
		}
	}
}

// TestRegistryCallNotDisabled 验证 CALL/SHELL 不在禁用列表（DEC-0505）。
func TestRegistryCallNotDisabled(t *testing.T) {
	for _, op := range []Opcode{CALL, SHELL} {
		m := Lookup(op)
		if m == nil {
			t.Errorf("opcode %d not registered", op)
			continue
		}
		if m.IsDisabled() {
			t.Errorf("opcode %d (%s) should NOT be disabled (DEC-0505)", op, m.Mnemonic)
		}
	}
}

// TestRegistrySysNullUnlockSafe 验证 SYS_NULL 可用于解锁脚本（特例）。
func TestRegistrySysNullUnlockSafe(t *testing.T) {
	m := Lookup(SYS_NULL)
	if m == nil {
		t.Fatal("SYS_NULL not registered")
	}
	if !m.IsUnlockSafe() {
		t.Error("SYS_NULL should be unlock-safe (special exception)")
	}
}

// TestRegistryAllHaveMnemonic 验证所有已注册指令都有非空助记符。
func TestRegistryAllHaveMnemonic(t *testing.T) {
	for _, m := range AllRegistered() {
		if m.Mnemonic == "" {
			t.Errorf("opcode %d has empty mnemonic", m.Opcode)
		}
	}
}

// TestRegistryAllHaveCategory 验证所有已注册指令都有合法类别 [1-18]。
func TestRegistryAllHaveCategory(t *testing.T) {
	for _, m := range AllRegistered() {
		if m.Category < 1 || m.Category > 18 {
			t.Errorf("opcode %d (%s) has invalid category %d (must be 1-18)", m.Opcode, m.Mnemonic, m.Category)
		}
	}
}

// TestRegistryMustLookupPanic 验证 MustLookup 对未注册 opcode 会 panic。
func TestRegistryMustLookupPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLookup of unregistered opcode should panic")
		}
	}()
	// opcode 254 永久保留，未注册
	MustLookup(Opcode(254))
}

// TestRegistryAllRegisteredOrdered 验证 AllRegistered 按 opcode 升序返回。
func TestRegistryAllRegisteredOrdered(t *testing.T) {
	all := AllRegistered()
	for i := 1; i < len(all); i++ {
		if all[i].Opcode <= all[i-1].Opcode {
			t.Errorf("AllRegistered not ordered at index %d: %d <= %d", i, all[i].Opcode, all[i-1].Opcode)
		}
	}
}

// TestRegistryOpcodeInBasicRange 验证基础段 [0-169] 已注册的 opcode 数量合理。
func TestRegistryOpcodeInBasicRange(t *testing.T) {
	count := 0
	for _, m := range AllRegistered() {
		if m.Opcode.IsBasic() {
			count++
		}
	}
	// 基础段应有相当数量的注册指令
	if count < 50 {
		t.Errorf("too few basic segment opcodes registered: got %d", count)
	}
}

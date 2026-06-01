package script

import "testing"

// TestMetadataFields 验证 Metadata 辅助方法正确。
func TestMetadataFields(t *testing.T) {
	m := Metadata{
		Opcode:         NIL,
		Mnemonic:       "NIL",
		Category:       1,
		ArgCount:       ArgNone,
		ReturnCount:    1,
		AssocDataParam: -1,
		Availability:   availAll,
		Deterministic:  true,
		CostTier:       CostTierFree,
	}

	if m.HasAssocData() {
		t.Error("NIL should not have assoc data")
	}
	if !m.IsUnlockSafe() {
		t.Error("NIL should be unlock-safe")
	}
	if !m.IsPublic() {
		t.Error("NIL should be public")
	}
	if !m.IsPrivate() {
		t.Error("NIL should be private")
	}
	if m.IsDisabled() {
		t.Error("NIL should not be disabled")
	}

	// 测试有关联数据的指令
	m2 := Metadata{AssocDataParam: 0}
	if !m2.HasAssocData() {
		t.Error("metadata with AssocDataParam=0 should have assoc data")
	}

	// 测试禁用指令元数据
	m3 := Metadata{Availability: availDisabledPublic}
	if !m3.IsDisabled() {
		t.Error("should be disabled")
	}
	if !m3.IsPrivate() {
		t.Error("disabled instruction should still be available in private path")
	}
	if m3.IsPublic() {
		t.Error("disabled instruction should not be public")
	}
}

// TestArgModel 验证 ArgModel 辅助方法。
func TestArgModel(t *testing.T) {
	tests := []struct {
		m        ArgModel
		isFixed  bool
		isVariad bool
		isSemi   bool
	}{
		{ArgNone, false, false, false},
		{FixedArgs(1), true, false, false},
		{FixedArgs(3), true, false, false},
		{ArgVariadic, false, true, false},
		{ArgSemiFixed, false, false, true},
	}
	for _, tc := range tests {
		if got := tc.m.IsFixed(); got != tc.isFixed {
			t.Errorf("ArgModel(%d).IsFixed() = %v, want %v", tc.m, got, tc.isFixed)
		}
		if got := tc.m.IsVariadic(); got != tc.isVariad {
			t.Errorf("ArgModel(%d).IsVariadic() = %v, want %v", tc.m, got, tc.isVariad)
		}
		if got := tc.m.IsSemiFixed(); got != tc.isSemi {
			t.Errorf("ArgModel(%d).IsSemiFixed() = %v, want %v", tc.m, got, tc.isSemi)
		}
	}
}

// TestFixedArgsPanic 验证 FixedArgs(0) 会 panic。
func TestFixedArgsPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("FixedArgs(0) should panic")
		}
	}()
	FixedArgs(0)
}

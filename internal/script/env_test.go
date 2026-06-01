package script

import "testing"

// env_test.go 测试 Environment/WitnessProvider 接口以及 VM 全局变量/签名标注操作。

// mockEnv 实现 Environment 接口（测试用）。
type mockEnv struct {
	data map[string]Value
}

func newMockEnv(kvs ...interface{}) *mockEnv {
	m := &mockEnv{data: make(map[string]Value)}
	for i := 0; i+1 < len(kvs); i += 2 {
		k := kvs[i].(string)
		m.data[k] = kvs[i+1].(Value)
	}
	return m
}

func (m *mockEnv) Lookup(name string) (Value, error) {
	if v, ok := m.data[name]; ok {
		return v, nil
	}
	return NilValue(), nil
}

// mockWitness 实现 WitnessProvider 接口（测试用）。
type mockWitness struct {
	data     []byte
	coinbase bool
}

func (w *mockWitness) GetWitness() []byte { return w.data }
func (w *mockWitness) IsCoinbase() bool   { return w.coinbase }

// ─── 全局变量测试 ─────────────────────────────────────────────────────────────

func TestVarSetVar(t *testing.T) {
	vm := NewVM()
	// 初始值应为 nil
	if vm.GetGlobalVar(0).Typ() != TypeNil {
		t.Fatalf("expected Nil, got %s", vm.GetGlobalVar(0).Typ())
	}
	// 设置后读取
	vm.SetGlobalVar(0, IntValue(42))
	got := vm.GetGlobalVar(0)
	n, err := got.AsInt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}
	// 设置边界索引 255
	vm.SetGlobalVar(255, BoolValue(true))
	b, err := vm.GetGlobalVar(255).AsBool()
	if err != nil || !b {
		t.Fatal("expected true at index 255")
	}
}

// ─── 签名标注测试 ─────────────────────────────────────────────────────────────

func TestSigned(t *testing.T) {
	vm := NewVM()
	if vm.GetSigned(0) {
		t.Fatal("should be false initially")
	}
	vm.SetSigned(0)
	if !vm.GetSigned(0) {
		t.Fatal("should be true after SetSigned")
	}
	// 其他序位不受影响
	if vm.GetSigned(1) {
		t.Fatal("index 1 should still be false")
	}
	// 边界序位
	vm.SetSigned(255)
	if !vm.GetSigned(255) {
		t.Fatal("index 255 should be true after SetSigned")
	}
}

// ─── PublicEnd 标记测试 ───────────────────────────────────────────────────────

func TestPassedPublicEnd(t *testing.T) {
	vm := NewVM()
	if vm.PassedPublicEnd() {
		t.Fatal("should be false initially")
	}
	vm.MarkPublicEnd()
	if !vm.PassedPublicEnd() {
		t.Fatal("should be true after MarkPublicEnd")
	}
}

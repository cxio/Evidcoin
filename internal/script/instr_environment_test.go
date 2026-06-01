package script

import (
	"encoding/binary"
	"testing"
)

// instr_environment_test.go 测试环境指令 [128-137]。

// makeULEB128AttrParam 构造 ULEB128 附参的小端 uint64 字节（模拟 decodeAttrParams 输出）。
func makeULEB128AttrParam(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

// makeFrame 构造最简 InstrFrame。
func makeFrame(op Opcode, attrParams ...[]byte) *InstrFrame {
	return &InstrFrame{Op: op, AttrParams: attrParams}
}

// ─── VAR 指令测试 ─────────────────────────────────────────────────────────────

func TestExecVAR(t *testing.T) {
	cases := []struct {
		name  string
		idx   uint64
		value Value
	}{
		{"index 0 Int", 0, IntValue(100)},
		{"index 5 Bool", 5, BoolValue(true)},
		{"index 255 String", 255, StringValue("hello")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			vm.SetGlobalVar(int(tc.idx), tc.value)
			f := makeFrame(VAR, makeULEB128AttrParam(tc.idx))
			if err := execVAR(vm, f); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, err := vm.stack.Pop()
			if err != nil {
				t.Fatalf("stack pop error: %v", err)
			}
			if !got.Equal(tc.value) {
				t.Fatalf("expected %v, got %v", tc.value, got)
			}
		})
	}
}

// ─── SETVAR 指令测试 ──────────────────────────────────────────────────────────

func TestExecSETVAR(t *testing.T) {
	cases := []struct {
		name  string
		idx   uint64
		value Value
	}{
		{"set index 0 Int", 0, IntValue(999)},
		{"set index 10 Bytes", 10, BytesValue([]byte{1, 2, 3})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			// 将值压栈（SETVAR 从栈取）
			vm.stack.Push(tc.value)
			f := makeFrame(SETVAR, makeULEB128AttrParam(tc.idx))
			if err := execSETVAR(vm, f); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := vm.GetGlobalVar(int(tc.idx))
			if !got.Equal(tc.value) {
				t.Fatalf("expected %v, got %v", tc.value, got)
			}
		})
	}
}

// ─── SIGNED 指令测试 ──────────────────────────────────────────────────────────

func TestExecSIGNED(t *testing.T) {
	t.Run("未标注时压入 false", func(t *testing.T) {
		vm := NewVM()
		f := makeFrame(SIGNED, makeULEB128AttrParam(0))
		if err := execSIGNED(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v, _ := vm.stack.Pop()
		b, _ := v.AsBool()
		if b {
			t.Fatal("expected false")
		}
	})
	t.Run("标注后压入 true", func(t *testing.T) {
		vm := NewVM()
		vm.SetSigned(0)
		f := makeFrame(SIGNED, makeULEB128AttrParam(0))
		if err := execSIGNED(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v, _ := vm.stack.Pop()
		b, _ := v.AsBool()
		if !b {
			t.Fatal("expected true")
		}
	})
}

// ─── ENV 指令测试 ─────────────────────────────────────────────────────────────

func TestExecENV(t *testing.T) {
	t.Run("查询 BlockTime 返回期望值", func(t *testing.T) {
		env := newMockEnv("BlockTime", IntValue(12345))
		vm := NewVM(WithEnvironment(env))
		f := makeFrame(ENV, makeULEB128AttrParam(0)) // 0 → "BlockTime"
		if err := execENV(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v, _ := vm.stack.Pop()
		n, _ := v.AsInt()
		if n != 12345 {
			t.Fatalf("expected 12345, got %d", n)
		}
	})
	t.Run("env 为 nil 时压入 Nil", func(t *testing.T) {
		vm := NewVM()
		f := makeFrame(ENV, makeULEB128AttrParam(0))
		if err := execENV(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v, _ := vm.stack.Pop()
		if v.Typ() != TypeNil {
			t.Fatalf("expected Nil, got %s", v.Typ())
		}
	})
}

// ─── INOUT 指令测试 ───────────────────────────────────────────────────────────

func TestExecINOUT(t *testing.T) {
	t.Run("公共路径返回 ScriptError", func(t *testing.T) {
		vm := NewVM()
		frames := []InstrFrame{{Op: INOUT}}
		state := vm.Run(frames)
		if state != StateScriptError {
			t.Fatalf("expected ScriptError, got %s", state)
		}
	})
	t.Run("私有路径同样返回 ScriptError", func(t *testing.T) {
		vm := NewVM(WithMode(ModePrivate))
		frames := []InstrFrame{{Op: INOUT}}
		state := vm.Run(frames)
		if state != StateScriptError {
			t.Fatalf("expected ScriptError in private mode, got %s", state)
		}
	})
}

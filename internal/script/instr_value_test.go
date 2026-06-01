package script

import (
	"encoding/binary"
	"math"
	"testing"
)

// buildFrame 快速构造 InstrFrame（用于测试）。
func buildFrame(op Opcode, attrParams [][]byte, assocData []byte) InstrFrame {
	return InstrFrame{Op: op, AttrParams: attrParams, AssocData: assocData}
}

// runFrames 在新 VM 上执行帧序列，返回 VM（供后续断言）。
func runFrames(t *testing.T, frames []InstrFrame, opts ...VMOption) *VM {
	t.Helper()
	vm := NewVM(opts...)
	vm.Run(frames)
	return vm
}

// TestInstrValueNILTRUEFALSE 测试 NIL/TRUE/FALSE 压栈。
func TestInstrValueNILTRUEFALSE(t *testing.T) {
	cases := []struct {
		name string
		op   Opcode
		typ  Type
	}{
		{"NIL", NIL, TypeNil},
		{"TRUE", TRUE, TypeBool},
		{"FALSE", FALSE, TypeBool},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			frames := []InstrFrame{{Op: tc.op}}
			vm.Run(frames)
			if vm.stack.Len() != 1 {
				t.Fatalf("stack len = %d, want 1", vm.stack.Len())
			}
			v, _ := vm.stack.Top()
			if v.Typ() != tc.typ {
				t.Errorf("top type = %v, want %v", v.Typ(), tc.typ)
			}
			if tc.op == TRUE {
				b, _ := v.AsBool()
				if !b {
					t.Error("TRUE should push true")
				}
			}
			if tc.op == FALSE {
				b, _ := v.AsBool()
				if b {
					t.Error("FALSE should push false")
				}
			}
		})
	}
}

// TestInstrValueBYTE_LIT 测试 BYTE_LIT 压入单字节字面量。
func TestInstrValueBYTE_LIT(t *testing.T) {
	vm := NewVM()
	frames := []InstrFrame{{Op: BYTE_LIT, AttrParams: [][]byte{{0xab}}}}
	vm.Run(frames)
	v, _ := vm.stack.Top()
	b, _ := v.AsByte()
	if b != 0xab {
		t.Errorf("BYTE_LIT: got 0x%x, want 0xab", b)
	}
}

// TestInstrValueRUNE_LIT 测试 RUNE_LIT 压入 Unicode 码点。
func TestInstrValueRUNE_LIT(t *testing.T) {
	vm := NewVM()
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32('中'))
	frames := []InstrFrame{{Op: RUNE_LIT, AttrParams: [][]byte{buf}}}
	vm.Run(frames)
	v, _ := vm.stack.Top()
	r, _ := v.AsRune()
	if r != '中' {
		t.Errorf("RUNE_LIT: got %v, want '中'", r)
	}
}

// TestInstrValueINT_LIT 测试 INT_LIT 压入 int64 字面量。
func TestInstrValueINT_LIT(t *testing.T) {
	cases := []int64{0, 1, 127, 128, 255, 65535, -1}
	for _, want := range cases {
		vm := NewVM()
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(want))
		frames := []InstrFrame{{Op: INT_LIT, AttrParams: [][]byte{buf}}}
		vm.Run(frames)
		v, _ := vm.stack.Top()
		got, _ := v.AsInt()
		if got != want {
			t.Errorf("INT_LIT(%d): got %d", want, got)
		}
	}
}

// TestInstrValueFLOAT_LIT 测试 FLOAT_LIT 合法值压栈，NaN/Inf 触发 ScriptError。
func TestInstrValueFLOAT_LIT(t *testing.T) {
	// 合法值
	vm := NewVM()
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(3.14))
	frames := []InstrFrame{{Op: FLOAT_LIT, AttrParams: [][]byte{buf}}}
	vm.Run(frames)
	v, _ := vm.stack.Top()
	f, _ := v.AsFloat()
	if f != 3.14 {
		t.Errorf("FLOAT_LIT(3.14): got %v", f)
	}

	// NaN 字面量应触发 ScriptError
	for name, bits := range map[string]uint64{
		"NaN":  math.Float64bits(math.NaN()),
		"+Inf": math.Float64bits(math.Inf(1)),
		"-Inf": math.Float64bits(math.Inf(-1)),
	} {
		t.Run(name, func(t *testing.T) {
			vm2 := NewVM()
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, bits)
			vm2.Run([]InstrFrame{{Op: FLOAT_LIT, AttrParams: [][]byte{b}}})
			if vm2.State() != StateScriptError {
				t.Errorf("FLOAT_LIT(%s) state = %v, want ScriptError", name, vm2.State())
			}
		})
	}
}

// TestInstrValueSTRING_LIT 测试 STRING_LIT 压入字符串。
func TestInstrValueSTRING_LIT(t *testing.T) {
	vm := NewVM()
	content := []byte("hello world")
	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(content)))
	frames := []InstrFrame{{Op: STRING_LIT, AttrParams: [][]byte{lenBuf}, AssocData: content}}
	vm.Run(frames)
	v, _ := vm.stack.Top()
	s, _ := v.AsString()
	if s != "hello world" {
		t.Errorf("STRING_LIT: got %q", s)
	}
}

// TestInstrValueDATA_LIT 测试 DATA_LIT 压入字节序列。
func TestInstrValueDATA_LIT(t *testing.T) {
	vm := NewVM()
	data := []byte{0xde, 0xad, 0xbe, 0xef}
	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(data)))
	frames := []InstrFrame{{Op: DATA_LIT, AttrParams: [][]byte{lenBuf}, AssocData: data}}
	vm.Run(frames)
	v, _ := vm.stack.Top()
	b, _ := v.AsBytes()
	if len(b) != 4 || b[0] != 0xde {
		t.Errorf("DATA_LIT: got %v", b)
	}
}

// TestInstrValueBIGINT_LIT 测试 BIGINT_LIT 拒绝前导零和负零。
func TestInstrValueBIGINT_LIT(t *testing.T) {
	// 合法：正数 magnitude=[0x01]
	vm := NewVM()
	vm.Run([]InstrFrame{{Op: BIGINT_LIT, AttrParams: [][]byte{{0x01}}, AssocData: []byte{0x01}}})
	if vm.State() == StateScriptError {
		t.Error("valid BIGINT_LIT should not ScriptError")
	}

	// 拒绝前导零：magnitude=[0x00, 0x01]
	vm2 := NewVM()
	vm2.Run([]InstrFrame{{Op: BIGINT_LIT, AttrParams: [][]byte{{0x02}}, AssocData: []byte{0x00, 0x01}}})
	if vm2.State() != StateScriptError {
		t.Error("BIGINT_LIT with leading zero should ScriptError")
	}

	// 拒绝负零：符号位=1，magnitude 为空
	vm3 := NewVM()
	vm3.Run([]InstrFrame{{Op: BIGINT_LIT, AttrParams: [][]byte{{0x80}}, AssocData: []byte{}}})
	if vm3.State() != StateScriptError {
		t.Error("BIGINT_LIT negative zero should ScriptError")
	}
}

// TestInstrValueCODE_LIT 测试 CODE_LIT 压入字节序列（TypeCode）。
func TestInstrValueCODE_LIT(t *testing.T) {
	vm := NewVM()
	code := []byte{byte(NIL), byte(END)}
	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(code)))
	vm.Run([]InstrFrame{{Op: CODE_LIT, AttrParams: [][]byte{lenBuf}, AssocData: code}})
	v, _ := vm.stack.Top()
	if v.Typ() != TypeCode {
		t.Errorf("CODE_LIT: top type = %v, want Code", v.Typ())
	}
}

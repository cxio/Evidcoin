package script

import (
	"testing"
)

// instr_function_test.go 测试函数段指令 [170-224]。

// ─── FN_BASE58 往返编码测试 ───────────────────────────────────────────────────

func TestExecFN_BASE58_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"single byte", []byte{0x00}},
		{"hello", []byte("hello world")},
		{"binary", []byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02, 0x03}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			// Bytes → String
			vm.stack.Push(BytesValue(tc.data))
			f := &InstrFrame{Op: FN_BASE58}
			if err := execFN_BASE58(vm, f); err != nil {
				t.Fatalf("encode error: %v", err)
			}
			encoded, _ := vm.stack.Pop()

			// String → Bytes
			vm.stack.Push(encoded)
			if err := execFN_BASE58(vm, f); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			decoded, _ := vm.stack.Pop()
			b, _ := decoded.AsBytes()
			if len(b) != len(tc.data) {
				t.Fatalf("length mismatch: got %d, want %d", len(b), len(tc.data))
			}
			for i := range b {
				if b[i] != tc.data[i] {
					t.Fatalf("data mismatch at index %d", i)
				}
			}
		})
	}
}

// ─── FN_BASE64 往返编码测试 ───────────────────────────────────────────────────

func TestExecFN_BASE64_RoundTrip(t *testing.T) {
	data := []byte("hello, base64 URL encoding!")
	vm := NewVM()
	f := &InstrFrame{Op: FN_BASE64}

	vm.stack.Push(BytesValue(data))
	if err := execFN_BASE64(vm, f); err != nil {
		t.Fatalf("encode error: %v", err)
	}
	encoded, _ := vm.stack.Pop()
	s, _ := encoded.AsString()
	t.Logf("encoded: %s", s)

	vm.stack.Push(encoded)
	if err := execFN_BASE64(vm, f); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	decoded, _ := vm.stack.Pop()
	b, _ := decoded.AsBytes()
	if string(b) != string(data) {
		t.Fatalf("round-trip mismatch: got %q, want %q", b, data)
	}
}

// ─── FN_BASE32 往返编码测试 ───────────────────────────────────────────────────

func TestExecFN_BASE32_RoundTrip(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 255, 128, 0}
	vm := NewVM()
	f := &InstrFrame{Op: FN_BASE32}

	vm.stack.Push(BytesValue(data))
	if err := execFN_BASE32(vm, f); err != nil {
		t.Fatalf("encode error: %v", err)
	}
	encoded, _ := vm.stack.Pop()

	vm.stack.Push(encoded)
	if err := execFN_BASE32(vm, f); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	decoded, _ := vm.stack.Pop()
	b, _ := decoded.AsBytes()
	if len(b) != len(data) {
		t.Fatalf("length mismatch: got %d, want %d", len(b), len(data))
	}
	for i := range b {
		if b[i] != data[i] {
			t.Fatalf("data mismatch at index %d", i)
		}
	}
}

// ─── FN_HASH256 测试 ─────────────────────────────────────────────────────────

func TestExecFN_HASH256(t *testing.T) {
	input := []byte("hello hash256")
	cases := []struct {
		name string
		algo uint64
	}{
		{"BLAKE3", hashAlgoBLAKE3},
		{"BLAKE2b", hashAlgoBLAKE2},
		{"SHA2-256", hashAlgoSHA2},
		{"SHA3-256", hashAlgoSHA3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			vm.stack.Push(BytesValue(input))
			f := &InstrFrame{Op: FN_HASH256, AttrParams: [][]byte{makeULEB128AttrParam(tc.algo)}}
			if err := execFN_HASH256(vm, f); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			v, _ := vm.stack.Pop()
			b, _ := v.AsBytes()
			if len(b) != 32 {
				t.Fatalf("expected 32 bytes, got %d", len(b))
			}
		})
	}
}

// ─── FN_HASH512 测试 ─────────────────────────────────────────────────────────

func TestExecFN_HASH512(t *testing.T) {
	input := []byte("hello hash512")
	cases := []struct {
		name string
		algo uint64
	}{
		{"BLAKE3", hashAlgoBLAKE3},
		{"BLAKE2b-512", hashAlgoBLAKE2},
		{"SHA2-512", hashAlgoSHA2},
		{"SHA3-512", hashAlgoSHA3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			vm.stack.Push(BytesValue(input))
			f := &InstrFrame{Op: FN_HASH512, AttrParams: [][]byte{makeULEB128AttrParam(tc.algo)}}
			if err := execFN_HASH512(vm, f); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			v, _ := vm.stack.Pop()
			b, _ := v.AsBytes()
			if len(b) != 64 {
				t.Fatalf("expected 64 bytes, got %d", len(b))
			}
		})
	}
}

// ─── FN_HASH224 测试 ─────────────────────────────────────────────────────────

func TestExecFN_HASH224(t *testing.T) {
	input := []byte("hello hash224")
	cases := []struct {
		name string
		algo uint64
	}{
		{"BLAKE3-224", hashAlgoBLAKE3},
		{"BLAKE2b-224", hashAlgoBLAKE2},
		{"SHA2-224", hashAlgoSHA2},
		{"SHA3-224", hashAlgoSHA3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			vm.stack.Push(BytesValue(input))
			f := &InstrFrame{Op: FN_HASH224, AttrParams: [][]byte{makeULEB128AttrParam(tc.algo)}}
			if err := execFN_HASH224(vm, f); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			v, _ := vm.stack.Pop()
			b, _ := v.AsBytes()
			if len(b) != 28 {
				t.Fatalf("expected 28 bytes, got %d", len(b))
			}
		})
	}
}

// ─── FN_PUBHASH 测试 ─────────────────────────────────────────────────────────

func TestExecFN_PUBHASH(t *testing.T) {
	// 构造任意 33 字节假公钥（实际应为 EC 压缩公钥，这里仅测试接口）
	pubKey := make([]byte, 33)
	for i := range pubKey {
		pubKey[i] = byte(i + 1)
	}
	vm := NewVM()
	vm.stack.Push(BytesValue(pubKey))
	f := &InstrFrame{Op: FN_PUBHASH}
	if err := execFN_PUBHASH(vm, f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, _ := vm.stack.Pop()
	b, _ := v.AsBytes()
	if len(b) != 32 {
		t.Fatalf("expected 32 bytes pubhash, got %d", len(b))
	}
}

// ─── FN_PRINTF 测试 ──────────────────────────────────────────────────────────

func TestExecFN_PRINTF(t *testing.T) {
	t.Run("格式化整数", func(t *testing.T) {
		vm := NewVM()
		// 将实参预先加入实参区
		vm.args.Enqueue(StringValue("value=%d"))
		vm.args.Enqueue(IntValue(42))
		f := &InstrFrame{Op: FN_PRINTF}
		if err := execFN_PRINTF(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v, _ := vm.stack.Pop()
		s, _ := v.AsString()
		if s != "value=42" {
			t.Fatalf("expected 'value=42', got %q", s)
		}
	})
	t.Run("无实参格式串", func(t *testing.T) {
		vm := NewVM()
		vm.stack.Push(StringValue("hello"))
		f := &InstrFrame{Op: FN_PRINTF}
		if err := execFN_PRINTF(vm, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v, _ := vm.stack.Pop()
		s, _ := v.AsString()
		if s != "hello" {
			t.Fatalf("expected 'hello', got %q", s)
		}
	})
}

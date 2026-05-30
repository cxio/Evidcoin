package types

import (
	"bytes"
	"testing"
)

func TestHashConstruction(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		input   int // input length
		wantErr bool
	}{
		{"Hash32 exact", 32, 32, false},
		{"Hash32 short", 32, 31, true},
		{"Hash32 long", 32, 33, true},
		{"Hash48 exact", 48, 48, false},
		{"Hash48 short", 48, 47, true},
		{"Hash48 long", 48, 49, true},
		{"Hash64 exact", 64, 64, false},
		{"Hash64 short", 64, 63, true},
		{"Hash64 long", 64, 65, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := make([]byte, tt.input)
			for i := range in {
				in[i] = byte(i + 1)
			}
			var err error
			switch tt.size {
			case 32:
				_, err = NewHash32(in)
			case 48:
				_, err = NewHash48(in)
			case 64:
				_, err = NewHash64(in)
			}
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestHashBytesNoAlias(t *testing.T) {
	in := make([]byte, 48)
	for i := range in {
		in[i] = byte(i)
	}
	h, err := NewHash48(in)
	if err != nil {
		t.Fatal(err)
	}
	// 修改源切片不应影响已存储哈希。
	in[0] = 0xFF
	got := h.Bytes()
	if got[0] != 0 {
		t.Fatalf("hash aliases input buffer: got[0]=%d", got[0])
	}
	// 修改返回切片不应影响已存储哈希。
	got[1] = 0xFF
	again := h.Bytes()
	if again[1] != 1 {
		t.Fatalf("Bytes aliases internal array: again[1]=%d", again[1])
	}
}

func TestIDConstruction(t *testing.T) {
	in48 := make([]byte, 48)
	if _, err := NewBlockID(in48); err != nil {
		t.Fatalf("NewBlockID: %v", err)
	}
	if _, err := NewTxID(in48); err != nil {
		t.Fatalf("NewTxID: %v", err)
	}
	if _, err := NewBlockID(in48[:47]); err == nil {
		t.Fatal("expected error for short BlockID")
	}
	in32 := make([]byte, 32)
	if _, err := NewAddressHash(in32); err != nil {
		t.Fatalf("NewAddressHash: %v", err)
	}
	if _, err := NewMintHash(in32); err != nil {
		t.Fatalf("NewMintHash: %v", err)
	}
	in64 := make([]byte, 64)
	if _, err := NewAttachmentHash(in64); err != nil {
		t.Fatalf("NewAttachmentHash: %v", err)
	}
}

func TestIDBytesRoundTrip(t *testing.T) {
	in := make([]byte, 48)
	for i := range in {
		in[i] = byte(i * 3)
	}
	id := MustBlockID(in)
	if !bytes.Equal(id.Bytes(), in) {
		t.Fatal("BlockID round-trip mismatch")
	}
}

// TestIDTypeIsolation 说明 BlockID 与 TxID 是彼此独立的命名类型。
// 二者虽然同为 48 字节布局，但不能隐式赋值，必须显式转换；
// 本测试能编译仅因为使用了显式转换。
func TestIDTypeIsolation(t *testing.T) {
	id := MustBlockID(make([]byte, 48))
	tx := TxID(id) // 必须显式转换；直接赋值将无法通过编译
	if !bytes.Equal(tx.Bytes(), id.Bytes()) {
		t.Fatal("conversion changed bytes")
	}
}

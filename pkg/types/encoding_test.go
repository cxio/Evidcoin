package types

import (
	"bytes"
	"math"
	"math/big"
	"testing"
)

func TestCanonicalEncodingFixedWidth(t *testing.T) {
	if got := AppendUint16BE(nil, 0x0102); !bytes.Equal(got, []byte{0x01, 0x02}) {
		t.Errorf("AppendUint16BE = %x", got)
	}
	if got := AppendUint32BE(nil, 0x01020304); !bytes.Equal(got, []byte{0x01, 0x02, 0x03, 0x04}) {
		t.Errorf("AppendUint32BE = %x", got)
	}
	if got := AppendUint64BE(nil, 0x0102030405060708); !bytes.Equal(got, []byte{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Errorf("AppendUint64BE = %x", got)
	}
	if _, _, err := ReadUint32BE([]byte{1, 2, 3}); err == nil {
		t.Error("expected short error for ReadUint32BE")
	}
	v, n, err := ReadUint32BE([]byte{1, 2, 3, 4, 5})
	if err != nil || n != 4 || v != 0x01020304 {
		t.Errorf("ReadUint32BE = %x, %d, %v", v, n, err)
	}
}

func TestCanonicalEncodingVarintVectors(t *testing.T) {
	tests := []struct {
		v    uint64
		want []byte
	}{
		{0, []byte{0x00}},
		{127, []byte{0x7F}},
		{128, []byte{0x80, 0x01}},
		{16383, []byte{0xFF, 0x7F}},
		{16384, []byte{0x80, 0x80, 0x01}},
	}
	for _, tt := range tests {
		got := AppendVarUint(nil, tt.v)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendVarUint(%d) = %x, want %x", tt.v, got, tt.want)
		}
		dv, n, err := ReadVarUint(got)
		if err != nil || n != len(tt.want) || dv != tt.v {
			t.Errorf("ReadVarUint(%x) = %d, %d, %v", got, dv, n, err)
		}
	}
}

func TestCanonicalEncodingVarintMaxLen(t *testing.T) {
	got := AppendVarUint(nil, math.MaxUint64)
	if len(got) != 10 {
		t.Fatalf("max uint64 varint len = %d, want 10", len(got))
	}
	v, n, err := ReadVarUint(got)
	if err != nil || n != 10 || v != math.MaxUint64 {
		t.Fatalf("round-trip max uint64: %d, %d, %v", v, n, err)
	}
}

func TestCanonicalEncodingVarintRejects(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
	}{
		{"non-minimal 0x80 0x00", []byte{0x80, 0x00}},
		{"non-minimal 0xFF 0x00", []byte{0xFF, 0x00}},
		{"truncated", []byte{0x80}},
		{"overflow 11 bytes", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := ReadVarUint(tt.in); err == nil {
				t.Errorf("ReadVarUint(%x) accepted, want error", tt.in)
			}
		})
	}
}

func TestCanonicalEncodingBytes(t *testing.T) {
	got := AppendBytes(nil, []byte{0xAA, 0xBB, 0xCC})
	if !bytes.Equal(got, []byte{0x03, 0xAA, 0xBB, 0xCC}) {
		t.Errorf("AppendBytes = %x", got)
	}
	empty := AppendBytes(nil, nil)
	if !bytes.Equal(empty, []byte{0x00}) {
		t.Errorf("AppendBytes(empty) = %x", empty)
	}
	b, n, err := ReadBytes([]byte{0x02, 0x01, 0x02, 0xFF})
	if err != nil || n != 3 || !bytes.Equal(b, []byte{0x01, 0x02}) {
		t.Errorf("ReadBytes = %x, %d, %v", b, n, err)
	}
	if _, _, err := ReadBytes([]byte{0x05, 0x01}); err == nil {
		t.Error("expected short error for ReadBytes")
	}
}

func TestCanonicalEncodingOptional(t *testing.T) {
	absent := AppendOptional(nil, false, nil)
	if !bytes.Equal(absent, []byte{0x00}) {
		t.Errorf("optional absent = %x", absent)
	}
	present := AppendOptional(nil, true, func(dst []byte) []byte {
		return append(dst, 0x42)
	})
	if !bytes.Equal(present, []byte{0x01, 0x42}) {
		t.Errorf("optional present = %x", present)
	}
	if p, n, err := ReadOptionalMarker([]byte{0x00}); err != nil || p || n != 1 {
		t.Errorf("ReadOptionalMarker(absent) = %v, %d, %v", p, n, err)
	}
	if p, n, err := ReadOptionalMarker([]byte{0x01, 0x42}); err != nil || !p || n != 1 {
		t.Errorf("ReadOptionalMarker(present) = %v, %d, %v", p, n, err)
	}
	if _, _, err := ReadOptionalMarker([]byte{0x02}); err == nil {
		t.Error("expected error for invalid optional marker")
	}
}

func TestCanonicalEncodingList(t *testing.T) {
	got := AppendList(nil, []uint16{0x0102, 0x0304}, AppendUint16BE)
	want := []byte{0x02, 0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(got, want) {
		t.Errorf("AppendList = %x, want %x", got, want)
	}
}

func TestCanonicalEncodingBigInt(t *testing.T) {
	tests := []struct {
		name string
		in   *big.Int
		want []byte
	}{
		{"zero", big.NewInt(0), []byte{0x00}},
		{"one", big.NewInt(1), []byte{0x01, 0x01}},
		{"neg one", big.NewInt(-1), []byte{0x81, 0x01}},
		{"256", big.NewInt(256), []byte{0x02, 0x01, 0x00}},
		{"neg 256", big.NewInt(-256), []byte{0x82, 0x01, 0x00}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AppendBigInt(nil, tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("AppendBigInt(%s) = %x, want %x", tt.in, got, tt.want)
			}
			x, n, err := ReadBigInt(got)
			if err != nil || n != len(tt.want) || x.Cmp(tt.in) != 0 {
				t.Fatalf("ReadBigInt = %s, %d, %v", x, n, err)
			}
		})
	}
}

func TestCanonicalEncodingBigIntRejects(t *testing.T) {
	// 绝对值存在前导零（非最短表示）
	if _, _, err := ReadBigInt([]byte{0x02, 0x00, 0x01}); err == nil {
		t.Error("expected non-minimal error")
	}
	// 负零
	if _, _, err := ReadBigInt([]byte{0x80}); err == nil {
		t.Error("expected negative-zero error")
	}
	// 绝对值长度超过 127 字节
	big128 := new(big.Int).Lsh(big.NewInt(1), 128*8)
	if _, err := AppendBigInt(nil, big128); err == nil {
		t.Error("expected too-large error")
	}
}

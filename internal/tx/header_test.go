package tx

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// mustHash32 在测试中由原始字节构造 Hash32。
func mustHash32(t *testing.T, b []byte) types.Hash32 {
	t.Helper()
	h, err := types.NewHash32(b)
	if err != nil {
		t.Fatalf("NewHash32: %v", err)
	}
	return h
}

// sampleHeader 构造一个用于测试的普通交易头（无 MintPKHash）。
func sampleHeader(t *testing.T) *TxHeader {
	t.Helper()
	return &TxHeader{
		Version:     1,
		HashInputs:  mustHash32(t, bytes.Repeat([]byte{0x11}, 32)),
		HashOutputs: mustHash32(t, bytes.Repeat([]byte{0x22}, 32)),
		Timestamp:   1700000000000,
	}
}

// TestTxHeaderCanonicalLayout 校验普通交易头字段顺序与缺省 MintPKHash 编码。
func TestTxHeaderCanonicalLayout(t *testing.T) {
	h := sampleHeader(t)

	var want []byte
	want = binary.BigEndian.AppendUint16(want, 1)
	want = append(want, bytes.Repeat([]byte{0x11}, 32)...)
	want = append(want, bytes.Repeat([]byte{0x22}, 32)...)
	want = binary.BigEndian.AppendUint64(want, uint64(1700000000000))
	want = append(want, 0x00) // MintPKHash 缺省，varint(0)

	got, err := h.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	// 2 + 32 + 32 + 8 + 1 = 75
	if len(got) != 75 {
		t.Fatalf("缺省 MintPKHash 交易头尺寸 = %d, 期望 75", len(got))
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("CanonicalBytes 不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestTxHeaderMintPKHashPresent 校验 len==32 的 MintPKHash 编码。
func TestTxHeaderMintPKHashPresent(t *testing.T) {
	h := sampleHeader(t)
	h.MintPKHash = bytes.Repeat([]byte{0xAB}, 32)

	got, err := h.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	// 75 + 32 = 107
	if len(got) != 107 {
		t.Fatalf("含 MintPKHash 交易头尺寸 = %d, 期望 107", len(got))
	}
	if got[74] != 0x20 {
		t.Fatalf("MintPKHash 长度前缀 = %#x, 期望 0x20", got[74])
	}
	if !bytes.Equal(got[75:], bytes.Repeat([]byte{0xAB}, 32)) {
		t.Fatalf("MintPKHash 字节不匹配, got=%x", got[75:])
	}
}

// TestTxHeaderMintPKHashInvalidLength 校验非 {0,32} 长度被拒绝。
func TestTxHeaderMintPKHashInvalidLength(t *testing.T) {
	for _, n := range []int{1, 16, 31, 33, 48} {
		h := sampleHeader(t)
		h.MintPKHash = bytes.Repeat([]byte{0x01}, n)
		if _, err := h.CanonicalBytes(); err == nil {
			t.Fatalf("MintPKHash 长度 %d 应被拒绝", n)
		}
	}
}

// TestTxHeaderID 校验 TxID 算法、确定性与字段敏感性。
func TestTxHeaderID(t *testing.T) {
	h := sampleHeader(t)
	pre, err := h.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	want := crypto.HashTxHeader(pre)

	id, err := h.TxID()
	if err != nil {
		t.Fatalf("TxID: %v", err)
	}
	if id != want {
		t.Fatalf("TxID 与 tx.header 域哈希不一致")
	}
	if len(id.Bytes()) != 48 {
		t.Fatalf("TxID 长度 = %d, 期望 48", len(id.Bytes()))
	}

	// 确定性：相同字段相同 TxID。
	id2, _ := sampleHeader(t).TxID()
	if id != id2 {
		t.Fatal("相同交易头的 TxID 不一致")
	}

	// 字段敏感性：修改任一字段改变 TxID。
	h2 := sampleHeader(t)
	h2.Timestamp++
	if alt, _ := h2.TxID(); alt == id {
		t.Fatal("修改 Timestamp 未改变 TxID")
	}
	h3 := sampleHeader(t)
	h3.Version = 2
	if alt, _ := h3.TxID(); alt == id {
		t.Fatal("修改 Version 未改变 TxID")
	}
	h4 := sampleHeader(t)
	h4.HashOutputs = mustHash32(t, bytes.Repeat([]byte{0x33}, 32))
	if alt, _ := h4.TxID(); alt == id {
		t.Fatal("修改 HashOutputs 未改变 TxID")
	}
}

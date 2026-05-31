package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// fpBytes 构造 64 字节附件指纹测试值。
func fpBytes(b byte) types.AttachmentHash {
	var fp types.AttachmentHash
	for i := range fp {
		fp[i] = b
	}
	return fp
}

// ghBytes 构造 32 字节片组哈希测试值。
func ghBytes(b byte) types.TreeHash {
	var gh types.TreeHash
	for i := range gh {
		gh[i] = b
	}
	return gh
}

// TestAttachmentEncodeNoPiece 校验分片数量为 0 时省略片组哈希字段，
// 且总长字节统计整个结构（含自身）。
func TestAttachmentEncodeNoPiece(t *testing.T) {
	a := AttachmentID{
		Type:        [2]byte{0x01, 0x02},
		Fingerprint: fpBytes(0xAA),
		PieceCount:  0,
		Size:        1000,
	}
	got, err := a.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// body = Type(2) || Fingerprint(64) || PieceCount(2) || varint(1000)
	var body []byte
	body = append(body, 0x01, 0x02)
	body = append(body, fpBytes(0xAA).Bytes()...)
	body = types.AppendUint16BE(body, 0)
	body = types.AppendVarUint(body, 1000)
	want := append([]byte{byte(1 + len(body))}, body...)
	if !bytes.Equal(got, want) {
		t.Fatalf("无分片编码不匹配\n got=%x\nwant=%x", got, want)
	}
	if int(got[0]) != len(got) {
		t.Fatalf("总长字节应等于整个结构长度: total=%d len=%d", got[0], len(got))
	}
}

// TestAttachmentEncodeWithPiece 校验分片数量 ≥1 时编码片组哈希字段。
func TestAttachmentEncodeWithPiece(t *testing.T) {
	for _, pc := range []uint16{1, 5} {
		a := AttachmentID{
			Type:        [2]byte{0x03, 0x04},
			Fingerprint: fpBytes(0xBB),
			PieceCount:  pc,
			GroupHash:   ghBytes(0xCC),
			Size:        2 << 20,
		}
		got, err := a.Encode()
		if err != nil {
			t.Fatalf("Encode(pc=%d): %v", pc, err)
		}
		var body []byte
		body = append(body, 0x03, 0x04)
		body = append(body, fpBytes(0xBB).Bytes()...)
		body = types.AppendUint16BE(body, pc)
		body = append(body, ghBytes(0xCC).Bytes()...)
		body = types.AppendVarUint(body, 2<<20)
		want := append([]byte{byte(1 + len(body))}, body...)
		if !bytes.Equal(got, want) {
			t.Fatalf("含分片编码不匹配(pc=%d)\n got=%x\nwant=%x", pc, got, want)
		}
	}
}

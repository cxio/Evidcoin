package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestProofPayloadLayout 校验 Proof payload 编码顺序：Creator || Title || Content || AttachmentID，无接收者。
func TestProofPayloadLayout(t *testing.T) {
	p := Proof{
		Creator: []byte("creator"),
		Title:   []byte("title"),
		Content: []byte("content"),
	}
	got, err := p.Payload()
	if err != nil {
		t.Fatalf("Payload: %v", err)
	}
	var want []byte
	want = types.AppendBytes(want, []byte("creator"))
	want = types.AppendBytes(want, []byte("title"))
	want = types.AppendBytes(want, []byte("content"))
	want = types.AppendBytes(want, nil) // AttachmentID 缺省
	if !bytes.Equal(got, want) {
		t.Fatalf("Proof payload 不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestProofContentLimit 校验 Content 最多 2KB，Creator/Title <256。
func TestProofContentLimit(t *testing.T) {
	if _, err := (Proof{Content: bytes.Repeat([]byte{0}, 2049)}).Payload(); err == nil {
		t.Fatal("Content 2049 字节应被拒绝")
	}
	if _, err := (Proof{Creator: bytes.Repeat([]byte{0}, 256)}).Payload(); err == nil {
		t.Fatal("Creator 256 字节应被拒绝")
	}
	if _, err := (Proof{Title: bytes.Repeat([]byte{0}, 256)}).Payload(); err == nil {
		t.Fatal("Title 256 字节应被拒绝")
	}
}

// TestProofCreatorEmpty 校验 Creator 可空。
func TestProofCreatorEmpty(t *testing.T) {
	if _, err := (Proof{Title: []byte("t")}).Payload(); err != nil {
		t.Fatalf("空 Creator 应被允许: %v", err)
	}
}

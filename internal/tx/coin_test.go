package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestCoinPayloadLayout 校验 Coin payload 编码顺序：Receiver || Amount || Memo。
func TestCoinPayloadLayout(t *testing.T) {
	recv := bytes.Repeat([]byte{0xA0}, 32)
	memo := []byte("hello")
	c := Coin{Amount: 12345, Receiver: recv, Memo: memo}

	got, err := c.Payload()
	if err != nil {
		t.Fatalf("Payload: %v", err)
	}
	var want []byte
	want = types.AppendBytes(want, recv)    // Receiver(varint(len)||bytes)
	want = types.AppendVarUint(want, 12345) // Amount(varint, chx)
	want = types.AppendBytes(want, memo)    // Memo(varint(len)||bytes)
	if !bytes.Equal(got, want) {
		t.Fatalf("Coin payload 不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestCoinMemoDefault 校验缺省 Memo 以 varint(0) 编码并参与前像。
func TestCoinMemoDefault(t *testing.T) {
	c := Coin{Amount: 1, Receiver: []byte{0x01}}
	got, err := c.Payload()
	if err != nil {
		t.Fatalf("Payload: %v", err)
	}
	// Receiver(1B len + 1B) || Amount(1B) || Memo(1B len=0x00)
	if got[len(got)-1] != 0x00 {
		t.Fatalf("缺省 Memo 末字节应为 0x00, got=%x", got)
	}
	if len(got) != 1+2+1 {
		t.Fatalf("缺省 Memo payload 尺寸 = %d, 期望 4", len(got))
	}
}

// TestCoinReceiverEmpty 校验 Receiver 可为空（自定义脚本验证场景）。
func TestCoinReceiverEmpty(t *testing.T) {
	c := Coin{Amount: 1}
	if _, err := c.Payload(); err != nil {
		t.Fatalf("空 Receiver 应被允许: %v", err)
	}
}

// TestCoinFieldLimits 校验 Receiver/Memo 超过 255 字节被拒绝。
func TestCoinFieldLimits(t *testing.T) {
	if _, err := (Coin{Amount: 1, Receiver: bytes.Repeat([]byte{0}, 256)}).Payload(); err == nil {
		t.Fatal("Receiver 长度 256 应被拒绝")
	}
	if _, err := (Coin{Amount: 1, Memo: bytes.Repeat([]byte{0}, 256)}).Payload(); err == nil {
		t.Fatal("Memo 长度 256 应被拒绝")
	}
	// 边界 255 合法。
	if _, err := (Coin{Amount: 1, Receiver: bytes.Repeat([]byte{0}, 255), Memo: bytes.Repeat([]byte{0}, 255)}).Payload(); err != nil {
		t.Fatalf("255 字节字段应被允许: %v", err)
	}
}

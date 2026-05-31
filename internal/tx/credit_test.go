package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestCreditPayloadLayout 校验 Credit payload 编码顺序：
// Receiver || Creator || Title || Description || AttachmentID。
func TestCreditPayloadLayout(t *testing.T) {
	c := Credit{
		Receiver:    []byte("recv"),
		Creator:     []byte("creator"),
		Title:       []byte("title"),
		Description: []byte("desc"),
	}
	got, err := c.Payload()
	if err != nil {
		t.Fatalf("Payload: %v", err)
	}
	var want []byte
	want = types.AppendBytes(want, []byte("recv"))
	want = types.AppendBytes(want, []byte("creator"))
	want = types.AppendBytes(want, []byte("title"))
	want = types.AppendBytes(want, []byte("desc"))
	want = types.AppendBytes(want, nil) // AttachmentID 缺省 varint(0)
	if !bytes.Equal(got, want) {
		t.Fatalf("Credit payload 不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestCreditDescriptionLimit 校验 Description 最多 2KB。
func TestCreditDescriptionLimit(t *testing.T) {
	if _, err := (Credit{Description: bytes.Repeat([]byte{0}, 2049)}).Payload(); err == nil {
		t.Fatal("Description 2049 字节应被拒绝")
	}
	if _, err := (Credit{Description: bytes.Repeat([]byte{0}, 2048)}).Payload(); err != nil {
		t.Fatalf("Description 2048 字节应被允许: %v", err)
	}
}

// TestCreditShortFieldLimits 校验 Receiver/Creator/Title 均 <256 字节。
func TestCreditShortFieldLimits(t *testing.T) {
	big := bytes.Repeat([]byte{0}, 256)
	if _, err := (Credit{Receiver: big}).Payload(); err == nil {
		t.Fatal("Receiver 256 字节应被拒绝")
	}
	if _, err := (Credit{Creator: big}).Payload(); err == nil {
		t.Fatal("Creator 256 字节应被拒绝")
	}
	if _, err := (Credit{Title: big}).Payload(); err == nil {
		t.Fatal("Title 256 字节应被拒绝")
	}
}

// TestCreditAttachmentOptional 校验 AttachmentID 可选，存在时参与编码。
func TestCreditAttachmentOptional(t *testing.T) {
	att := []byte{0x05, 0x01, 0x02, 0x03, 0x04, 0x05}
	c := Credit{Title: []byte("t"), AttachmentID: att}
	got, err := c.Payload()
	if err != nil {
		t.Fatalf("Payload: %v", err)
	}
	if !bytes.Contains(got, att) {
		t.Fatal("AttachmentID 应参与编码")
	}
}

// TestCreditOutputCountLimit 校验每交易凭信输出 ≤2，第 3 笔被拒绝。
func TestCreditOutputCountLimit(t *testing.T) {
	credit := Output{Type: TypeCredit}
	coin := Output{Type: TypeCoin}
	if err := ValidateCreditOutputCount([]Output{credit, coin, credit}); err != nil {
		t.Fatalf("2 笔凭信输出应被允许: %v", err)
	}
	if err := ValidateCreditOutputCount([]Output{credit, credit, credit}); err == nil {
		t.Fatal("3 笔凭信输出应被拒绝")
	}
}

// TestCreditExpiry 校验 31 年过期边界：age > 31×87661 失效，边界相等仍可用。
func TestCreditExpiry(t *testing.T) {
	boundary := uint64(31 * types.BlocksPerYear)
	if CreditExpired(boundary) {
		t.Fatal("age == 31×87661 应仍可引用花销")
	}
	if !CreditExpired(boundary + 1) {
		t.Fatal("age > 31×87661 应失效")
	}
}

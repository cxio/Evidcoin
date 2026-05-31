package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// sampleRef 构造一个合法的输入短引用（TxIDPart 恰好 16 字节）。
func sampleRef(fill byte) OutPoint {
	return OutPoint{
		Year:     2024,
		TxIDPart: bytes.Repeat([]byte{fill}, MinTxIDPartLen),
		OutIndex: 1,
	}
}

// TestLeadInputKind 校验首领输入固定标记为币金输入。
func TestLeadInputKind(t *testing.T) {
	lead := LeadInput{Ref: sampleRef(0x01), UnlockScript: []byte{0xAA}}
	if lead.Kind() != InputCoin {
		t.Fatalf("LeadInput.Kind() = %d, 期望 InputCoin(%d)", lead.Kind(), InputCoin)
	}
}

// TestInputCanonicalContainsUnlockScript 校验输入规范编码包含解锁脚本，
// 且定制验证签名字节（置于 UnlockScript）作为普通输入字节参与编码。
func TestInputCanonicalContainsUnlockScript(t *testing.T) {
	sig := []byte("custom-signature-bytes")
	in := Inputs{
		Lead: LeadInput{Ref: sampleRef(0x01), UnlockScript: sig},
	}
	b, err := in.canonicalList()
	if err != nil {
		t.Fatalf("canonicalList: %v", err)
	}
	if !bytes.Contains(b, sig) {
		t.Fatalf("规范编码未包含 UnlockScript 字节")
	}
}

// TestInputTxIDPartLength 校验 TxIDPart 必须 >=16 字节。
func TestInputTxIDPartLength(t *testing.T) {
	short := Inputs{
		Lead: LeadInput{
			Ref:          OutPoint{Year: 1, TxIDPart: bytes.Repeat([]byte{0x01}, 15), OutIndex: 0},
			UnlockScript: nil,
		},
	}
	if _, err := short.canonicalList(); err == nil {
		t.Fatal("TxIDPart 长度 15 应被拒绝")
	}
	ok := Inputs{Lead: LeadInput{Ref: sampleRef(0x01)}}
	if _, err := ok.canonicalList(); err != nil {
		t.Fatalf("TxIDPart 长度 16 应被接受: %v", err)
	}
}

// TestRestInputProofRejected 校验 Proof 输入类型被结构验证拒绝。
func TestRestInputProofRejected(t *testing.T) {
	in := Inputs{
		Lead: LeadInput{Ref: sampleRef(0x01)},
		Rest: []RestInput{{Kind: InputProof, Ref: sampleRef(0x02)}},
	}
	if err := in.Validate(); err == nil {
		t.Fatal("Proof 输入类型应被拒绝")
	}

	// Coin / Credit 输入类型应被接受。
	okCoin := Inputs{Lead: LeadInput{Ref: sampleRef(0x01)}, Rest: []RestInput{{Kind: InputCoin, Ref: sampleRef(0x02)}}}
	if err := okCoin.Validate(); err != nil {
		t.Fatalf("Coin 输入应被接受: %v", err)
	}
	okCredit := Inputs{Lead: LeadInput{Ref: sampleRef(0x01)}, Rest: []RestInput{{Kind: InputCredit, Ref: sampleRef(0x02)}}}
	if err := okCredit.Validate(); err != nil {
		t.Fatalf("Credit 输入应被接受: %v", err)
	}
}

// TestUnlockScriptLimit 校验解锁脚本超过 MaxUnlockScript 被拒绝。
func TestUnlockScriptLimit(t *testing.T) {
	in := Inputs{
		Lead: LeadInput{Ref: sampleRef(0x01), UnlockScript: bytes.Repeat([]byte{0x00}, types.MaxUnlockScript+1)},
	}
	if _, err := in.canonicalList(); err == nil {
		t.Fatal("超长 UnlockScript 应被拒绝")
	}
}

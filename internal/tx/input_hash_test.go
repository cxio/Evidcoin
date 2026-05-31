package tx

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
)

// leadPK 是测试用的首领输入源公钥哈希（32 字节）。
var leadPK = bytes.Repeat([]byte{0xCD}, 32)

// TestHashInputsVector 校验 HashInputs = BLAKE3-256(ListHash || LeadPKHash)。
func TestHashInputsVector(t *testing.T) {
	in := Inputs{
		Lead: LeadInput{Ref: sampleRef(0x01), UnlockScript: []byte{0xAA}},
		Rest: []RestInput{{Kind: InputCoin, Ref: sampleRef(0x02), UnlockScript: []byte{0xBB}}},
	}

	lh, err := in.ListHash()
	if err != nil {
		t.Fatalf("ListHash: %v", err)
	}
	want := crypto.HashInputRoot(lh.Bytes(), leadPK)

	got, err := in.HashInputs(leadPK)
	if err != nil {
		t.Fatalf("HashInputs: %v", err)
	}
	if got != want {
		t.Fatalf("HashInputs 与 BLAKE3-256(ListHash||LeadPKHash) 不一致")
	}
}

// TestHashInputsUnlockScriptSensitivity 校验修改任一输入 UnlockScript 改变 ListHash 与 HashInputs。
func TestHashInputsUnlockScriptSensitivity(t *testing.T) {
	build := func(unlock []byte) Inputs {
		return Inputs{
			Lead: LeadInput{Ref: sampleRef(0x01), UnlockScript: []byte{0xAA}},
			Rest: []RestInput{{Kind: InputCoin, Ref: sampleRef(0x02), UnlockScript: unlock}},
		}
	}
	a, _ := build([]byte{0x01}).HashInputs(leadPK)
	b, _ := build([]byte{0x02}).HashInputs(leadPK)
	if a == b {
		t.Fatal("修改 UnlockScript 未改变 HashInputs")
	}
}

// TestHashInputsOrderSensitivity 校验输入列表顺序变化导致 HashInputs 变化。
func TestHashInputsOrderSensitivity(t *testing.T) {
	r1 := RestInput{Kind: InputCoin, Ref: sampleRef(0x02), UnlockScript: []byte{0x01}}
	r2 := RestInput{Kind: InputCoin, Ref: sampleRef(0x03), UnlockScript: []byte{0x02}}
	in1 := Inputs{Lead: LeadInput{Ref: sampleRef(0x01)}, Rest: []RestInput{r1, r2}}
	in2 := Inputs{Lead: LeadInput{Ref: sampleRef(0x01)}, Rest: []RestInput{r2, r1}}
	h1, _ := in1.HashInputs(leadPK)
	h2, _ := in2.HashInputs(leadPK)
	if h1 == h2 {
		t.Fatal("输入顺序变化未改变 HashInputs")
	}
}

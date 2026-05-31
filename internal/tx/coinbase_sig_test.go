package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

func testTxID(b byte) types.TxID {
	raw := make([]byte, 48)
	for i := range raw {
		raw[i] = b
	}
	return types.MustTxID(raw)
}

// TestCoinbaseSignatureMessageLayout 校验 Coinbase 签名消息布局（DEC-0102 §5）。
func TestCoinbaseSignatureMessageLayout(t *testing.T) {
	cs := testChainScope(nil)
	txid := testTxID(0x42)
	got := CoinbaseSignatureMessage(cs, txid)

	want := crypto.SignatureMessageTag()
	want = cs.appendCanonical(want)
	want = append(want, 0x00) // chk_type = 0
	want = append(want, txid.Bytes()...)

	if string(got) != string(want) {
		t.Fatalf("coinbase sig message mismatch\n got=%x\nwant=%x", got, want)
	}
	// 末尾 48 字节必须等于完整 CoinbaseTxID。
	tail := got[len(got)-48:]
	if string(tail) != string(txid.Bytes()) {
		t.Fatal("coinbase sig message must end with full 48-byte TxID")
	}
}

// TestCoinbaseSignatureMessageNoAuthFlag 校验 Coinbase 不走授权种类路径：
// 改变 auth_flag 类比值不影响 Coinbase 签名消息（其根本不含 auth_flag 字段）。
func TestCoinbaseSignatureMessageChkTypeZero(t *testing.T) {
	cs := testChainScope(nil)
	got := CoinbaseSignatureMessage(cs, testTxID(0x01))
	prefix := crypto.SignatureMessageTag()
	prefix = cs.appendCanonical(prefix)
	// chk_type 字节紧随 ChainScope。
	if got[len(prefix)] != 0x00 {
		t.Fatalf("coinbase chk_type byte = %x, want 0x00", got[len(prefix)])
	}
}

// TestCheckRootSignatureMessageReproducible 校验 CheckRoot 签名消息确定可复现，
// 且独立于 Coinbase 签名消息（不带 signature.message 域标签）。
func TestCheckRootSignatureMessageReproducible(t *testing.T) {
	raw := make([]byte, 48)
	for i := range raw {
		raw[i] = byte(i)
	}
	cr, err := types.NewCheckRoot(raw)
	if err != nil {
		t.Fatal(err)
	}
	a := CheckRootSignatureMessage(cr)
	b := CheckRootSignatureMessage(cr)
	if string(a) != string(b) {
		t.Fatal("CheckRoot sig message must be reproducible")
	}
	if string(a) != string(cr.Bytes()) {
		t.Fatal("CheckRoot sig message must equal the 48-byte CheckRoot")
	}
	// 必须独立于 Coinbase 域：不以 signature.message 域标签开头。
	tag := crypto.SignatureMessageTag()
	if len(a) >= len(tag) && string(a[:len(tag)]) == string(tag) {
		t.Fatal("CheckRoot sig message must not carry signature.message domain tag")
	}
}

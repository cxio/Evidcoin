package crypto

import (
	"bytes"
	"testing"
)

func TestAddressHashSingleLength(t *testing.T) {
	h := AddressHashSingle([]byte("pubkey-material"))
	if len(h.Bytes()) != 32 {
		t.Fatalf("single address hash len = %d, want 32", len(h.Bytes()))
	}
}

func TestAddressHashSingleDistinct(t *testing.T) {
	a := AddressHashSingle([]byte("pubkey-a"))
	b := AddressHashSingle([]byte("pubkey-b"))
	if bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("different pubkeys produced identical address hash")
	}
}

func TestAddressHashMultiOrderIndependent(t *testing.T) {
	k1 := []byte("key-1")
	k2 := []byte("key-2")
	k3 := []byte("key-3")
	h1, err := AddressHashMulti(2, 3, [][]byte{k1, k2, k3})
	if err != nil {
		t.Fatal(err)
	}
	h2, err := AddressHashMulti(2, 3, [][]byte{k3, k1, k2})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(h1.Bytes(), h2.Bytes()) {
		t.Fatal("multisig hash must be independent of input order")
	}
	if len(h1.Bytes()) != 32 {
		t.Fatalf("multi address hash len = %d, want 32", len(h1.Bytes()))
	}
}

func TestAddressHashMultiRatioInvalid(t *testing.T) {
	keys := [][]byte{[]byte("a"), []byte("b")}
	tests := []struct {
		m, n uint8
		keys [][]byte
	}{
		{0, 2, keys},
		{2, 0, nil},
		{3, 2, [][]byte{[]byte("a"), []byte("b")}},
		{2, 3, keys}, // n != len(keys)
	}
	for _, tt := range tests {
		if _, err := AddressHashMulti(tt.m, tt.n, tt.keys); err == nil {
			t.Errorf("AddressHashMulti(%d,%d,...) accepted, want error", tt.m, tt.n)
		}
	}
}

func TestAddressHashMultiDuplicate(t *testing.T) {
	dup := []byte("same-key")
	if _, err := AddressHashMulti(2, 3, [][]byte{dup, dup, []byte("other")}); err == nil {
		t.Fatal("duplicate public key must be rejected")
	}
}

func TestEncodeDecodeAddressRoundTrip(t *testing.T) {
	h := AddressHashSingle([]byte("round-trip-key"))
	for _, net := range []Network{Mainnet, Testnet, Devnet} {
		addr, err := EncodeAddress(net, h)
		if err != nil {
			t.Fatalf("EncodeAddress(%s): %v", net, err)
		}
		if addr[:2] != string(net) {
			t.Errorf("address %q missing prefix %s", addr, net)
		}
		gotNet, gotHash, err := DecodeAddress(addr)
		if err != nil {
			t.Fatalf("DecodeAddress(%q): %v", addr, err)
		}
		if gotNet != net {
			t.Errorf("decoded net = %s, want %s", gotNet, net)
		}
		if !bytes.Equal(gotHash.Bytes(), h.Bytes()) {
			t.Errorf("decoded hash mismatch for %s", net)
		}
	}
}

func TestDecodeAddressRejects(t *testing.T) {
	h := AddressHashSingle([]byte("key"))
	addr, _ := EncodeAddress(Mainnet, h)

	// 前缀错误
	if _, _, err := DecodeAddress("Zx" + addr[2:]); err == nil {
		t.Error("expected unknown network error")
	}
	// 校验和损坏：通过篡改载荷改变末尾字符附近内容
	bad := addr[:len(addr)-1] + flipBase58Char(addr[len(addr)-1])
	if _, _, err := DecodeAddress(bad); err == nil {
		t.Error("expected checksum/length error for corrupted address")
	}
	// 非法 base58（包含 '0'，不在 Bitcoin 字母表内）
	if _, _, err := DecodeAddress("Cx0OIl"); err == nil {
		t.Error("expected malformed error for invalid base58")
	}
	// 过短
	if _, _, err := DecodeAddress("C"); err == nil {
		t.Error("expected malformed error for too-short address")
	}
}

func flipBase58Char(c byte) string {
	if c == '2' {
		return "3"
	}
	return "2"
}

func TestSingleAndMultiIndistinguishable(t *testing.T) {
	single := AddressHashSingle([]byte("k"))
	multi, err := AddressHashMulti(1, 2, [][]byte{[]byte("a"), []byte("b")})
	if err != nil {
		t.Fatal(err)
	}
	// 二者都是纯 32 字节哈希，不含结构标记。
	if len(single.Bytes()) != len(multi.Bytes()) {
		t.Fatal("single and multisig address hashes differ in length")
	}
}

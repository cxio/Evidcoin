package crypto_test

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// TestSHA3_384Deterministic 测试 SHA3-384 输出确定性。
func TestSHA3_384Deterministic(t *testing.T) {
	data := []byte("hello evidcoin")
	h1 := crypto.SHA3_384(data)
	h2 := crypto.SHA3_384(data)
	if h1 != h2 {
		t.Error("SHA3_384 is not deterministic")
	}
}

// TestSHA3_384Different 测试不同输入产生不同哈希。
func TestSHA3_384Different(t *testing.T) {
	h1 := crypto.SHA3_384([]byte("a"))
	h2 := crypto.SHA3_384([]byte("b"))
	if h1 == h2 {
		t.Error("SHA3_384 collision on different inputs")
	}
}

// TestBLAKE3_256Deterministic 测试 BLAKE3-256 确定性。
func TestBLAKE3_256Deterministic(t *testing.T) {
	data := []byte("test data")
	h1 := crypto.BLAKE3_256(data)
	h2 := crypto.BLAKE3_256(data)
	if h1 != h2 {
		t.Error("BLAKE3_256 is not deterministic")
	}
}

// TestPubKeyHashDeterministic 测试公钥哈希确定性。
func TestPubKeyHashDeterministic(t *testing.T) {
	pub := []byte("fake-public-key-bytes")
	h1 := crypto.PubKeyHash(pub)
	h2 := crypto.PubKeyHash(pub)
	if h1 != h2 {
		t.Error("PubKeyHash is not deterministic")
	}
}

// TestPubKeyHashLen 测试公钥哈希长度为 32 字节。
func TestPubKeyHashLen(t *testing.T) {
	pub := []byte("some-public-key")
	h := crypto.PubKeyHash(pub)
	if len(h) != types.PubKeyHashLen {
		t.Errorf("PubKeyHash length = %d, want %d", len(h), types.PubKeyHashLen)
	}
}

// TestSignVerifyRoundtrip 测试签名与验证往返一致性。
func TestSignVerifyRoundtrip(t *testing.T) {
	pub, priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	data := []byte("sign this message")
	sig := crypto.Sign(priv, data)
	if !crypto.Verify(pub, data, sig) {
		t.Error("Verify() returned false for valid signature")
	}
}

// TestSignVerifyTampered 测试篡改数据后验证失败。
func TestSignVerifyTampered(t *testing.T) {
	pub, priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	data := []byte("original message")
	sig := crypto.Sign(priv, data)
	tampered := []byte("tampered message")
	if crypto.Verify(pub, tampered, sig) {
		t.Error("Verify() returned true for tampered data")
	}
}

// TestPublicKeyBytesRoundtrip 测试公钥序列化往返一致性。
func TestPublicKeyBytesRoundtrip(t *testing.T) {
	pub, _, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	b := crypto.PublicKeyBytes(pub)
	pub2, err := crypto.PublicKeyFromBytes(b)
	if err != nil {
		t.Fatalf("PublicKeyFromBytes() error = %v", err)
	}
	b2 := crypto.PublicKeyBytes(pub2)
	if !bytes.Equal(b, b2) {
		t.Error("public key bytes roundtrip mismatch")
	}
}

// TestSHA3_512Deterministic 测试 SHA3-512 确定性。
func TestSHA3_512Deterministic(t *testing.T) {
	data := []byte("test sha3-512")
	h1 := crypto.SHA3_512(data)
	h2 := crypto.SHA3_512(data)
	if h1 != h2 {
		t.Error("SHA3_512 is not deterministic")
	}
}

// TestBLAKE3_512Deterministic 测试 BLAKE3-512 确定性。
func TestBLAKE3_512Deterministic(t *testing.T) {
	data := []byte("test blake3-512")
	h1 := crypto.BLAKE3_512(data)
	h2 := crypto.BLAKE3_512(data)
	if h1 != h2 {
		t.Error("BLAKE3_512 is not deterministic")
	}
}

// TestPrivateKeyBytesRoundtrip 测试私钥序列化往返一致性。
func TestPrivateKeyBytesRoundtrip(t *testing.T) {
	_, priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	b := crypto.PrivateKeyBytes(priv)
	priv2, err := crypto.PrivateKeyFromBytes(b)
	if err != nil {
		t.Fatalf("PrivateKeyFromBytes() error = %v", err)
	}
	b2 := crypto.PrivateKeyBytes(priv2)
	if !bytes.Equal(b, b2) {
		t.Error("private key bytes roundtrip mismatch")
	}
}

// TestMultiSigPubKeyHashDifferentM 测试不同 m 值产生不同哈希。
func TestMultiSigPubKeyHashDifferentM(t *testing.T) {
	pkhs := []types.PubKeyHash{{1}, {2}, {3}}
	h1 := crypto.MultiSigPubKeyHash(2, pkhs)
	h2 := crypto.MultiSigPubKeyHash(3, pkhs)
	if h1 == h2 {
		t.Error("MultiSigPubKeyHash should differ for different m values")
	}
}

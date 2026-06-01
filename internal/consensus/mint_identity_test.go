package consensus

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
)

// TestVerifyMintIdentityWithPKHash 断言含 MintPKHash 路径：
// MintPubKey 的公钥哈希必须等于 MintPKHash。
func TestVerifyMintIdentityWithPKHash(t *testing.T) {
	pubKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	pkHash := crypto.AddressHashSingle(pubKey)

	// 匹配：通过。
	if err := VerifyMintIdentityWithPKHash(pubKey, pkHash.Bytes()); err != nil {
		t.Fatalf("expected match, got %v", err)
	}

	// 不匹配：拒绝。
	wrong := pkHash.Bytes()
	wrong[0] ^= 0xFF
	if err := VerifyMintIdentityWithPKHash(pubKey, wrong); err != ErrMintIdentityMismatch {
		t.Fatalf("expected ErrMintIdentityMismatch, got %v", err)
	}
}

// TestVerifyMintIdentityWithLeadInput 断言不含 MintPKHash 路径：
// MintPubKey 公钥哈希作 LeadPKHash，须满足 BLAKE3-256(ListHash || LeadPKHash) == 输入根。
func TestVerifyMintIdentityWithLeadInput(t *testing.T) {
	pubKey := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	leadPKHash := crypto.AddressHashSingle(pubKey)

	lh := crypto.HashInputList([]byte("input-list-payload"))
	wantRoot := crypto.HashInputRoot(lh.Bytes(), leadPKHash.Bytes())

	// 匹配：通过。
	if err := VerifyMintIdentityWithLeadInput(pubKey, lh, wantRoot); err != nil {
		t.Fatalf("expected match, got %v", err)
	}

	// 输入根不匹配：拒绝。
	bad := wantRoot
	bad[0] ^= 0xFF
	if err := VerifyMintIdentityWithLeadInput(pubKey, lh, bad); err != ErrInputRootMismatch {
		t.Fatalf("expected ErrInputRootMismatch, got %v", err)
	}
}

// TestVerifyCoinbaseMintEligibility 断言 Coinbase 只要显式设置非零 MintPKHash 即可参与；
// 全零 MintPKHash 视为未设置，拒绝。
func TestVerifyCoinbaseMintEligibility(t *testing.T) {
	var set [32]byte
	copy(set[:], bytes.Repeat([]byte{0x09}, 32))
	if err := VerifyCoinbaseMintEligibility(set); err != nil {
		t.Fatalf("expected eligible, got %v", err)
	}

	var zero [32]byte
	if err := VerifyCoinbaseMintEligibility(zero); err != ErrCoinbaseMintPKHashMissing {
		t.Fatalf("expected ErrCoinbaseMintPKHashMissing, got %v", err)
	}
}

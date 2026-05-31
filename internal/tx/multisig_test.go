package tx

import (
	"crypto/hmac"
	"crypto/sha256"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 多签/单签验证测试桩：确定性 HMAC 签名，验签仅依赖公钥材料，便于多签独立验证。
// 不绑定具体 ML-DSA 库（DEC-0104 已冻结 circl 为生产配置，接入在后续阶段）。

type tPub struct{ raw []byte }

func (p tPub) Algorithm() crypto.AlgorithmID { return crypto.AlgMLDSA65 }
func (p tPub) Bytes() []byte                 { return append([]byte(nil), p.raw...) }

type tSig struct{ raw []byte }

func (s tSig) Algorithm() crypto.AlgorithmID { return crypto.AlgMLDSA65 }
func (s tSig) Bytes() []byte                 { return append([]byte(nil), s.raw...) }

// tKeypair 由种子派生确定性公钥与签名函数。签名为 HMAC(key=pub.raw, message)，
// 验签端可仅凭公钥重算，故每个签名者相互独立。
func tKeypair(seed string) (tPub, func(msg []byte) tSig) {
	material := sha256.Sum256([]byte("pub:" + seed))
	pub := tPub{raw: material[:]}
	sign := func(msg []byte) tSig {
		mac := hmac.New(sha256.New, pub.raw)
		mac.Write(msg)
		return tSig{raw: mac.Sum(nil)}
	}
	return pub, sign
}

type tVerifier struct{}

func (tVerifier) Verify(pub crypto.PublicKey, message []byte, sig crypto.Signature) (bool, error) {
	if pub.Algorithm() != crypto.AlgMLDSA65 || sig.Algorithm() != crypto.AlgMLDSA65 {
		return false, crypto.ErrAlgorithmMismatch
	}
	mac := hmac.New(sha256.New, pub.Bytes())
	mac.Write(message)
	return hmac.Equal(mac.Sum(nil), sig.Bytes()), nil
}

func TestVerifySingleValid(t *testing.T) {
	pub, sign := tKeypair("alice")
	receiver := crypto.AddressHashSingle(pub.Bytes())
	msg := []byte("signature message bytes")
	if err := VerifySingle(tVerifier{}, receiver, pub, sign(msg), msg); err != nil {
		t.Fatalf("valid single sig should verify, got %v", err)
	}
}

func TestVerifySingleReceiverMismatch(t *testing.T) {
	pub, sign := tKeypair("alice")
	other, _ := tKeypair("mallory")
	receiver := crypto.AddressHashSingle(other.Bytes())
	msg := []byte("msg")
	if err := VerifySingle(tVerifier{}, receiver, pub, sign(msg), msg); err != ErrReceiverMismatch {
		t.Fatalf("expected ErrReceiverMismatch, got %v", err)
	}
}

func TestVerifySingleBadSignature(t *testing.T) {
	pub, sign := tKeypair("alice")
	receiver := crypto.AddressHashSingle(pub.Bytes())
	msg := []byte("msg")
	sig := sign(msg)
	sig.raw[0] ^= 0xFF // 篡改签名
	if err := VerifySingle(tVerifier{}, receiver, pub, sig, msg); err != ErrSignatureInvalid {
		t.Fatalf("expected ErrSignatureInvalid, got %v", err)
	}
}

func TestVerifySingleAlgorithmMismatch(t *testing.T) {
	pub, _ := tKeypair("alice")
	receiver := crypto.AddressHashSingle(pub.Bytes())
	msg := []byte("msg")
	bad := mismatchSig{}
	if err := VerifySingle(tVerifier{}, receiver, pub, bad, msg); err != crypto.ErrAlgorithmMismatch {
		t.Fatalf("expected ErrAlgorithmMismatch, got %v", err)
	}
}

type mismatchSig struct{}

func (mismatchSig) Algorithm() crypto.AlgorithmID { return crypto.AlgUnknown }
func (mismatchSig) Bytes() []byte                 { return nil }

// multisigFixture 构造一个 m-of-n 配置：前 m 个种子为签名者，其余为补全方。
func multisigFixture(t *testing.T, m, n int, msg []byte) (types.AddressHash, MultisigWitness) {
	t.Helper()
	seeds := []string{"k0", "k1", "k2", "k3", "k4"}
	if n > len(seeds) {
		t.Fatalf("fixture supports at most %d keys", len(seeds))
	}
	allPubs := make([][]byte, n)
	var sigs []crypto.Signature
	var signedPubs []crypto.PublicKey
	var completion [][]byte
	for i := 0; i < n; i++ {
		pub, sign := tKeypair(seeds[i])
		allPubs[i] = pub.Bytes()
		if i < m {
			sigs = append(sigs, sign(msg))
			signedPubs = append(signedPubs, pub)
		} else {
			bh := crypto.PubKeyBaseHash(pub.Bytes())
			completion = append(completion, bh[:])
		}
	}
	// 补全集须字典序升序。
	sortBaseHashes(completion)
	receiver, err := crypto.AddressHashMulti(uint8(m), uint8(n), allPubs)
	if err != nil {
		t.Fatalf("derive receiver: %v", err)
	}
	return receiver, MultisigWitness{Signatures: sigs, PublicKeys: signedPubs, Completion: completion}
}

func sortBaseHashes(s [][]byte) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && bytesLess(s[j], s[j-1]); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func bytesLess(a, b []byte) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func TestVerifyMultisigValid(t *testing.T) {
	msg := []byte("multisig message")
	receiver, w := multisigFixture(t, 2, 3, msg)
	if err := VerifyMultisig(tVerifier{}, receiver, w, msg); err != nil {
		t.Fatalf("valid 2-of-3 should verify, got %v", err)
	}
}

// TestVerifyMultisigPairingOrderIndependent 验证排序规则 1 与规则 3 的区分：
// 交换签名/公钥配对顺序仍验证通过（派生哈希内部排序，验签按配对索引）。
func TestVerifyMultisigPairingOrderIndependent(t *testing.T) {
	msg := []byte("order test")
	receiver, w := multisigFixture(t, 2, 3, msg)
	w.Signatures[0], w.Signatures[1] = w.Signatures[1], w.Signatures[0]
	w.PublicKeys[0], w.PublicKeys[1] = w.PublicKeys[1], w.PublicKeys[0]
	if err := VerifyMultisig(tVerifier{}, receiver, w, msg); err != nil {
		t.Fatalf("swapped pairing should still verify, got %v", err)
	}
}

func TestVerifyMultisigConfigInvalid(t *testing.T) {
	msg := []byte("msg")
	// m=0：仅补全集，无签名。
	bh := crypto.PubKeyBaseHash([]byte("x"))
	w := MultisigWitness{Completion: [][]byte{bh[:]}}
	var zero types.AddressHash
	if err := VerifyMultisig(tVerifier{}, zero, w, msg); err != ErrMultisigConfig {
		t.Fatalf("expected ErrMultisigConfig, got %v", err)
	}
}

func TestVerifyMultisigSetMismatch(t *testing.T) {
	msg := []byte("msg")
	receiver, w := multisigFixture(t, 2, 3, msg)
	w.PublicKeys = w.PublicKeys[:1] // 公钥集与签名集数量不符
	if err := VerifyMultisig(tVerifier{}, receiver, w, msg); err != ErrMultisigSetMismatch {
		t.Fatalf("expected ErrMultisigSetMismatch, got %v", err)
	}
}

func TestVerifyMultisigCompletionNotSorted(t *testing.T) {
	msg := []byte("msg")
	receiver, w := multisigFixture(t, 1, 3, msg) // 补全集 2 项
	if len(w.Completion) != 2 {
		t.Fatalf("fixture should have 2 completion hashes, got %d", len(w.Completion))
	}
	w.Completion[0], w.Completion[1] = w.Completion[1], w.Completion[0] // 打乱为降序
	if err := VerifyMultisig(tVerifier{}, receiver, w, msg); err != ErrCompletionNotSorted {
		t.Fatalf("expected ErrCompletionNotSorted, got %v", err)
	}
}

func TestVerifyMultisigReceiverMismatch(t *testing.T) {
	msg := []byte("msg")
	_, w := multisigFixture(t, 2, 3, msg)
	var wrong types.AddressHash
	if err := VerifyMultisig(tVerifier{}, wrong, w, msg); err != ErrReceiverMismatch {
		t.Fatalf("expected ErrReceiverMismatch, got %v", err)
	}
}

func TestVerifyMultisigBadSignature(t *testing.T) {
	msg := []byte("msg")
	receiver, w := multisigFixture(t, 2, 3, msg)
	bad := w.Signatures[1].(tSig)
	bad.raw[0] ^= 0xFF
	w.Signatures[1] = bad
	if err := VerifyMultisig(tVerifier{}, receiver, w, msg); err != ErrSignatureInvalid {
		t.Fatalf("expected ErrSignatureInvalid, got %v", err)
	}
}

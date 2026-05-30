package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"testing"
)

// testSigner 是一个确定性的基于 HMAC 的测试桩，用于验证 Signer / Verifier
// 接口契约，而不直接绑定具体 ML-DSA 库
// （DEC-0104 已冻结 circl 为生产配置；该绑定在后续阶段实现）。

const algTest AlgorithmID = 200

type testPublicKey struct{ raw []byte }

func (p testPublicKey) Algorithm() AlgorithmID { return algTest }
func (p testPublicKey) Bytes() []byte {
	out := make([]byte, len(p.raw))
	copy(out, p.raw)
	return out
}

type testSignature struct{ raw []byte }

func (s testSignature) Algorithm() AlgorithmID { return algTest }
func (s testSignature) Bytes() []byte {
	out := make([]byte, len(s.raw))
	copy(out, s.raw)
	return out
}

type testSigner struct {
	secret []byte
	pub    testPublicKey
}

func newTestSigner(secret string) testSigner {
	pubMaterial := sha256.Sum256([]byte("pub:" + secret))
	return testSigner{
		secret: []byte(secret),
		pub:    testPublicKey{raw: pubMaterial[:]},
	}
}

func (s testSigner) PublicKey() PublicKey { return s.pub }

func (s testSigner) Sign(message []byte) (Signature, error) {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(s.pub.raw)
	mac.Write(message)
	return testSignature{raw: mac.Sum(nil)}, nil
}

type testVerifier struct{ secret []byte }

func (v testVerifier) Verify(pub PublicKey, message []byte, sig Signature) (bool, error) {
	if pub.Algorithm() != algTest || sig.Algorithm() != algTest {
		return false, ErrAlgorithmMismatch
	}
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(pub.Bytes())
	mac.Write(message)
	want := mac.Sum(nil)
	return hmac.Equal(want, sig.Bytes()), nil
}

func TestSignerVerifyValid(t *testing.T) {
	signer := newTestSigner("secret-1")
	verifier := testVerifier{secret: []byte("secret-1")}
	msg := []byte("signature message bytes")

	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := verifier.Verify(signer.PublicKey(), msg, sig)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("valid signature failed to verify")
	}
}

func TestSignerVerifyTamperedMessage(t *testing.T) {
	signer := newTestSigner("secret-2")
	verifier := testVerifier{secret: []byte("secret-2")}
	sig, _ := signer.Sign([]byte("original"))
	ok, err := verifier.Verify(signer.PublicKey(), []byte("tampered"), sig)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("tampered message must not verify")
	}
}

func TestSignerVerifyAlgorithmMismatch(t *testing.T) {
	signer := newTestSigner("secret-3")
	verifier := testVerifier{secret: []byte("secret-3")}
	msg := []byte("msg")
	sig, _ := signer.Sign(msg)

	mismatch := mismatchSignature{testSignature(sig.(testSignature))}
	if _, err := verifier.Verify(signer.PublicKey(), msg, mismatch); err != ErrAlgorithmMismatch {
		t.Fatalf("expected ErrAlgorithmMismatch, got %v", err)
	}
}

type mismatchSignature struct{ testSignature }

func (mismatchSignature) Algorithm() AlgorithmID { return AlgMLDSA65 }

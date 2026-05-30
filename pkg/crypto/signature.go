package crypto

import "errors"

// 签名抽象（DEC-0104 固定 ML-DSA-65 / cloudflare-circl profile）。上层只依赖这些
// 接口，不直接绑定具体后量子签名库，便于未来单独评估迁移（A-2 观察项）。

// AlgorithmID identifies a signature algorithm profile.
type AlgorithmID uint8

const (
	// AlgUnknown is the zero-value, invalid algorithm.
	AlgUnknown AlgorithmID = 0
	// AlgMLDSA65 is the ML-DSA-65 profile (cloudflare/circl, DEC-0104).
	AlgMLDSA65 AlgorithmID = 1
)

// ErrAlgorithmMismatch is returned when a verifier and signature disagree on
// the algorithm profile.
var ErrAlgorithmMismatch = errors.New("crypto: signature algorithm mismatch")

// PublicKey is an algorithm-tagged public key whose canonical byte encoding
// feeds public-key hashing and address derivation.
type PublicKey interface {
	// Algorithm returns the signature algorithm profile.
	Algorithm() AlgorithmID
	// Bytes returns the canonical public key encoding (a fresh copy).
	Bytes() []byte
}

// Signature is an algorithm-tagged signature value.
type Signature interface {
	// Algorithm returns the signature algorithm profile.
	Algorithm() AlgorithmID
	// Bytes returns the canonical signature encoding (a fresh copy).
	Bytes() []byte
}

// Signer produces signatures over a signature message byte sequence
// (the message profile is defined in 第 08 章 / DEC-0102).
type Signer interface {
	// PublicKey returns the signer's public key.
	PublicKey() PublicKey
	// Sign signs the given message bytes.
	Sign(message []byte) (Signature, error)
}

// Verifier verifies signatures against a public key.
type Verifier interface {
	// Verify reports whether sig is a valid signature of message under pub.
	// It returns ErrAlgorithmMismatch when the algorithm profiles disagree.
	Verify(pub PublicKey, message []byte, sig Signature) (bool, error)
}

// Package types provides the foundation value types, fixed-length hashes,
// identifiers, protocol constants and canonical byte encoding shared by the
// protocol layer and the script layer. It has no internal dependencies.
package types

// 固定长度哈希类型。这些类型仅承载字节数组，算法语义（SHA3-384、BLAKE3-256 等）
// 由 pkg/crypto 按用途绑定，参见第 02 章域标签全集。

// Hash32 is a fixed 32-byte hash value (e.g. BLAKE3-256 / SHA3-256 output).
type Hash32 [32]byte

// Hash48 is a fixed 48-byte hash value (e.g. SHA3-384 output).
type Hash48 [48]byte

// Hash64 is a fixed 64-byte hash value (e.g. SHA3-512 output).
type Hash64 [64]byte

// NewHash32 constructs a Hash32 from b, which must be exactly 32 bytes long.
// The returned value owns a copy of the input so the caller may reuse b safely.
func NewHash32(b []byte) (Hash32, error) {
	var h Hash32
	if len(b) != len(h) {
		return h, newLengthError("Hash32", len(h), len(b))
	}
	copy(h[:], b)
	return h, nil
}

// NewHash48 constructs a Hash48 from b, which must be exactly 48 bytes long.
func NewHash48(b []byte) (Hash48, error) {
	var h Hash48
	if len(b) != len(h) {
		return h, newLengthError("Hash48", len(h), len(b))
	}
	copy(h[:], b)
	return h, nil
}

// NewHash64 constructs a Hash64 from b, which must be exactly 64 bytes long.
func NewHash64(b []byte) (Hash64, error) {
	var h Hash64
	if len(b) != len(h) {
		return h, newLengthError("Hash64", len(h), len(b))
	}
	copy(h[:], b)
	return h, nil
}

// Bytes returns a fresh copy of the hash bytes. The copy prevents aliasing of
// the internal array, so mutating the result never affects the original value.
func (h Hash32) Bytes() []byte {
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

// Bytes returns a fresh copy of the hash bytes.
func (h Hash48) Bytes() []byte {
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

// Bytes returns a fresh copy of the hash bytes.
func (h Hash64) Bytes() []byte {
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

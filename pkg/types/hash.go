// Package types 提供基础值类型、定长哈希、标识符、协议常量与规范字节编码，
// 供协议层与脚本层共享。该包不依赖任何内部包。
package types

// 固定长度哈希类型。这些类型仅承载字节数组，算法语义（SHA3-384、BLAKE3-256 等）
// 由 pkg/crypto 按用途绑定，参见第 02 章域标签全集。

// Hash32 是固定 32 字节的哈希值（例如 BLAKE3-256 / SHA3-256 输出）。
type Hash32 [32]byte

// Hash48 是固定 48 字节的哈希值（例如 SHA3-384 输出）。
type Hash48 [48]byte

// Hash64 是固定 64 字节的哈希值（例如 SHA3-512 输出）。
type Hash64 [64]byte

// NewHash32 从 b 构造 Hash32，b 的长度必须恰好为 32 字节。
// 返回值会拷贝输入数据，因此调用方可安全复用 b。
func NewHash32(b []byte) (Hash32, error) {
	var h Hash32
	if len(b) != len(h) {
		return h, newLengthError("Hash32", len(h), len(b))
	}
	copy(h[:], b)
	return h, nil
}

// NewHash48 从 b 构造 Hash48，b 的长度必须恰好为 48 字节。
func NewHash48(b []byte) (Hash48, error) {
	var h Hash48
	if len(b) != len(h) {
		return h, newLengthError("Hash48", len(h), len(b))
	}
	copy(h[:], b)
	return h, nil
}

// NewHash64 从 b 构造 Hash64，b 的长度必须恰好为 64 字节。
func NewHash64(b []byte) (Hash64, error) {
	var h Hash64
	if len(b) != len(h) {
		return h, newLengthError("Hash64", len(h), len(b))
	}
	copy(h[:], b)
	return h, nil
}

// Bytes 返回哈希字节的新副本。通过拷贝避免与内部数组别名，
// 因此修改返回结果不会影响原值。
func (h Hash32) Bytes() []byte {
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

// Bytes 返回哈希字节的新副本。
func (h Hash48) Bytes() []byte {
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

// Bytes 返回哈希字节的新副本。
func (h Hash64) Bytes() []byte {
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

// Package crypto 封装 Evidcoin 使用的密码学原语。
package crypto

import (
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
	"lukechampine.com/blake3"

	"github.com/cxio/evidcoin/pkg/types"
)

// SHA3_384 计算数据的 SHA3-384 哈希，返回 Hash384。
// 用途：区块 ID、交易 ID（TxID）、CheckRoot、公钥哈希路径等。
func SHA3_384(data []byte) types.Hash384 {
	return sha3.Sum384(data)
}

// SHA3_512 计算数据的 SHA3-512 哈希，返回 64 字节数组。
// 用途：暂保留备用（铸凭哈希使用 BLAKE3-512）。
func SHA3_512(data []byte) [64]byte {
	return sha3.Sum512(data)
}

// SHA3_256 计算数据的 SHA3-256 哈希，返回 Hash256。
// 用途：公钥哈希双重哈希的外层。
func SHA3_256(data []byte) types.Hash256 {
	var h types.Hash256
	d := sha3.Sum256(data)
	copy(h[:], d[:])
	return h
}

// BLAKE3_256 计算数据的 BLAKE3-256 哈希，返回 Hash256。
// 用途：UTXO/UTCO 树内部节点、输出哈希树内部节点。
func BLAKE3_256(data []byte) types.Hash256 {
	var h types.Hash256
	sum := blake3.Sum256(data)
	copy(h[:], sum[:])
	return h
}

// BLAKE3_512 计算数据的 BLAKE3-512 哈希，返回 Hash512。
// 用途：铸凭哈希（MintHash）。
func BLAKE3_512(data []byte) types.Hash512 {
	var h types.Hash512
	sum := blake3.Sum512(data)
	copy(h[:], sum[:])
	return h
}

// BLAKE2b_512 计算数据的 BLAKE2b-512 哈希，返回 64 字节切片。
// 用途：公钥哈希双重哈希的内层。
func BLAKE2b_512(data []byte) []byte {
	h, _ := blake2b.New512(nil) // 无 key，不会失败
	h.Write(data)
	return h.Sum(nil)
}

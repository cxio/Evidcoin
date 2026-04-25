package types

import "encoding/hex"

// Hash384 48 字节哈希，用于区块 ID、交易 ID、CheckRoot、公钥哈希路径等。
// 算法：SHA3-384
type Hash384 [Hash384Len]byte

// Hash512 64 字节哈希，用于铸凭哈希（MintHash）。
// 算法：BLAKE3-512
type Hash512 [Hash512Len]byte

// Hash256 32 字节哈希，用于 UTXO/UTCO 树内部节点、输出哈希树根等。
// 算法：BLAKE3-256
type Hash256 [Hash256Len]byte

// PubKeyHash 32 字节公钥哈希。
// 算法：SHA3-256( BLAKE2b-512( publicKey ) )
type PubKeyHash [PubKeyHashLen]byte

// IsZero 判断哈希是否为全零。
func (h Hash384) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回哈希的十六进制字符串表示。
func (h Hash384) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero 判断哈希是否为全零。
func (h Hash512) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回哈希的十六进制字符串表示。
func (h Hash512) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero 判断哈希是否为全零。
func (h Hash256) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回哈希的十六进制字符串表示。
func (h Hash256) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero 判断哈希是否为全零。
func (h PubKeyHash) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

package crypto

import "github.com/cxio/evidcoin/pkg/types"

// PubKeyHash 计算公钥哈希。
// 算法：SHA3-256( BLAKE2b-512( publicKey ) )
// 输出 32 字节，用作地址的内核。
func PubKeyHash(pubKey []byte) types.PubKeyHash {
	inner := BLAKE2b_512(pubKey)
	outer := SHA3_256(inner)
	var pkh types.PubKeyHash
	copy(pkh[:], outer[:])
	return pkh
}

// MultiSigPubKeyHash 计算多重签名的复合公钥哈希。
// 算法：SHA3-256( BLAKE2b-512( [m, N] || PKH_1 || PKH_2 || ... || PKH_N ) )
// @m: 所需最小签名数
// @pkhs: 所有参与者的公钥哈希（按排序顺序）
func MultiSigPubKeyHash(m int, pkhs []types.PubKeyHash) types.PubKeyHash {
	// 构造 [m, N] 前缀 + 有序公钥哈希串联
	N := len(pkhs)
	payload := make([]byte, 2+N*types.PubKeyHashLen)
	payload[0] = byte(m)
	payload[1] = byte(N)
	for i, pkh := range pkhs {
		copy(payload[2+i*types.PubKeyHashLen:], pkh[:])
	}
	inner := BLAKE2b_512(payload)
	outer := SHA3_256(inner)
	var result types.PubKeyHash
	copy(result[:], outer[:])
	return result
}

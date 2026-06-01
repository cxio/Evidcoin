package consensus

import (
	"bytes"
	"crypto/subtle"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 铸造者身份两路规则（第 11 章 §3，DEC-0301）。

// VerifyMintIdentityWithPKHash 校验「交易头含 MintPKHash」路径：
// MintPubKey 的单签公钥哈希必须等于 mintPKHash；不要求该公钥参与输入根。
// mintPKHash 必须为 32 字节。
func VerifyMintIdentityWithPKHash(mintPubKey, mintPKHash []byte) error {
	if len(mintPKHash) != 32 {
		return ErrMintIdentityMismatch
	}
	got := crypto.AddressHashSingle(mintPubKey)
	if subtle.ConstantTimeCompare(got.Bytes(), mintPKHash) != 1 {
		return ErrMintIdentityMismatch
	}
	return nil
}

// VerifyMintIdentityWithLeadInput 校验「交易头不含 MintPKHash」路径：
// MintPubKey 的单签公钥哈希作为 LeadPKHash，必须满足输入根验证
// BLAKE3-256(ListHash || LeadPKHash) == expectedInputRoot（第 04 章 §3.2）。
func VerifyMintIdentityWithLeadInput(mintPubKey []byte, listHash types.Hash48, expectedInputRoot types.Hash32) error {
	leadPKHash := crypto.AddressHashSingle(mintPubKey)
	gotRoot := crypto.HashInputRoot(listHash.Bytes(), leadPKHash.Bytes())
	if gotRoot != expectedInputRoot {
		return ErrInputRootMismatch
	}
	return nil
}

// VerifyCoinbaseMintEligibility 校验 Coinbase 铸凭资格：Coinbase 无输入项、无 LeadPKHash，
// 只要显式设置了（非全零）MintPKHash 即可参与竞争（第 11 章 §3，DEC-0301/DEC-0302）。
// 调用方须另行保证该 Coinbase 为已确认交易。
func VerifyCoinbaseMintEligibility(mintPKHash [32]byte) error {
	var zero [32]byte
	if bytes.Equal(mintPKHash[:], zero[:]) {
		return ErrCoinbaseMintPKHashMissing
	}
	return nil
}

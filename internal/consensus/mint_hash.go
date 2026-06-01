package consensus

import (
	"bytes"
	"math/big"
	"slices"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// mintMix 是铸凭哈希前像 X 字段使用的混淆常量（DEC-0301）。
const mintMix uint64 = 0x517cc1b727220a95

// MintHashPreimage 描述铸凭哈希前像的各组成字段（字段顺序由 DEC-0301 冻结）。
// 域标签 "mint.hash" 由 crypto.HashMint 内部前置，本结构不包含域标签。
type MintHashPreimage struct {
	// MintPubKey 是铸造者公钥的规范字节（MintPKHash 或 LeadPKHash 的源公钥）。
	MintPubKey []byte
	// MintTxID 是铸凭交易完整 48 字节 TxID。
	MintTxID types.TxID
	// Stakes 是 -32 号区块头币权销毁值（聪时），大端编码。
	Stakes uint64
	// RefMintHash 是评参区块 Coinbase 记录的铸凭哈希；创世/初段无 Minter 时全零。
	RefMintHash types.MintHash
	// BlockHeight 是铸凭交易实际所在区块高度，仅用于推导 X，与 Stakes 无关。
	BlockHeight uint32
}

// encodeMintX 计算 X = BE(minimal_unsigned(BlockHeight × Mix))。
// 用无损大整数最短大端编码，避免定宽溢出；X 仅由 BlockHeight 与 Mix 决定，
// 与 Stakes 无关（DEC-0301）。乘积为 0 时返回空切片（最短无符号编码）。
func encodeMintX(blockHeight uint32) []byte {
	prod := new(big.Int).Mul(
		new(big.Int).SetUint64(uint64(blockHeight)),
		new(big.Int).SetUint64(mintMix),
	)
	// big.Int.Bytes 返回大端最短无符号字节；零值返回空切片，符合最短编码定义。
	return prod.Bytes()
}

// CanonicalBytes 按 DEC-0301 冻结顺序拼装铸凭哈希前像（不含域标签）：
//
//	MintPubKey || MintTxID || Stakes(BE u64) || RefMintHash || X
func (p MintHashPreimage) CanonicalBytes() []byte {
	x := encodeMintX(p.BlockHeight)
	out := make([]byte, 0, len(p.MintPubKey)+48+8+32+len(x))
	out = append(out, p.MintPubKey...)
	out = append(out, p.MintTxID.Bytes()...)
	out = types.AppendUint64BE(out, p.Stakes)
	out = append(out, p.RefMintHash.Bytes()...)
	out = append(out, x...)
	return out
}

// ComputeMintHash 计算铸凭哈希：BLAKE3-256(DomainTag("mint.hash") || 前像)（DEC-0301）。
func ComputeMintHash(p MintHashPreimage) types.MintHash {
	return crypto.HashMint(p.CanonicalBytes())
}

// MintCandidate 是择优排序中的候选者关键字段（用于三级比较）。
type MintCandidate struct {
	// MintHash 是候选者的铸凭哈希（一级排序键）。
	MintHash types.MintHash
	// TxID 是铸凭交易完整 48 字节 TxID（二级排序键）。
	TxID types.TxID
	// MintPubKey 是铸造者公钥规范字节（三级排序键）。
	MintPubKey []byte
}

// CompareMintCandidates 按择优三级升序比较两个候选者（DEC-0301）：
// 先按 MintHash 32 字节无符号字典序，相等按完整 TxID，再按 MintPubKey 字节序。
// 返回负值表示 a 优于（小于）b，0 表示三级全等，正值表示 a 劣于 b。
func CompareMintCandidates(a, b MintCandidate) int {
	if c := bytes.Compare(a.MintHash[:], b.MintHash[:]); c != 0 {
		return c
	}
	at := a.TxID
	bt := b.TxID
	if c := bytes.Compare(at[:], bt[:]); c != 0 {
		return c
	}
	return bytes.Compare(a.MintPubKey, b.MintPubKey)
}

// RankMintCandidates 就地按择优三级升序排序候选者（值小者优先，DEC-0301）。
func RankMintCandidates(candidates []MintCandidate) {
	slices.SortFunc(candidates, CompareMintCandidates)
}

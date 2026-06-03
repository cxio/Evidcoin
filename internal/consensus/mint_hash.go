package consensus

import (
	"bytes"
	"cmp"
	"math/big"
	"slices"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// mintMix 是铸凭哈希前像 X 字段使用的混淆常量（DEC-0301）。
const mintMix uint64 = 0x517cc1b727220a95

// MintHashPreimage 描述铸凭挑战种子（ChallengeSeed）前像的各组成字段（DEC-0301 冻结）。
// ChallengeSeed = BLAKE3-256(前像)，无域标签；最终 MintHash 由 Equi-X 哈希列表计算。
type MintHashPreimage struct {
	// MintPubKey 是铸造者公钥的规范字节（MintPKHash 或 LeadPKHash 的源公钥）。
	MintPubKey []byte
	// MintTxID 是铸凭交易完整 48 字节 TxID。
	MintTxID types.TxID
	// Stakes 是 -32 号区块头币权销毁值（聪时），大端编码。
	Stakes uint64
	// RefMintHash 是评参区块 Coinbase 记录的铸凭哈希；创世/初段无 Minter 时全零。
	RefMintHash types.MintHash
	// BlockHeight 是当前待铸区块高度，仅用于推导 X，与 Stakes 无关。
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

// CanonicalBytes 按 DEC-0301 冻结顺序拼装挑战种子前像（不含域标签）：
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

// ComputeChallengeSeed 计算 Equi-X 挑战种子：BLAKE3-256(前像)，无域标签（DEC-0301）。
// 前像字段顺序见 MintHashPreimage.CanonicalBytes。
func ComputeChallengeSeed(p MintHashPreimage) []byte {
	return crypto.HashMintChallengeSeed(p.CanonicalBytes())
}

// ComputeMintHash 根据 Equi-X 哈希列表计算最终铸凭哈希（DEC-0301）：
//
//	MintHash = BLAKE3-256( DomainTag("mint.hash") || HashList[0] || HashList[1] || ... )
func ComputeMintHash(hashList [][]byte) types.MintHash {
	var buf []byte
	for _, h := range hashList {
		buf = append(buf, h...)
	}
	return crypto.HashMint(buf)
}

// MintCandidate 是择优排序中的候选者关键字段（用于四级比较）。
type MintCandidate struct {
	// Nonce 是 Equi-X 求解时使用的 nonce（一级排序键，升序）。
	Nonce uint64
	// MintHash 是候选者的铸凭哈希（二级排序键）。
	MintHash types.MintHash
	// TxID 是铸凭交易完整 48 字节 TxID（三级排序键）。
	TxID types.TxID
	// MintPubKey 是铸造者公钥规范字节（四级排序键）。
	MintPubKey []byte
}

// CompareMintCandidates 按择优四级升序比较两个候选者（DEC-0301）：
// 先按 Nonce 升序，再按 MintHash 32 字节无符号字典序，相等按完整 TxID，再按 MintPubKey 字节序。
// 返回负值表示 a 优于（小于）b，0 表示四级全等，正值表示 a 劣于 b。
func CompareMintCandidates(a, b MintCandidate) int {
	if c := cmp.Compare(a.Nonce, b.Nonce); c != 0 {
		return c
	}
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

// RankMintCandidates 就地按择优四级升序排序候选者（值小者优先，DEC-0301）。
func RankMintCandidates(candidates []MintCandidate) {
	slices.SortFunc(candidates, CompareMintCandidates)
}


// Package crypto 固化协议级哈希配置、域隔离标签、公钥/地址编码与后量子签名
// （ML-DSA-65）抽象。该包仅依赖 pkg/types 与第三方密码库。
package crypto

import (
	"crypto/sha3"
	"hash"

	"github.com/cxio/evidcoin/pkg/types"
	"lukechampine.com/blake3"
)

// domainPrefix 是所有域标签统一使用的固定命名空间前缀（DEC-0002）。
const domainPrefix = "Evidcoin/v1/"

// 域标签名称（完整 17 项：DEC-0002 的 12 项 + DEC-0201 的两个空根标签 + 三个输出摘要标签）。
const (
	tagNameBlockHeader        = "block.header"
	tagNameTxHeader           = "tx.header"
	tagNameTreeLeaf           = "tree.leaf"
	tagNameTreeBranch         = "tree.branch"
	tagNameCheckRoot          = "checkroot"
	tagNameUTXOLeaf           = "utxo.leaf"
	tagNameUTCOLeaf           = "utco.leaf"
	tagNameMintHash           = "mint.hash"
	tagNameSignatureMsg       = "signature.message"
	tagNameAttachment         = "attachment.fingerprint"
	tagNameAddressSingle      = "address.single"
	tagNameAddressMulti       = "address.multi"
	tagNameUTXOEmpty          = "utxo.empty"
	tagNameUTCOEmpty          = "utco.empty"
	tagNameOutputDigestAcct   = "output.digest.account"
	tagNameOutputDigestConten = "output.digest.content"
	tagNameOutputDigestScript = "output.digest.script"
)

// 预计算域标签（`"Evidcoin/v1/" || name || 0x00`）。这些是协议唯一权威标签；
// 调用方不得向哈希 API 传入任意自定义标签，域隔离由下方按用途函数绑定。
var (
	tagBlockHeader        = DomainTag(tagNameBlockHeader)
	tagTxHeader           = DomainTag(tagNameTxHeader)
	tagTreeLeaf           = DomainTag(tagNameTreeLeaf)
	tagTreeBranch         = DomainTag(tagNameTreeBranch)
	tagCheckRoot          = DomainTag(tagNameCheckRoot)
	tagUTXOLeaf           = DomainTag(tagNameUTXOLeaf)
	tagUTCOLeaf           = DomainTag(tagNameUTCOLeaf)
	tagMintHash           = DomainTag(tagNameMintHash)
	tagSignatureMsg       = DomainTag(tagNameSignatureMsg)
	tagAttachment         = DomainTag(tagNameAttachment)
	tagAddressSingle      = DomainTag(tagNameAddressSingle)
	tagAddressMulti       = DomainTag(tagNameAddressMulti)
	tagUTXOEmpty          = DomainTag(tagNameUTXOEmpty)
	tagUTCOEmpty          = DomainTag(tagNameUTCOEmpty)
	tagOutputDigestAcct   = DomainTag(tagNameOutputDigestAcct)
	tagOutputDigestConten = DomainTag(tagNameOutputDigestConten)
	tagOutputDigestScript = DomainTag(tagNameOutputDigestScript)
)

// DomainTag 根据用途名称构造域标签："Evidcoin/v1/" || name || 0x00
// （DEC-0002）。该标签必须作为哈希原像的首段。
func DomainTag(name string) []byte {
	tag := make([]byte, 0, len(domainPrefix)+len(name)+1)
	tag = append(tag, domainPrefix...)
	tag = append(tag, name...)
	tag = append(tag, 0x00)
	return tag
}

// sum 按顺序将各段写入 h 并返回摘要。
func sum(h hash.Hash, parts ...[]byte) []byte {
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

func sha3_384(parts ...[]byte) []byte { return sum(sha3.New384(), parts...) }
func sha3_512(parts ...[]byte) []byte { return sum(sha3.New512(), parts...) }

// blake3_256 返回拼接输入的 32 字节 BLAKE3 摘要。BLAKE3 从不使用 keyed
// 模式（DEC-0002）；隔离完全依赖域标签。
func blake3_256(parts ...[]byte) [32]byte {
	h := blake3.New(32, nil)
	for _, p := range parts {
		_, _ = h.Write(p)
	}
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// HashBlockHeader 对区块头原像做哈希（SHA3-384 + block.header）。
func HashBlockHeader(data []byte) types.BlockID {
	id, _ := types.NewBlockID(sha3_384(tagBlockHeader, data))
	return id
}

// HashTxHeader 对交易头原像做哈希（SHA3-384 + tx.header）。
func HashTxHeader(data []byte) types.TxID {
	id, _ := types.NewTxID(sha3_384(tagTxHeader, data))
	return id
}

// HashCheckRoot 对校验根原像做哈希（SHA3-384 + checkroot）。
func HashCheckRoot(data []byte) types.CheckRoot {
	r, _ := types.NewCheckRoot(sha3_384(tagCheckRoot, data))
	return r
}

// HashTreeLeaf 对通用树叶子 payload 做哈希（SHA3-384 + tree.leaf）。
func HashTreeLeaf(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagTreeLeaf, data))
	return h
}

// HashTreeBranch 对通用树分支原像做哈希（BLAKE3-256 + tree.branch）。
// data 为 left || right 的拼接。
func HashTreeBranch(data []byte) types.TreeHash {
	return types.TreeHash(blake3_256(tagTreeBranch, data))
}

// HashUTXOLeaf 对 UTXO 末端叶子 payload 做哈希（SHA3-384 + utxo.leaf）。
func HashUTXOLeaf(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagUTXOLeaf, data))
	return h
}

// HashUTCOLeaf 对 UTCO 末端叶子 payload 做哈希（SHA3-384 + utco.leaf）。
func HashUTCOLeaf(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagUTCOLeaf, data))
	return h
}

// HashAttachment 对附件完整指纹做哈希（SHA3-512 + attachment.fingerprint）。
func HashAttachment(data []byte) types.AttachmentHash {
	h, _ := types.NewAttachmentHash(sha3_512(tagAttachment, data))
	return h
}

// HashMint 对铸凭哈希列表拼接做最终哈希（BLAKE3-256 + mint.hash，32 字节，DEC-0301）。
// data 为 Equi-X hashList 各项顺序拼接的字节。
func HashMint(data []byte) types.MintHash {
	return types.MintHash(blake3_256(tagMintHash, data))
}

// HashMintChallengeSeed 计算铸凭 Equi-X 挑战种子（纯 BLAKE3-256，无域标签，DEC-0301）。
// ChallengeSeed 是 Equi-X 内部挑战值，不单独命名，不分配域标签（DEC-0002）。
// preimage 为 MintHashPreimage.CanonicalBytes() 的输出。
func HashMintChallengeSeed(preimage []byte) []byte {
	h := blake3_256(preimage)
	return h[:]
}

// SignatureMessageTag 返回 signature.message 域标签字节，供签名消息配置使用
// （DEC-0102，第 08 章）。
func SignatureMessageTag() []byte {
	out := make([]byte, len(tagSignatureMsg))
	copy(out, tagSignatureMsg)
	return out
}

// EmptyUTXORoot 返回 UTXO 空状态树根：BLAKE3-256(DomainTag("utxo.empty")).
func EmptyUTXORoot() types.TreeHash {
	return types.TreeHash(blake3_256(tagUTXOEmpty))
}

// EmptyUTCORoot 返回 UTCO 空状态树根：BLAKE3-256(DomainTag("utco.empty")).
func EmptyUTCORoot() types.TreeHash {
	return types.TreeHash(blake3_256(tagUTCOEmpty))
}

// HashOutputDigestAccount 对输出项接收者片段计算摘要（SHA3-384 + output.digest.account，DEC-0002）。
// 当输出项配置 bit7（DigestAccount）置位时，以此摘要替代原始接收者字节参与输出项叶哈希前像。
func HashOutputDigestAccount(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagOutputDigestAcct, data))
	return h
}

// HashOutputDigestContent 对输出项内容片段计算摘要（SHA3-384 + output.digest.content，DEC-0002）。
// 当输出项配置 bit6（DigestContent）置位时，以此摘要替代原始内容字节参与输出项叶哈希前像。
func HashOutputDigestContent(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagOutputDigestConten, data))
	return h
}

// HashOutputDigestScript 对输出项锁定脚本片段计算摘要（SHA3-384 + output.digest.script，DEC-0002）。
// 当输出项配置 bit5（DigestScript）置位时，以此摘要替代原始脚本字节参与输出项叶哈希前像。
func HashOutputDigestScript(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(tagOutputDigestScript, data))
	return h
}

// HashInputList 计算交易输入项列表的串联哈希 ListHash（第 04 章 §3.3）：
// SHA3-384(data)，无域标签。这是交易输入根的专用规则：按 proposal 04 §3.3 与
// DEC-0002 域标签全集，输入根未分配域标签，故此处不前置域标签。
func HashInputList(data []byte) types.Hash48 {
	h, _ := types.NewHash48(sha3_384(data))
	return h
}

// HashInputRoot 计算交易输入根 HashInputs（第 04 章 §3.3）：
// BLAKE3-256( listHash || leadPKHash )，无域标签（输入根专用规则）。
func HashInputRoot(listHash, leadPKHash []byte) types.Hash32 {
	return types.Hash32(blake3_256(listHash, leadPKHash))
}

// HashAttachmentPieceLeaf 对附件分片树叶子做哈希。这是唯一不带域标签的例外
// （DEC-0002）：BLAKE3-256(2-byte seq || BLAKE3-256(piece))，
// 原像长度 34 字节且不含域标签，以便外部文件分享工具复用。
func HashAttachmentPieceLeaf(seq uint16, piece []byte) types.TreeHash {
	pieceHash := blake3_256(piece)
	var seqBytes [2]byte
	seqBytes[0] = byte(seq >> 8)
	seqBytes[1] = byte(seq)
	return types.TreeHash(blake3_256(seqBytes[:], pieceHash[:]))
}

// HashAttachmentPieceBranch 对附件分片树分支做哈希：
// BLAKE3-256(left || right)，不带域标签（DEC-0002 例外）。
func HashAttachmentPieceBranch(left, right types.TreeHash) types.TreeHash {
	l := types.Hash32(left)
	r := types.Hash32(right)
	return types.TreeHash(blake3_256(l[:], r[:]))
}

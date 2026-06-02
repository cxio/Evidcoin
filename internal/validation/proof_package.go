package validation

import (
	"github.com/cxio/evidcoin/internal/blockchain"
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// ProofPackage 是区块证明包（第 13 章 §5，DEC-0601）。
// 铸造者和管理层协作后由管理层组装并广播，接收方可快速预验证而无需完整区块下载。
// 共 8 个字段，字段顺序固定；不含 UTXO/UTCO 状态证明——接收方只能与本地状态比较。
// P2P 解码（Decode）属于 C-10 待决内容，本版本不实现。
type ProofPackage struct {
	// BlockHeader 是候选区块的完整头部（DEC-0601 字段 1）。
	BlockHeader blockchain.BlockHeader
	// CoinbaseTx 是本区块的 Coinbase 交易头（DEC-0601 字段 2）。
	CoinbaseTx tx.CoinbaseHeader
	// CoinbaseTxIndex 是 Coinbase 在交易树中的下标，按协议始终为 0（DEC-0601 字段 3）。
	CoinbaseTxIndex uint32
	// CoinbaseMerklePath 是 Coinbase 的交易树 Merkle 路径（DEC-0601 字段 4）。
	CoinbaseMerklePath hashtree.Proof
	// TreeRoot 是本区块完整交易树根（DEC-0601 字段 5）。
	TreeRoot []byte
	// UTXORoot 是完成本区块交易后的 UTXO 状态指纹（DEC-0601 字段 6，DEC-0201）。
	UTXORoot types.TreeHash
	// UTCORoot 是完成本区块交易后的 UTCO 状态指纹（DEC-0601 字段 7，DEC-0201）。
	UTCORoot types.TreeHash
	// MinterCheckRootSignature 是铸造者对 BlockHeader.CheckRoot 的签名（DEC-0601 字段 8）。
	// 签名覆盖范围：CheckRoot.Bytes()（DEC-0102 §5）；不计入 CoinbaseTxID 哈希范围。
	MinterCheckRootSignature []byte
}

// Encode 将证明包序列化为规范字节（DEC-0601）。
// 各字段的编码顺序固定为字段 1..8；可变长数据以 varint(len)||bytes 格式前缀。
// CoinbaseHeader.CanonicalBytes() 如遇非法结构（如创世规则违反）则返回错误。
func (pp *ProofPackage) Encode() ([]byte, error) {
	dst := make([]byte, 0, 512)

	// 字段 1：BlockHeader
	hb := pp.BlockHeader.CanonicalBytes()
	dst = types.AppendBytes(dst, hb)

	// 字段 2：CoinbaseTx
	cb, err := pp.CoinbaseTx.CanonicalBytes()
	if err != nil {
		return nil, err
	}
	dst = types.AppendBytes(dst, cb)

	// 字段 3：CoinbaseTxIndex（uint32 大端，固定 4 字节）
	dst = types.AppendUint32BE(dst, pp.CoinbaseTxIndex)

	// 字段 4：CoinbaseMerklePath（自定义 Merkle 路径编码）
	dst = appendProof(dst, pp.CoinbaseMerklePath)

	// 字段 5：TreeRoot（varint(len) || bytes）
	dst = types.AppendBytes(dst, pp.TreeRoot)

	// 字段 6：UTXORoot（32 字节，原样追加）
	dst = append(dst, pp.UTXORoot[:]...)

	// 字段 7：UTCORoot（32 字节，原样追加）
	dst = append(dst, pp.UTCORoot[:]...)

	// 字段 8：MinterCheckRootSignature（varint(len) || bytes）
	dst = types.AppendBytes(dst, pp.MinterCheckRootSignature)

	return dst, nil
}

// appendProof 将 Merkle 路径追加到 dst 并返回扩展后的切片。
// 编码格式：
//
//	varint(len(LeafHash)) || LeafHash
//	varint(len(Siblings))
//	for each step: Direction(1 byte) || varint(len(Hash)) || Hash
//	varint(len(Root)) || Root
func appendProof(dst []byte, p hashtree.Proof) []byte {
	dst = types.AppendBytes(dst, p.LeafHash)
	dst = types.AppendVarUint(dst, uint64(len(p.Siblings)))
	for _, s := range p.Siblings {
		dst = append(dst, byte(s.Direction))
		dst = types.AppendBytes(dst, s.Hash)
	}
	dst = types.AppendBytes(dst, p.Root)
	return dst
}

package services

import "github.com/cxio/evidcoin/pkg/types"

// TxIDPrefixLen 是区块概要中 TxID 前缀的固定字节长度（16 字节，DEC-0602）。
// 不设 TxIDPrefixLen 字段，不协商其它长度；固定 16 字节。
const TxIDPrefixLen = 16

// TxIDPrefix 是完整 TxID 的前 16 字节，用于网络同步优化（DEC-0602）。
// 定长数组类型防止与完整 TxID 混用。
type TxIDPrefix [TxIDPrefixLen]byte

// NewTxIDPrefix 从完整 TxID 提取前 16 字节作为前缀（DEC-0602）。
func NewTxIDPrefix(txID types.TxID) TxIDPrefix {
	var p TxIDPrefix
	// 取 TxID 的前 TxIDPrefixLen 字节；TxID 为 48 字节，前缀长度 16，不越界。
	copy(p[:], txID[:TxIDPrefixLen])
	return p
}

// BlockSummary 是网络区块概要格式（DEC-0602）。
//
// 编码格式：BlockID || varint(TxCount) || TxIDPrefix*
//
// TxIDPrefixes 按区块交易序列顺序排列，包含 Coinbase（索引 0）。
// 每个前缀固定为 16 字节（TxID 的前 16 字节），不带长度前缀。
//
// 区块概要仅作网络同步优化，不属于共识数据，不需要发布方单独签名。
// 节点不得因短前缀无法解析就接受不完整区块；最终验证必须用完整 TxID 序列
// 重算交易树根（DEC-0602）。
type BlockSummary struct {
	// BlockID is the block identifier (SHA3-384, 48 bytes).
	BlockID types.BlockID
	// TxCount is the total transaction count in the block (including Coinbase).
	TxCount uint64
	// TxIDPrefixes contains the first 16 bytes of each transaction's TxID,
	// ordered by in-block sequence position with Coinbase at index 0.
	TxIDPrefixes []TxIDPrefix
}

// NewBlockSummary 从完整 TxID 序列构造区块概要。
// txIDs 必须按区块交易顺序排列，Coinbase 位于首位（索引 0）。
// TxIDPrefixes 由每个 TxID 的前 16 字节自动提取。
func NewBlockSummary(blockID types.BlockID, txIDs []types.TxID) BlockSummary {
	prefixes := make([]TxIDPrefix, len(txIDs))
	for i, id := range txIDs {
		prefixes[i] = NewTxIDPrefix(id)
	}
	return BlockSummary{
		BlockID:      blockID,
		TxCount:      uint64(len(txIDs)),
		TxIDPrefixes: prefixes,
	}
}

// Encode 将区块概要序列化为规范字节：BlockID(48B) || varint(TxCount) || TxIDPrefix*(16B each)。
//
// 每个 TxIDPrefix 原样追加为 16 字节（定长，无额外长度前缀）。
// 编码结果为确定性字节序列，可供网络同步使用。
// 注意：此编码不进入共识；最终验证须从完整 TxID 序列重算交易树根。
func (s *BlockSummary) Encode() []byte {
	// 预估容量：48（BlockID）+ 10（varint 上限）+ 16*n（前缀）。
	dst := make([]byte, 0, 48+10+TxIDPrefixLen*len(s.TxIDPrefixes))
	dst = append(dst, s.BlockID.Bytes()...)
	dst = types.AppendVarUint(dst, s.TxCount)
	for _, p := range s.TxIDPrefixes {
		dst = append(dst, p[:]...)
	}
	return dst
}

// CollisionFallback 是碰撞回退响应（DEC-0602）。
//
// 当接收方在区块概要的某个序位发现本地候选交易有多个前缀匹配（碰撞）时，
// 按交易序位请求碰撞回退；发布方对指定序位返回完整 48 字节 TxID。
//
// CollisionFallback 不属于基础 BlockSummary 本体，是独立的按需响应。
// 接收方收到完整 TxID 后，可精确匹配候选交易并继续组装区块。
type CollisionFallback struct {
	// BlockID identifies the block whose summary triggered the collision.
	BlockID types.BlockID
	// TxIndex is the in-block sequence position (0-based) of the colliding prefix.
	TxIndex uint32
	// FullTxID is the complete 48-byte TxID for the specified sequence position.
	FullTxID types.TxID
}

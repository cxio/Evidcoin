package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// 链身份材料（第 03 章 §1、§2）。本层只提供身份材料字节，不定义签名消息语义：
// 完整签名消息 MixData = ChainIdentity.Bytes() || TxMSG 由第 08 章组装。
// 身份字段按 conception 原样拼接（不加长度前缀），以便上层签名字节精确匹配规范。

// boundIDPrefixLen 是 Bound-ID 取用的 BlockID 前缀字节数。
const boundIDPrefixLen = 20

// BoundID 是主链绑定标识（分叉后首块），格式 <Height><BlockID[:20]>，可选（第 03 章 §1）。
// 用于分叉后绑定主链，避免新交易在支链上被重放。
type BoundID struct {
	// Height 是分叉后首块的高度。
	Height uint32
	// BlockPrefix 是分叉后首块 BlockID 的前 20 字节。
	BlockPrefix [boundIDPrefixLen]byte
}

// NewBoundID 由分叉后首块高度与完整 BlockID 构造 BoundID，取 BlockID 前 20 字节。
func NewBoundID(height uint32, blockID types.BlockID) BoundID {
	var b BoundID
	b.Height = height
	copy(b.BlockPrefix[:], blockID.Bytes()[:boundIDPrefixLen])
	return b
}

// AppendTo 将 BoundID 以 <Height(4B 大端)><BlockPrefix(20B)> 追加到 dst。
func (b BoundID) AppendTo(dst []byte) []byte {
	dst = types.AppendUint32BE(dst, b.Height)
	return append(dst, b.BlockPrefix[:]...)
}

// ChainIdentity 是链身份材料：Protocol-ID、Chain-ID、Genesis-ID 与可选 Bound-ID（第 03 章 §1）。
type ChainIdentity struct {
	// ProtocolID 区分本链与其它链，主网值为 types.ProtocolID（"Evidcoin@v1"）。
	ProtocolID string
	// ChainID 是运行态标识：mainnet / testnet / devnet。
	ChainID string
	// GenesisID 是创世块 ID（具体值由 C-9 裁决，本层不固化）。
	GenesisID types.BlockID
	// Bound 是可选的主链绑定标识；分叉成为独立链后不再必须。
	Bound *BoundID
}

// Bytes 返回链身份材料的规范拼接：
//
//	Protocol-ID || Chain-ID || Genesis-ID || [Bound-ID]
//
// 该字节序列即 MixData 的前段身份材料（第 03 章 §2），按 conception 原样拼接，
// Bound-ID 缺失时完全省略（不写占位）。
func (id ChainIdentity) Bytes() []byte {
	dst := make([]byte, 0, len(id.ProtocolID)+len(id.ChainID)+48+24)
	dst = append(dst, id.ProtocolID...)
	dst = append(dst, id.ChainID...)
	dst = append(dst, id.GenesisID.Bytes()...)
	if id.Bound != nil {
		dst = id.Bound.AppendTo(dst)
	}
	return dst
}

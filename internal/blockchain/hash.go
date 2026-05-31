package blockchain

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// ID 计算区块头的 BlockID（第 05 章 §1.1）：
//
//	BlockID = SHA3-384( DomainTag("block.header") || CanonicalBytes() )
//
// 年块前像包含 YearBlock 字段，非年块前像不含该字段（与省略规则一致）。
func (h *BlockHeader) ID() types.BlockID {
	return crypto.HashBlockHeader(h.CanonicalBytes())
}

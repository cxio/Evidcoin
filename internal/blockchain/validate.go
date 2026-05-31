package blockchain

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// 最小入块验证（第 05 章 §4）。核心仅做最小合法性验证：高度连续性、PrevBlock 衔接、
// 同高度冲突拒绝。区块本身的完整合法性（交易、签名、状态、PoH）不在本层。

// validateNext 校验 h 是否可作为当前链的下一个区块头入块。
// 空链时要求 h 为合法创世头；非空链时要求高度为 tip+1 且 PrevBlock 衔接 tip。
func (c *Chain) validateNext(h *BlockHeader) error {
	tip, err := c.store.Tip()
	if errors.Is(err, ErrHeaderNotFound) {
		// 空链：首块必须是创世头（高度 0、PrevBlock 全零）。
		if h.Height != 0 || h.PrevBlock != (types.BlockID{}) {
			return ErrNotGenesis
		}
		return nil
	}
	if err != nil {
		return err
	}
	// 非空链：高度必须为 tip+1。
	if h.Height != tip.Height+1 {
		// 高度 ≤ tip：该高度已敲定，视为同高度/历史冲突，拒绝且不替换。
		if h.Height <= tip.Height {
			return ErrHeightConflict
		}
		// 高度 > tip+1：缺失中间头，拒绝跨高度衔接。
		return ErrHeightNotSequential
	}
	// PrevBlock 必须等于当前 tip 的 ID。
	if h.PrevBlock != tip.ID() {
		return ErrPrevBlockMismatch
	}
	return nil
}

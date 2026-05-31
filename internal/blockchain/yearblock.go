package blockchain

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// 年块边界与恢复衔接（第 05 章 §5）。区块头因支持年块哈希，可只维持年块衔接式连续性，
// 用户可配置忽略任意年度的完整区块头数据；完整性缺口由 Blockqs（第 15 章）回填。

// IsYearBoundary 报告 height 是否位于年度边界（height % BlocksPerYear == 0）。
// 创世（高度 0）亦为年度边界。
func IsYearBoundary(height uint32) bool {
	return height%types.BlocksPerYear == 0
}

// LastYearBoundary 返回 ≤ height 的最近年度边界高度。
func LastYearBoundary(height uint32) uint32 {
	return height - height%types.BlocksPerYear
}

// YearBlockHeader 返回 ≤ height 的最近年度边界处的区块头（年块）。
// 若该年块未存储，返回 ErrYearBlockMissing（调用方可经 Blockqs 回填后重试）。
func (c *Chain) YearBlockHeader(height uint32) (*BlockHeader, error) {
	boundary := LastYearBoundary(height)
	h, err := c.store.ByHeight(boundary)
	if errors.Is(err, ErrHeaderNotFound) {
		return nil, ErrYearBlockMissing
	}
	return h, err
}

// ValidateRecoveredHeader 校验恢复头 h 与其前驱 prev、后继 next 正确衔接：
// 三者高度必须连续，且 h.PrevBlock 指向 prev、next.PrevBlock 指向 h。
// 用于经 Blockqs 回填的区块头与已有链段的衔接校验（第 05 章 §5）。
func ValidateRecoveredHeader(prev, h, next *BlockHeader) error {
	if h.Height != prev.Height+1 || next.Height != h.Height+1 {
		return ErrHeightNotSequential
	}
	if h.PrevBlock != prev.ID() {
		return ErrPrevBlockMismatch
	}
	if next.PrevBlock != h.ID() {
		return ErrPrevBlockMismatch
	}
	return nil
}

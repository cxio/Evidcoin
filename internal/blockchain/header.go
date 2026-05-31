// Package blockchain 实现 Layer 1 区块链核心：区块头结构与规范编码、BlockID 计算、
// CheckRoot 合并、年块边界、最小入块衔接验证与区块尺寸限额曲线。
// 本包只管理区块头链与最小验证，不执行交易、脚本、状态转移、PoH 或网络逻辑；
// 仅依赖 pkg/types 与 pkg/crypto，禁止依赖任何上层内部包（第 05 章）。
package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// BlockHeader 是区块头结构（第 05 章 §1、DEC-0003）。字段编码顺序固定，
// 不含时间戳（时间戳由高度与出块间隔从创世时间戳推导）。
type BlockHeader struct {
	// Version 是区块版本号，创世固定为 1。
	Version uint32
	// Height 是区块高度，代替时间戳信息。
	Height uint32
	// PrevBlock 是前一区块 ID（SHA3-384）。
	PrevBlock types.BlockID
	// CheckRoot 是校验根（SHA3-384，见 checkroot.go）。
	CheckRoot types.CheckRoot
	// Stakes 是该区块收录全部交易的币权（币量×币龄）合计，
	// 单位「聪时」，溢出截断。经济计算语义由第 07 章承载，本层只固定字段。
	Stakes uint64
	// YearBlock 是前一年块哈希。仅当 IsYearBlock() 为真（Height % BlocksPerYear == 0）
	// 时才参与编码与 BlockID 前像；非年块完全省略（不编码全零占位，DEC-0003）。
	// 创世（高度 0）为年块，但因无前一年块故此值全零。
	YearBlock types.Hash48
}

// IsYearBlock 报告该区块头是否位于年度边界（Height % BlocksPerYear == 0）。
// 年块（含创世）的 YearBlock 字段参与编码，非年块则省略该字段。
func (h *BlockHeader) IsYearBlock() bool {
	return h.Height%types.BlocksPerYear == 0
}

// CanonicalBytes 返回区块头的规范编码字节（DEC-0003）。
// 字段顺序固定为 Version || Height || PrevBlock || CheckRoot || Stakes，
// 年块在末尾追加 YearBlock。常规区块头 112 字节，年块 160 字节。
func (h *BlockHeader) CanonicalBytes() []byte {
	dst := make([]byte, 0, 160)
	dst = types.AppendUint32BE(dst, h.Version)
	dst = types.AppendUint32BE(dst, h.Height)
	dst = append(dst, h.PrevBlock.Bytes()...)
	dst = append(dst, h.CheckRoot.Bytes()...)
	dst = types.AppendUint64BE(dst, h.Stakes)
	if h.IsYearBlock() {
		dst = append(dst, h.YearBlock.Bytes()...)
	}
	return dst
}

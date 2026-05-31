package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// 创世区块头工件（第 05 章 §9）。本层只固定创世区块头的确定字段、编码边界与验证规则；
// 不计算交易树，故 CheckRoot 具体值由调用方（依赖第 06、14 章 Coinbase 工件）提供。
//
// 待决 C-9：mainnet 的创世 BlockID（Genesis-ID）与创世时间戳尚未冻结，
// 本包不硬编码其数值；裁决前依赖这些数值的上层任务（创世硬编码、初段评参）阻塞。

// GenesisVersion 是创世区块头的版本号，固定为 1（第 05 章 §9、DEC-0003）。
const GenesisVersion uint32 = 1

// NewGenesisHeader 构造创世区块头的确定字段（第 05 章 §9）：
// Version=1、Height=0、PrevBlock 全零、Stakes=0、YearBlock 存在但全零
// （高度 0 为年块，但无前一年块故值全零）。
//
// checkRoot 由调用方传入：它按常规计算（仅一笔 Coinbase 交易，关联 UTXO/UTCO 为空根），
// 依赖创世 Coinbase 交易树根与空状态根（第 06、14、09 章）；本层不在此固定其值。
// 创世 BlockID（Genesis-ID）与创世时间戳属待决 C-9，本函数不涉及其数值。
func NewGenesisHeader(checkRoot types.CheckRoot) *BlockHeader {
	return &BlockHeader{
		Version:   GenesisVersion,
		Height:    0,
		PrevBlock: types.BlockID{}, // 无前一区块
		CheckRoot: checkRoot,
		Stakes:    0,              // 无币权销毁
		YearBlock: types.Hash48{}, // 年块字段存在但全零（无前一年块）
	}
}

// ValidateGenesisHeader 校验 h 是否符合创世区块头工件的确定边界规则（第 05 章 §9）：
// Version==1、Height==0、PrevBlock 全零、Stakes==0、YearBlock 全零。
// 不符返回 ErrInvalidGenesisHeader。
//
// 本校验不涉及 CheckRoot 具体值（依赖 Coinbase 工件，第 06、14 章），
// 也不涉及 Genesis-ID/创世时间戳（待决 C-9，未冻结）。
func ValidateGenesisHeader(h *BlockHeader) error {
	if h.Version != GenesisVersion ||
		h.Height != 0 ||
		h.PrevBlock != (types.BlockID{}) ||
		h.Stakes != 0 ||
		h.YearBlock != (types.Hash48{}) {
		return ErrInvalidGenesisHeader
	}
	return nil
}

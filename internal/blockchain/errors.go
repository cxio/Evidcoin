package blockchain

import "errors"

// 区块链核心层错误定义（第 05 章 §4）。
// ErrHeaderNotFound 属存储层语义，定义在 store.go。

var (
	// ErrNotGenesis 表示空链接收的首块不是合法创世头（高度须为 0 且 PrevBlock 全零）。
	ErrNotGenesis = errors.New("blockchain: first header must be genesis")
	// ErrHeightNotSequential 表示新头高度不等于 tip+1（含缺失中间头的跨高度衔接）。
	ErrHeightNotSequential = errors.New("blockchain: header height not sequential")
	// ErrHeightConflict 表示该高度已存在区块头（同高度二次提交），核心一律拒绝且不自动替换。
	ErrHeightConflict = errors.New("blockchain: header height conflict")
	// ErrPrevBlockMismatch 表示新头 PrevBlock 未正确衔接当前 tip 的 ID。
	ErrPrevBlockMismatch = errors.New("blockchain: prev block does not link to tip")
	// ErrBlockIDMismatch 表示调用方声明的 BlockID 与重算结果不一致。
	ErrBlockIDMismatch = errors.New("blockchain: block id mismatch")
)

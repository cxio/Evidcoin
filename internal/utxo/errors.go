package utxo

import "errors"

// UTXO 状态层错误定义（第 09 章、DEC-0201）。

var (
	// ErrNotFound 表示按完整 OutPoint 查询或花费时未命中状态集。
	ErrNotFound = errors.New("utxo: outpoint not found")
	// ErrDuplicate 表示插入的 OutPoint 已存在于状态集。
	ErrDuplicate = errors.New("utxo: outpoint already exists")
	// ErrAlreadySpent 表示对已花费的 entry 再次花费。
	ErrAlreadySpent = errors.New("utxo: outpoint already spent")
	// ErrInputKindInvalid 表示传入 UTXO apply 的输入来源类别不是币金
	// （凭信/存证输入不应进入 Coin 状态转移）。
	ErrInputKindInvalid = errors.New("utxo: input kind must be coin")
)

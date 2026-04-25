// Package utxo 管理 Evidcoin 未花费交易输出（UTXO）集合。
package utxo

import "github.com/cxio/evidcoin/pkg/types"

// Outpoint 指向特定交易的特定输出。
type Outpoint struct {
	TxID     types.Hash384 // 交易 ID（48 字节）
	OutIndex int           // 输出序位（从 0 开始）
}

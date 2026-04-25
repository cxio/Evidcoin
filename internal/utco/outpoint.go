// Package utco 管理 Evidcoin 未转移凭信输出（UTCO）集合。
package utco

import "github.com/cxio/evidcoin/pkg/types"

// Outpoint 指向特定交易的特定凭信输出。
type Outpoint struct {
	TxID     types.Hash384
	OutIndex int
}

package blockchain

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// CheckRoot 合并（第 05 章 §2）。本层只组合已给定的根，不在核心层计算交易树或状态树。

// StateFingerprintProvider 提供指定**已完成高度**的 UTXO/UTCO 集指纹。
// 具体实现由第 09 章状态层承载；本层只依赖该读取契约以组合 CheckRoot。
type StateFingerprintProvider interface {
	// StateFingerprint 返回 completedHeight 区块执行完成后的 UTXO/UTCO 指纹。
	StateFingerprint(completedHeight uint32) (utxoRoot, utcoRoot types.Hash48, err error)
}

// ComputeCheckRoot 组合校验根（第 05 章 §2）：
//
//	CheckRoot = SHA3-384( DomainTag("checkroot") || TreeRoot || UTXORoot || UTCORoot )
//
// treeRoot 为区块交易树根（第 04 章 §3.1），UTXORoot/UTCORoot 为前一区块完成后的
// 状态指纹。UTXO 与 UTCO 顺序固定，不可调换。
func ComputeCheckRoot(treeRoot []byte, utxoRoot, utcoRoot types.Hash48) types.CheckRoot {
	pre := make([]byte, 0, len(treeRoot)+len(utxoRoot)+len(utcoRoot))
	pre = append(pre, treeRoot...)
	pre = append(pre, utxoRoot.Bytes()...)
	pre = append(pre, utcoRoot.Bytes()...)
	return crypto.HashCheckRoot(pre)
}

// ComputeCheckRootAt 按区块高度组合 CheckRoot：状态根取**前一区块完成后**的指纹。
// height == 0（创世）无前一区块，使用空状态指纹（第 05 章空根规则，不查询 provider）；
// height > 0 读取上一高度 height-1 完成后的状态指纹（链式状态约束，第 09 章）。
func ComputeCheckRootAt(height uint32, treeRoot []byte, provider StateFingerprintProvider) (types.CheckRoot, error) {
	var utxoRoot, utcoRoot types.Hash48
	if height == 0 {
		utxoRoot = crypto.EmptyUTXORoot()
		utcoRoot = crypto.EmptyUTCORoot()
	} else {
		var err error
		utxoRoot, utcoRoot, err = provider.StateFingerprint(height - 1)
		if err != nil {
			return types.CheckRoot{}, err
		}
	}
	return ComputeCheckRoot(treeRoot, utxoRoot, utcoRoot), nil
}

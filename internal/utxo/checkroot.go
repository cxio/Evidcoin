package utxo

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// ComputeCheckRoot 计算区块头 CheckRoot 字段：
//
//	CheckRoot = SHA3-384( TxTreeRoot ‖ UTXOFingerprint ‖ UTCOFingerprint )
func ComputeCheckRoot(txTreeRoot types.Hash384, utxoFP, utcoFP types.Hash256) types.Hash384 {
	var buf []byte
	buf = append(buf, txTreeRoot[:]...)
	buf = append(buf, utxoFP[:]...)
	buf = append(buf, utcoFP[:]...)
	return crypto.SHA3_384(buf)
}

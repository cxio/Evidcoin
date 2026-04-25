package tx

import (
	"encoding/binary"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// ComputeTxID 计算交易 ID：SHA3-384( TxHeader.Bytes() )。
func ComputeTxID(header *TxHeader) types.Hash384 {
	return crypto.SHA3_384(txHeaderBytes(header))
}

// txHeaderBytes 将交易头序列化为字节（用于 TxID 计算）。
func txHeaderBytes(h *TxHeader) []byte {
	buf := make([]byte, 0, 8+8+types.Hash256Len*2)
	buf = AppendVarint(buf, uint64(h.Version))
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(h.Timestamp))
	buf = append(buf, ts...)
	buf = append(buf, h.HashInputs[:]...)
	buf = append(buf, h.HashOutputs[:]...)
	return buf
}

// ComputeLeadHash 计算首领输入哈希：BLAKE3-256( Year || TxID[48] || OutIndex )。
func ComputeLeadHash(lead *LeadInput) types.Hash256 {
	buf := make([]byte, 0, 8+types.Hash384Len+8)
	buf = AppendVarint(buf, uint64(lead.Year))
	buf = append(buf, lead.TxID[:]...)
	buf = AppendVarint(buf, uint64(lead.OutIndex))
	return crypto.BLAKE3_256(buf)
}

// ComputeRestHash 计算非首领输入集合哈希：BLAKE3-256( rest1 || rest2 || ... )。
func ComputeRestHash(rests []RestInput) types.Hash256 {
	var buf []byte
	for _, r := range rests {
		buf = AppendVarint(buf, uint64(r.Year))
		buf = append(buf, r.TxIDPart[:]...)
		buf = AppendVarint(buf, uint64(r.OutIndex))
		if r.TransferIndex >= 0 {
			buf = AppendVarint(buf, uint64(r.TransferIndex))
		}
	}
	return crypto.BLAKE3_256(buf)
}

// ComputeInputHash 计算输入集根哈希：BLAKE3-256( LeadHash || RestHash )。
func ComputeInputHash(lead *LeadInput, rests []RestInput) types.Hash256 {
	leadHash := ComputeLeadHash(lead)
	restHash := ComputeRestHash(rests)
	combined := append(leadHash[:], restHash[:]...)
	return crypto.BLAKE3_256(combined)
}

// ComputeOutputHash 计算输出集根哈希（二元哈希树，BLAKE3-256 内部节点，SHA3-384 叶子）。
// OutputHash = BLAKE3-256( BinaryTree( SHA3-384(output_0), SHA3-384(output_1), ... ) )
func ComputeOutputHash(outputs []Output) types.Hash256 {
	if len(outputs) == 0 {
		return types.Hash256{}
	}
	// 计算各叶子哈希
	leaves := make([][]byte, len(outputs))
	for i, o := range outputs {
		h := outputLeafHash(&o)
		leaves[i] = h[:]
	}
	return binaryTreeHash(leaves)
}

// outputLeafHash 计算单个输出的叶子哈希（SHA3-384）。
func outputLeafHash(o *Output) types.Hash384 {
	// 序列化输出的关键字段用于哈希
	buf := make([]byte, 0, 64)
	buf = AppendVarint(buf, uint64(o.Serial))
	buf = append(buf, o.Config)
	buf = append(buf, EncodeInt64(o.Amount)...)
	buf = append(buf, o.Address[:]...)
	buf = append(buf, o.LockScript...)
	buf = append(buf, o.Memo...)
	buf = append(buf, o.Creator...)
	buf = append(buf, o.Title...)
	buf = append(buf, o.Desc...)
	buf = append(buf, o.Content...)
	buf = append(buf, o.AttachID...)
	return crypto.SHA3_384(buf)
}

// binaryTreeHash 对叶子列表计算类 Merkle 二元哈希树根（BLAKE3-256 内部节点）。
func binaryTreeHash(leaves [][]byte) types.Hash256 {
	if len(leaves) == 1 {
		return crypto.BLAKE3_256(leaves[0])
	}
	mid := len(leaves) / 2
	left := binaryTreeHash(leaves[:mid])
	right := binaryTreeHash(leaves[mid:])
	combined := append(left[:], right[:]...)
	return crypto.BLAKE3_256(combined)
}

// ComputeCheckRoot 计算 CheckRoot。
// CheckRoot = SHA3-384( TxTreeRoot || UTXOFingerprint || UTCOFingerprint )
func ComputeCheckRoot(txTreeRoot types.Hash256, utxoFP, utcoFP types.Hash256) types.Hash384 {
	buf := make([]byte, 0, types.Hash256Len*3)
	buf = append(buf, txTreeRoot[:]...)
	buf = append(buf, utxoFP[:]...)
	buf = append(buf, utcoFP[:]...)
	return crypto.SHA3_384(buf)
}

// ComputeTxTreeRoot 计算区块内所有交易 ID 的哈希树根。
// 每个叶子为 3 字节序号前缀 + TxID（48 字节）。
func ComputeTxTreeRoot(txIDs []types.Hash384) types.Hash256 {
	if len(txIDs) == 0 {
		return types.Hash256{}
	}
	leaves := make([][]byte, len(txIDs))
	for i, txID := range txIDs {
		leaf := make([]byte, 3+types.Hash384Len)
		// 3 字节大端序号
		leaf[0] = byte(i >> 16)
		leaf[1] = byte(i >> 8)
		leaf[2] = byte(i)
		copy(leaf[3:], txID[:])
		leaves[i] = leaf
	}
	return binaryTreeHash(leaves)
}

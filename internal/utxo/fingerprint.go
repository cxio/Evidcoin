package utxo

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 状态指纹末端叶子（第 09 章 §3-§4、DEC-0201）。
//
// 末端叶前像为 `TxID || Count || FlagBytes`，叶哈希 = `SHA3-384(DomainTag("utxo.leaf") || 前像)`。
// 其中：
//   - Count 是该 TxID 的有效（未花费）输出数量，以 ULEB128 变长整数编码（DEC-0001）。
//   - FlagBytes 第 i 位对应输出序位 i，每字节低位优先；1=未花费，0=已花费/无效；
//     尾部未用位为 0。
//   - 输出详情（金额/接收者/脚本）属缓存集，不进入前像。

// flagOutputs 由同一 TxID 的若干 entry 计算状态位集合与有效输出数。
//
// FlagBytes 长度仅由有效（未花费）输出的最大序位决定：UTXO 集语义上只含未花费
// 成员，已花费输出已移出集合，故其序位既不置位也不参与长度，确保逆向推导
// （第 09 章 §7）可在仅有未花费集合时复现同一叶。无有效输出时返回 count=0、
// flagBytes=nil（该 TxID 不产生叶，第 09 章 §6）。
func flagOutputs(entries []Entry) (count uint64, flagBytes []byte) {
	maxSerial := -1
	for _, e := range entries {
		if e.Spent {
			continue
		}
		if int(e.OutIndex) > maxSerial {
			maxSerial = int(e.OutIndex)
		}
	}
	if maxSerial < 0 {
		return 0, nil
	}
	flagBytes = make([]byte, maxSerial/8+1)
	for _, e := range entries {
		if e.Spent {
			continue
		}
		i := e.OutIndex
		flagBytes[i/8] |= 1 << (i % 8)
		count++
	}
	return count, flagBytes
}

// leafPreimage 构造末端叶前像（不含域标签）：`TxID || varuint(Count) || FlagBytes`。
func leafPreimage(txid types.TxID, count uint64, flagBytes []byte) []byte {
	dst := make([]byte, 0, len(txid)+10+len(flagBytes))
	dst = append(dst, txid[:]...)
	dst = types.AppendVarUint(dst, count)
	dst = append(dst, flagBytes...)
	return dst
}

// leafHash 计算 UTXO 末端叶哈希：`SHA3-384(DomainTag("utxo.leaf") || leafPreimage)`。
func leafHash(txid types.TxID, count uint64, flagBytes []byte) types.Hash48 {
	return crypto.HashUTXOLeaf(leafPreimage(txid, count, flagBytes))
}

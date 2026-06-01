package utco

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 状态指纹末端叶子（第 09 章 §3-§4、DEC-0201）。
//
// 末端叶前像为 `TxID || Count || FlagBytes`，叶哈希 = `SHA3-384(DomainTag("utco.leaf") || 前像)`。
// 结构与 UTXO 同形，但域标签为 utco.leaf，与 UTXO 隔离，类型与 API 不可混用。
//   - Count 是该 TxID 的有效（未转出）输出数量，以 ULEB128 变长整数编码（DEC-0001）。
//   - FlagBytes 第 i 位对应输出序位 i，每字节低位优先；1=未转出，0=已转出/无效；
//     尾部未用位为 0。
//   - 凭信详情（持有人/创建者/标题等）属缓存集，不进入前像。
//
// 过期凭信的清理由 expiry.go 的 ExpireAt 在区块结算时完成；调用方应先清理过期项
// 再计算指纹，故此处有效性判定仅依据 Spent（未转出）。

// flagOutputs 由同一 TxID 的若干 entry 计算状态位集合与有效输出数。
//
// FlagBytes 长度仅由有效（未转出）输出的最大序位决定：UTCO 集语义上只含未转出
// 成员，已转出/已删输出不参与长度，确保逆向推导（第 09 章 §7）可在仅有未转出
// 集合时复现同一叶。无有效输出时返回 count=0、flagBytes=nil（该 TxID 删叶，§6）。
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

// leafHash 计算 UTCO 末端叶哈希：`SHA3-384(DomainTag("utco.leaf") || leafPreimage)`。
func leafHash(txid types.TxID, count uint64, flagBytes []byte) types.Hash48 {
	return crypto.HashUTCOLeaf(leafPreimage(txid, count, flagBytes))
}

package utco

import (
	"bytes"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// Resolve 将交易输入的短引用（tx.OutPoint：Year + TxIDPart 前缀 + OutIndex）
// 解析为 UTCO 集中的完整 entry（第 09 章 §1、proposal 06 §5、DEC-0101）。
//
// 碰撞策略与 UTXO 一致：同一年度内对所有有效 entry，取 TxID 以 TxIDPart 为前缀者，
// 按完整 TxID 字节升序，首个匹配即引用，不区分是否碰撞；选定 TxID 后按 OutIndex
// 定位，若无有效输出则返回缺失。
//
// 对 UTCO，有效性额外要求未过期：currentHeight 下 age > CreditMaxAge 的凭信
// 不参与解析（第 07 章 §5）。已转出（Spent）凭信亦不参与解析。
func (s *Store) Resolve(ref tx.OutPoint, currentHeight uint32) (Entry, error) {
	bestTxID, found := s.firstMatchTxID(ref.Year, ref.TxIDPart, currentHeight)
	if !found {
		return Entry{}, ErrNotFound
	}
	op := OutPoint{Year: ref.Year, TxID: bestTxID, OutIndex: ref.OutIndex}
	e, ok := s.entries[op]
	if !ok || e.Spent || e.Expired(currentHeight) {
		return Entry{}, ErrNotFound
	}
	return e, nil
}

// firstMatchTxID 在指定年度内，返回所有有效（未转出且未过期）entry 中 TxID 以
// part 为前缀、且字节序最小的那个 TxID。无匹配时 found 为 false。
func (s *Store) firstMatchTxID(year uint64, part []byte, currentHeight uint32) (best types.TxID, found bool) {
	for _, e := range s.entries {
		if e.Spent || e.Year != year || e.Expired(currentHeight) {
			continue
		}
		if !txIDHasPrefix(e.TxID, part) {
			continue
		}
		if !found || bytes.Compare(e.TxID[:], best[:]) < 0 {
			best = e.TxID
			found = true
		}
	}
	return best, found
}

// txIDHasPrefix 报告 txid 是否以 part 为前缀。part 长度超过 48 字节时恒为 false。
func txIDHasPrefix(txid types.TxID, part []byte) bool {
	if len(part) > len(txid) {
		return false
	}
	return bytes.Equal(txid[:len(part)], part)
}

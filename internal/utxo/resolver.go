package utxo

import (
	"bytes"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// Resolve 将交易输入的短引用（tx.OutPoint：Year + TxIDPart 前缀 + OutIndex）
// 解析为 UTXO 集中的完整 entry（第 09 章 §1、proposal 06 §5、DEC-0101）。
//
// 碰撞策略为固定协议规则：在同一年度内，对所有未花费 entry，取 TxID 以 TxIDPart
// 为前缀者，按完整 TxID 字节升序排列，**首个匹配即引用，不区分是否碰撞**。
// 选定 TxID 后再按 OutIndex 定位有效输出；若该序位无有效输出则返回缺失错误
// （不向后回退到下一个 TxID）。引用错误属用户责任，应预查询或延长引用字节数。
//
// 仅未花费（有效）entry 参与解析；无匹配返回 ErrNotFound。
func (s *Store) Resolve(ref tx.OutPoint) (Entry, error) {
	bestTxID, found := s.firstMatchTxID(ref.Year, ref.TxIDPart)
	if !found {
		return Entry{}, ErrNotFound
	}
	op := OutPoint{Year: ref.Year, TxID: bestTxID, OutIndex: ref.OutIndex}
	e, ok := s.entries[op]
	if !ok || e.Spent {
		return Entry{}, ErrNotFound
	}
	return e, nil
}

// firstMatchTxID 在指定年度内，返回所有未花费 entry 中 TxID 以 part 为前缀、
// 且字节序最小的那个 TxID。无匹配时 found 为 false。
func (s *Store) firstMatchTxID(year uint64, part []byte) (best types.TxID, found bool) {
	for _, e := range s.entries {
		if e.Spent || e.Year != year {
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

package utxo

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// storeKey 用交易 ID 前 8 字节 + 输出序位生成内部键（快速路由）。
type storeKey struct {
	txPrefix [8]byte
	outIndex int
}

func keyOf(op Outpoint) storeKey {
	var k storeKey
	copy(k.txPrefix[:], op.TxID[:8])
	k.outIndex = op.OutIndex
	return k
}

// UTXOStore 是内存中的 UTXO 集合。
// 实际生产中可替换为磁盘实现；此处提供内存版本供测试与共识层使用。
type UTXOStore struct {
	entries map[storeKey][]*UTXOEntry // 同前缀的多个 entry（哈希碰撞极少）
}

// NewStore 创建空的 UTXOStore。
func NewStore() *UTXOStore {
	return &UTXOStore{entries: make(map[storeKey][]*UTXOEntry)}
}

// Insert 插入一条新的未花费输出项。
func (s *UTXOStore) Insert(e *UTXOEntry) {
	k := keyOf(Outpoint{TxID: e.TxID, OutIndex: e.OutIndex})
	s.entries[k] = append(s.entries[k], e)
}

// Lookup 查找指定输出点；返回 entry 与是否找到。
func (s *UTXOStore) Lookup(op Outpoint) (*UTXOEntry, bool) {
	for _, e := range s.entries[keyOf(op)] {
		if e.TxID == op.TxID && e.OutIndex == op.OutIndex {
			return e, true
		}
	}
	return nil, false
}

// Spend 将指定输出点标记为已花费。若未找到或已花费则返回错误。
func (s *UTXOStore) Spend(op Outpoint) error {
	for _, e := range s.entries[keyOf(op)] {
		if e.TxID == op.TxID && e.OutIndex == op.OutIndex {
			if e.Spent {
				return errors.New("utxo already spent")
			}
			e.Spent = true
			return nil
		}
	}
	return errors.New("utxo not found")
}

// AllByTxID 返回指定交易的所有输出（含已花费，用于指纹叶子节点构造）。
func (s *UTXOStore) AllByTxID(txID types.Hash384) []*UTXOEntry {
	k := keyOf(Outpoint{TxID: txID})
	return s.entries[k]
}

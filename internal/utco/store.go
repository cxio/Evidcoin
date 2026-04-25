package utco

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

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

// UTCOStore 是内存中的 UTCO 集合。
type UTCOStore struct {
	entries map[storeKey][]*UTCOEntry
}

// NewStore 创建空 UTCOStore。
func NewStore() *UTCOStore {
	return &UTCOStore{entries: make(map[storeKey][]*UTCOEntry)}
}

// Insert 插入一条新的未转移凭信输出。
func (s *UTCOStore) Insert(e *UTCOEntry) {
	k := keyOf(Outpoint{TxID: e.TxID, OutIndex: e.OutIndex})
	s.entries[k] = append(s.entries[k], e)
}

// Lookup 查找指定输出点。
func (s *UTCOStore) Lookup(op Outpoint) (*UTCOEntry, bool) {
	for _, e := range s.entries[keyOf(op)] {
		if e.TxID == op.TxID && e.OutIndex == op.OutIndex {
			return e, true
		}
	}
	return nil, false
}

// Transfer 将指定凭信输出标记为已转移。
func (s *UTCOStore) Transfer(op Outpoint) error {
	for _, e := range s.entries[keyOf(op)] {
		if e.TxID == op.TxID && e.OutIndex == op.OutIndex {
			if e.Transferred {
				return errors.New("utco already transferred")
			}
			e.Transferred = true
			e.TransferCnt++
			return nil
		}
	}
	return errors.New("utco not found")
}

// AllByTxID 返回指定交易的所有凭信输出（含已转移，用于指纹叶子节点构造）。
func (s *UTCOStore) AllByTxID(txID types.Hash384) []*UTCOEntry {
	k := keyOf(Outpoint{TxID: txID})
	return s.entries[k]
}

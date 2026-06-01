package utco

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

func wrap(b []byte) []byte {
	t, _ := hashtree.BuildTree([][]byte{b})
	return t.Root()
}

func wrapN(b []byte, n int) []byte {
	for i := 0; i < n; i++ {
		b = wrap(b)
	}
	return b
}

func branch(children ...[]byte) []byte {
	t, _ := hashtree.BuildTree(children)
	return t.Root()
}

func leafBytes(txid types.TxID) []byte {
	return leafHash(txid, 1, []byte{0x01}).Bytes()
}

func txIDSet(base types.TxID, idx int, val byte) types.TxID {
	b := base
	b[idx] = val
	return b
}

func TestFingerprintRootEmpty(t *testing.T) {
	s := NewStore()
	if s.Root() != crypto.EmptyUTCORoot() {
		t.Fatalf("empty store must return EmptyUTCORoot")
	}
}

func TestFingerprintRootSingle(t *testing.T) {
	s := NewStore()
	txid := testTxID(0x10)
	_ = s.Put(Entry{Year: 2025, TxID: txid, OutIndex: 0})
	want := wrapN(leafBytes(txid), terminalLevel+1)
	if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("single root mismatch")
	}
}

func TestFingerprintRootYearAscending(t *testing.T) {
	txLow := testTxID(0x10)
	txHigh := testTxID(0x20)
	nodeLow := wrapN(leafBytes(txLow), terminalLevel)
	nodeHigh := wrapN(leafBytes(txHigh), terminalLevel)
	want := branch(nodeLow, nodeHigh)

	s := NewStore()
	_ = s.Put(Entry{Year: 2026, TxID: txHigh, OutIndex: 0})
	_ = s.Put(Entry{Year: 2025, TxID: txLow, OutIndex: 0})
	if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("year-ascending root mismatch")
	}
}

func TestFingerprintRootRoutingIndexSplits(t *testing.T) {
	base := testTxID(0x10)
	cases := []struct {
		idx   int
		level int
	}{
		{7, 1}, {11, 2}, {15, 3},
	}
	for _, c := range cases {
		a := txIDSet(base, c.idx, 0x01)
		b := txIDSet(base, c.idx, 0x02)
		depth := terminalLevel - c.level
		split := branch(wrapN(leafBytes(a), depth), wrapN(leafBytes(b), depth))
		want := wrapN(split, c.level)
		s := NewStore()
		_ = s.Put(Entry{Year: 2025, TxID: b, OutIndex: 0})
		_ = s.Put(Entry{Year: 2025, TxID: a, OutIndex: 0})
		if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
			t.Fatalf("routing index %d must split at level %d", c.idx, c.level)
		}
	}
}

func TestFingerprintRootNonRoutingSameGroup(t *testing.T) {
	base := testTxID(0x10)
	a := txIDSet(base, 8, 0x01)
	b := txIDSet(base, 8, 0x02)
	terminal := branch(leafBytes(a), leafBytes(b))
	want := wrapN(terminal, terminalLevel)
	s := NewStore()
	_ = s.Put(Entry{Year: 2025, TxID: b, OutIndex: 0})
	_ = s.Put(Entry{Year: 2025, TxID: a, OutIndex: 0})
	if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("non-routing byte must keep same terminal group")
	}
}

func TestFingerprintRootSpentExcluded(t *testing.T) {
	s := NewStore()
	keep := testTxID(0x10)
	gone := testTxID(0x50)
	_ = s.Put(Entry{Year: 2025, TxID: keep, OutIndex: 0})
	spent := Entry{Year: 2025, TxID: gone, OutIndex: 0}
	_ = s.Put(spent)
	_ = s.Spend(spent.OutPoint())
	want := wrapN(leafBytes(keep), terminalLevel+1)
	if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("spent output must not affect root")
	}
}

// UTCO 根与 UTXO 根语义隔离：相同逻辑数据因叶域标签不同而根不同。
func TestFingerprintRootDomainSeparated(t *testing.T) {
	txid := testTxID(0x10)
	s := NewStore()
	_ = s.Put(Entry{Year: 2025, TxID: txid, OutIndex: 0})
	utcoRoot := s.Root()
	// 以 UTXO 叶域标签构造等价单条目根。
	var pre []byte
	pre = append(pre, txid[:]...)
	pre = types.AppendVarUint(pre, 1)
	pre = append(pre, 0x01)
	utxoLeaf := crypto.HashUTXOLeaf(pre).Bytes()
	utxoRoot := wrapN(utxoLeaf, terminalLevel+1)
	if bytes.Equal(utcoRoot.Bytes(), utxoRoot) {
		t.Fatalf("utco/utxo state roots must be domain-separated")
	}
}

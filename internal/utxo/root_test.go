package utxo

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// wrap 对单节点做一次通用二叉树单叶归一化（tree.branch），返回 32 字节根。
func wrap(b []byte) []byte {
	t, _ := hashtree.BuildTree([][]byte{b})
	return t.Root()
}

// wrapN 连续 n 次单叶归一化。
func wrapN(b []byte, n int) []byte {
	for i := 0; i < n; i++ {
		b = wrap(b)
	}
	return b
}

// branch 以通用二叉树合并若干有序子节点。
func branch(children ...[]byte) []byte {
	t, _ := hashtree.BuildTree(children)
	return t.Root()
}

// leafBytes 返回「单输出、序位 0 有效」TxID 的末端叶哈希字节。
func leafBytes(txid types.TxID) []byte {
	return leafHash(txid, 1, []byte{0x01}).Bytes()
}

// txIDSet 基于 base 复制并覆盖指定字节，构造与 base 仅在该字节不同的 TxID。
func txIDSet(base types.TxID, idx int, val byte) types.TxID {
	b := base
	b[idx] = val
	return b
}

func TestFingerprintRootEmpty(t *testing.T) {
	s := NewStore()
	if s.Root() != crypto.EmptyUTXORoot() {
		t.Fatalf("empty store must return EmptyUTXORoot")
	}
}

// 单条目根 = 叶经五层（末端 + [15]/[11]/[7]/年度四中间层）单叶归一化。
func TestFingerprintRootSingle(t *testing.T) {
	s := NewStore()
	txid := testTxID(0x10)
	_ = s.Put(Entry{Year: 2025, TxID: txid, OutIndex: 0})
	want := wrapN(leafBytes(txid), terminalLevel+1) // 5 次
	if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("single root mismatch")
	}
}

// 顶层按年度数值升序：低年度子树在前。
func TestFingerprintRootYearAscending(t *testing.T) {
	txLow := testTxID(0x10)
	txHigh := testTxID(0x20)
	// 年度子节点：单 TxID 经 [7]/[11]/[15]/末端四中间层归一化（terminalLevel 次）。
	nodeLow := wrapN(leafBytes(txLow), terminalLevel)
	nodeHigh := wrapN(leafBytes(txHigh), terminalLevel)
	want := branch(nodeLow, nodeHigh) // 2025 在前，2026 在后

	// 乱序插入仍应得到年度升序的根。
	s := NewStore()
	_ = s.Put(Entry{Year: 2026, TxID: txHigh, OutIndex: 0})
	_ = s.Put(Entry{Year: 2025, TxID: txLow, OutIndex: 0})
	if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("year-ascending root mismatch")
	}
}

// 三级中间分层使用 0-based [7]/[11]/[15]：在路由字节处不同的两个 TxID 必须分组。
func TestFingerprintRootRoutingIndexSplits(t *testing.T) {
	base := testTxID(0x10)
	cases := []struct {
		idx   int
		level int // 分裂所在层（1=[7], 2=[11], 3=[15]）
	}{
		{7, 1}, {11, 2}, {15, 3},
	}
	for _, c := range cases {
		a := txIDSet(base, c.idx, 0x01)
		b := txIDSet(base, c.idx, 0x02) // a < b（仅该字节不同）
		// 分裂层下方的单 TxID 子节点深度 = terminalLevel - level。
		depth := terminalLevel - c.level
		split := branch(wrapN(leafBytes(a), depth), wrapN(leafBytes(b), depth))
		want := wrapN(split, c.level) // 分裂层之上各单子层再归一化 level 次
		s := NewStore()
		_ = s.Put(Entry{Year: 2025, TxID: b, OutIndex: 0})
		_ = s.Put(Entry{Year: 2025, TxID: a, OutIndex: 0})
		if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
			t.Fatalf("routing index %d must split at level %d", c.idx, c.level)
		}
	}
}

// 非路由字节（如第 9 个字节 idx=8）不同的两个 TxID 落入同一末端分组，按完整 TxID 排序。
func TestFingerprintRootNonRoutingSameGroup(t *testing.T) {
	base := testTxID(0x10)
	a := txIDSet(base, 8, 0x01)
	b := txIDSet(base, 8, 0x02) // 仅第 9 字节不同，[7]/[11]/[15] 相同 => 同末端组
	// 末端组内按完整 TxID 升序，其上有 [15]/[11]/[7]/年度四中间层。
	terminal := branch(leafBytes(a), leafBytes(b))
	want := wrapN(terminal, terminalLevel)
	s := NewStore()
	_ = s.Put(Entry{Year: 2025, TxID: b, OutIndex: 0})
	_ = s.Put(Entry{Year: 2025, TxID: a, OutIndex: 0})
	if got := s.Root(); !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("non-routing byte must keep same terminal group")
	}
}

// 已花费输出不参与状态根。
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

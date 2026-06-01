package utxo

import (
	"errors"
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// txIDWithPrefix 构造一个以 prefix 开头、其余字节填充 suffix 的 TxID，
// 用于制造短引用前缀碰撞（不同 suffix 产生不同完整 TxID，按 suffix 排序）。
func txIDWithPrefix(prefix []byte, suffix byte) types.TxID {
	var b [48]byte
	copy(b[:], prefix)
	for i := len(prefix); i < 48; i++ {
		b[i] = suffix
	}
	return types.MustTxID(b[:])
}

func commonPrefix() []byte {
	p := make([]byte, tx.MinTxIDPartLen)
	for i := range p {
		p[i] = byte(0x40 + i)
	}
	return p
}

func TestResolverFullOutpoint(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 2, 100)
	_ = s.Put(e)
	ref := tx.OutPoint{Year: 2025, TxIDPart: e.TxID.Bytes(), OutIndex: 2}
	got, err := s.Resolve(ref)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TxID != e.TxID || got.OutIndex != 2 {
		t.Fatalf("resolved wrong entry: %+v", got)
	}
}

func TestResolverPartialUnique(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	_ = s.Put(e)
	ref := tx.OutPoint{Year: 2025, TxIDPart: e.TxID.Bytes()[:tx.MinTxIDPartLen], OutIndex: 0}
	got, err := s.Resolve(ref)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TxID != e.TxID {
		t.Fatalf("resolved wrong entry")
	}
}

// TestResolverCollisionFirstMatch 验证短引用前缀碰撞时按 TxID 升序取首个匹配
// （proposal 06 §5 / DEC-0101：首个匹配即引用，不区分是否碰撞）。
func TestResolverCollisionFirstMatch(t *testing.T) {
	s := NewStore()
	prefix := commonPrefix()
	low := txIDWithPrefix(prefix, 0x01)  // 较小 TxID
	high := txIDWithPrefix(prefix, 0xFF) // 较大 TxID
	_ = s.Put(newCoinEntry(2025, high, 0, 200))
	_ = s.Put(newCoinEntry(2025, low, 0, 100))
	ref := tx.OutPoint{Year: 2025, TxIDPart: prefix, OutIndex: 0}
	got, err := s.Resolve(ref)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TxID != low {
		t.Fatalf("collision must resolve to smallest TxID; got amount=%d", got.Amount)
	}
}

func TestResolverNoMatch(t *testing.T) {
	s := NewStore()
	_ = s.Put(newCoinEntry(2025, testTxID(0x10), 0, 100))
	ref := tx.OutPoint{Year: 2025, TxIDPart: commonPrefix(), OutIndex: 0}
	if _, err := s.Resolve(ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResolverWrongYear(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	_ = s.Put(e)
	ref := tx.OutPoint{Year: 2024, TxIDPart: e.TxID.Bytes(), OutIndex: 0}
	if _, err := s.Resolve(ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong year, got %v", err)
	}
}

// 已花费（无效）项不参与有效解析：碰撞前缀下，较小但已花费的 TxID 被跳过，
// 解析落到下一个有效 TxID。
func TestResolverSpentNotResolved(t *testing.T) {
	s := NewStore()
	prefix := commonPrefix()
	low := txIDWithPrefix(prefix, 0x01)
	high := txIDWithPrefix(prefix, 0xFF)
	_ = s.Put(newCoinEntry(2025, low, 0, 100))
	_ = s.Put(newCoinEntry(2025, high, 0, 200))
	if err := s.Spend(OutPoint{Year: 2025, TxID: low, OutIndex: 0}); err != nil {
		t.Fatalf("Spend: %v", err)
	}
	ref := tx.OutPoint{Year: 2025, TxIDPart: prefix, OutIndex: 0}
	got, err := s.Resolve(ref)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TxID != high {
		t.Fatalf("spent low TxID must be skipped; resolved %v", got.TxID)
	}
}

// 引用到首个匹配 TxID 后，其指定 OutIndex 若无有效项则返回缺失（不向后回退）。
func TestResolverWrongOutIndex(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	_ = s.Put(e)
	ref := tx.OutPoint{Year: 2025, TxIDPart: e.TxID.Bytes(), OutIndex: 5}
	if _, err := s.Resolve(ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for invalid out index, got %v", err)
	}
}

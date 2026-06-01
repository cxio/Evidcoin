package utco

import (
	"errors"
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
)

// expiredHeight 返回令 CreatedHeight=created 的凭信刚好过期（age = max+1）的当前高度。
func expiredHeight(created uint32) uint32 {
	return created + tx.CreditMaxAge + 1
}

// boundaryHeight 返回令 CreatedHeight=created 的凭信处于边界（age = max，仍有效）的当前高度。
func boundaryHeight(created uint32) uint32 {
	return created + tx.CreditMaxAge
}

func TestExpireAtRemovesExpired(t *testing.T) {
	s := NewStore()
	e := newCreditEntry(2025, testTxID(0x10), 0)
	e.CreatedHeight = 1
	_ = s.Put(e)
	n := s.ExpireAt(expiredHeight(1))
	if n != 1 {
		t.Fatalf("ExpireAt removed = %d, want 1", n)
	}
	if _, err := s.Get(e.OutPoint()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired credit must be removed, got %v", err)
	}
}

func TestExpireAtBoundaryRetained(t *testing.T) {
	s := NewStore()
	e := newCreditEntry(2025, testTxID(0x10), 0)
	e.CreatedHeight = 1
	_ = s.Put(e)
	if n := s.ExpireAt(boundaryHeight(1)); n != 0 {
		t.Fatalf("boundary-equal credit must not expire, removed = %d", n)
	}
	if _, err := s.Get(e.OutPoint()); err != nil {
		t.Fatalf("boundary credit must be retained: %v", err)
	}
}

// 同 TxID 仍有其它未过期凭信时保留叶：仅删除过期输出，未过期输出保留。
func TestExpireAtPartialGroupRetained(t *testing.T) {
	s := NewStore()
	txid := testTxID(0x10)
	expired := newCreditEntry(2025, txid, 0)
	expired.CreatedHeight = 1
	fresh := newCreditEntry(2025, txid, 1)
	fresh.CreatedHeight = expiredHeight(1) // 与当前高度相同，age=0
	_ = s.Put(expired)
	_ = s.Put(fresh)
	if n := s.ExpireAt(expiredHeight(1)); n != 1 {
		t.Fatalf("ExpireAt removed = %d, want 1", n)
	}
	if _, err := s.Get(expired.OutPoint()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired output must be removed, got %v", err)
	}
	if _, err := s.Get(fresh.OutPoint()); err != nil {
		t.Fatalf("fresh output of same TxID must be retained: %v", err)
	}
}

// 同 TxID 无任何有效凭信时删除残留叶：已转出 + 过期则整组移除。
func TestExpireAtResidualLeafRemoved(t *testing.T) {
	s := NewStore()
	txid := testTxID(0x10)
	spent := newCreditEntry(2025, txid, 0)
	spent.CreatedHeight = 1
	expired := newCreditEntry(2025, txid, 1)
	expired.CreatedHeight = 1
	_ = s.Put(spent)
	_ = s.Put(expired)
	_ = s.Spend(spent.OutPoint())
	if n := s.ExpireAt(expiredHeight(1)); n != 1 {
		t.Fatalf("ExpireAt removed = %d, want 1", n)
	}
	// 整组无有效凭信，残留的已转出叶也应被清除。
	if _, err := s.Get(spent.OutPoint()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("residual spent output must be removed, got %v", err)
	}
	if _, err := s.Get(expired.OutPoint()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired output must be removed, got %v", err)
	}
}

// 同 TxID 仍有有效凭信时保留已转出残留输出（叶仍存在）。
func TestExpireAtResidualRetainedWhenValidExists(t *testing.T) {
	s := NewStore()
	txid := testTxID(0x10)
	spent := newCreditEntry(2025, txid, 0)
	spent.CreatedHeight = 1
	valid := newCreditEntry(2025, txid, 1)
	valid.CreatedHeight = 1
	_ = s.Put(spent)
	_ = s.Put(valid)
	_ = s.Spend(spent.OutPoint())
	if n := s.ExpireAt(boundaryHeight(1)); n != 0 {
		t.Fatalf("nothing expired at boundary, removed = %d", n)
	}
	if _, err := s.Get(spent.OutPoint()); err != nil {
		t.Fatalf("spent residual must be retained while valid exists: %v", err)
	}
}

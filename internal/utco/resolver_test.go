package utco

import (
	"errors"
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

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

const resolveHeight = 1000

func TestResolverFullOutpoint(t *testing.T) {
	s := NewStore()
	e := newCreditEntry(2025, testTxID(0x10), 1)
	_ = s.Put(e)
	ref := tx.OutPoint{Year: 2025, TxIDPart: e.TxID.Bytes(), OutIndex: 1}
	got, err := s.Resolve(ref, resolveHeight)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TxID != e.TxID || got.OutIndex != 1 {
		t.Fatalf("resolved wrong entry: %+v", got)
	}
}

func TestResolverPartialUnique(t *testing.T) {
	s := NewStore()
	e := newCreditEntry(2025, testTxID(0x10), 0)
	_ = s.Put(e)
	ref := tx.OutPoint{Year: 2025, TxIDPart: e.TxID.Bytes()[:tx.MinTxIDPartLen], OutIndex: 0}
	got, err := s.Resolve(ref, resolveHeight)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TxID != e.TxID {
		t.Fatalf("resolved wrong entry")
	}
}

func TestResolverCollisionFirstMatch(t *testing.T) {
	s := NewStore()
	prefix := commonPrefix()
	low := txIDWithPrefix(prefix, 0x01)
	high := txIDWithPrefix(prefix, 0xFF)
	_ = s.Put(newCreditEntry(2025, high, 0))
	_ = s.Put(newCreditEntry(2025, low, 0))
	ref := tx.OutPoint{Year: 2025, TxIDPart: prefix, OutIndex: 0}
	got, err := s.Resolve(ref, resolveHeight)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TxID != low {
		t.Fatalf("collision must resolve to smallest TxID")
	}
}

func TestResolverNoMatch(t *testing.T) {
	s := NewStore()
	_ = s.Put(newCreditEntry(2025, testTxID(0x10), 0))
	ref := tx.OutPoint{Year: 2025, TxIDPart: commonPrefix(), OutIndex: 0}
	if _, err := s.Resolve(ref, resolveHeight); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResolverSpentNotResolved(t *testing.T) {
	s := NewStore()
	prefix := commonPrefix()
	low := txIDWithPrefix(prefix, 0x01)
	high := txIDWithPrefix(prefix, 0xFF)
	_ = s.Put(newCreditEntry(2025, low, 0))
	_ = s.Put(newCreditEntry(2025, high, 0))
	_ = s.Spend(OutPoint{Year: 2025, TxID: low, OutIndex: 0})
	ref := tx.OutPoint{Year: 2025, TxIDPart: prefix, OutIndex: 0}
	got, err := s.Resolve(ref, resolveHeight)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TxID != high {
		t.Fatalf("spent low TxID must be skipped")
	}
}

// 过期 Credit 不可解析为有效输入（age > CreditMaxAge）。
func TestResolverExpiredNotResolved(t *testing.T) {
	s := NewStore()
	e := newCreditEntry(2025, testTxID(0x10), 0)
	e.CreatedHeight = 1
	_ = s.Put(e)
	ref := tx.OutPoint{Year: 2025, TxIDPart: e.TxID.Bytes(), OutIndex: 0}
	// 边界相等仍可解析。
	if _, err := s.Resolve(ref, e.CreatedHeight+tx.CreditMaxAge); err != nil {
		t.Fatalf("boundary-equal credit must resolve, got %v", err)
	}
	// 超过边界一格则过期，不可解析。
	if _, err := s.Resolve(ref, e.CreatedHeight+tx.CreditMaxAge+1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired credit must not resolve, got %v", err)
	}
}

func TestResolverWrongOutIndex(t *testing.T) {
	s := NewStore()
	e := newCreditEntry(2025, testTxID(0x10), 0)
	_ = s.Put(e)
	ref := tx.OutPoint{Year: 2025, TxIDPart: e.TxID.Bytes(), OutIndex: 9}
	if _, err := s.Resolve(ref, resolveHeight); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for invalid out index, got %v", err)
	}
}

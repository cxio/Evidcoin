package utxo

import (
	"errors"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func newCoinEntry(year uint64, txid types.TxID, outIndex uint64, amount uint64) Entry {
	return Entry{
		Year:          year,
		TxID:          txid,
		OutIndex:      outIndex,
		Amount:        types.Amount(amount),
		Receiver:      []byte("r"),
		LockScript:    []byte{0x01},
		CreatedHeight: 1,
	}
}

func TestStorePutGet(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 2, 100)
	if err := s.Put(e); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(e.OutPoint())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Amount != e.Amount || got.TxID != e.TxID || got.OutIndex != e.OutIndex {
		t.Fatalf("Get returned wrong entry: %+v", got)
	}
}

func TestStoreGetMissing(t *testing.T) {
	s := NewStore()
	_, err := s.Get(OutPoint{Year: 2025, TxID: testTxID(0x10), OutIndex: 0})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStorePutDuplicate(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	if err := s.Put(e); err != nil {
		t.Fatalf("first Put: %v", err)
	}
	if err := s.Put(e); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate on second Put, got %v", err)
	}
}

func TestStoreSpend(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	if err := s.Put(e); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Spend(e.OutPoint()); err != nil {
		t.Fatalf("Spend: %v", err)
	}
	got, err := s.Get(e.OutPoint())
	if err != nil {
		t.Fatalf("Get after spend: %v", err)
	}
	if !got.Spent {
		t.Fatalf("entry must be marked spent")
	}
}

func TestStoreSpendTwiceRejected(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	if err := s.Put(e); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Spend(e.OutPoint()); err != nil {
		t.Fatalf("first Spend: %v", err)
	}
	if err := s.Spend(e.OutPoint()); !errors.Is(err, ErrAlreadySpent) {
		t.Fatalf("expected ErrAlreadySpent on second spend, got %v", err)
	}
}

func TestStoreSpendMissing(t *testing.T) {
	s := NewStore()
	err := s.Spend(OutPoint{Year: 2025, TxID: testTxID(0x10), OutIndex: 0})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStoreValidEntries(t *testing.T) {
	s := NewStore()
	a := newCoinEntry(2025, testTxID(0x10), 0, 100)
	b := newCoinEntry(2025, testTxID(0x20), 0, 200)
	_ = s.Put(a)
	_ = s.Put(b)
	if err := s.Spend(a.OutPoint()); err != nil {
		t.Fatalf("Spend: %v", err)
	}
	valid := s.ValidEntries()
	if len(valid) != 1 {
		t.Fatalf("ValidEntries len = %d, want 1", len(valid))
	}
	if valid[0].TxID != b.TxID {
		t.Fatalf("ValidEntries returned wrong entry")
	}
}

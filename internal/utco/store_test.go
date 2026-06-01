package utco

import (
	"errors"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func newCreditEntry(year uint64, txid types.TxID, outIndex uint64) Entry {
	return Entry{
		Year:          year,
		TxID:          txid,
		OutIndex:      outIndex,
		Receiver:      []byte("recv"),
		Creator:       []byte("creator"),
		Title:         []byte("title"),
		Description:   []byte("desc"),
		AttachmentID:  []byte("att"),
		LockScript:    []byte{0x01},
		CreatedHeight: 1,
	}
}

func TestStorePutGet(t *testing.T) {
	s := NewStore()
	e := newCreditEntry(2025, testTxID(0x10), 0)
	if err := s.Put(e); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(e.OutPoint())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TxID != e.TxID || got.OutIndex != e.OutIndex {
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
	e := newCreditEntry(2025, testTxID(0x10), 0)
	_ = s.Put(e)
	if err := s.Put(e); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

// Credit 为一次性转移/消费：转移即消费旧 UTCO（Spent），不维护多次转移计数。
func TestStoreSpendOnce(t *testing.T) {
	s := NewStore()
	e := newCreditEntry(2025, testTxID(0x10), 0)
	_ = s.Put(e)
	if err := s.Spend(e.OutPoint()); err != nil {
		t.Fatalf("Spend: %v", err)
	}
	got, _ := s.Get(e.OutPoint())
	if !got.Spent {
		t.Fatalf("entry must be marked spent after transfer")
	}
	if err := s.Spend(e.OutPoint()); !errors.Is(err, ErrAlreadySpent) {
		t.Fatalf("expected ErrAlreadySpent on second transfer, got %v", err)
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
	a := newCreditEntry(2025, testTxID(0x10), 0)
	b := newCreditEntry(2025, testTxID(0x20), 0)
	_ = s.Put(a)
	_ = s.Put(b)
	_ = s.Spend(a.OutPoint())
	valid := s.ValidEntries()
	if len(valid) != 1 || valid[0].TxID != b.TxID {
		t.Fatalf("ValidEntries = %+v, want only b", valid)
	}
}

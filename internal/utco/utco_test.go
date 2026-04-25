package utco_test

import (
	"testing"

	"github.com/cxio/evidcoin/internal/utco"
	"github.com/cxio/evidcoin/pkg/types"
)

func TestUTCOStore_InsertAndLookup(t *testing.T) {
	store := utco.NewStore()
	entry := &utco.UTCOEntry{
		TxID:        types.Hash384{0x01},
		OutIndex:    0,
		TxYear:      2026,
		Address:     [32]byte{0xAA},
		Transferred: false,
	}
	store.Insert(entry)
	op := utco.Outpoint{TxID: entry.TxID, OutIndex: 0}
	got, ok := store.Lookup(op)
	if !ok || got.Address != entry.Address {
		t.Error("UTCO lookup failed")
	}
}

func TestUTCOStore_Transfer(t *testing.T) {
	store := utco.NewStore()
	entry := &utco.UTCOEntry{TxID: types.Hash384{0x02}, OutIndex: 0}
	store.Insert(entry)
	op := utco.Outpoint{TxID: entry.TxID, OutIndex: 0}

	if err := store.Transfer(op); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	got, ok := store.Lookup(op)
	if !ok || !got.Transferred {
		t.Error("UTCO should be marked as transferred")
	}
}

func TestUTCOStore_TransferNotFound(t *testing.T) {
	store := utco.NewStore()
	op := utco.Outpoint{TxID: types.Hash384{0xFF}, OutIndex: 0}
	if err := store.Transfer(op); err == nil {
		t.Error("expected error transferring non-existent entry")
	}
}

func TestUTCOStore_TransferAlreadyTransferred(t *testing.T) {
	store := utco.NewStore()
	entry := &utco.UTCOEntry{TxID: types.Hash384{0x03}, OutIndex: 0}
	store.Insert(entry)
	op := utco.Outpoint{TxID: entry.TxID, OutIndex: 0}
	_ = store.Transfer(op)
	if err := store.Transfer(op); err == nil {
		t.Error("expected error double-transferring")
	}
}

func TestUTCOFingerprint_Deterministic(t *testing.T) {
	s1 := utco.NewStore()
	s2 := utco.NewStore()
	e := &utco.UTCOEntry{TxID: types.Hash384{0xCC}, TxYear: 2026, OutIndex: 0}
	s1.Insert(e)
	s2.Insert(e)
	if utco.ComputeFingerprint(s1, 2026) != utco.ComputeFingerprint(s2, 2026) {
		t.Error("UTCO fingerprint must be deterministic")
	}
}

func TestUTCOFingerprint_ChangesAfterTransfer(t *testing.T) {
	store := utco.NewStore()
	txID := types.Hash384{0xDD}
	entry := &utco.UTCOEntry{TxID: txID, TxYear: 2026, OutIndex: 0}
	store.Insert(entry)

	fp1 := utco.ComputeFingerprint(store, 2026)
	_ = store.Transfer(utco.Outpoint{TxID: txID, OutIndex: 0})
	fp2 := utco.ComputeFingerprint(store, 2026)

	if fp1 == fp2 {
		t.Error("UTCO fingerprint must change after transfer")
	}
}

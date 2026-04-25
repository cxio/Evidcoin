// internal/utxo/utxo_test.go
package utxo_test

import (
	"testing"

	"github.com/cxio/evidcoin/internal/utxo"
	"github.com/cxio/evidcoin/pkg/types"
)

func TestUTXOStore_InsertAndLookup(t *testing.T) {
	store := utxo.NewStore()
	entry := &utxo.UTXOEntry{
		TxID:     types.Hash384{0x01},
		OutIndex: 0,
		TxYear:   2026,
		Amount:   1000,
	}
	op := utxo.Outpoint{TxID: entry.TxID, OutIndex: 0}
	store.Insert(entry)

	got, ok := store.Lookup(op)
	if !ok {
		t.Fatal("expected entry to be found")
	}
	if got.Amount != 1000 {
		t.Errorf("amount = %d, want 1000", got.Amount)
	}
}

func TestUTXOStore_Spend(t *testing.T) {
	store := utxo.NewStore()
	entry := &utxo.UTXOEntry{TxID: types.Hash384{0x02}, OutIndex: 0, Amount: 500}
	store.Insert(entry)
	op := utxo.Outpoint{TxID: entry.TxID, OutIndex: 0}

	if err := store.Spend(op); err != nil {
		t.Fatalf("Spend: %v", err)
	}
	got, ok := store.Lookup(op)
	if !ok {
		t.Fatal("spent entry should still be findable (for fingerprint)")
	}
	if !got.Spent {
		t.Error("entry should be marked as spent")
	}
}

func TestUTXOStore_SpendNotFound(t *testing.T) {
	store := utxo.NewStore()
	op := utxo.Outpoint{TxID: types.Hash384{0xFF}, OutIndex: 0}
	if err := store.Spend(op); err == nil {
		t.Error("expected error spending non-existent entry")
	}
}

func TestUTXOStore_SpendAlreadySpent(t *testing.T) {
	store := utxo.NewStore()
	entry := &utxo.UTXOEntry{TxID: types.Hash384{0x03}, OutIndex: 0}
	store.Insert(entry)
	op := utxo.Outpoint{TxID: entry.TxID, OutIndex: 0}
	_ = store.Spend(op)
	if err := store.Spend(op); err == nil {
		t.Error("expected error double-spending")
	}
}

func TestFingerprint_SingleEntry(t *testing.T) {
	store := utxo.NewStore()
	txID := types.Hash384{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		0x11, 0x12, 0x13, 0x14}
	entry := &utxo.UTXOEntry{
		TxID:     txID,
		TxYear:   2026,
		OutIndex: 0,
		Amount:   100,
	}
	store.Insert(entry)

	fp := utxo.ComputeFingerprint(store, 2026)
	if fp == (types.Hash256{}) {
		t.Error("fingerprint should not be zero")
	}
}

func TestFingerprint_Deterministic(t *testing.T) {
	store1 := utxo.NewStore()
	store2 := utxo.NewStore()
	txID := types.Hash384{0xAA}
	e := &utxo.UTXOEntry{TxID: txID, TxYear: 2026, OutIndex: 0, Amount: 50}
	store1.Insert(e)
	store2.Insert(e)

	fp1 := utxo.ComputeFingerprint(store1, 2026)
	fp2 := utxo.ComputeFingerprint(store2, 2026)
	if fp1 != fp2 {
		t.Error("fingerprint must be deterministic")
	}
}

func TestFingerprint_ChangesAfterSpend(t *testing.T) {
	store := utxo.NewStore()
	txID := types.Hash384{0xBB}
	entry := &utxo.UTXOEntry{TxID: txID, TxYear: 2026, OutIndex: 0, Amount: 200}
	store.Insert(entry)

	fp1 := utxo.ComputeFingerprint(store, 2026)
	_ = store.Spend(utxo.Outpoint{TxID: txID, OutIndex: 0})
	fp2 := utxo.ComputeFingerprint(store, 2026)

	if fp1 == fp2 {
		t.Error("fingerprint must change after spend")
	}
}

func TestComputeCheckRoot(t *testing.T) {
	var txTreeRoot types.Hash384
	var utxoFP types.Hash256
	var utcoFP types.Hash256
	// 全零输入，结果应确定且非零
	result := utxo.ComputeCheckRoot(txTreeRoot, utxoFP, utcoFP)
	if result == (types.Hash384{}) {
		t.Error("CheckRoot should not be zero")
	}
}

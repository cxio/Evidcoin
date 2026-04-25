package test

import (
	"testing"

	"github.com/cxio/evidcoin/internal/utco"
	"github.com/cxio/evidcoin/internal/utxo"
	"github.com/cxio/evidcoin/pkg/types"
)

// TestCheckRoot_ChangesWithUTXO 验证 UTXO 花费后 CheckRoot 变化。
func TestCheckRoot_ChangesWithUTXO(t *testing.T) {
	utxoStore := utxo.NewStore()
	utcoStore := utco.NewStore()
	txID := types.Hash384{0x10}

	utxoStore.Insert(&utxo.UTXOEntry{TxID: txID, OutIndex: 0, TxYear: 2026, Amount: 999})

	fp1u := utxo.ComputeRootFingerprint(utxoStore)
	fp1c := utco.ComputeRootFingerprint(utcoStore)
	cr1 := utxo.ComputeCheckRoot(types.Hash384{}, fp1u, fp1c)

	_ = utxoStore.Spend(utxo.Outpoint{TxID: txID, OutIndex: 0})

	fp2u := utxo.ComputeRootFingerprint(utxoStore)
	cr2 := utxo.ComputeCheckRoot(types.Hash384{}, fp2u, fp1c)

	if cr1 == cr2 {
		t.Error("CheckRoot must change when UTXO is spent")
	}
}

// TestCheckRoot_ChangesWithUTCO 验证 UTCO 转移后 CheckRoot 变化。
func TestCheckRoot_ChangesWithUTCO(t *testing.T) {
	utxoStore := utxo.NewStore()
	utcoStore := utco.NewStore()
	txID := types.Hash384{0x20}

	utcoStore.Insert(&utco.UTCOEntry{TxID: txID, OutIndex: 0, TxYear: 2026})

	fp1u := utxo.ComputeRootFingerprint(utxoStore)
	fp1c := utco.ComputeRootFingerprint(utcoStore)
	cr1 := utxo.ComputeCheckRoot(types.Hash384{}, fp1u, fp1c)

	_ = utcoStore.Transfer(utco.Outpoint{TxID: txID, OutIndex: 0})

	fp2c := utco.ComputeRootFingerprint(utcoStore)
	cr2 := utxo.ComputeCheckRoot(types.Hash384{}, fp1u, fp2c)

	if cr1 == cr2 {
		t.Error("CheckRoot must change when UTCO is transferred")
	}
}

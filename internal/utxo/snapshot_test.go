package utxo

import (
	"context"
	"errors"
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// testBlockID 构造一个填充字节的 48 字节区块标识，用于快照身份断言。
func testBlockID(fill byte) types.BlockID {
	b := make([]byte, 48)
	for i := range b {
		b[i] = fill
	}
	return types.MustBlockID(b)
}

// 快照须绑定高度、BlockID、链身份（ChainID + GenesisID）与拍照时刻的状态 root。
func TestSnapshotBindsIdentity(t *testing.T) {
	s := NewStore()
	_ = s.Put(newCoinEntry(2025, testTxID(0x10), 0, 100))
	gen := testBlockID(0x01)
	blk := testBlockID(0x02)

	snap := s.Snapshot("evidcoin-main", gen, blk, 42)

	if snap.ChainID != "evidcoin-main" {
		t.Fatalf("ChainID = %q, want evidcoin-main", snap.ChainID)
	}
	if snap.GenesisID != gen {
		t.Fatalf("GenesisID mismatch")
	}
	if snap.BlockID != blk {
		t.Fatalf("BlockID mismatch")
	}
	if snap.Height != 42 {
		t.Fatalf("Height = %d, want 42", snap.Height)
	}
	if snap.StateRoot != s.Root() {
		t.Fatalf("StateRoot must equal store root at snapshot time")
	}
}

// 应用失败后留下的部分变更（已花费标记）须能经 Restore 回滚。
func TestSnapshotRestoreAfterFailedApply(t *testing.T) {
	s := NewStore()
	e1 := newCoinEntry(2025, testTxID(0x10), 0, 100)
	e2 := newCoinEntry(2025, testTxID(0x20), 0, 200)
	_ = s.Put(e1)
	_ = s.Put(e2)
	snap := s.Snapshot("c", testBlockID(0x01), testBlockID(0x02), 1)
	rootBefore := s.Root()

	// 第一个输入合法消费（标记 Spent），第二个来源类别非法触发失败。
	b := Batch{Spends: []SpendRef{
		{Kind: tx.InputCoin, Ref: fullRef(2025, e1.TxID, 0)},
		{Kind: tx.InputCredit, Ref: fullRef(2025, e2.TxID, 0)},
	}}
	if err := Apply(context.Background(), s, nil, b); err == nil {
		t.Fatalf("expected apply failure on invalid input kind")
	}
	if got, _ := s.Get(e1.OutPoint()); !got.Spent {
		t.Fatalf("precondition: e1 should be spent after partial apply")
	}

	s.Restore(snap)

	if got, _ := s.Get(e1.OutPoint()); got.Spent {
		t.Fatalf("restore must un-spend e1")
	}
	if s.Root() != rootBefore {
		t.Fatalf("restore must recover pre-batch root")
	}
}

// 回滚须移除批次新插入的输出，且不影响批次前已有状态。
func TestSnapshotRestoreRemovesBatchInserts(t *testing.T) {
	s := NewStore()
	e1 := newCoinEntry(2025, testTxID(0x10), 0, 100)
	_ = s.Put(e1)
	snap := s.Snapshot("c", testBlockID(0x01), testBlockID(0x02), 1)

	out := coinOutput(2025, testTxID(0x30), 0, 500)
	if err := Apply(context.Background(), s, nil, Batch{Outputs: []NewOutput{out}}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	inserted := OutPoint{Year: 2025, TxID: testTxID(0x30), OutIndex: 0}
	if _, err := s.Get(inserted); err != nil {
		t.Fatalf("precondition: inserted output must be present: %v", err)
	}

	s.Restore(snap)

	if _, err := s.Get(inserted); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restore must remove batch-inserted output, got %v", err)
	}
	if got, err := s.Get(e1.OutPoint()); err != nil || got.Spent {
		t.Fatalf("pre-batch entry must remain unspent: err=%v entry=%+v", err, got)
	}
}

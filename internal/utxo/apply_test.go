package utxo

import (
	"context"
	"errors"
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// stubVerifier 是测试用脚本验证器替身：可配置返回错误并记录调用次数。
type stubVerifier struct {
	err   error
	calls int
}

func (v *stubVerifier) VerifyCoinSpend(_ context.Context, _ Entry, _ []byte) error {
	v.calls++
	return v.err
}

func fullRef(year uint64, txid types.TxID, outIndex uint64) tx.OutPoint {
	return tx.OutPoint{Year: year, TxIDPart: txid.Bytes(), OutIndex: outIndex}
}

func coinOutput(year uint64, txid types.TxID, serial uint32, amount uint64) NewOutput {
	return NewOutput{
		Year:   year,
		TxID:   txid,
		Height: 10,
		Output: tx.Output{Serial: serial, Type: tx.TypeCoin, LockScript: []byte{0x01}},
		Coin:   tx.Coin{Amount: types.Amount(amount), Receiver: []byte("r")},
	}
}

func TestApplyConsumesInput(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	_ = s.Put(e)
	v := &stubVerifier{}
	b := Batch{Spends: []SpendRef{{Kind: tx.InputCoin, Ref: fullRef(2025, e.TxID, 0)}}}
	if err := Apply(context.Background(), s, v, b); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, _ := s.Get(e.OutPoint())
	if !got.Spent {
		t.Fatalf("consumed input must be marked spent")
	}
	if v.calls != 1 {
		t.Fatalf("verifier calls = %d, want 1", v.calls)
	}
}

func TestApplyScriptFailureRejected(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	_ = s.Put(e)
	verifyErr := errors.New("bad script")
	v := &stubVerifier{err: verifyErr}
	b := Batch{Spends: []SpendRef{{Kind: tx.InputCoin, Ref: fullRef(2025, e.TxID, 0)}}}
	if err := Apply(context.Background(), s, v, b); !errors.Is(err, verifyErr) {
		t.Fatalf("expected script error, got %v", err)
	}
	got, _ := s.Get(e.OutPoint())
	if got.Spent {
		t.Fatalf("entry must not be spent when script verification fails")
	}
}

func TestApplyDoubleSpendRejected(t *testing.T) {
	s := NewStore()
	e := newCoinEntry(2025, testTxID(0x10), 0, 100)
	_ = s.Put(e)
	b := Batch{Spends: []SpendRef{
		{Kind: tx.InputCoin, Ref: fullRef(2025, e.TxID, 0)},
		{Kind: tx.InputCoin, Ref: fullRef(2025, e.TxID, 0)},
	}}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); err == nil {
		t.Fatalf("expected rejection for double spend within batch")
	}
}

func TestApplyInsertsCoinOutput(t *testing.T) {
	s := NewStore()
	out := coinOutput(2025, testTxID(0x30), 0, 500)
	b := Batch{Outputs: []NewOutput{out}}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := s.Get(OutPoint{Year: 2025, TxID: testTxID(0x30), OutIndex: 0})
	if err != nil {
		t.Fatalf("coin output must enter UTXO: %v", err)
	}
	if got.Amount != types.Amount(500) {
		t.Fatalf("inserted amount = %d, want 500", got.Amount)
	}
}

func TestApplyNonCoinOutputSkipped(t *testing.T) {
	s := NewStore()
	out := NewOutput{
		Year:   2025,
		TxID:   testTxID(0x30),
		Height: 10,
		Output: tx.Output{Serial: 0, Type: tx.TypeProof, LockScript: []byte{0x01}},
	}
	b := Batch{Outputs: []NewOutput{out}}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := s.Get(OutPoint{Year: 2025, TxID: testTxID(0x30), OutIndex: 0}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-coin output must not enter UTXO, got %v", err)
	}
}

func TestApplyRejectsNonCoinInput(t *testing.T) {
	for _, kind := range []tx.InputKind{tx.InputCredit, tx.InputProof} {
		s := NewStore()
		e := newCoinEntry(2025, testTxID(0x10), 0, 100)
		_ = s.Put(e)
		b := Batch{Spends: []SpendRef{{Kind: kind, Ref: fullRef(2025, e.TxID, 0)}}}
		if err := Apply(context.Background(), s, &stubVerifier{}, b); !errors.Is(err, ErrInputKindInvalid) {
			t.Fatalf("kind %d: expected ErrInputKindInvalid, got %v", kind, err)
		}
	}
}

// 同一区块 A 输出被 B 输入引用必须拒绝：输入只能引用已确认历史区块的 UTXO，
// 不能引用同批次新产生的输出（无论 A 在 B 之前还是之后）。
func TestApplySameBlockReferenceRejected(t *testing.T) {
	sameBlockTx := testTxID(0x30)
	out := coinOutput(2025, sameBlockTx, 0, 500)
	spend := SpendRef{Kind: tx.InputCoin, Ref: fullRef(2025, sameBlockTx, 0)}

	// 输出在前、引用在后。
	s := NewStore()
	b := Batch{Outputs: []NewOutput{out}, Spends: []SpendRef{spend}}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); !errors.Is(err, ErrNotFound) {
		t.Fatalf("same-block reference must be rejected, got %v", err)
	}
}

package utco

import (
	"context"
	"errors"
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

type stubVerifier struct {
	err   error
	calls int
}

func (v *stubVerifier) VerifyCreditTransfer(_ context.Context, _ Entry, _ []byte) error {
	v.calls++
	return v.err
}

func fullRef(year uint64, txid types.TxID, outIndex uint64) tx.OutPoint {
	return tx.OutPoint{Year: year, TxIDPart: txid.Bytes(), OutIndex: outIndex}
}

// creditCreation 构造一个新建凭信输出。
func creditCreation(year uint64, txid types.TxID, serial uint32, receiver string) NewOutput {
	return NewOutput{
		Year:   year,
		TxID:   txid,
		Height: 10,
		Output: tx.Output{Serial: serial, Type: tx.TypeCredit, LockScript: []byte{0x01}},
		Credit: tx.Credit{
			Receiver:     []byte(receiver),
			Creator:      []byte("creator"),
			Title:        []byte("title"),
			Description:  []byte("desc"),
			AttachmentID: []byte("att"),
		},
	}
}

const applyHeight = 100

func TestApplyCreatesCredit(t *testing.T) {
	s := NewStore()
	b := Batch{CurrentHeight: applyHeight, Creations: []NewOutput{creditCreation(2025, testTxID(0x30), 0, "alice")}}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := s.Get(OutPoint{Year: 2025, TxID: testTxID(0x30), OutIndex: 0})
	if err != nil {
		t.Fatalf("new credit must enter UTCO: %v", err)
	}
	if string(got.Receiver) != "alice" {
		t.Fatalf("receiver = %q, want alice", got.Receiver)
	}
}

func TestApplyTransferConsumesAndInserts(t *testing.T) {
	s := NewStore()
	old := newCreditEntry(2025, testTxID(0x10), 0)
	_ = s.Put(old)
	v := &stubVerifier{}
	newOut := creditCreation(2026, testTxID(0x50), 0, "bob")
	b := Batch{
		CurrentHeight: applyHeight,
		Transfers:     []Transfer{{Kind: tx.InputCredit, Ref: fullRef(2025, old.TxID, 0), NewOutput: &newOut}},
	}
	if err := Apply(context.Background(), s, v, b); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	oldGot, _ := s.Get(old.OutPoint())
	if !oldGot.Spent {
		t.Fatalf("transferred credit must consume old UTCO")
	}
	newGot, err := s.Get(OutPoint{Year: 2026, TxID: testTxID(0x50), OutIndex: 0})
	if err != nil {
		t.Fatalf("transfer must insert new UTCO: %v", err)
	}
	if string(newGot.Receiver) != "bob" {
		t.Fatalf("new holder = %q, want bob", newGot.Receiver)
	}
	if v.calls != 1 {
		t.Fatalf("verifier calls = %d, want 1", v.calls)
	}
}

func TestApplyTransferDestroys(t *testing.T) {
	// Transfer.NewOutput == nil 视为凭信销毁：消费旧 UTCO，不产生新 UTCO（第 07 章 §5）。
	s := NewStore()
	old := newCreditEntry(2025, testTxID(0x10), 0)
	_ = s.Put(old)
	b := Batch{
		CurrentHeight: applyHeight,
		Transfers:     []Transfer{{Kind: tx.InputCredit, Ref: fullRef(2025, old.TxID, 0), NewOutput: nil}},
	}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); err != nil {
		t.Fatalf("Apply destroy: %v", err)
	}
	oldGot, _ := s.Get(old.OutPoint())
	if !oldGot.Spent {
		t.Fatalf("destroyed credit must be marked spent")
	}
	// 不应产生任何新 UTCO。
	if _, err := s.Get(OutPoint{Year: 2026, TxID: testTxID(0x50), OutIndex: 0}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("destroy must not insert new UTCO, got %v", err)
	}
}

func TestApplyTransferImmutableChangeAllowed(t *testing.T) {
	// 转移后新输出的字段（Creator/Title 等）可自由变更——约束由脚本决定，状态层不限制（第 07 章 §5）。
	s := NewStore()
	old := newCreditEntry(2025, testTxID(0x10), 0)
	_ = s.Put(old)
	newOut := creditCreation(2026, testTxID(0x50), 0, "bob")
	newOut.Credit.Creator = []byte("new-creator")
	newOut.Credit.Title = []byte("new-title")
	b := Batch{
		CurrentHeight: applyHeight,
		Transfers:     []Transfer{{Kind: tx.InputCredit, Ref: fullRef(2025, old.TxID, 0), NewOutput: &newOut}},
	}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); err != nil {
		t.Fatalf("Apply must allow changed credit fields, got %v", err)
	}
	newGot, err := s.Get(OutPoint{Year: 2026, TxID: testTxID(0x50), OutIndex: 0})
	if err != nil {
		t.Fatalf("new UTCO must be inserted: %v", err)
	}
	if string(newGot.Creator) != "new-creator" {
		t.Fatalf("creator = %q, want new-creator", newGot.Creator)
	}
}

func TestApplyTransferScriptFailure(t *testing.T) {
	s := NewStore()
	old := newCreditEntry(2025, testTxID(0x10), 0)
	_ = s.Put(old)
	verifyErr := errors.New("bad sig")
	newOut := creditCreation(2026, testTxID(0x50), 0, "bob")
	b := Batch{
		CurrentHeight: applyHeight,
		Transfers:     []Transfer{{Kind: tx.InputCredit, Ref: fullRef(2025, old.TxID, 0), NewOutput: &newOut}},
	}
	if err := Apply(context.Background(), s, &stubVerifier{err: verifyErr}, b); !errors.Is(err, verifyErr) {
		t.Fatalf("expected script error, got %v", err)
	}
	oldGot, _ := s.Get(old.OutPoint())
	if oldGot.Spent {
		t.Fatalf("old credit must not be consumed when script fails")
	}
}

func TestApplyRejectsNonCreditInput(t *testing.T) {
	for _, kind := range []tx.InputKind{tx.InputCoin, tx.InputProof} {
		s := NewStore()
		old := newCreditEntry(2025, testTxID(0x10), 0)
		_ = s.Put(old)
		newOut := creditCreation(2026, testTxID(0x50), 0, "bob")
		b := Batch{
			CurrentHeight: applyHeight,
			Transfers:     []Transfer{{Kind: kind, Ref: fullRef(2025, old.TxID, 0), NewOutput: &newOut}},
		}
		if err := Apply(context.Background(), s, &stubVerifier{}, b); !errors.Is(err, ErrInputKindInvalid) {
			t.Fatalf("kind %d: expected ErrInputKindInvalid, got %v", kind, err)
		}
	}
}

// TestApplyNonCreditCreationSkipped 校验存证/币金输出不进入 UTCO（只有凭信进入）。
func TestApplyNonCreditCreationSkipped(t *testing.T) {
	s := NewStore()
	// 存证类（TypeProof）输出不应进入 UTCO。
	out := NewOutput{
		Year:   2025,
		TxID:   testTxID(0x30),
		Height: 10,
		Output: tx.Output{Serial: 0, Type: tx.TypeProof, LockScript: []byte{0x01}},
	}
	b := Batch{CurrentHeight: applyHeight, Creations: []NewOutput{out}}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := s.Get(OutPoint{Year: 2025, TxID: testTxID(0x30), OutIndex: 0}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("proof output must not enter UTCO, got %v", err)
	}
}

func TestApplySameBlockReferenceRejected(t *testing.T) {
	sameBlockTx := testTxID(0x30)
	creation := creditCreation(2025, sameBlockTx, 0, "alice")
	newOut := creditCreation(2026, testTxID(0x50), 0, "alice")
	transfer := Transfer{Kind: tx.InputCredit, Ref: fullRef(2025, sameBlockTx, 0), NewOutput: &newOut}
	s := NewStore()
	b := Batch{CurrentHeight: applyHeight, Creations: []NewOutput{creation}, Transfers: []Transfer{transfer}}
	if err := Apply(context.Background(), s, &stubVerifier{}, b); !errors.Is(err, ErrNotFound) {
		t.Fatalf("same-block reference must be rejected, got %v", err)
	}
}

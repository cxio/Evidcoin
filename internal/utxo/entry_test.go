package utxo

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// testTxID 构造一个测试用 TxID：首字节为 fill，其余递增填充，便于区分与排序。
func testTxID(fill byte) types.TxID {
	var b [48]byte
	for i := range b {
		b[i] = fill + byte(i)
	}
	return types.MustTxID(b[:])
}

func TestEntryOutPoint(t *testing.T) {
	e := Entry{
		Year:          2025,
		TxID:          testTxID(0x10),
		OutIndex:      3,
		Amount:        types.Amount(500),
		Receiver:      []byte("receiver-pkh"),
		LockScript:    []byte{0x01, 0x02},
		CreatedHeight: 12345,
	}
	op := e.OutPoint()
	if op.Year != 2025 || op.OutIndex != 3 {
		t.Fatalf("OutPoint year/index mismatch: %+v", op)
	}
	if op.TxID != e.TxID {
		t.Fatalf("OutPoint TxID mismatch")
	}
	if e.Spent {
		t.Fatalf("new entry must default to unspent")
	}
}

func TestEntryFieldsRetained(t *testing.T) {
	e := Entry{
		Year:          1,
		TxID:          testTxID(0x20),
		OutIndex:      0,
		Amount:        types.Amount(99),
		Receiver:      []byte("r"),
		LockScript:    []byte{0xAA},
		CreatedHeight: 7,
	}
	if e.Amount != types.Amount(99) {
		t.Errorf("amount = %d, want 99", e.Amount)
	}
	if !bytes.Equal(e.Receiver, []byte("r")) {
		t.Errorf("receiver mismatch")
	}
	if !bytes.Equal(e.LockScript, []byte{0xAA}) {
		t.Errorf("lock script mismatch")
	}
	if e.CreatedHeight != 7 {
		t.Errorf("created height = %d, want 7", e.CreatedHeight)
	}
}

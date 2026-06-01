package utco

import (
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// testTxID 构造一个测试用 TxID：首字节为 fill，其余递增填充。
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
		OutIndex:      1,
		Receiver:      []byte("recv"),
		Creator:       []byte("creator"),
		Title:         []byte("title"),
		Description:   []byte("desc"),
		AttachmentID:  []byte("att"),
		LockScript:    []byte{0x01},
		CreatedHeight: 50,
	}
	op := e.OutPoint()
	if op.Year != 2025 || op.OutIndex != 1 || op.TxID != e.TxID {
		t.Fatalf("OutPoint mismatch: %+v", op)
	}
	if e.Spent {
		t.Fatalf("new entry must default to not spent")
	}
}

func TestEntryExpiry(t *testing.T) {
	const created = 1000
	cases := []struct {
		name          string
		currentHeight uint32
		wantExpired   bool
	}{
		{"young", created + 10, false},
		{"boundary equal", created + tx.CreditMaxAge, false},
		{"one below boundary", created + tx.CreditMaxAge - 1, false},
		{"one above boundary", created + tx.CreditMaxAge + 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := Entry{CreatedHeight: created}
			if got := e.Expired(c.currentHeight); got != c.wantExpired {
				t.Fatalf("Expired(%d) = %v, want %v (age=%d, max=%d)",
					c.currentHeight, got, c.wantExpired, e.Age(c.currentHeight), tx.CreditMaxAge)
			}
		})
	}
}

package utxo

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// validEntry 构造一条有效（未花费）UTXO 记录，仅设定指纹相关字段。
func validEntry(txid types.TxID, outIndex uint64) Entry {
	return Entry{TxID: txid, OutIndex: outIndex}
}

func TestFingerprintLeafFlagBits(t *testing.T) {
	txid := testTxID(0x10)
	// 序位 0 与 2 有效，序位 1 缺席（无效）。低位优先：bit0|bit2 = 0x05。
	entries := []Entry{validEntry(txid, 0), validEntry(txid, 2)}
	count, flags := flagOutputs(entries)
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if len(flags) != 1 || flags[0] != 0x05 {
		t.Fatalf("flags = % x, want [05]", flags)
	}
}

func TestFingerprintLeafSpentBitCleared(t *testing.T) {
	txid := testTxID(0x10)
	spent := validEntry(txid, 1)
	spent.Spent = true
	entries := []Entry{validEntry(txid, 0), spent}
	count, flags := flagOutputs(entries)
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	// 已花费序位 1 不计入长度也不置位：仅 bit0 => 0x01。
	if len(flags) != 1 || flags[0] != 0x01 {
		t.Fatalf("flags = % x, want [01]", flags)
	}
}

// 已花费的高序位输出不参与长度计算（UTXO 集只含未花费成员，逆向推导可复现）。
func TestFingerprintLeafLengthFromValidOnly(t *testing.T) {
	txid := testTxID(0x10)
	highSpent := validEntry(txid, 8)
	highSpent.Spent = true
	entries := []Entry{validEntry(txid, 0), highSpent}
	_, flags := flagOutputs(entries)
	if len(flags) != 1 || flags[0] != 0x01 {
		t.Fatalf("flags = % x, want [01] (spent high serial must not extend length)", flags)
	}
}

func TestFingerprintLeafTrailingBitsZero(t *testing.T) {
	txid := testTxID(0x10)
	// 仅序位 9 有效：需 2 字节，bit9 => 第二字节 0x02，首字节 0x00。
	entries := []Entry{validEntry(txid, 9)}
	_, flags := flagOutputs(entries)
	if len(flags) != 2 || flags[0] != 0x00 || flags[1] != 0x02 {
		t.Fatalf("flags = % x, want [00 02]", flags)
	}
}

func TestFingerprintLeafPreimageAndHash(t *testing.T) {
	txid := testTxID(0x10)
	entries := []Entry{validEntry(txid, 0), validEntry(txid, 2)}
	count, flags := flagOutputs(entries)
	// 期望前像（不含域标签）：TxID || varuint(count) || FlagBytes。
	var want []byte
	want = append(want, txid[:]...)
	want = types.AppendVarUint(want, count)
	want = append(want, flags...)
	if got := leafPreimage(txid, count, flags); !bytes.Equal(got, want) {
		t.Fatalf("preimage = % x, want % x", got, want)
	}
	// 叶哈希 = SHA3-384(DomainTag("utxo.leaf") || preimage)。
	wantHash := crypto.HashUTXOLeaf(want)
	if got := leafHash(txid, count, flags); got != wantHash {
		t.Fatalf("leaf hash mismatch")
	}
}

func TestFingerprintLeafCountChangesHash(t *testing.T) {
	txid := testTxID(0x10)
	base := leafHash(txid, 2, []byte{0x05})
	if leafHash(txid, 3, []byte{0x05}) == base {
		t.Fatalf("changing count must change leaf hash")
	}
	if leafHash(txid, 2, []byte{0x07}) == base {
		t.Fatalf("changing flag bits must change leaf hash")
	}
}

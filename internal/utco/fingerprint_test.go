package utco

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// validEntry 构造一条有效（未转出）UTCO 记录，仅设定指纹相关字段。
func validEntry(txid types.TxID, outIndex uint64) Entry {
	return Entry{TxID: txid, OutIndex: outIndex}
}

func TestFingerprintLeafFlagBits(t *testing.T) {
	txid := testTxID(0x10)
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
	if len(flags) != 1 || flags[0] != 0x01 {
		t.Fatalf("flags = % x, want [01]", flags)
	}
}

func TestFingerprintLeafLengthFromValidOnly(t *testing.T) {
	txid := testTxID(0x10)
	highSpent := validEntry(txid, 8)
	highSpent.Spent = true
	entries := []Entry{validEntry(txid, 0), highSpent}
	_, flags := flagOutputs(entries)
	if len(flags) != 1 || flags[0] != 0x01 {
		t.Fatalf("flags = % x, want [01] (transferred high serial must not extend length)", flags)
	}
}

func TestFingerprintLeafTrailingBitsZero(t *testing.T) {
	txid := testTxID(0x10)
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
	var want []byte
	want = append(want, txid[:]...)
	want = types.AppendVarUint(want, count)
	want = append(want, flags...)
	if got := leafPreimage(txid, count, flags); !bytes.Equal(got, want) {
		t.Fatalf("preimage = % x, want % x", got, want)
	}
	// UTCO 域标签隔离：叶哈希 = SHA3-384(DomainTag("utco.leaf") || preimage)。
	wantHash := crypto.HashUTCOLeaf(want)
	if got := leafHash(txid, count, flags); got != wantHash {
		t.Fatalf("leaf hash mismatch")
	}
}

// UTCO 与 UTXO 同前像但域标签不同，叶哈希必须隔离。
func TestFingerprintLeafDomainSeparated(t *testing.T) {
	txid := testTxID(0x10)
	var pre []byte
	pre = append(pre, txid[:]...)
	pre = types.AppendVarUint(pre, 1)
	pre = append(pre, 0x01)
	if crypto.HashUTCOLeaf(pre) == crypto.HashUTXOLeaf(pre) {
		t.Fatalf("utco/utxo leaf domains must differ")
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

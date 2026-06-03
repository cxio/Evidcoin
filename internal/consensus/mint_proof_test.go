package consensus

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func sampleMintProof() MintProof {
	var mh types.MintHash
	copy(mh[:], bytes.Repeat([]byte{0x7A}, 32))
	return MintProof{
		TxHeight:   123456,
		TxID:       types.MustTxID(bytes.Repeat([]byte{0x11}, 48)),
		Nonce:      0xDEADBEEFCAFE0001,
		Solution:   []byte{0x0A, 0x0B, 0x0C, 0x0D},
		MintPubKey: []byte{0xDE, 0xAD, 0xBE, 0xEF},
		MintHash:   mh,
		Signature:  []byte{0x01, 0x02, 0x03, 0x04, 0x05},
	}
}

// TestMintProofRoundTrip 断言编码后解码可还原所有七字段。
func TestMintProofRoundTrip(t *testing.T) {
	p := sampleMintProof()
	enc := p.CanonicalBytes()
	got, n, err := ReadMintProof(enc)
	if err != nil {
		t.Fatalf("ReadMintProof error: %v", err)
	}
	if n != len(enc) {
		t.Fatalf("consumed %d bytes, want %d", n, len(enc))
	}
	if got.TxHeight != p.TxHeight {
		t.Errorf("TxHeight = %d, want %d", got.TxHeight, p.TxHeight)
	}
	if got.TxID != p.TxID {
		t.Errorf("TxID mismatch")
	}
	if got.Nonce != p.Nonce {
		t.Errorf("Nonce = %d, want %d", got.Nonce, p.Nonce)
	}
	if !bytes.Equal(got.Solution, p.Solution) {
		t.Errorf("Solution mismatch")
	}
	if !bytes.Equal(got.MintPubKey, p.MintPubKey) {
		t.Errorf("MintPubKey mismatch")
	}
	if got.MintHash != p.MintHash {
		t.Errorf("MintHash mismatch")
	}
	if !bytes.Equal(got.Signature, p.Signature) {
		t.Errorf("Signature mismatch")
	}
}

// TestMintProofFieldOrder 断言七字段冻结顺序：
// TxHeight(u32 BE) || TxID[48] || Nonce(u64 BE) || varint(len)||Solution ||
// varint(len)||MintPubKey || MintHash[32] || varint(len)||Signature。
func TestMintProofFieldOrder(t *testing.T) {
	p := sampleMintProof()
	got := p.CanonicalBytes()

	var want []byte
	want = types.AppendUint32BE(want, p.TxHeight)
	want = append(want, p.TxID.Bytes()...)
	want = types.AppendUint64BE(want, p.Nonce)
	want = types.AppendBytes(want, p.Solution)
	want = types.AppendBytes(want, p.MintPubKey)
	want = append(want, p.MintHash.Bytes()...)
	want = types.AppendBytes(want, p.Signature)

	if !bytes.Equal(got, want) {
		t.Fatalf("encoding = %x, want %x", got, want)
	}
}

// TestReadMintProofTooShort 断言截断输入返回错误。
func TestReadMintProofTooShort(t *testing.T) {
	p := sampleMintProof()
	enc := p.CanonicalBytes()
	for _, cut := range []int{0, 1, 4, 10, 52} {
		if cut >= len(enc) {
			continue
		}
		if _, _, err := ReadMintProof(enc[:cut]); err == nil {
			t.Errorf("ReadMintProof(truncated to %d) expected error", cut)
		}
	}
}

// TestReadMintProofTrailing 断言尾随多余字节被拒绝。
func TestReadMintProofTrailing(t *testing.T) {
	p := sampleMintProof()
	enc := append(p.CanonicalBytes(), 0xFF)
	if _, _, err := ReadMintProof(enc); err != ErrMintProofTrailing {
		t.Fatalf("expected ErrMintProofTrailing, got %v", err)
	}
}


package validation

import (
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// TestMintRequestFields 检验 MintRequest 字段可正确赋值读取。
func TestMintRequestFields(t *testing.T) {
	req := MintRequest{
		MintProofBytes: []byte{0x01, 0x02},
		Signature:      []byte{0xAA},
	}
	if len(req.MintProofBytes) != 2 {
		t.Errorf("MintProofBytes len: got %d, want 2", len(req.MintProofBytes))
	}
	if len(req.Signature) != 1 {
		t.Errorf("Signature len: got %d, want 1", len(req.Signature))
	}
}

// TestMintInfoResponseFields 检验 MintInfoResponse 字段可正确赋值读取。
func TestMintInfoResponseFields(t *testing.T) {
	r := MintInfoResponse{
		TxFeeTotal:        100,
		GroupRewardAddr:   []byte{0xA0},
		ServiceRewardAddr: []byte{0xB0},
		MintAmount:        500,
		AwardSlots:        [18]byte{0x01},
	}
	if r.TxFeeTotal != 100 {
		t.Errorf("TxFeeTotal: got %d, want 100", r.TxFeeTotal)
	}
	if r.MintAmount != 500 {
		t.Errorf("MintAmount: got %d, want 500", r.MintAmount)
	}
	if r.AwardSlots[0] != 0x01 {
		t.Errorf("AwardSlots[0]: got %02x, want 01", r.AwardSlots[0])
	}
}

// TestCoinbaseSubmissionFields 检验 CoinbaseSubmission 字段可正确赋值读取。
func TestCoinbaseSubmissionFields(t *testing.T) {
	sub := CoinbaseSubmission{
		CoinbaseHeader: tx.CoinbaseHeader{
			BlockHeight: 1,
			Minter:      []byte{0x01},
			BurnCoin:    10,
		},
		MinterSig: []byte{0xCC},
	}
	if sub.CoinbaseHeader.BlockHeight != 1 {
		t.Errorf("CoinbaseHeader.BlockHeight: got %d, want 1", sub.CoinbaseHeader.BlockHeight)
	}
	if len(sub.MinterSig) != 1 {
		t.Errorf("MinterSig len: got %d, want 1", len(sub.MinterSig))
	}
}

// TestInclusionResponseFields 检验 InclusionResponse 字段可正确赋值读取。
func TestInclusionResponseFields(t *testing.T) {
	root := []byte{0x01, 0x02, 0x03}
	r := InclusionResponse{
		CoinbaseMerklePath: hashtree.Proof{
			LeafHash: []byte{0xAB},
			Siblings: nil,
			Root:     root,
		},
		TreeRoot: root,
		UTXORoot: types.TreeHash{0x11},
		UTCORoot: types.TreeHash{0x22},
	}
	if len(r.TreeRoot) != 3 {
		t.Errorf("TreeRoot len: got %d, want 3", len(r.TreeRoot))
	}
	if r.UTXORoot[0] != 0x11 {
		t.Errorf("UTXORoot[0]: got %02x, want 11", r.UTXORoot[0])
	}
	if r.UTCORoot[0] != 0x22 {
		t.Errorf("UTCORoot[0]: got %02x, want 22", r.UTCORoot[0])
	}
}

// TestBlockSignatureFields 检验 BlockSignature 字段可正确赋值读取。
func TestBlockSignatureFields(t *testing.T) {
	txID := types.TxID{0xBB}
	sig := BlockSignature{
		TxID:         txID,
		CheckRootSig: []byte{0xDD, 0xEE},
	}
	if sig.TxID != txID {
		t.Errorf("BlockSignature.TxID mismatch")
	}
	if len(sig.CheckRootSig) != 2 {
		t.Errorf("CheckRootSig len: got %d, want 2", len(sig.CheckRootSig))
	}
}

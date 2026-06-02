package services

import (
	"testing"

	"github.com/cxio/evidcoin/internal/validation"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// TestBlockqs 验证 Blockqs 6 类响应数据 profile（第 15 章 §3，DEC-0603）。
func TestBlockqs(t *testing.T) {
	t.Run("state_kind_values", func(t *testing.T) {
		// UTXO 为 1，UTCO 为 2；两值互不相同。
		if StateKindUTXO != 1 {
			t.Errorf("StateKindUTXO = %d, want 1", StateKindUTXO)
		}
		if StateKindUTCO != 2 {
			t.Errorf("StateKindUTCO = %d, want 2", StateKindUTCO)
		}
		if StateKindUTXO == StateKindUTCO {
			t.Error("StateKindUTXO and StateKindUTCO must be distinct")
		}
	})

	t.Run("tx_lookup_response_fields", func(t *testing.T) {
		// TxLookupResponse 携带：年度、TxID、完整交易数据、区块高度、区块内序位。
		txid := types.MustTxID(make([]byte, 48))
		resp := TxLookupResponse{
			Year:        2025,
			TxID:        txid,
			TxData:      []byte{0x01, 0x02},
			BlockHeight: 1000,
			TxIndex:     3,
		}
		if resp.Year != 2025 {
			t.Errorf("Year = %d, want 2025", resp.Year)
		}
		if resp.TxID != txid {
			t.Error("TxID mismatch")
		}
		if len(resp.TxData) != 2 {
			t.Errorf("TxData len = %d, want 2", len(resp.TxData))
		}
		if resp.BlockHeight != 1000 {
			t.Errorf("BlockHeight = %d, want 1000", resp.BlockHeight)
		}
		if resp.TxIndex != 3 {
			t.Errorf("TxIndex = %d, want 3", resp.TxIndex)
		}
	})

	t.Run("tx_proof_response_fields", func(t *testing.T) {
		// TxProofResponse 携带：TxID、区块高度、Merkle 验证路径。
		txid := types.MustTxID(make([]byte, 48))
		proof := hashtree.Proof{
			LeafHash: []byte{0xAA},
			Root:     []byte{0xBB},
		}
		resp := TxProofResponse{
			TxID:        txid,
			BlockHeight: 500,
			Proof:       proof,
		}
		if resp.TxID != txid {
			t.Error("TxID mismatch")
		}
		if resp.BlockHeight != 500 {
			t.Errorf("BlockHeight = %d, want 500", resp.BlockHeight)
		}
		if len(resp.Proof.LeafHash) != 1 || resp.Proof.LeafHash[0] != 0xAA {
			t.Error("Proof.LeafHash mismatch")
		}
	})

	t.Run("block_tx_list_response_full_mode", func(t *testing.T) {
		// BlockTxListResponse 非概要模式：携带完整 TxID 列表，IsSummary=false，Summary=nil。
		blockID := types.MustBlockID(make([]byte, 48))
		txid1 := types.MustTxID(make([]byte, 48))
		txid2 := types.MustTxID(append(make([]byte, 47), 0x01))

		resp := BlockTxListResponse{
			BlockID:     blockID,
			BlockHeight: 100,
			TxIDs:       []types.TxID{txid1, txid2},
			IsSummary:   false,
		}
		if resp.IsSummary {
			t.Error("IsSummary should be false for full mode")
		}
		if resp.Summary != nil {
			t.Error("Summary should be nil in full mode")
		}
		if len(resp.TxIDs) != 2 {
			t.Errorf("TxIDs len = %d, want 2", len(resp.TxIDs))
		}
	})

	t.Run("block_tx_list_response_summary_mode", func(t *testing.T) {
		// BlockTxListResponse 概要模式：携带 BlockSummary，IsSummary=true，TxIDs=nil。
		blockID := types.MustBlockID(make([]byte, 48))
		summary := &BlockSummary{
			BlockID:      blockID,
			TxCount:      1,
			TxIDPrefixes: []TxIDPrefix{{}},
		}
		resp := BlockTxListResponse{
			BlockID:     blockID,
			BlockHeight: 100,
			Summary:     summary,
			IsSummary:   true,
		}
		if !resp.IsSummary {
			t.Error("IsSummary should be true for summary mode")
		}
		if resp.Summary == nil {
			t.Error("Summary should not be nil in summary mode")
		}
		if len(resp.TxIDs) != 0 {
			t.Errorf("TxIDs should be empty in summary mode, got len=%d", len(resp.TxIDs))
		}
	})

	t.Run("state_proof_response_fields", func(t *testing.T) {
		// StateProofResponse 携带：条目列表、UTXO 根、UTCO 根。
		utxoRoot := types.TreeHash{0x11}
		utcoRoot := types.TreeHash{0x22}
		entry := StateProofEntry{
			Kind:     StateKindUTXO,
			IsValid:  true,
			OutIndex: 0,
		}
		resp := StateProofResponse{
			Entries:  []StateProofEntry{entry},
			UTXORoot: utxoRoot,
			UTCORoot: utcoRoot,
		}
		if len(resp.Entries) != 1 {
			t.Errorf("Entries len = %d, want 1", len(resp.Entries))
		}
		if resp.Entries[0].Kind != StateKindUTXO {
			t.Errorf("Entry Kind = %d, want %d", resp.Entries[0].Kind, StateKindUTXO)
		}
		if resp.UTXORoot != utxoRoot {
			t.Error("UTXORoot mismatch")
		}
		if resp.UTCORoot != utcoRoot {
			t.Error("UTCORoot mismatch")
		}
	})

	t.Run("recent_block_proofs_minimum_31", func(t *testing.T) {
		// RecentBlockProofsResponse 必须包含至少 31 个区块证明包（DEC-0601）。
		cases := []struct {
			name    string
			count   int
			wantErr bool
		}{
			{"empty", 0, true},
			{"count_1", 1, true},
			{"count_30", 30, true},
			{"count_31", 31, false},
			{"count_50", 50, false},
			{"count_240", 240, false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				packages := make([]validation.ProofPackage, tc.count)
				resp := RecentBlockProofsResponse{ProofPackages: packages}
				err := ValidateRecentBlockProofs(resp)
				if tc.wantErr && err == nil {
					t.Errorf("count=%d: expected ErrRecentBlockProofsInsufficient, got nil", tc.count)
				}
				if !tc.wantErr && err != nil {
					t.Errorf("count=%d: unexpected error: %v", tc.count, err)
				}
				if tc.wantErr && err != nil && err != ErrRecentBlockProofsInsufficient {
					t.Errorf("count=%d: got %v, want ErrRecentBlockProofsInsufficient", tc.count, err)
				}
			})
		}
	})

	t.Run("attachment_index_small", func(t *testing.T) {
		// AttachmentIndexResponse 小附件模式：Data 非空，IsLargeAttachment=false。
		fp := types.AttachmentHash{}
		resp := AttachmentIndexResponse{
			Fingerprint:       fp,
			IsLargeAttachment: false,
			Data:              []byte{0xDE, 0xAD},
			FragmentCount:     0,
		}
		if resp.IsLargeAttachment {
			t.Error("IsLargeAttachment should be false for small attachment")
		}
		if len(resp.Data) == 0 {
			t.Error("Data should be non-empty for small attachment")
		}
		if resp.FragmentCount != 0 {
			t.Errorf("FragmentCount = %d, want 0 for small attachment", resp.FragmentCount)
		}
	})

	t.Run("attachment_index_large", func(t *testing.T) {
		// AttachmentIndexResponse 大附件模式：FragmentIndex 非空，IsLargeAttachment=true。
		fp := types.AttachmentHash{}
		resp := AttachmentIndexResponse{
			Fingerprint:       fp,
			IsLargeAttachment: true,
			FragmentIndex:     []byte{0x01, 0x02, 0x03},
			FragmentCount:     10,
		}
		if !resp.IsLargeAttachment {
			t.Error("IsLargeAttachment should be true for large attachment")
		}
		if len(resp.Data) != 0 {
			t.Errorf("Data should be empty for large attachment, got len=%d", len(resp.Data))
		}
		if len(resp.FragmentIndex) == 0 {
			t.Error("FragmentIndex should be non-empty for large attachment")
		}
		if resp.FragmentCount != 10 {
			t.Errorf("FragmentCount = %d, want 10", resp.FragmentCount)
		}
	})
}

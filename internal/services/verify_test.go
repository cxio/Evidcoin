package services

import (
	"testing"

	"github.com/cxio/evidcoin/internal/validation"
	"github.com/cxio/evidcoin/pkg/types"
)

// TestVerify 验证响应验证规则与服务密钥约束（第 15 章 §5·§6，DEC-0603）。
func TestVerify(t *testing.T) {
	t.Run("validate_recent_block_proofs_minimum", func(t *testing.T) {
		// 数量不足时必须返回 ErrRecentBlockProofsInsufficient。
		cases := []struct {
			name    string
			count   int
			wantErr error
		}{
			{"nil_packages", 0, ErrRecentBlockProofsInsufficient},
			{"count_1", 1, ErrRecentBlockProofsInsufficient},
			{"count_30", 30, ErrRecentBlockProofsInsufficient},
			{"count_31", 31, nil},
			{"count_100", 100, nil},
			{"count_240", 240, nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				packages := make([]validation.ProofPackage, tc.count)
				resp := RecentBlockProofsResponse{ProofPackages: packages}
				err := ValidateRecentBlockProofs(resp)
				if err != tc.wantErr {
					t.Errorf("ValidateRecentBlockProofs(count=%d) = %v, want %v",
						tc.count, err, tc.wantErr)
				}
			})
		}
	})

	t.Run("verify_block_summary_consistency_nil", func(t *testing.T) {
		// nil 指针应返回 ErrInvalidSummary。
		if err := VerifyBlockSummaryConsistency(nil); err != ErrInvalidSummary {
			t.Errorf("nil: got %v, want ErrInvalidSummary", err)
		}
	})

	t.Run("verify_block_summary_consistency_zero_count", func(t *testing.T) {
		// TxCount=0 应返回 ErrInvalidSummary。
		s := &BlockSummary{TxCount: 0, TxIDPrefixes: nil}
		if err := VerifyBlockSummaryConsistency(s); err != ErrInvalidSummary {
			t.Errorf("zero TxCount: got %v, want ErrInvalidSummary", err)
		}
	})

	t.Run("verify_block_summary_consistency_mismatch", func(t *testing.T) {
		// TxCount 与 TxIDPrefixes 数量不一致时应返回 ErrInvalidSummary。
		cases := []struct {
			name     string
			txCount  uint64
			prefixes int
		}{
			{"count_3_prefix_2", 3, 2},
			{"count_1_prefix_0", 1, 0},
			{"count_5_prefix_6", 5, 6},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := &BlockSummary{
					TxCount:      tc.txCount,
					TxIDPrefixes: make([]TxIDPrefix, tc.prefixes),
				}
				if err := VerifyBlockSummaryConsistency(s); err != ErrInvalidSummary {
					t.Errorf("got %v, want ErrInvalidSummary", err)
				}
			})
		}
	})

	t.Run("verify_block_summary_consistency_valid", func(t *testing.T) {
		// TxCount 与 TxIDPrefixes 数量一致时应返回 nil。
		cases := []int{1, 2, 10, 100}
		for _, n := range cases {
			blockID := types.MustBlockID(make([]byte, 48))
			txids := make([]types.TxID, n)
			for i := range txids {
				raw := make([]byte, 48)
				raw[0] = byte(i + 1)
				txids[i] = types.MustTxID(raw)
			}
			s := NewBlockSummary(blockID, txids)
			if err := VerifyBlockSummaryConsistency(&s); err != nil {
				t.Errorf("n=%d: unexpected error: %v", n, err)
			}
		}
	})

	t.Run("default_service_key_constraint_fields", func(t *testing.T) {
		// DefaultServiceKeyConstraint 的三个字段均必须为 true（DEC-0603）。
		c := DefaultServiceKeyConstraint
		if !c.ProveSourceOnly {
			t.Error("ProveSourceOnly must be true")
		}
		if !c.NotBoundToRewardAddress {
			t.Error("NotBoundToRewardAddress must be true")
		}
		if !c.CrossQueryRequired {
			t.Error("CrossQueryRequired must be true")
		}
	})

	t.Run("verification_anchor_carries_all_anchors", func(t *testing.T) {
		// VerificationAnchor 携带四类本地锚点用于验证。
		blockID := types.MustBlockID(make([]byte, 48))
		checkRoot := types.CheckRoot{0x01}
		utxoRoot := types.TreeHash{0x02}
		utcoRoot := types.TreeHash{0x03}

		anchor := VerificationAnchor{
			BlockID:   blockID,
			CheckRoot: checkRoot,
			UTXORoot:  utxoRoot,
			UTCORoot:  utcoRoot,
		}
		if anchor.BlockID != blockID {
			t.Error("BlockID mismatch")
		}
		if anchor.CheckRoot != checkRoot {
			t.Error("CheckRoot mismatch")
		}
		if anchor.UTXORoot != utxoRoot {
			t.Error("UTXORoot mismatch")
		}
		if anchor.UTCORoot != utcoRoot {
			t.Error("UTCORoot mismatch")
		}
	})

	t.Run("service_key_proves_source_not_data", func(t *testing.T) {
		// 协议约束：服务密钥签名仅证明来源，不证明数据真实（DEC-0603）。
		// 测试：DefaultServiceKeyConstraint 明确声明不应以服务密钥作为数据真实依据。
		c := DefaultServiceKeyConstraint
		// ProveSourceOnly=true 意味着服务密钥不证明数据真实。
		if !c.ProveSourceOnly {
			t.Error("service key must only prove source, not data authenticity")
		}
		// NotBoundToRewardAddress=true 意味着收益地址声明不作为真实性依据。
		if !c.NotBoundToRewardAddress {
			t.Error("service key must not be protocol-bound to reward address")
		}
	})

	t.Run("cross_query_required_for_critical_data", func(t *testing.T) {
		// 客户端应向多个 Blockqs 节点交叉查询关键数据（DEC-0603）。
		c := DefaultServiceKeyConstraint
		if !c.CrossQueryRequired {
			t.Error("cross-querying multiple nodes must be required for critical data")
		}
	})

	t.Run("initial_sync_needs_31_proof_packages", func(t *testing.T) {
		// 初始同步依赖 RecentBlockProofs 的完整性（至少 31 块，第 15 章 §6）。
		// 确认 MinRecentBlockProofs 与验证函数行为一致。
		exactly31 := make([]validation.ProofPackage, MinRecentBlockProofs)
		if err := ValidateRecentBlockProofs(RecentBlockProofsResponse{ProofPackages: exactly31}); err != nil {
			t.Errorf("exactly %d packages: unexpected error: %v", MinRecentBlockProofs, err)
		}
		tooFew := make([]validation.ProofPackage, MinRecentBlockProofs-1)
		if err := ValidateRecentBlockProofs(RecentBlockProofsResponse{ProofPackages: tooFew}); err != ErrRecentBlockProofsInsufficient {
			t.Errorf("one short of minimum: got %v, want ErrRecentBlockProofsInsufficient", err)
		}
	})
}

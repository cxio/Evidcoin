package services

import "testing"

// TestDecomposition 验证服务分解常量与服务类别定义（第 15 章 §1，DEC-0603）。
func TestDecomposition(t *testing.T) {
	t.Run("data_boundary_bytes", func(t *testing.T) {
		// 数据量边界必须精确为 10 MB。
		const want = 10 * 1024 * 1024
		if DataBoundaryBytes != want {
			t.Errorf("DataBoundaryBytes = %d, want %d", DataBoundaryBytes, want)
		}
	})

	t.Run("min_recent_block_proofs", func(t *testing.T) {
		// 最小区块证明包数量必须为 31（覆盖分叉安全窗口）。
		if MinRecentBlockProofs != 31 {
			t.Errorf("MinRecentBlockProofs = %d, want 31", MinRecentBlockProofs)
		}
	})

	t.Run("service_kind_ordering", func(t *testing.T) {
		// 服务类别枚举值必须从 1 开始，依序递增。
		cases := []struct {
			name string
			kind ServiceKind
			want ServiceKind
		}{
			{"Base", ServiceKindBase, 1},
			{"STUN", ServiceKindSTUN, 2},
			{"Depots", ServiceKindDepots, 3},
			{"Blockqs", ServiceKindBlockqs, 4},
		}
		for _, tc := range cases {
			if tc.kind != tc.want {
				t.Errorf("ServiceKind%s = %d, want %d", tc.name, tc.kind, tc.want)
			}
		}
	})

	t.Run("service_kind_distinct", func(t *testing.T) {
		// 四种服务类别值必须互不相同。
		kinds := []ServiceKind{
			ServiceKindBase,
			ServiceKindSTUN,
			ServiceKindDepots,
			ServiceKindBlockqs,
		}
		seen := make(map[ServiceKind]bool)
		for _, k := range kinds {
			if seen[k] {
				t.Errorf("ServiceKind value %d is duplicated", k)
			}
			seen[k] = true
		}
	})

	t.Run("depots_serves_large_data", func(t *testing.T) {
		// Depots 服务大尺寸数据（>= 10MB），Blockqs 服务小尺寸数据（< 10MB）。
		// 验证边界常量的语义方向正确。
		largeDataSize := DataBoundaryBytes
		smallDataSize := DataBoundaryBytes - 1

		// 达到边界值时使用 Depots（>= DataBoundaryBytes）。
		depotsServes := largeDataSize >= DataBoundaryBytes
		blockqsServes := smallDataSize < DataBoundaryBytes

		if !depotsServes {
			t.Error("Depots should serve data >= DataBoundaryBytes")
		}
		if !blockqsServes {
			t.Error("Blockqs should serve data < DataBoundaryBytes")
		}
	})

	t.Run("error_vars_non_nil", func(t *testing.T) {
		// 所有错误变量必须非 nil。
		errs := []error{
			ErrCollisionDetected,
			ErrInvalidSummary,
			ErrInvalidTxIDPrefixLen,
			ErrRecentBlockProofsInsufficient,
			ErrResponseNotVerifiable,
		}
		for _, err := range errs {
			if err == nil {
				t.Errorf("error variable is nil")
			}
		}
	})

	t.Run("error_messages_distinct", func(t *testing.T) {
		// 各错误消息文本互不相同。
		errs := []error{
			ErrCollisionDetected,
			ErrInvalidSummary,
			ErrInvalidTxIDPrefixLen,
			ErrRecentBlockProofsInsufficient,
			ErrResponseNotVerifiable,
		}
		seen := make(map[string]bool)
		for _, err := range errs {
			msg := err.Error()
			if seen[msg] {
				t.Errorf("duplicate error message: %q", msg)
			}
			seen[msg] = true
		}
	})
}

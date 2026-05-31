package blockchain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestIsYearBoundary 校验年度边界识别（height % BlocksPerYear == 0）。
func TestIsYearBoundary(t *testing.T) {
	cases := []struct {
		height uint32
		want   bool
	}{
		{0, true}, // 创世为年度边界
		{1, false},
		{types.BlocksPerYear - 1, false},
		{types.BlocksPerYear, true},
		{types.BlocksPerYear + 1, false},
		{2 * types.BlocksPerYear, true},
	}
	for _, tc := range cases {
		if got := IsYearBoundary(tc.height); got != tc.want {
			t.Errorf("IsYearBoundary(%d) = %v, 期望 %v", tc.height, got, tc.want)
		}
	}
}

// TestLastYearBoundary 校验返回 ≤ height 的最近年度边界高度。
func TestLastYearBoundary(t *testing.T) {
	cases := []struct {
		height uint32
		want   uint32
	}{
		{0, 0},
		{1, 0},
		{types.BlocksPerYear - 1, 0},
		{types.BlocksPerYear, types.BlocksPerYear},
		{types.BlocksPerYear + 5, types.BlocksPerYear},
		{2*types.BlocksPerYear + 3, 2 * types.BlocksPerYear},
	}
	for _, tc := range cases {
		if got := LastYearBoundary(tc.height); got != tc.want {
			t.Errorf("LastYearBoundary(%d) = %d, 期望 %d", tc.height, got, tc.want)
		}
	}
}

// TestYearBlockHeaderReference 校验非年度边界高度返回最近年块引用。
func TestYearBlockHeaderReference(t *testing.T) {
	s := newMemStore()
	g := genesisHeader(t)
	if err := s.Put(g); err != nil {
		t.Fatalf("Put: %v", err)
	}
	c := NewChain(s)

	yb, err := c.YearBlockHeader(100) // 年内高度，最近年块为创世（边界 0）
	if err != nil {
		t.Fatalf("YearBlockHeader: %v", err)
	}
	if yb.ID() != g.ID() {
		t.Fatal("最近年块引用不是创世头")
	}
}

// TestYearBlockHeaderMissing 校验缺失年块时返回明确错误 ErrYearBlockMissing。
func TestYearBlockHeaderMissing(t *testing.T) {
	s := newMemStore()
	g := genesisHeader(t)
	if err := s.Put(g); err != nil {
		t.Fatalf("Put: %v", err)
	}
	c := NewChain(s)

	// 第 2 年高度，边界 BlocksPerYear 未存储。
	if _, err := c.YearBlockHeader(types.BlocksPerYear + 10); !errors.Is(err, ErrYearBlockMissing) {
		t.Fatalf("缺失年块应返回 ErrYearBlockMissing, got %v", err)
	}
}

// TestValidateRecoveredHeader 校验恢复头必须与前后头衔接。
func TestValidateRecoveredHeader(t *testing.T) {
	g := genesisHeader(t)
	h1 := nextHeader(t, g, 0x11)
	h2 := nextHeader(t, h1, 0x22)

	// 合法衔接。
	if err := ValidateRecoveredHeader(g, h1, h2); err != nil {
		t.Fatalf("合法恢复头衔接应通过, got %v", err)
	}

	// 前向衔接断裂：h 的 PrevBlock 不指向 prev。
	badPrev := &BlockHeader{
		Version:   1,
		Height:    1,
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0xFF}, 48)),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x11}, 48)),
	}
	if err := ValidateRecoveredHeader(g, badPrev, h2); !errors.Is(err, ErrPrevBlockMismatch) {
		t.Fatalf("前向衔接断裂应返回 ErrPrevBlockMismatch, got %v", err)
	}

	// 后向衔接断裂：next 的 PrevBlock 不指向 h。
	badNext := &BlockHeader{
		Version:   1,
		Height:    2,
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0xFF}, 48)),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
	}
	if err := ValidateRecoveredHeader(g, h1, badNext); !errors.Is(err, ErrPrevBlockMismatch) {
		t.Fatalf("后向衔接断裂应返回 ErrPrevBlockMismatch, got %v", err)
	}

	// 高度不连续。
	if err := ValidateRecoveredHeader(g, h2, h2); !errors.Is(err, ErrHeightNotSequential) {
		t.Fatalf("高度不连续应返回 ErrHeightNotSequential, got %v", err)
	}
}

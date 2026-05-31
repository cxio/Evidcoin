package blockchain

import (
	"errors"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 创世区块头工件边界测试（proposal 05 §9）。
// 本层只固定创世头的确定字段、编码边界与验证规则；
// CheckRoot 具体值依赖 Coinbase 工件（第 06、14 章），
// Genesis-ID/创世时间戳属待决 C-9，不在本包硬编码。

func TestNewGenesisHeaderFields(t *testing.T) {
	var cr types.CheckRoot // CheckRoot 由调用方提供，此处用零值占位

	h := NewGenesisHeader(cr)

	if h.Version != GenesisVersion {
		t.Errorf("Version = %d, 期望 %d", h.Version, GenesisVersion)
	}
	if h.Height != 0 {
		t.Errorf("Height = %d, 期望 0", h.Height)
	}
	if h.PrevBlock != (types.BlockID{}) {
		t.Error("PrevBlock 应为全零（创世无前一区块）")
	}
	if h.Stakes != 0 {
		t.Errorf("Stakes = %d, 期望 0", h.Stakes)
	}
	if h.YearBlock != (types.Hash48{}) {
		t.Error("YearBlock 应全零（高度 0 为年块但无前一年块）")
	}
}

// TestNewGenesisHeaderIsYearBlock 创世（高度 0）为年块，编码须含 YearBlock，达 160 字节。
func TestNewGenesisHeaderIsYearBlock(t *testing.T) {
	h := NewGenesisHeader(types.CheckRoot{})
	if !h.IsYearBlock() {
		t.Fatal("创世应为年块（0 % BlocksPerYear == 0）")
	}
	if got := len(h.CanonicalBytes()); got != 160 {
		t.Errorf("创世头规范编码 = %d 字节, 期望 160（年块）", got)
	}
}

// TestValidateGenesisHeaderAcceptsValid 合法创世头工件应通过校验，且不受 CheckRoot 取值影响。
func TestValidateGenesisHeaderAcceptsValid(t *testing.T) {
	var cr types.CheckRoot
	cr[0] = 0xAB // CheckRoot 非零不应影响工件边界校验

	if err := ValidateGenesisHeader(NewGenesisHeader(cr)); err != nil {
		t.Fatalf("合法创世头被拒绝：%v", err)
	}
}

// TestValidateGenesisHeaderRejectsInvalid 篡改任一确定字段都应返回 ErrInvalidGenesisHeader。
func TestValidateGenesisHeaderRejectsInvalid(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(h *BlockHeader)
	}{
		{"wrong_version", func(h *BlockHeader) { h.Version = 2 }},
		{"nonzero_height", func(h *BlockHeader) { h.Height = 1 }},
		{"nonzero_prevblock", func(h *BlockHeader) { h.PrevBlock[0] = 1 }},
		{"nonzero_stakes", func(h *BlockHeader) { h.Stakes = 1 }},
		{"nonzero_yearblock", func(h *BlockHeader) { h.YearBlock[0] = 1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGenesisHeader(types.CheckRoot{})
			tt.mutate(h)
			err := ValidateGenesisHeader(h)
			if !errors.Is(err, ErrInvalidGenesisHeader) {
				t.Errorf("期望 ErrInvalidGenesisHeader, 得到 %v", err)
			}
		})
	}
}

package blockchain

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestBlockIDLength 校验 BlockID 输出 48 字节。
func TestBlockIDLength(t *testing.T) {
	h := &BlockHeader{
		Version:   1,
		Height:    5,
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x11}, 48)),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
		Stakes:    9,
	}
	if got := len(h.ID().Bytes()); got != 48 {
		t.Fatalf("BlockID 长度 = %d, 期望 48", got)
	}
}

// TestBlockIDDeterministic 校验相同字段得到相同 BlockID。
func TestBlockIDDeterministic(t *testing.T) {
	build := func() *BlockHeader {
		return &BlockHeader{
			Version:   1,
			Height:    5,
			PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x11}, 48)),
			CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
			Stakes:    9,
		}
	}
	// 用两个独立构建的实例比较，验证 ID 仅由字段决定（确定性）。
	first := build().ID()
	second := build().ID()
	if first != second {
		t.Fatal("相同字段的 BlockID 不一致")
	}
}

// TestBlockIDChangesWithEachField 校验改变任一字段都会改变 BlockID。
func TestBlockIDChangesWithEachField(t *testing.T) {
	base := &BlockHeader{
		Version:   1,
		Height:    100,
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x11}, 48)),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
		Stakes:    42,
		YearBlock: mustHash48(t, bytes.Repeat([]byte{0x00}, 48)),
	}
	baseID := base.ID()

	mutations := map[string]func(*BlockHeader){
		"Version":   func(h *BlockHeader) { h.Version = 2 },
		"Height":    func(h *BlockHeader) { h.Height = 101 },
		"PrevBlock": func(h *BlockHeader) { h.PrevBlock = types.MustBlockID(bytes.Repeat([]byte{0xAA}, 48)) },
		"CheckRoot": func(h *BlockHeader) { h.CheckRoot = mustCheckRoot(t, bytes.Repeat([]byte{0xBB}, 48)) },
		"Stakes":    func(h *BlockHeader) { h.Stakes = 43 },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			h := *base
			mutate(&h)
			if h.ID() == baseID {
				t.Fatalf("修改字段 %s 未改变 BlockID", name)
			}
		})
	}
}

// TestBlockIDYearBlockFieldAffectsID 校验年块的 YearBlock 字段参与 BlockID 前像。
func TestBlockIDYearBlockFieldAffectsID(t *testing.T) {
	mk := func(yb byte) *BlockHeader {
		return &BlockHeader{
			Version:   1,
			Height:    types.BlocksPerYear,
			PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x11}, 48)),
			CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x22}, 48)),
			Stakes:    1,
			YearBlock: mustHash48(t, bytes.Repeat([]byte{yb}, 48)),
		}
	}
	if mk(0x01).ID() == mk(0x02).ID() {
		t.Fatal("年块的 YearBlock 字段未影响 BlockID")
	}
}

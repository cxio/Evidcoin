package blockchain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// genesisHeader 构造一个合法创世头：高度 0、PrevBlock 全零。
func genesisHeader(t *testing.T) *BlockHeader {
	t.Helper()
	return &BlockHeader{
		Version:   1,
		Height:    0,
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0xA0}, 48)),
		Stakes:    0,
	}
}

// nextHeader 基于前一头构造合法后继头（高度 +1，PrevBlock 衔接前一头 ID）。
func nextHeader(t *testing.T, prev *BlockHeader, tag byte) *BlockHeader {
	t.Helper()
	return &BlockHeader{
		Version:   1,
		Height:    prev.Height + 1,
		PrevBlock: prev.ID(),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{tag}, 48)),
		Stakes:    uint64(prev.Height + 1),
	}
}

// TestChainInitGenesis 校验空链可用创世头初始化，且 tip 即创世。
func TestChainInitGenesis(t *testing.T) {
	c := NewChain(newMemStore())
	g := genesisHeader(t)
	id, err := c.AddHeader(g)
	if err != nil {
		t.Fatalf("AddHeader(genesis): %v", err)
	}
	if id != g.ID() {
		t.Fatal("AddHeader 返回的 ID 与重算不一致")
	}
	tip, err := c.Tip()
	if err != nil {
		t.Fatalf("Tip: %v", err)
	}
	if tip.ID() != g.ID() {
		t.Fatal("tip 不是创世头")
	}
}

// TestChainRejectNonGenesisFirst 校验首块必须是创世（高度 0、PrevBlock 全零）。
func TestChainRejectNonGenesisFirst(t *testing.T) {
	c := NewChain(newMemStore())
	bad := &BlockHeader{Version: 1, Height: 5, CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x01}, 48))}
	if _, err := c.AddHeader(bad); !errors.Is(err, ErrNotGenesis) {
		t.Fatalf("首块高度非 0 应返回 ErrNotGenesis, got %v", err)
	}

	c2 := NewChain(newMemStore())
	bad2 := &BlockHeader{
		Version:   1,
		Height:    0,
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x09}, 48)), // 非全零
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x01}, 48)),
	}
	if _, err := c2.AddHeader(bad2); !errors.Is(err, ErrNotGenesis) {
		t.Fatalf("创世 PrevBlock 非全零应返回 ErrNotGenesis, got %v", err)
	}
}

// TestChainHeightMustBeSequential 校验新头高度必须为 tip+1；跨高度（缺失中间头）拒绝衔接。
func TestChainHeightMustBeSequential(t *testing.T) {
	c := NewChain(newMemStore())
	g := genesisHeader(t)
	if _, err := c.AddHeader(g); err != nil {
		t.Fatalf("AddHeader(genesis): %v", err)
	}
	// 直接提交高度 2（缺失高度 1）。
	gap := &BlockHeader{
		Version:   1,
		Height:    2,
		PrevBlock: g.ID(),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x02}, 48)),
	}
	if _, err := c.AddHeader(gap); !errors.Is(err, ErrHeightNotSequential) {
		t.Fatalf("跨高度衔接应返回 ErrHeightNotSequential, got %v", err)
	}
}

// TestChainPrevBlockMustMatchTip 校验 PrevBlock 必须等于当前 tip 的 ID。
func TestChainPrevBlockMustMatchTip(t *testing.T) {
	c := NewChain(newMemStore())
	g := genesisHeader(t)
	if _, err := c.AddHeader(g); err != nil {
		t.Fatalf("AddHeader(genesis): %v", err)
	}
	bad := &BlockHeader{
		Version:   1,
		Height:    1,
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0xFF}, 48)), // 错误衔接
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{0x03}, 48)),
	}
	if _, err := c.AddHeader(bad); !errors.Is(err, ErrPrevBlockMismatch) {
		t.Fatalf("PrevBlock 不衔接应返回 ErrPrevBlockMismatch, got %v", err)
	}
}

// TestChainSameHeightConflict 校验同高度二次提交被拒，且 tip 不被自动替换。
func TestChainSameHeightConflict(t *testing.T) {
	c := NewChain(newMemStore())
	g := genesisHeader(t)
	if _, err := c.AddHeader(g); err != nil {
		t.Fatalf("AddHeader(genesis): %v", err)
	}
	h1 := nextHeader(t, g, 0x11)
	if _, err := c.AddHeader(h1); err != nil {
		t.Fatalf("AddHeader(h1): %v", err)
	}
	// 另一个高度 1 的不同区块头。
	h1b := nextHeader(t, g, 0x22)
	if _, err := c.AddHeader(h1b); !errors.Is(err, ErrHeightConflict) {
		t.Fatalf("同高度冲突应返回 ErrHeightConflict, got %v", err)
	}
	tip, err := c.Tip()
	if err != nil {
		t.Fatalf("Tip: %v", err)
	}
	if tip.ID() != h1.ID() {
		t.Fatal("同高度冲突后 tip 不应被替换")
	}
}

// TestChainAddHeaderWithIDMismatch 校验接收外部声明 ID 的 API 在重算不匹配时拒绝。
func TestChainAddHeaderWithIDMismatch(t *testing.T) {
	c := NewChain(newMemStore())
	g := genesisHeader(t)
	wrongID := types.MustBlockID(bytes.Repeat([]byte{0xEE}, 48))
	if _, err := c.AddHeaderWithID(g, wrongID); !errors.Is(err, ErrBlockIDMismatch) {
		t.Fatalf("声明 ID 不匹配应返回 ErrBlockIDMismatch, got %v", err)
	}
	// 正确声明 ID 应成功。
	if _, err := c.AddHeaderWithID(g, g.ID()); err != nil {
		t.Fatalf("声明 ID 正确应成功, got %v", err)
	}
}

// TestChainQuery 校验按高度与按 ID 查询。
func TestChainQuery(t *testing.T) {
	c := NewChain(newMemStore())
	g := genesisHeader(t)
	if _, err := c.AddHeader(g); err != nil {
		t.Fatalf("AddHeader(genesis): %v", err)
	}
	h1 := nextHeader(t, g, 0x11)
	if _, err := c.AddHeader(h1); err != nil {
		t.Fatalf("AddHeader(h1): %v", err)
	}
	byH, err := c.HeaderByHeight(1)
	if err != nil || byH.ID() != h1.ID() {
		t.Fatalf("HeaderByHeight(1) = %v, err=%v", byH, err)
	}
	byID, err := c.HeaderByID(g.ID())
	if err != nil || byID.Height != 0 {
		t.Fatalf("HeaderByID(genesis) = %v, err=%v", byID, err)
	}
}

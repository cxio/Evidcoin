package blockchain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// storeTestHeader 构造高度为 height 的测试区块头，CheckRoot 由 tag 区分。
func storeTestHeader(t *testing.T, height uint32, tag byte) *BlockHeader {
	t.Helper()
	return &BlockHeader{
		Version:   1,
		Height:    height,
		PrevBlock: types.MustBlockID(bytes.Repeat([]byte{0x11}, 48)),
		CheckRoot: mustCheckRoot(t, bytes.Repeat([]byte{tag}, 48)),
		Stakes:    uint64(height),
	}
}

// TestHeaderStoreByHeight 校验按高度查询。
func TestHeaderStoreByHeight(t *testing.T) {
	s := newMemStore()
	h := storeTestHeader(t, 5, 0x22)
	if err := s.Put(h); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.ByHeight(5)
	if err != nil {
		t.Fatalf("ByHeight: %v", err)
	}
	if got.ID() != h.ID() {
		t.Fatal("ByHeight 返回的区块头不匹配")
	}
}

// TestHeaderStoreByID 校验按 BlockID 查询。
func TestHeaderStoreByID(t *testing.T) {
	s := newMemStore()
	h := storeTestHeader(t, 5, 0x22)
	if err := s.Put(h); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.ByID(h.ID())
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Height != 5 {
		t.Fatalf("ByID 返回高度 = %d, 期望 5", got.Height)
	}
}

// TestHeaderStoreTip 校验 tip 返回最高高度区块头。
func TestHeaderStoreTip(t *testing.T) {
	s := newMemStore()
	for _, h := range []*BlockHeader{
		storeTestHeader(t, 0, 0x01),
		storeTestHeader(t, 2, 0x02),
		storeTestHeader(t, 1, 0x03),
	} {
		if err := s.Put(h); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}
	tip, err := s.Tip()
	if err != nil {
		t.Fatalf("Tip: %v", err)
	}
	if tip.Height != 2 {
		t.Fatalf("Tip 高度 = %d, 期望 2", tip.Height)
	}
}

// TestHeaderStoreMissing 校验缺失头返回 ErrHeaderNotFound。
func TestHeaderStoreMissing(t *testing.T) {
	s := newMemStore()
	if _, err := s.ByHeight(9); !errors.Is(err, ErrHeaderNotFound) {
		t.Fatalf("ByHeight 缺失应返回 ErrHeaderNotFound, got %v", err)
	}
	missing := types.MustBlockID(bytes.Repeat([]byte{0xEE}, 48))
	if _, err := s.ByID(missing); !errors.Is(err, ErrHeaderNotFound) {
		t.Fatalf("ByID 缺失应返回 ErrHeaderNotFound, got %v", err)
	}
	if _, err := s.Tip(); !errors.Is(err, ErrHeaderNotFound) {
		t.Fatalf("空 store 的 Tip 应返回 ErrHeaderNotFound, got %v", err)
	}
}

package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// memStore 是测试用内存 HeaderStore 实现（DEC：测试替身，非生产存储）。
// 放在 _test.go 中以避免被误当作生产存储引用。
type memStore struct {
	byID     map[types.BlockID]*BlockHeader
	byHeight map[uint32]*BlockHeader
	tipH     uint32
	hasTip   bool
}

// 编译期断言 memStore 满足 HeaderStore 接口。
var _ HeaderStore = (*memStore)(nil)

func newMemStore() *memStore {
	return &memStore{
		byID:     make(map[types.BlockID]*BlockHeader),
		byHeight: make(map[uint32]*BlockHeader),
	}
}

func (s *memStore) Put(h *BlockHeader) error {
	cp := *h
	s.byID[h.ID()] = &cp
	s.byHeight[h.Height] = &cp
	if !s.hasTip || h.Height > s.tipH {
		s.tipH = h.Height
		s.hasTip = true
	}
	return nil
}

func (s *memStore) ByID(id types.BlockID) (*BlockHeader, error) {
	h, ok := s.byID[id]
	if !ok {
		return nil, ErrHeaderNotFound
	}
	return h, nil
}

func (s *memStore) ByHeight(height uint32) (*BlockHeader, error) {
	h, ok := s.byHeight[height]
	if !ok {
		return nil, ErrHeaderNotFound
	}
	return h, nil
}

func (s *memStore) Tip() (*BlockHeader, error) {
	if !s.hasTip {
		return nil, ErrHeaderNotFound
	}
	return s.byHeight[s.tipH], nil
}

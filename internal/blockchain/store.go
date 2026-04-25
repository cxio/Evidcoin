package blockchain

import (
	"errors"
	"sync"

	"github.com/cxio/evidcoin/pkg/types"
)

// ErrNotFound 区块头不存在。
var ErrNotFound = errors.New("block header not found")

// HeaderStore 区块头存储接口。
type HeaderStore interface {
	// Get 按高度获取区块头。
	Get(height int32) (*BlockHeader, error)

	// GetByHash 按区块哈希获取区块头。
	GetByHash(hash types.Hash384) (*BlockHeader, error)

	// Put 存储一个区块头。
	Put(header *BlockHeader) error

	// Has 检查指定高度的区块头是否存在。
	Has(height int32) bool

	// Tip 返回当前链顶区块头。
	Tip() (*BlockHeader, error)

	// YearBlock 获取指定年份的年块区块头（year 从 1 开始）。
	YearBlock(year int) (*BlockHeader, error)
}

// MemoryStore 基于内存的区块头存储实现（用于测试）。
type MemoryStore struct {
	mu      sync.RWMutex
	byHeight map[int32]*BlockHeader
	byHash  map[types.Hash384]*BlockHeader
	tip     *BlockHeader
}

// NewMemoryStore 创建新的内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byHeight: make(map[int32]*BlockHeader),
		byHash:  make(map[types.Hash384]*BlockHeader),
	}
}

// Get 按高度获取区块头。
func (s *MemoryStore) Get(height int32) (*BlockHeader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.byHeight[height]
	if !ok {
		return nil, ErrNotFound
	}
	return h, nil
}

// GetByHash 按哈希获取区块头。
func (s *MemoryStore) GetByHash(hash types.Hash384) (*BlockHeader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.byHash[hash]
	if !ok {
		return nil, ErrNotFound
	}
	return h, nil
}

// Put 存储区块头。
func (s *MemoryStore) Put(header *BlockHeader) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := header.Hash()
	s.byHeight[header.Height] = header
	s.byHash[hash] = header
	if s.tip == nil || header.Height > s.tip.Height {
		s.tip = header
	}
	return nil
}

// Has 检查高度是否存在。
func (s *MemoryStore) Has(height int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.byHeight[height]
	return ok
}

// Tip 返回链顶区块头。
func (s *MemoryStore) Tip() (*BlockHeader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.tip == nil {
		return nil, ErrNotFound
	}
	return s.tip, nil
}

// YearBlock 获取指定年份的年块（year 从 1 开始）。
func (s *MemoryStore) YearBlock(year int) (*BlockHeader, error) {
	height := int32(year * types.BlocksPerYear)
	return s.Get(height)
}

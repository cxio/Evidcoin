package blockchain

import (
	"errors"
	"sync"

	"github.com/cxio/evidcoin/pkg/types"
)

// 区块提交错误类型
var (
	// ErrConflict 该高度已存在区块。
	ErrConflict = errors.New("block already exists at this height")
	// ErrInvalidPrevBlock PrevBlock 不匹配当前链顶。
	ErrInvalidPrevBlock = errors.New("invalid previous block hash")
	// ErrInvalidHeight 区块高度不连续。
	ErrInvalidHeight = errors.New("invalid block height")
	// ErrInvalidVersion 版本号不兼容。
	ErrInvalidVersion = errors.New("unsupported block version")
	// ErrInvalidYearBlock 年块字段不正确。
	ErrInvalidYearBlock = errors.New("invalid year block reference")
	// ErrInvalidCheckRoot CheckRoot 为零。
	ErrInvalidCheckRoot = errors.New("check root must not be zero")
)

// CurrentVersion 当前支持的最高区块版本号。
const CurrentVersion = int32(1)

// Blockchain 区块头链核心，负责区块头链的管理与查询。
type Blockchain struct {
	mu       sync.RWMutex
	store    HeaderStore
	blockqs  BlockqsClient
	identity *ChainIdentity
	// subscribers 新区块订阅者列表
	subMu       sync.Mutex
	subscribers []chan<- *BlockHeader
}

// Config 区块链初始化配置。
type Config struct {
	Store    HeaderStore
	Blockqs  BlockqsClient
	Identity *ChainIdentity
	// Genesis 创世区块头（必须提供）
	Genesis *BlockHeader
}

// New 创建一个新的 Blockchain 实例。
// 如果存储为空，则写入创世区块。
func New(cfg Config) (*Blockchain, error) {
	if cfg.Store == nil {
		cfg.Store = NewMemoryStore()
	}
	if cfg.Blockqs == nil {
		cfg.Blockqs = NoopBlockqsClient()
	}
	bc := &Blockchain{
		store:    cfg.Store,
		blockqs:  cfg.Blockqs,
		identity: cfg.Identity,
	}
	// 若存储为空且提供了创世块，写入创世块
	if cfg.Genesis != nil && !cfg.Store.Has(0) {
		if err := cfg.Store.Put(cfg.Genesis); err != nil {
			return nil, err
		}
	}
	return bc, nil
}

// SubmitBlock 提交新区块到链上。
// 区块的内容合法性由外部调用者保证。
// 返回 ErrConflict 如果该高度已有区块。
func (bc *Blockchain) SubmitBlock(header *BlockHeader) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if err := bc.validateHeader(header); err != nil {
		return err
	}
	if bc.store.Has(header.Height) {
		return ErrConflict
	}
	if err := bc.store.Put(header); err != nil {
		return err
	}
	bc.notifySubscribers(header)
	return nil
}

// ReplaceBlock 替换当前链顶区块（用于分叉切换）。
// 要求 height 必须是当前链顶高度。
func (bc *Blockchain) ReplaceBlock(height int32, header *BlockHeader) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	tip, err := bc.store.Tip()
	if err != nil {
		return err
	}
	if height != tip.Height {
		return errors.New("can only replace the current chain tip")
	}
	return bc.store.Put(header)
}

// validateHeader 验证区块头的结构合法性（不验证交易内容）。
func (bc *Blockchain) validateHeader(header *BlockHeader) error {
	// 版本号检查
	if header.Version < 1 || header.Version > CurrentVersion {
		return ErrInvalidVersion
	}
	// CheckRoot 非零
	if header.CheckRoot.IsZero() {
		return ErrInvalidCheckRoot
	}
	// 对于非创世块，验证高度和 PrevBlock 连续性
	if header.Height > 0 {
		tip, err := bc.store.Tip()
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if tip != nil {
			if header.Height != tip.Height+1 {
				return ErrInvalidHeight
			}
			tipHash := tip.Hash()
			if header.PrevBlock != tipHash {
				return ErrInvalidPrevBlock
			}
		}
	}
	// 年块字段验证
	if header.IsYearBlock() {
		year := int(header.Height) / types.BlocksPerYear
		prevYearHeader, err := bc.store.YearBlock(year - 1)
		if err == nil {
			prevYearHash := prevYearHeader.Hash()
			if header.YearBlock != prevYearHash {
				return ErrInvalidYearBlock
			}
		}
		// 若前一年块不存在（初始同步场景），暂允许通过
	}
	return nil
}

// HeaderByHeight 按高度查询区块头。
// 如果本地缺失，自动从 Blockqs 获取。
func (bc *Blockchain) HeaderByHeight(height int32) (*BlockHeader, error) {
	bc.mu.RLock()
	h, err := bc.store.Get(height)
	bc.mu.RUnlock()
	if err == nil {
		return h, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	// 从 Blockqs 获取
	return bc.blockqs.FetchHeader(height)
}

// HeaderByHash 按哈希查询区块头。
func (bc *Blockchain) HeaderByHash(hash types.Hash384) (*BlockHeader, error) {
	bc.mu.RLock()
	h, err := bc.store.GetByHash(hash)
	bc.mu.RUnlock()
	if err == nil {
		return h, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return bc.blockqs.FetchHeaderByHash(hash)
}

// ChainTip 返回当前链顶区块头。
func (bc *Blockchain) ChainTip() (*BlockHeader, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.store.Tip()
}

// ChainHeight 返回当前链顶高度，若链为空返回 -1。
func (bc *Blockchain) ChainHeight() int32 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	tip, err := bc.store.Tip()
	if err != nil {
		return -1
	}
	return tip.Height
}

// Identity 返回当前节点的链标识信息。
func (bc *Blockchain) Identity() *ChainIdentity {
	return bc.identity
}

// Subscribe 订阅新区块事件。
func (bc *Blockchain) Subscribe(ch chan<- *BlockHeader) {
	bc.subMu.Lock()
	defer bc.subMu.Unlock()
	bc.subscribers = append(bc.subscribers, ch)
}

// notifySubscribers 通知所有订阅者（非阻塞）。
func (bc *Blockchain) notifySubscribers(header *BlockHeader) {
	bc.subMu.Lock()
	defer bc.subMu.Unlock()
	for _, ch := range bc.subscribers {
		select {
		case ch <- header:
		default:
			// 订阅者缓冲满则跳过，不阻塞
		}
	}
}

// SyncHeaders 同步指定范围的区块头（从 Blockqs 拉取并存储）。
func (bc *Blockchain) SyncHeaders(from, to int32) error {
	headers, err := bc.blockqs.FetchHeaders(from, to)
	if err != nil {
		return err
	}
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, h := range headers {
		if !bc.store.Has(h.Height) {
			if err := bc.store.Put(h); err != nil {
				return err
			}
		}
	}
	return nil
}

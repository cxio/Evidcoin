# Phase 2: Blockchain Core（区块链核心）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `internal/blockchain` 包，提供区块头链的管理、验证、存储与查询能力。

**Architecture:** Layer 1 核心层。依赖 Layer 0（`pkg/types`, `pkg/crypto`）。Core 专注区块头链管理，不含共识逻辑、交易校验或脚本执行。外部提交者负责内容合法性，Core 仅执行结构性验证。

**Tech Stack:** Go 1.25+，接口抽象存储后端（默认内存实现），Blockqs 连接器接口（本阶段 stub 实现）。

---

## 目录结构（预期）

```
internal/blockchain/
  header.go          # BlockHeader 结构定义与哈希计算
  identity.go        # ChainIdentity 链标识
  store.go           # HeaderStore 接口 + 内存实现
  blockchain.go      # Blockchain 核心结构与提交/查询逻辑
  blockqs.go         # BlockqsClient 接口 stub
  fork.go            # 分叉检测与手动切换
  blockchain_test.go # 单元测试
```

---

## Task 1: 区块头结构（internal/blockchain/header.go）

**Files:**
- Create: `internal/blockchain/header.go`

**Step 1: 编写区块头结构**

```go
// Package blockchain 实现 Evidcoin 区块头链的管理与维护。
package blockchain

import (
	"encoding/binary"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// BlockHeader 区块头结构（112 字节常规，年块额外 +48 字节）。
type BlockHeader struct {
	Version   int32        // 协议版本号（4 字节）
	Height    int32        // 区块高度，从 0 开始（4 字节）
	PrevBlock types.Hash384 // 前一区块 SHA3-384 哈希（48 字节）
	CheckRoot types.Hash384 // 校验根（48 字节）
	Stakes    uint64       // 币权销毁量，单位：聪时（8 字节）
	YearBlock types.Hash384 // 前一年块哈希（仅 Height % 87661 == 0 时存在）
}

// IsYearBlock 返回该区块是否为年块（Height 是 BlocksPerYear 的整数倍且不为 0）。
func (h *BlockHeader) IsYearBlock() bool {
	return h.Height > 0 && int(h.Height)%types.BlocksPerYear == 0
}

// Hash 计算区块头的 SHA3-384 哈希，即区块 ID。
func (h *BlockHeader) Hash() types.Hash384 {
	return crypto.SHA3_384(h.Bytes())
}

// Bytes 将区块头序列化为字节序列（用于哈希计算）。
// 格式：Version(4) + Height(4) + PrevBlock(48) + CheckRoot(48) + Stakes(8) [+ YearBlock(48)]
func (h *BlockHeader) Bytes() []byte {
	size := 112
	if h.IsYearBlock() {
		size += 48
	}
	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf[0:4], uint32(h.Version))
	binary.BigEndian.PutUint32(buf[4:8], uint32(h.Height))
	copy(buf[8:56], h.PrevBlock[:])
	copy(buf[56:104], h.CheckRoot[:])
	binary.BigEndian.PutUint64(buf[104:112], h.Stakes)
	if h.IsYearBlock() {
		copy(buf[112:160], h.YearBlock[:])
	}
	return buf
}

// BlockTime 按高度计算区块时间戳（Unix 毫秒）。
// 时间戳不存储于区块头，通过公式确定性计算。
func BlockTime(genesisTimestamp int64, height int32) int64 {
	return genesisTimestamp + int64(height)*int64(types.BlockInterval.Milliseconds())
}
```

**Step 2: 运行构建**

```bash
go build ./internal/blockchain/...
```

---

## Task 2: 链标识（internal/blockchain/identity.go）

**Files:**
- Create: `internal/blockchain/identity.go`

**Step 1: 编写链标识结构**

```go
package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// ChainIdentity 链标识信息，用于 P2P 握手与交易签名。
type ChainIdentity struct {
	// ProtocolID 协议标识，如 "Evidcoin@V1"
	ProtocolID string
	// ChainID 运行态标识，如 "mainnet"
	ChainID string
	// GenesisID 创世区块 ID（SHA3-384）
	GenesisID types.Hash384
	// BoundID 主链绑定（可选），取 -29 号区块 ID 前 20 字节
	// 为 nil 时表示未绑定
	BoundID []byte
}

// Bytes 将链标识序列化为字节序列，用于参与交易签名前置。
// 格式：ProtocolID_len(1) + ProtocolID + ChainID_len(1) + ChainID + GenesisID(48) + BoundID
func (ci *ChainIdentity) Bytes() []byte {
	pid := []byte(ci.ProtocolID)
	cid := []byte(ci.ChainID)
	size := 1 + len(pid) + 1 + len(cid) + types.Hash384Len
	if len(ci.BoundID) > 0 {
		size += len(ci.BoundID)
	}
	buf := make([]byte, 0, size)
	buf = append(buf, byte(len(pid)))
	buf = append(buf, pid...)
	buf = append(buf, byte(len(cid)))
	buf = append(buf, cid...)
	buf = append(buf, ci.GenesisID[:]...)
	if len(ci.BoundID) > 0 {
		buf = append(buf, ci.BoundID...)
	}
	return buf
}
```

**Step 2: 运行构建**

```bash
go build ./internal/blockchain/...
```

---

## Task 3: 存储接口与内存实现（internal/blockchain/store.go）

**Files:**
- Create: `internal/blockchain/store.go`

**Step 1: 编写 HeaderStore 接口与内存实现**

```go
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
```

**Step 2: 运行构建**

```bash
go build ./internal/blockchain/...
```

---

## Task 4: Blockqs 连接器接口（internal/blockchain/blockqs.go）

**Files:**
- Create: `internal/blockchain/blockqs.go`

**Step 1: 编写接口与 stub 实现**

```go
package blockchain

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// ErrBlockqsUnavailable Blockqs 服务不可用。
var ErrBlockqsUnavailable = errors.New("blockqs service unavailable")

// BlockqsClient 区块查询服务客户端接口。
// 本阶段提供 stub 实现，真实实现在阶段 8（服务接口）完成。
type BlockqsClient interface {
	// FetchHeader 从远程服务获取指定高度的区块头。
	FetchHeader(height int32) (*BlockHeader, error)

	// FetchHeaders 批量获取区块头（用于同步）。
	FetchHeaders(from, to int32) ([]*BlockHeader, error)

	// FetchHeaderByHash 按哈希获取区块头。
	FetchHeaderByHash(hash types.Hash384) (*BlockHeader, error)
}

// noopBlockqsClient 空实现，用于不需要远程获取的场景（测试或离线模式）。
type noopBlockqsClient struct{}

func (n *noopBlockqsClient) FetchHeader(height int32) (*BlockHeader, error) {
	return nil, ErrBlockqsUnavailable
}

func (n *noopBlockqsClient) FetchHeaders(from, to int32) ([]*BlockHeader, error) {
	return nil, ErrBlockqsUnavailable
}

func (n *noopBlockqsClient) FetchHeaderByHash(hash types.Hash384) (*BlockHeader, error) {
	return nil, ErrBlockqsUnavailable
}

// NoopBlockqsClient 返回一个不执行任何操作的 stub 客户端。
func NoopBlockqsClient() BlockqsClient {
	return &noopBlockqsClient{}
}
```

**Step 2: 运行构建**

```bash
go build ./internal/blockchain/...
```

---

## Task 5: 区块链核心逻辑（internal/blockchain/blockchain.go）

**Files:**
- Create: `internal/blockchain/blockchain.go`

**Step 1: 编写核心结构与方法**

```go
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
	subMu      sync.Mutex
	subscribers []chan<- *BlockHeader
}

// Config 区块链初始化配置。
type Config struct {
	Store    HeaderStore
	Blockqs  BlockqsClient
	Identity *ChainIdentity
	// Genesis 创世区块头（必须提供）
	Genesis  *BlockHeader
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
		store:   cfg.Store,
		blockqs: cfg.Blockqs,
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
```

**Step 2: 运行构建**

```bash
go build ./internal/blockchain/...
```

---

## Task 6: 分叉处理（internal/blockchain/fork.go）

**Files:**
- Create: `internal/blockchain/fork.go`

**Step 1: 编写分叉检测与切换接口**

```go
package blockchain

import "github.com/cxio/evidcoin/pkg/types"

// ForkInfo 分叉信息。
type ForkInfo struct {
	// ForkHeight 分叉发生的高度（两链首次出现分歧的高度）
	ForkHeight int32
	// LocalTip 本地链顶区块头
	LocalTip *BlockHeader
	// RemoteTip 远程链顶区块头
	RemoteTip *BlockHeader
	// RemoteLength 远程链自分叉点后的区块数
	RemoteLength int
}

// DetectFork 检测与指定节点之间的分叉。
// peers 为目标节点的地址列表（本阶段为 stub，真实实现在阶段 7/8）。
func (bc *Blockchain) DetectFork(peers []string) (*ForkInfo, error) {
	// 本阶段返回 nil（stub），共识层实现真实逻辑
	return nil, nil
}

// SwitchChain 切换到替代链。
// 这是一个需要用户明确确认的危险操作。
// forkHeight 为分叉高度，headers 为分叉点之后的替代链区块头序列。
func (bc *Blockchain) SwitchChain(forkHeight int32, headers []*BlockHeader) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 验证替代链的区块头连续性
	for i := 1; i < len(headers); i++ {
		prevHash := headers[i-1].Hash()
		if headers[i].PrevBlock != prevHash {
			return ErrInvalidPrevBlock
		}
		if headers[i].Height != headers[i-1].Height+1 {
			return ErrInvalidHeight
		}
	}
	// 写入替代链区块头
	for _, h := range headers {
		if err := bc.store.Put(h); err != nil {
			return err
		}
	}
	return nil
}

// BootstrapResult 初始验证结果。
type BootstrapResult struct {
	// ChainTip 验证通过的链顶区块头
	ChainTip *BlockHeader
	// Height 当前链高度
	Height int32
	// Confidence 置信度（基于多源一致性，0.0–1.0）
	Confidence float64
	// Sources 参与验证的数据源数量
	Sources int
}

// BootstrapVerify 初始主链验证（stub 实现）。
// 从多个数据源获取主链信息并交叉验证，确认目标主链的合法性。
func (bc *Blockchain) BootstrapVerify(sources []string) (*BootstrapResult, error) {
	// 本阶段返回空结果，真实实现在服务接口阶段完成
	tip, err := bc.store.Tip()
	if err != nil {
		return nil, err
	}
	return &BootstrapResult{
		ChainTip:   tip,
		Height:     tip.Height,
		Confidence: 1.0,
		Sources:    0,
	}, nil
}

// BoundIDFromHeader 从指定区块头计算 BoundID（取区块 ID 前 20 字节）。
// 通常取 -29 号区块（当前高度减 29）的区块 ID 前 20 字节。
func BoundIDFromHeader(h *BlockHeader) []byte {
	hash := h.Hash()
	bound := make([]byte, 20)
	copy(bound, hash[:20])
	return bound
}

// UpdateBoundID 更新链标识中的 BoundID（分叉确定后调用）。
func (bc *Blockchain) UpdateBoundID(height int32) error {
	h, err := bc.HeaderByHeight(height)
	if err != nil {
		return err
	}
	if bc.identity != nil {
		bc.identity.BoundID = BoundIDFromHeader(h)
	}
	return nil
}

// GenesisID 返回创世区块的区块 ID。
func (bc *Blockchain) GenesisID() (types.Hash384, error) {
	genesis, err := bc.HeaderByHeight(0)
	if err != nil {
		return types.Hash384{}, err
	}
	return genesis.Hash(), nil
}
```

**Step 2: 运行构建**

```bash
go build ./internal/blockchain/...
```

---

## Task 7: 单元测试（internal/blockchain/blockchain_test.go）

**Files:**
- Create: `internal/blockchain/blockchain_test.go`

**Step 1: 编写表驱动测试**

```go
package blockchain_test

import (
	"testing"

	"github.com/cxio/evidcoin/internal/blockchain"
	"github.com/cxio/evidcoin/pkg/types"
)

// makeGenesis 创建测试用创世区块头。
func makeGenesis() *blockchain.BlockHeader {
	return &blockchain.BlockHeader{
		Version:   1,
		Height:    0,
		PrevBlock: types.Hash384{}, // 创世块前一哈希为零
		CheckRoot: types.Hash384{1},
		Stakes:    0,
	}
}

// makeNext 在 parent 后创建合法的下一区块头。
func makeNext(parent *blockchain.BlockHeader) *blockchain.BlockHeader {
	return &blockchain.BlockHeader{
		Version:   1,
		Height:    parent.Height + 1,
		PrevBlock: parent.Hash(),
		CheckRoot: types.Hash384{byte(parent.Height + 2)},
		Stakes:    100,
	}
}

// TestSubmitBlockSuccess 测试正常提交区块。
func TestSubmitBlockSuccess(t *testing.T) {
	genesis := makeGenesis()
	bc, err := blockchain.New(blockchain.Config{Genesis: genesis})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	block1 := makeNext(genesis)
	if err := bc.SubmitBlock(block1); err != nil {
		t.Errorf("SubmitBlock() error = %v", err)
	}
}

// TestSubmitBlockConflict 测试同高度提交返回 ErrConflict。
func TestSubmitBlockConflict(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	block1 := makeNext(genesis)
	bc.SubmitBlock(block1)

	// 再次提交同一区块
	err := bc.SubmitBlock(block1)
	if err == nil {
		t.Error("expected ErrConflict, got nil")
	}
}

// TestSubmitBlockInvalidPrevBlock 测试 PrevBlock 不匹配被拒绝。
func TestSubmitBlockInvalidPrevBlock(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	bad := &blockchain.BlockHeader{
		Version:   1,
		Height:    1,
		PrevBlock: types.Hash384{0xff}, // 错误的前一哈希
		CheckRoot: types.Hash384{1},
	}
	err := bc.SubmitBlock(bad)
	if err == nil {
		t.Error("expected error for invalid PrevBlock, got nil")
	}
}

// TestSubmitBlockInvalidHeight 测试非连续高度被拒绝。
func TestSubmitBlockInvalidHeight(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	bad := &blockchain.BlockHeader{
		Version:   1,
		Height:    5, // 应为 1
		PrevBlock: genesis.Hash(),
		CheckRoot: types.Hash384{1},
	}
	err := bc.SubmitBlock(bad)
	if err == nil {
		t.Error("expected error for invalid height, got nil")
	}
}

// TestChainHeightAfterSubmit 测试提交后链高度正确更新。
func TestChainHeightAfterSubmit(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	if h := bc.ChainHeight(); h != 0 {
		t.Errorf("initial height = %d, want 0", h)
	}
	bc.SubmitBlock(makeNext(genesis))
	if h := bc.ChainHeight(); h != 1 {
		t.Errorf("height after submit = %d, want 1", h)
	}
}

// TestHeaderByHeight 测试按高度查询区块头。
func TestHeaderByHeight(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})
	h, err := bc.HeaderByHeight(0)
	if err != nil {
		t.Fatalf("HeaderByHeight(0) error = %v", err)
	}
	if h.Height != 0 {
		t.Errorf("Height = %d, want 0", h.Height)
	}
}

// TestHeaderByHash 测试按哈希查询区块头。
func TestHeaderByHash(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})
	hash := genesis.Hash()
	h, err := bc.HeaderByHash(hash)
	if err != nil {
		t.Fatalf("HeaderByHash() error = %v", err)
	}
	if h.Hash() != hash {
		t.Error("HeaderByHash returned wrong header")
	}
}

// TestBlockTimeDeterminism 测试区块时间戳计算确定性。
func TestBlockTimeDeterminism(t *testing.T) {
	genesis := int64(1700000000000)
	t1 := blockchain.BlockTime(genesis, 0)
	t2 := blockchain.BlockTime(genesis, 0)
	if t1 != t2 {
		t.Error("BlockTime is not deterministic")
	}
	t3 := blockchain.BlockTime(genesis, 1)
	expected := genesis + int64(types.BlockInterval.Milliseconds())
	if t3 != expected {
		t.Errorf("BlockTime(1) = %d, want %d", t3, expected)
	}
}

// TestIsYearBlock 测试年块判断逻辑。
func TestIsYearBlock(t *testing.T) {
	cases := []struct {
		height int32
		want   bool
	}{
		{0, false},
		{1, false},
		{int32(types.BlocksPerYear), true},
		{int32(types.BlocksPerYear * 2), true},
		{int32(types.BlocksPerYear) + 1, false},
	}
	for _, tc := range cases {
		h := &blockchain.BlockHeader{Height: tc.height}
		if got := h.IsYearBlock(); got != tc.want {
			t.Errorf("Height %d IsYearBlock() = %v, want %v", tc.height, got, tc.want)
		}
	}
}

// TestSubscribeNewBlock 测试新区块订阅通知。
func TestSubscribeNewBlock(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	ch := make(chan *blockchain.BlockHeader, 1)
	bc.Subscribe(ch)

	block1 := makeNext(genesis)
	bc.SubmitBlock(block1)

	select {
	case received := <-ch:
		if received.Height != 1 {
			t.Errorf("received block height = %d, want 1", received.Height)
		}
	default:
		t.Error("expected notification on channel, got none")
	}
}

// TestSwitchChainValidates 测试 SwitchChain 验证区块头连续性。
func TestSwitchChainValidates(t *testing.T) {
	genesis := makeGenesis()
	bc, _ := blockchain.New(blockchain.Config{Genesis: genesis})

	// 构造一个不连续的替代链
	bad := []*blockchain.BlockHeader{
		{Version: 1, Height: 1, PrevBlock: genesis.Hash(), CheckRoot: types.Hash384{9}},
		{Version: 1, Height: 3, PrevBlock: types.Hash384{0xab}, CheckRoot: types.Hash384{10}}, // 高度不连续
	}
	err := bc.SwitchChain(0, bad)
	if err == nil {
		t.Error("expected error for non-contiguous fork chain, got nil")
	}
}
```

**Step 2: 运行所有测试**

```bash
go test ./internal/blockchain/... -v
```
预期：所有测试 PASS。

**Step 3: 检查覆盖率**

```bash
go test ./internal/blockchain/... -cover
```
预期：覆盖率 ≥ 80%。

**Step 4: Commit**

```bash
git add internal/blockchain/
git commit -m "feat: implement blockchain core with header validation, storage, and subscription"
```

---

## Task 8: 整体验收

**Step 1: 完整构建**

```bash
go build ./...
```

**Step 2: 全量测试**

```bash
go test ./... -cover
```

**Step 3: 代码格式与静态分析**

```bash
go fmt ./... && golangci-lint run ./internal/blockchain/...
```

---

## 注意事项

1. **创世块哈希**：创世块的 `PrevBlock` 为全零哈希，这是惯例设计。`validateHeader` 中对高度 0 的区块不执行 PrevBlock 连续性检查。

2. **年块验证宽松**：初始同步阶段，前一年块可能尚未在本地存储，此时验证逻辑暂允许跳过年块引用检查，避免同步阻塞。

3. **存储后端**：本阶段仅实现内存存储（`MemoryStore`）。阶段 8 引入持久化存储（如 LevelDB/BadgerDB）时，只需实现 `HeaderStore` 接口即可无缝替换。

4. **Blockqs stub**：本阶段的 `noopBlockqsClient` 总返回 `ErrBlockqsUnavailable`，真实网络实现将在阶段 8 完成。

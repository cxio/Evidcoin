# Phase 2：区块链核心 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现极简的区块头链管理——包括区块头结构、BlockID 计算、HeaderStore 接口与内存实现、区块提交/查询、年块机制、事件订阅、分叉切换、初始验证框架。

**Architecture:** `internal/blockchain` 包，仅依赖 `pkg/types` 和 `pkg/crypto`。Core 不包含共识或交易校验逻辑，仅负责区块头链的结构性管理。外部组件通过 SubmitBlock 接口提交已验证的区块头。

**Tech Stack:** Go 1.25+, pkg/types (Hash512, constants), pkg/crypto (SHA512Sum)

---

## 前置依赖

本 Phase 假设 Phase 1 已完成，以下类型和函数已可用：

```go
// pkg/types
type Hash512 [64]byte                // 64 字节哈希
func (h Hash512) IsZero() bool       // 判断是否全零
const HashLength = 64                 // 哈希长度
const BlockInterval = 6 * time.Minute // 出块间隔（6 分钟）
const BlocksPerYear = 87661           // 每年区块数

// pkg/crypto
func SHA512Sum(data []byte) Hash512   // SHA-512 哈希计算
```

> **注意：** 如果 Phase 1 的具体 API 与以上描述有差异，请在实现时以 `pkg/types` 和 `pkg/crypto` 的实际导出符号为准，适当调整本计划中的调用方式。

---

## Task 1: 区块头结构与 BlockID 计算

**Files:**
- Create: `internal/blockchain/header.go`
- Create: `internal/blockchain/header_test.go`
- Create: `config/genesis.go`

本 Task 实现 `BlockHeader` 结构体、二进制序列化、BlockID 计算、基本字段验证，以及创世块硬编码常量。

### Step 1: 写失败测试

创建 `internal/blockchain/header_test.go`：

```go
package blockchain

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 测试常规区块头序列化为 140 字节
func TestBlockHeader_Serialize_Regular(t *testing.T) {
	h := &BlockHeader{
		Version:   1,
		PrevBlock: types.Hash512{1, 2, 3},
		CheckRoot: types.Hash512{4, 5, 6},
		Stakes:    100,
		Height:    10,
	}

	data := h.Serialize()
	if len(data) != 140 {
		t.Errorf("Serialize() length = %d, want 140", len(data))
	}
}

// 测试年块区块头序列化为 204 字节（140 + 64）
func TestBlockHeader_Serialize_YearBlock(t *testing.T) {
	h := &BlockHeader{
		Version:   1,
		PrevBlock: types.Hash512{1, 2, 3},
		CheckRoot: types.Hash512{4, 5, 6},
		Stakes:    200,
		Height:    87661, // 第一个年块
		YearBlock: types.Hash512{7, 8, 9},
	}

	data := h.Serialize()
	if len(data) != 204 {
		t.Errorf("Serialize() length = %d, want 204", len(data))
	}
}

// 测试 BlockID 计算：对序列化数据做 SHA-512
func TestBlockHeader_BlockID(t *testing.T) {
	h := &BlockHeader{
		Version:   1,
		PrevBlock: types.Hash512{1, 2, 3},
		CheckRoot: types.Hash512{4, 5, 6},
		Stakes:    100,
		Height:    10,
	}

	id := h.BlockID()
	if id.IsZero() {
		t.Error("BlockID() returned zero hash")
	}

	// 手动验证：对序列化数据做 SHA-512 应得到相同结果
	expected := crypto.SHA512Sum(h.Serialize())
	if id != expected {
		t.Error("BlockID() != SHA512Sum(Serialize())")
	}
}

// 测试 IsYearBlock 判断
func TestBlockHeader_IsYearBlock(t *testing.T) {
	tests := []struct {
		name   string
		height int32
		want   bool
	}{
		{"genesis", 0, true},
		{"regular", 100, false},
		{"first_year", 87661, true},
		{"second_year", 87661 * 2, true},
		{"not_year", 87660, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &BlockHeader{Height: tt.height}
			if got := h.IsYearBlock(); got != tt.want {
				t.Errorf("IsYearBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 测试 BlockTime 计算
func TestBlockTime(t *testing.T) {
	// Height=0 时应返回创世时间戳
	t0 := BlockTime(0)
	if t0 != GenesisTimestamp {
		t.Errorf("BlockTime(0) = %d, want GenesisTimestamp %d", t0, GenesisTimestamp)
	}

	// Height=1 时应返回创世时间戳 + 6 分钟
	t1 := BlockTime(1)
	expected := GenesisTimestamp + int64(6*time.Minute/time.Second)
	if t1 != expected {
		t.Errorf("BlockTime(1) = %d, want %d", t1, expected)
	}

	// Height=10 时应返回创世时间戳 + 60 分钟
	t10 := BlockTime(10)
	expected10 := GenesisTimestamp + 10*int64(6*time.Minute/time.Second)
	if t10 != expected10 {
		t.Errorf("BlockTime(10) = %d, want %d", t10, expected10)
	}
}

// 测试 Validate 基本字段验证
func TestBlockHeader_Validate(t *testing.T) {
	validPrev := types.Hash512{1, 2, 3}
	validCheck := types.Hash512{4, 5, 6}

	tests := []struct {
		name    string
		header  *BlockHeader
		wantErr bool
	}{
		{
			name: "valid_regular",
			header: &BlockHeader{
				Version:   1,
				PrevBlock: validPrev,
				CheckRoot: validCheck,
				Stakes:    100,
				Height:    10,
			},
			wantErr: false,
		},
		{
			name: "negative_stakes",
			header: &BlockHeader{
				Version:   1,
				PrevBlock: validPrev,
				CheckRoot: validCheck,
				Stakes:    -1,
				Height:    10,
			},
			wantErr: true,
		},
		{
			name: "zero_checkroot",
			header: &BlockHeader{
				Version:   1,
				PrevBlock: validPrev,
				CheckRoot: types.Hash512{}, // 全零
				Stakes:    100,
				Height:    10,
			},
			wantErr: true,
		},
		{
			name: "negative_height",
			header: &BlockHeader{
				Version:   1,
				PrevBlock: validPrev,
				CheckRoot: validCheck,
				Stakes:    100,
				Height:    -1,
			},
			wantErr: true,
		},
		{
			name: "genesis_zero_prevblock",
			header: &BlockHeader{
				Version:   1,
				PrevBlock: types.Hash512{}, // 创世块允许零 PrevBlock
				CheckRoot: validCheck,
				Stakes:    0,
				Height:    0,
			},
			wantErr: false,
		},
		{
			name: "non_genesis_zero_prevblock",
			header: &BlockHeader{
				Version:   1,
				PrevBlock: types.Hash512{}, // 非创世块不允许零 PrevBlock
				CheckRoot: validCheck,
				Stakes:    100,
				Height:    5,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.header.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 测试创世块常量已定义
func TestGenesisBlock(t *testing.T) {
	if GenesisTimestamp == 0 {
		t.Error("GenesisTimestamp is zero")
	}
	if GenesisBlock.Height != 0 {
		t.Errorf("GenesisBlock.Height = %d, want 0", GenesisBlock.Height)
	}
	if GenesisBlock.CheckRoot.IsZero() {
		t.Error("GenesisBlock.CheckRoot is zero")
	}
	if !GenesisBlock.PrevBlock.IsZero() {
		t.Error("GenesisBlock.PrevBlock should be zero")
	}

	// 创世块的 BlockID 应为非零
	id := GenesisBlock.BlockID()
	if id.IsZero() {
		t.Error("GenesisBlock.BlockID() is zero")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/blockchain/ -run "TestBlockHeader|TestBlockTime|TestGenesisBlock"
```

预期输出：编译失败，`BlockHeader` 等类型未定义。

### Step 3: 写最小实现

创建 `internal/blockchain/header.go`：

```go
package blockchain

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 区块头版本号范围
const (
	MinVersion = 1 // 最小协议版本
	MaxVersion = 1 // 当前最大协议版本
)

// 区块头序列化大小
const (
	HeaderBaseSize = 140 // 常规区块头大小：4+64+64+4+4
	YearBlockSize  = 64  // 年块字段大小
)

// BlockHeader 区块头结构。
// 区块头是区块链的核心数据结构，通过 PrevBlock 字段形成链式结构。
// 常规大小 140 字节，年块额外增加 64 字节。
type BlockHeader struct {
	Version   int32        // 协议版本号
	PrevBlock types.Hash512 // 前一区块哈希
	CheckRoot types.Hash512 // 校验根 = SHA-512(TxTreeRoot || UTXOFingerprint || UTCOFingerprint)
	Stakes    int32        // 币权销毁量（币*天）
	Height    int32        // 区块高度（从 0 开始）
	YearBlock types.Hash512 // 仅当 Height % BlocksPerYear == 0 时存在
}

// Serialize 将区块头序列化为二进制格式。
// 格式：Version(4) + PrevBlock(64) + CheckRoot(64) + Stakes(4) + Height(4) + [YearBlock(64)]
func (h *BlockHeader) Serialize() []byte {
	size := HeaderBaseSize
	if h.IsYearBlock() {
		size += YearBlockSize
	}

	buf := make([]byte, size)
	offset := 0

	// Version: 4 字节，小端序
	binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(h.Version))
	offset += 4

	// PrevBlock: 64 字节
	copy(buf[offset:offset+types.HashLength], h.PrevBlock[:])
	offset += types.HashLength

	// CheckRoot: 64 字节
	copy(buf[offset:offset+types.HashLength], h.CheckRoot[:])
	offset += types.HashLength

	// Stakes: 4 字节，小端序
	binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(h.Stakes))
	offset += 4

	// Height: 4 字节，小端序
	binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(h.Height))
	offset += 4

	// YearBlock: 年块时追加 64 字节
	if h.IsYearBlock() {
		copy(buf[offset:offset+types.HashLength], h.YearBlock[:])
	}

	return buf
}

// BlockID 计算区块 ID。
// 区块 ID = SHA-512(Serialize())
func (h *BlockHeader) BlockID() types.Hash512 {
	return crypto.SHA512Sum(h.Serialize())
}

// IsYearBlock 判断该区块是否为年块。
// 当 Height 是 BlocksPerYear 的整数倍时为年块（包括创世块）。
func (h *BlockHeader) IsYearBlock() bool {
	return h.Height >= 0 && h.Height%types.BlocksPerYear == 0
}

// BlockTime 计算指定高度的区块时间戳（Unix 毫秒）。
// BlockTime = GenesisTimestamp + Height × BlockInterval
func BlockTime(height int32) int64 {
	return GenesisTimestamp + int64(height)*int64(types.BlockInterval/time.Millisecond)
}

// 区块头基本字段验证错误
var (
	ErrNegativeHeight    = errors.New("block height is negative")
	ErrNegativeStakes    = errors.New("block stakes is negative")
	ErrZeroCheckRoot     = errors.New("block checkroot is zero")
	ErrZeroPrevBlock     = errors.New("non-genesis block has zero prevblock")
	ErrInvalidVersion    = errors.New("block version out of range")
)

// Validate 对区块头执行基本字段验证。
// 仅检查字段自身的合法性，不验证与链上其他区块的关系。
func (h *BlockHeader) Validate() error {
	// 高度非负
	if h.Height < 0 {
		return ErrNegativeHeight
	}

	// 版本号范围
	if h.Version < MinVersion || h.Version > MaxVersion {
		return ErrInvalidVersion
	}

	// Stakes 非负
	if h.Stakes < 0 {
		return ErrNegativeStakes
	}

	// CheckRoot 非零（创世块也应有 CheckRoot）
	if h.CheckRoot.IsZero() {
		return ErrZeroCheckRoot
	}

	// 非创世块的 PrevBlock 不能为零
	if h.Height > 0 && h.PrevBlock.IsZero() {
		return ErrZeroPrevBlock
	}

	return nil
}
```

创建 `config/genesis.go`：

```go
package config

import (
	"github.com/cxio/evidcoin/pkg/types"
)

// 创世时间戳（Unix 毫秒）。
// 约定为 2026-06-01 12:34:56.789 UTC。
const GenesisTimestamp int64 = 1780317296789
```

在 `internal/blockchain/header.go` 中添加创世块相关定义（也可以放在单独文件 `genesis.go` 中，此处集中到 blockchain 包内以便引用）：

创建 `internal/blockchain/genesis.go`：

```go
package blockchain

import (
	"github.com/cxio/evidcoin/pkg/types"
)

// GenesisTimestamp 创世时间戳（Unix 毫秒）。
// 2026-06-01 00:00:00 UTC
const GenesisTimestamp int64 = 1780317296789

// GenesisBlock 创世区块头。
// PrevBlock 为零值（无前置区块），Height 为 0，YearBlock 为零值（首个年块无前置年块）。
var GenesisBlock = BlockHeader{
	Version:   1,
	PrevBlock: types.Hash512{},    // 无前置区块
	CheckRoot: genesisCheckRoot(), // 创世校验根
	Stakes:    0,                  // 创世块无币权销毁
	Height:    0,
	YearBlock: types.Hash512{},    // 首个年块，无前置年块引用
}

// genesisCheckRoot 返回创世块的校验根。
// 创世块的 CheckRoot 为硬编码常量值。
func genesisCheckRoot() types.Hash512 {
	var h types.Hash512
	// 使用固定的创世标识字节填充
	// 实际值将在网络启动前确定
	copy(h[:], []byte("evidcoin-genesis-checkroot-2026-06-01"))
	return h
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/blockchain/ -run "TestBlockHeader|TestBlockTime|TestGenesisBlock"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/blockchain/header.go internal/blockchain/header_test.go internal/blockchain/genesis.go
git commit -m "feat(blockchain): add BlockHeader struct, BlockID, serialization, and genesis constants"
```

---

## Task 2: HeaderStore 接口与内存实现

**Files:**
- Create: `internal/blockchain/store.go`
- Create: `internal/blockchain/store_test.go`

本 Task 定义 HeaderStore 存储接口并提供基于内存的实现。

### Step 1: 写失败测试

创建 `internal/blockchain/store_test.go`：

```go
package blockchain

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：创建测试用区块头
func makeTestHeader(height int32, prevBlock types.Hash512) *BlockHeader {
	var checkRoot types.Hash512
	checkRoot[0] = byte(height + 1) // 确保非零
	return &BlockHeader{
		Version:   1,
		PrevBlock: prevBlock,
		CheckRoot: checkRoot,
		Stakes:    int32(height) * 10,
		Height:    height,
	}
}

// 测试 Put 和 Get
func TestMemoryHeaderStore_PutGet(t *testing.T) {
	store := NewMemoryHeaderStore()
	h := makeTestHeader(5, types.Hash512{1})

	if err := store.Put(h); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, err := store.Get(5)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Height != 5 {
		t.Errorf("Get().Height = %d, want 5", got.Height)
	}
}

// 测试 GetByHash
func TestMemoryHeaderStore_GetByHash(t *testing.T) {
	store := NewMemoryHeaderStore()
	h := makeTestHeader(3, types.Hash512{1})

	if err := store.Put(h); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	hash := h.BlockID()
	got, err := store.GetByHash(hash)
	if err != nil {
		t.Fatalf("GetByHash() error = %v", err)
	}
	if got.Height != 3 {
		t.Errorf("GetByHash().Height = %d, want 3", got.Height)
	}
}

// 测试 Has
func TestMemoryHeaderStore_Has(t *testing.T) {
	store := NewMemoryHeaderStore()
	h := makeTestHeader(7, types.Hash512{1})

	if store.Has(7) {
		t.Error("Has(7) should be false before Put")
	}

	if err := store.Put(h); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	if !store.Has(7) {
		t.Error("Has(7) should be true after Put")
	}
}

// 测试 Tip：返回最高高度的区块头
func TestMemoryHeaderStore_Tip(t *testing.T) {
	store := NewMemoryHeaderStore()

	// 空存储时 Tip 应返回错误
	_, err := store.Tip()
	if err == nil {
		t.Error("Tip() on empty store should return error")
	}

	// 依次存入不同高度的区块头
	h0 := makeTestHeader(0, types.Hash512{})
	h1 := makeTestHeader(1, h0.BlockID())
	h2 := makeTestHeader(2, h1.BlockID())

	for _, h := range []*BlockHeader{h0, h1, h2} {
		if err := store.Put(h); err != nil {
			t.Fatalf("Put(height=%d) error = %v", h.Height, err)
		}
	}

	tip, err := store.Tip()
	if err != nil {
		t.Fatalf("Tip() error = %v", err)
	}
	if tip.Height != 2 {
		t.Errorf("Tip().Height = %d, want 2", tip.Height)
	}
}

// 测试 Get 不存在的高度
func TestMemoryHeaderStore_GetNotFound(t *testing.T) {
	store := NewMemoryHeaderStore()
	_, err := store.Get(999)
	if err == nil {
		t.Error("Get(999) on empty store should return error")
	}
}

// 测试 Delete
func TestMemoryHeaderStore_Delete(t *testing.T) {
	store := NewMemoryHeaderStore()
	h := makeTestHeader(5, types.Hash512{1})

	if err := store.Put(h); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if !store.Has(5) {
		t.Fatal("Has(5) should be true after Put")
	}

	if err := store.Delete(5); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.Has(5) {
		t.Error("Has(5) should be false after Delete")
	}

	// 按哈希也应查不到
	hash := h.BlockID()
	_, err := store.GetByHash(hash)
	if err == nil {
		t.Error("GetByHash after Delete should return error")
	}
}

// 测试 YearBlock
func TestMemoryHeaderStore_YearBlock(t *testing.T) {
	store := NewMemoryHeaderStore()

	// 存入年块
	h := makeTestHeader(87661, types.Hash512{1}) // year=1 的年块
	if err := store.Put(h); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, err := store.YearBlock(1)
	if err != nil {
		t.Fatalf("YearBlock(1) error = %v", err)
	}
	if got.Height != 87661 {
		t.Errorf("YearBlock(1).Height = %d, want 87661", got.Height)
	}

	// 查询不存在的年块
	_, err = store.YearBlock(99)
	if err == nil {
		t.Error("YearBlock(99) should return error")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/blockchain/ -run "TestMemoryHeaderStore"
```

预期输出：编译失败，`HeaderStore`、`MemoryHeaderStore` 等未定义。

### Step 3: 写最小实现

创建 `internal/blockchain/store.go`：

```go
package blockchain

import (
	"errors"
	"sync"

	"github.com/cxio/evidcoin/pkg/types"
)

// 存储相关错误
var (
	ErrHeaderNotFound = errors.New("header not found")
	ErrEmptyChain     = errors.New("chain is empty")
)

// HeaderStore 区块头存储接口。
type HeaderStore interface {
	// Get 按高度获取区块头。
	Get(height int) (*BlockHeader, error)

	// GetByHash 按区块哈希获取区块头。
	GetByHash(hash types.Hash512) (*BlockHeader, error)

	// Put 存储一个区块头。
	Put(header *BlockHeader) error

	// Delete 按高度删除区块头。
	Delete(height int) error

	// Has 检查指定高度的区块头是否存在。
	Has(height int) bool

	// Tip 返回当前最高高度的区块头。
	Tip() (*BlockHeader, error)

	// YearBlock 获取指定年份的年块区块头。
	// year 从 0 开始，year=0 表示创世块，year=1 表示 Height=87661 的区块。
	YearBlock(year int) (*BlockHeader, error)
}

// MemoryHeaderStore 基于内存的区块头存储实现。
// 适用于测试和轻客户端场景。
type MemoryHeaderStore struct {
	mu        sync.RWMutex
	byHeight  map[int]*BlockHeader
	byHash    map[types.Hash512]*BlockHeader
	tipHeight int // 当前最高高度，-1 表示空链
}

// NewMemoryHeaderStore 创建一个新的内存区块头存储。
func NewMemoryHeaderStore() *MemoryHeaderStore {
	return &MemoryHeaderStore{
		byHeight:  make(map[int]*BlockHeader),
		byHash:    make(map[types.Hash512]*BlockHeader),
		tipHeight: -1,
	}
}

// Get 按高度获取区块头。
func (s *MemoryHeaderStore) Get(height int) (*BlockHeader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	h, ok := s.byHeight[height]
	if !ok {
		return nil, ErrHeaderNotFound
	}
	return h, nil
}

// GetByHash 按区块哈希获取区块头。
func (s *MemoryHeaderStore) GetByHash(hash types.Hash512) (*BlockHeader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	h, ok := s.byHash[hash]
	if !ok {
		return nil, ErrHeaderNotFound
	}
	return h, nil
}

// Put 存储一个区块头。
// 同时建立高度索引和哈希索引，并更新链顶高度。
func (s *MemoryHeaderStore) Put(header *BlockHeader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	height := int(header.Height)
	hash := header.BlockID()

	s.byHeight[height] = header
	s.byHash[hash] = header

	if height > s.tipHeight {
		s.tipHeight = height
	}

	return nil
}

// Delete 按高度删除区块头。
func (s *MemoryHeaderStore) Delete(height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	h, ok := s.byHeight[height]
	if !ok {
		return ErrHeaderNotFound
	}

	hash := h.BlockID()
	delete(s.byHeight, height)
	delete(s.byHash, hash)

	// 如果删除的是链顶，向下查找新的链顶
	if height == s.tipHeight {
		s.tipHeight = -1
		for h := range s.byHeight {
			if h > s.tipHeight {
				s.tipHeight = h
			}
		}
	}

	return nil
}

// Has 检查指定高度的区块头是否存在。
func (s *MemoryHeaderStore) Has(height int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.byHeight[height]
	return ok
}

// Tip 返回当前最高高度的区块头。
func (s *MemoryHeaderStore) Tip() (*BlockHeader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tipHeight < 0 {
		return nil, ErrEmptyChain
	}
	return s.byHeight[s.tipHeight], nil
}

// YearBlock 获取指定年份的年块区块头。
func (s *MemoryHeaderStore) YearBlock(year int) (*BlockHeader, error) {
	height := year * types.BlocksPerYear
	return s.Get(height)
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/blockchain/ -run "TestMemoryHeaderStore"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/blockchain/store.go internal/blockchain/store_test.go
git commit -m "feat(blockchain): add HeaderStore interface and MemoryHeaderStore implementation"
```

---

## Task 3: Blockchain 核心结构

**Files:**
- Create: `internal/blockchain/blockchain.go`
- Create: `internal/blockchain/blockchain_test.go`

本 Task 实现 Blockchain 核心结构体、构造函数、区块提交（含结构性验证）、查询方法。

### Step 1: 写失败测试

创建 `internal/blockchain/blockchain_test.go`：

```go
package blockchain

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：创建初始化好的 Blockchain 实例
func newTestBlockchain(t *testing.T) *Blockchain {
	t.Helper()
	store := NewMemoryHeaderStore()
	bc, err := New(store, &GenesisBlock)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return bc
}

// 辅助函数：创建可提交的下一个区块头
func nextHeader(bc *Blockchain, height int32) *BlockHeader {
	tip, _ := bc.ChainTip()
	var checkRoot types.Hash512
	checkRoot[0] = byte(height + 1)

	h := &BlockHeader{
		Version:   1,
		PrevBlock: tip.BlockID(),
		CheckRoot: checkRoot,
		Stakes:    int32(height) * 5,
		Height:    height,
	}
	return h
}

// 测试构造函数自动存入创世块
func TestNew(t *testing.T) {
	bc := newTestBlockchain(t)

	if bc.ChainHeight() != 0 {
		t.Errorf("ChainHeight() = %d, want 0", bc.ChainHeight())
	}

	tip, err := bc.ChainTip()
	if err != nil {
		t.Fatalf("ChainTip() error = %v", err)
	}
	if tip.Height != 0 {
		t.Errorf("ChainTip().Height = %d, want 0", tip.Height)
	}
}

// 测试成功提交区块
func TestBlockchain_SubmitBlock(t *testing.T) {
	bc := newTestBlockchain(t)
	h := nextHeader(bc, 1)

	if err := bc.SubmitBlock(h); err != nil {
		t.Fatalf("SubmitBlock() error = %v", err)
	}

	if bc.ChainHeight() != 1 {
		t.Errorf("ChainHeight() = %d, want 1", bc.ChainHeight())
	}
}

// 测试 PrevBlock 不匹配时拒绝
func TestBlockchain_SubmitBlock_WrongPrevBlock(t *testing.T) {
	bc := newTestBlockchain(t)

	h := &BlockHeader{
		Version:   1,
		PrevBlock: types.Hash512{99, 99, 99}, // 错误的 PrevBlock
		CheckRoot: types.Hash512{1},
		Stakes:    10,
		Height:    1,
	}
	err := bc.SubmitBlock(h)
	if err == nil {
		t.Error("SubmitBlock() should fail with wrong PrevBlock")
	}
}

// 测试 Height 不连续时拒绝
func TestBlockchain_SubmitBlock_WrongHeight(t *testing.T) {
	bc := newTestBlockchain(t)

	tip, _ := bc.ChainTip()
	h := &BlockHeader{
		Version:   1,
		PrevBlock: tip.BlockID(),
		CheckRoot: types.Hash512{1},
		Stakes:    10,
		Height:    5, // 应该是 1
	}
	err := bc.SubmitBlock(h)
	if err == nil {
		t.Error("SubmitBlock() should fail with non-sequential height")
	}
}

// 测试重复提交同高度区块被拒绝
func TestBlockchain_SubmitBlock_Duplicate(t *testing.T) {
	bc := newTestBlockchain(t)
	h := nextHeader(bc, 1)

	if err := bc.SubmitBlock(h); err != nil {
		t.Fatalf("first SubmitBlock() error = %v", err)
	}

	// 重复提交
	err := bc.SubmitBlock(h)
	if err == nil {
		t.Error("SubmitBlock() should fail on duplicate height")
	}
}

// 测试无效版本号被拒绝
func TestBlockchain_SubmitBlock_InvalidVersion(t *testing.T) {
	bc := newTestBlockchain(t)
	tip, _ := bc.ChainTip()

	h := &BlockHeader{
		Version:   999, // 无效版本
		PrevBlock: tip.BlockID(),
		CheckRoot: types.Hash512{1},
		Stakes:    10,
		Height:    1,
	}
	err := bc.SubmitBlock(h)
	if err == nil {
		t.Error("SubmitBlock() should fail with invalid version")
	}
}

// 测试按高度查询
func TestBlockchain_HeaderByHeight(t *testing.T) {
	bc := newTestBlockchain(t)
	h := nextHeader(bc, 1)
	bc.SubmitBlock(h)

	got, err := bc.HeaderByHeight(1)
	if err != nil {
		t.Fatalf("HeaderByHeight() error = %v", err)
	}
	if got.Height != 1 {
		t.Errorf("HeaderByHeight(1).Height = %d, want 1", got.Height)
	}
}

// 测试按哈希查询
func TestBlockchain_HeaderByHash(t *testing.T) {
	bc := newTestBlockchain(t)
	h := nextHeader(bc, 1)
	bc.SubmitBlock(h)

	hash := h.BlockID()
	got, err := bc.HeaderByHash(hash)
	if err != nil {
		t.Fatalf("HeaderByHash() error = %v", err)
	}
	if got.Height != 1 {
		t.Errorf("HeaderByHash().Height = %d, want 1", got.Height)
	}
}

// 测试连续提交多个区块
func TestBlockchain_SubmitMultipleBlocks(t *testing.T) {
	bc := newTestBlockchain(t)

	for i := int32(1); i <= 10; i++ {
		h := nextHeader(bc, i)
		if err := bc.SubmitBlock(h); err != nil {
			t.Fatalf("SubmitBlock(height=%d) error = %v", i, err)
		}
	}

	if bc.ChainHeight() != 10 {
		t.Errorf("ChainHeight() = %d, want 10", bc.ChainHeight())
	}
}

// 测试 ReplaceBlock 替换链顶
func TestBlockchain_ReplaceBlock(t *testing.T) {
	bc := newTestBlockchain(t)

	// 先提交 height=1
	h1 := nextHeader(bc, 1)
	bc.SubmitBlock(h1)

	// 用不同内容替换 height=1
	tip0, _ := bc.HeaderByHeight(0)
	replacement := &BlockHeader{
		Version:   1,
		PrevBlock: tip0.BlockID(),
		CheckRoot: types.Hash512{99, 88, 77}, // 不同的 CheckRoot
		Stakes:    999,
		Height:    1,
	}

	err := bc.ReplaceBlock(1, replacement)
	if err != nil {
		t.Fatalf("ReplaceBlock() error = %v", err)
	}

	// 确认已替换
	got, _ := bc.HeaderByHeight(1)
	if got.Stakes != 999 {
		t.Errorf("replaced block Stakes = %d, want 999", got.Stakes)
	}
}

// 测试 ReplaceBlock 只能替换链顶
func TestBlockchain_ReplaceBlock_NotTip(t *testing.T) {
	bc := newTestBlockchain(t)

	h1 := nextHeader(bc, 1)
	bc.SubmitBlock(h1)
	h2 := nextHeader(bc, 2)
	bc.SubmitBlock(h2)

	// 尝试替换 height=1（不是链顶，链顶是 2）
	tip0, _ := bc.HeaderByHeight(0)
	replacement := &BlockHeader{
		Version:   1,
		PrevBlock: tip0.BlockID(),
		CheckRoot: types.Hash512{99},
		Stakes:    999,
		Height:    1,
	}

	err := bc.ReplaceBlock(1, replacement)
	if err == nil {
		t.Error("ReplaceBlock() should fail when not replacing tip")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/blockchain/ -run "TestNew|TestBlockchain_"
```

预期输出：编译失败，`Blockchain`、`New` 等未定义。

### Step 3: 写最小实现

创建 `internal/blockchain/blockchain.go`：

```go
package blockchain

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cxio/evidcoin/pkg/types"
)

// 区块链操作相关错误
var (
	ErrPrevBlockMismatch = errors.New("prevblock does not match chain tip")
	ErrHeightMismatch    = errors.New("block height is not sequential")
	ErrDuplicateHeight   = errors.New("block at this height already exists")
	ErrNotChainTip       = errors.New("can only replace chain tip")
	ErrYearBlockMismatch = errors.New("yearblock reference is incorrect")
)

// Blockchain 区块链核心结构。
// 管理区块头链的存储、验证与查询，不包含共识或交易校验逻辑。
type Blockchain struct {
	mu    sync.RWMutex
	store HeaderStore
}

// New 创建一个新的 Blockchain 实例。
// 将创世块存入存储。如果存储中已有创世块则验证一致性。
func New(store HeaderStore, genesis *BlockHeader) (*Blockchain, error) {
	bc := &Blockchain{
		store: store,
	}

	// 如果存储中已有创世块，验证一致性
	if store.Has(0) {
		existing, err := store.Get(0)
		if err != nil {
			return nil, fmt.Errorf("get existing genesis: %w", err)
		}
		if existing.BlockID() != genesis.BlockID() {
			return nil, errors.New("genesis block mismatch with existing chain")
		}
		return bc, nil
	}

	// 存入创世块
	if err := store.Put(genesis); err != nil {
		return nil, fmt.Errorf("store genesis block: %w", err)
	}

	return bc, nil
}

// SubmitBlock 提交一个新区块头到链上。
// 执行结构性验证：PrevBlock 一致性、Height 连续性、Version 兼容性、
// YearBlock 正确性（年度边界时）、基本字段合法性。
func (bc *Blockchain) SubmitBlock(header *BlockHeader) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 基本字段验证
	if err := header.Validate(); err != nil {
		return fmt.Errorf("validate header: %w", err)
	}

	// 获取当前链顶
	tip, err := bc.store.Tip()
	if err != nil {
		return fmt.Errorf("get chain tip: %w", err)
	}

	// Height 连续性
	expectedHeight := tip.Height + 1
	if header.Height != expectedHeight {
		return ErrHeightMismatch
	}

	// 同一高度不允许重复提交
	if bc.store.Has(int(header.Height)) {
		return ErrDuplicateHeight
	}

	// PrevBlock 一致性
	tipID := tip.BlockID()
	if header.PrevBlock != tipID {
		return ErrPrevBlockMismatch
	}

	// 年块验证
	if header.IsYearBlock() {
		year := int(header.Height) / types.BlocksPerYear
		if year > 0 {
			prevYear, err := bc.store.YearBlock(year - 1)
			if err != nil {
				return fmt.Errorf("get previous year block: %w", err)
			}
			if header.YearBlock != prevYear.BlockID() {
				return ErrYearBlockMismatch
			}
		}
	}

	// 存储
	if err := bc.store.Put(header); err != nil {
		return fmt.Errorf("store header: %w", err)
	}

	// 通知订阅者（见 Task 5 实现）
	bc.notifySubscribers(header)

	return nil
}

// ReplaceBlock 替换指定高度的区块头（用于分叉切换）。
// 要求目标高度必须是当前链顶。
func (bc *Blockchain) ReplaceBlock(height int, header *BlockHeader) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 基本字段验证
	if err := header.Validate(); err != nil {
		return fmt.Errorf("validate header: %w", err)
	}

	// 只能替换链顶
	tip, err := bc.store.Tip()
	if err != nil {
		return fmt.Errorf("get chain tip: %w", err)
	}
	if int(tip.Height) != height {
		return ErrNotChainTip
	}

	// PrevBlock 需指向 height-1 的区块
	if height > 0 {
		prev, err := bc.store.Get(height - 1)
		if err != nil {
			return fmt.Errorf("get previous block: %w", err)
		}
		if header.PrevBlock != prev.BlockID() {
			return ErrPrevBlockMismatch
		}
	}

	// 删除旧区块，存入新区块
	if err := bc.store.Delete(height); err != nil {
		return fmt.Errorf("delete old header: %w", err)
	}
	if err := bc.store.Put(header); err != nil {
		return fmt.Errorf("store replacement header: %w", err)
	}

	return nil
}

// HeaderByHeight 按高度查询区块头。
func (bc *Blockchain) HeaderByHeight(height int) (*BlockHeader, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.store.Get(height)
}

// HeaderByHash 按哈希查询区块头。
func (bc *Blockchain) HeaderByHash(hash types.Hash512) (*BlockHeader, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.store.GetByHash(hash)
}

// ChainTip 返回当前链顶区块头。
func (bc *Blockchain) ChainTip() (*BlockHeader, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.store.Tip()
}

// ChainHeight 返回当前链高度。
// 空链返回 -1。
func (bc *Blockchain) ChainHeight() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	tip, err := bc.store.Tip()
	if err != nil {
		return -1
	}
	return int(tip.Height)
}

// notifySubscribers 通知所有订阅者新区块已提交。
// 占位实现，将在 Task 5 中完善。
func (bc *Blockchain) notifySubscribers(header *BlockHeader) {
	// 见 Task 5
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/blockchain/ -run "TestNew|TestBlockchain_"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/blockchain/blockchain.go internal/blockchain/blockchain_test.go
git commit -m "feat(blockchain): add Blockchain core with SubmitBlock, ReplaceBlock, and query methods"
```

---

## Task 4: 年块机制

**Files:**
- Create: `internal/blockchain/yearblock.go`
- Create: `internal/blockchain/yearblock_test.go`

本 Task 实现年块相关的辅助函数和验证逻辑。

### Step 1: 写失败测试

创建 `internal/blockchain/yearblock_test.go`：

```go
package blockchain

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 测试 YearFromHeight 计算
func TestYearFromHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int32
		want   int
	}{
		{"genesis", 0, 0},
		{"before_first_year", 87660, 0},
		{"first_year", 87661, 1},
		{"mid_second_year", 100000, 1},
		{"second_year", 87661 * 2, 2},
		{"tenth_year", 87661 * 10, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := YearFromHeight(tt.height); got != tt.want {
				t.Errorf("YearFromHeight(%d) = %d, want %d", tt.height, got, tt.want)
			}
		})
	}
}

// 测试 IsYearBoundary 判断
func TestIsYearBoundary(t *testing.T) {
	tests := []struct {
		name   string
		height int32
		want   bool
	}{
		{"genesis", 0, true},
		{"regular", 100, false},
		{"first_year_boundary", 87661, true},
		{"just_before", 87660, false},
		{"just_after", 87662, false},
		{"second_year", 87661 * 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsYearBoundary(tt.height); got != tt.want {
				t.Errorf("IsYearBoundary(%d) = %v, want %v", tt.height, got, tt.want)
			}
		})
	}
}

// 测试 YearBlockHeight 计算
func TestYearBlockHeight(t *testing.T) {
	tests := []struct {
		name string
		year int
		want int32
	}{
		{"year_0", 0, 0},
		{"year_1", 1, 87661},
		{"year_5", 5, 87661 * 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := YearBlockHeight(tt.year); got != tt.want {
				t.Errorf("YearBlockHeight(%d) = %d, want %d", tt.year, got, tt.want)
			}
		})
	}
}

// 测试年块提交与验证：年块 YearBlock 字段应正确引用前一年块
func TestBlockchain_SubmitYearBlock(t *testing.T) {
	bc := newTestBlockchain(t)

	// 从 height=1 构建链直到 height=87661（第一个年块）
	for i := int32(1); i < types.BlocksPerYear; i++ {
		h := nextHeader(bc, i)
		if err := bc.SubmitBlock(h); err != nil {
			t.Fatalf("SubmitBlock(height=%d) error = %v", i, err)
		}
	}

	// 提交年块（height=87661）
	tip, _ := bc.ChainTip()
	genesis, _ := bc.HeaderByHeight(0)

	yearHeader := &BlockHeader{
		Version:   1,
		PrevBlock: tip.BlockID(),
		CheckRoot: types.Hash512{42},
		Stakes:    500,
		Height:    int32(types.BlocksPerYear),
		YearBlock: genesis.BlockID(), // 引用前一年块（创世块）
	}

	if err := bc.SubmitBlock(yearHeader); err != nil {
		t.Fatalf("SubmitBlock(yearblock) error = %v", err)
	}

	if bc.ChainHeight() != types.BlocksPerYear {
		t.Errorf("ChainHeight() = %d, want %d", bc.ChainHeight(), types.BlocksPerYear)
	}
}

// 测试年块 YearBlock 字段不正确时被拒绝
func TestBlockchain_SubmitYearBlock_WrongReference(t *testing.T) {
	bc := newTestBlockchain(t)

	// 构建链到 height=87660
	for i := int32(1); i < types.BlocksPerYear; i++ {
		h := nextHeader(bc, i)
		if err := bc.SubmitBlock(h); err != nil {
			t.Fatalf("SubmitBlock(height=%d) error = %v", i, err)
		}
	}

	// 提交带错误 YearBlock 的年块
	tip, _ := bc.ChainTip()
	yearHeader := &BlockHeader{
		Version:   1,
		PrevBlock: tip.BlockID(),
		CheckRoot: types.Hash512{42},
		Stakes:    500,
		Height:    int32(types.BlocksPerYear),
		YearBlock: types.Hash512{0xFF}, // 错误引用
	}

	err := bc.SubmitBlock(yearHeader)
	if err == nil {
		t.Error("SubmitBlock() should reject year block with wrong YearBlock reference")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/blockchain/ -run "TestYearFromHeight|TestIsYearBoundary|TestYearBlockHeight|TestBlockchain_SubmitYearBlock"
```

预期输出：编译失败，`YearFromHeight` 等未定义。

### Step 3: 写最小实现

创建 `internal/blockchain/yearblock.go`：

```go
package blockchain

import (
	"github.com/cxio/evidcoin/pkg/types"
)

// YearFromHeight 计算给定高度对应的年份（从 0 开始）。
// year = height / BlocksPerYear
func YearFromHeight(height int32) int {
	return int(height) / types.BlocksPerYear
}

// IsYearBoundary 判断给定高度是否为年度边界。
// 当 height 是 BlocksPerYear 的整数倍时为年度边界。
func IsYearBoundary(height int32) bool {
	return height >= 0 && int(height)%types.BlocksPerYear == 0
}

// YearBlockHeight 返回指定年份的年块高度。
// year=0 → height=0（创世块），year=1 → height=87661，以此类推。
func YearBlockHeight(year int) int32 {
	return int32(year * types.BlocksPerYear)
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/blockchain/ -run "TestYearFromHeight|TestIsYearBoundary|TestYearBlockHeight|TestBlockchain_SubmitYearBlock"
```

预期输出：全部 PASS。

> **注意：** 年块提交测试 `TestBlockchain_SubmitYearBlock` 需构建 87660 个区块，运行时间较长。
> 如需缩短测试时间，可考虑在测试中使用一个较小的 `BlocksPerYear` 值模拟。
> 但为确保真实场景的正确性，建议至少保留一个完整测试。
> 如果测试超时，可使用 `go test -timeout 300s` 延长超时时间。

### Step 5: 提交

```bash
git add internal/blockchain/yearblock.go internal/blockchain/yearblock_test.go
git commit -m "feat(blockchain): add year block helpers and year boundary validation"
```

---

## Task 5: 事件订阅

**Files:**
- Modify: `internal/blockchain/blockchain.go` — 添加订阅/取消订阅逻辑
- Create: `internal/blockchain/subscribe_test.go`

本 Task 实现新区块提交的事件通知机制。

### Step 1: 写失败测试

创建 `internal/blockchain/subscribe_test.go`：

```go
package blockchain

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// 测试订阅新区块事件
func TestBlockchain_Subscribe(t *testing.T) {
	bc := newTestBlockchain(t)

	ch := make(chan *BlockHeader, 10)
	bc.Subscribe(ch)

	// 提交一个区块
	h := nextHeader(bc, 1)
	if err := bc.SubmitBlock(h); err != nil {
		t.Fatalf("SubmitBlock() error = %v", err)
	}

	// 验证收到通知
	select {
	case got := <-ch:
		if got.Height != 1 {
			t.Errorf("received header Height = %d, want 1", got.Height)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for block notification")
	}
}

// 测试多个订阅者
func TestBlockchain_MultipleSubscribers(t *testing.T) {
	bc := newTestBlockchain(t)

	ch1 := make(chan *BlockHeader, 10)
	ch2 := make(chan *BlockHeader, 10)
	bc.Subscribe(ch1)
	bc.Subscribe(ch2)

	h := nextHeader(bc, 1)
	bc.SubmitBlock(h)

	// 两个订阅者都应收到通知
	for i, ch := range []chan *BlockHeader{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Height != 1 {
				t.Errorf("subscriber %d: Height = %d, want 1", i, got.Height)
			}
		case <-time.After(time.Second):
			t.Errorf("subscriber %d: timeout", i)
		}
	}
}

// 测试取消订阅
func TestBlockchain_Unsubscribe(t *testing.T) {
	bc := newTestBlockchain(t)

	ch := make(chan *BlockHeader, 10)
	bc.Subscribe(ch)
	bc.Unsubscribe(ch)

	h := nextHeader(bc, 1)
	bc.SubmitBlock(h)

	// 取消订阅后不应收到通知
	select {
	case <-ch:
		t.Error("should not receive notification after Unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// 预期超时，正确
	}
}

// 测试满 channel 不阻塞提交
func TestBlockchain_Subscribe_FullChannel(t *testing.T) {
	bc := newTestBlockchain(t)

	// 创建容量为 1 的 channel
	ch := make(chan *BlockHeader, 1)
	bc.Subscribe(ch)

	// 连续提交 3 个区块
	for i := int32(1); i <= 3; i++ {
		h := nextHeader(bc, i)
		err := bc.SubmitBlock(h)
		if err != nil {
			t.Fatalf("SubmitBlock(height=%d) error = %v", i, err)
		}
	}

	// 提交不应阻塞，即使 channel 已满
	if bc.ChainHeight() != 3 {
		t.Errorf("ChainHeight() = %d, want 3", bc.ChainHeight())
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/blockchain/ -run "TestBlockchain_Subscribe|TestBlockchain_Unsubscribe"
```

预期输出：编译失败，`Subscribe`、`Unsubscribe` 等方法未定义。

### Step 3: 写最小实现

修改 `internal/blockchain/blockchain.go`——在 `Blockchain` 结构体中添加订阅者列表，并实现 `Subscribe`、`Unsubscribe` 和 `notifySubscribers` 方法。

在 `Blockchain` 结构体中添加字段：

```go
// Blockchain 区块链核心结构。
// 管理区块头链的存储、验证与查询，不包含共识或交易校验逻辑。
type Blockchain struct {
	mu          sync.RWMutex
	store       HeaderStore
	subscribers []chan<- *BlockHeader // 新增：订阅者列表
}
```

添加方法：

```go
// Subscribe 订阅新区块事件。
// 当新区块被成功提交时，区块头将发送到指定 channel。
// 如果 channel 已满，通知将被跳过（不阻塞提交流程）。
func (bc *Blockchain) Subscribe(ch chan<- *BlockHeader) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.subscribers = append(bc.subscribers, ch)
}

// Unsubscribe 取消订阅新区块事件。
func (bc *Blockchain) Unsubscribe(ch chan<- *BlockHeader) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for i, sub := range bc.subscribers {
		if sub == ch {
			bc.subscribers = append(bc.subscribers[:i], bc.subscribers[i+1:]...)
			return
		}
	}
}

// notifySubscribers 通知所有订阅者新区块已提交。
// 使用非阻塞发送，如果某个 channel 已满则跳过。
// 注意：调用者已持有 bc.mu 写锁。
func (bc *Blockchain) notifySubscribers(header *BlockHeader) {
	for _, ch := range bc.subscribers {
		select {
		case ch <- header:
		default:
			// channel 已满，跳过
		}
	}
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/blockchain/ -run "TestBlockchain_Subscribe|TestBlockchain_Unsubscribe"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/blockchain/blockchain.go internal/blockchain/subscribe_test.go
git commit -m "feat(blockchain): add block event subscription and notification"
```

---

## Task 6: 分叉切换

**Files:**
- Create: `internal/blockchain/fork.go`
- Create: `internal/blockchain/fork_test.go`

本 Task 实现 `SwitchChain` 方法，用于将本地链切换到替代链（分叉切换）。

### Step 1: 写失败测试

创建 `internal/blockchain/fork_test.go`：

```go
package blockchain

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：构建从指定高度开始的替代链区块头列表
func buildAltChain(base *BlockHeader, count int) []*BlockHeader {
	headers := make([]*BlockHeader, count)
	prevID := base.BlockID()

	for i := 0; i < count; i++ {
		height := base.Height + int32(i) + 1
		var checkRoot types.Hash512
		checkRoot[0] = byte(height)
		checkRoot[1] = 0xAA // 标记为替代链

		h := &BlockHeader{
			Version:   1,
			PrevBlock: prevID,
			CheckRoot: checkRoot,
			Stakes:    int32(height) * 3,
			Height:    height,
		}

		// 年块处理
		if h.IsYearBlock() {
			// 简化：测试中不涉及年块高度
		}

		headers[i] = h
		prevID = h.BlockID()
	}
	return headers
}

// 测试成功的分叉切换
func TestBlockchain_SwitchChain(t *testing.T) {
	bc := newTestBlockchain(t)

	// 构建主链：height 1-5
	for i := int32(1); i <= 5; i++ {
		h := nextHeader(bc, i)
		bc.SubmitBlock(h)
	}

	// 从 height=2 开始分叉（分叉高度 = 2，保留 height 0-2）
	base, _ := bc.HeaderByHeight(2)
	altHeaders := buildAltChain(base, 4) // 替代链 height 3-6

	err := bc.SwitchChain(2, altHeaders)
	if err != nil {
		t.Fatalf("SwitchChain() error = %v", err)
	}

	// 链高度应为 6
	if bc.ChainHeight() != 6 {
		t.Errorf("ChainHeight() = %d, want 6", bc.ChainHeight())
	}

	// height=3 应为替代链的区块
	h3, _ := bc.HeaderByHeight(3)
	if h3.CheckRoot[1] != 0xAA {
		t.Error("height=3 should be from alt chain")
	}
}

// 测试空替代链时返回错误
func TestBlockchain_SwitchChain_EmptyHeaders(t *testing.T) {
	bc := newTestBlockchain(t)

	err := bc.SwitchChain(0, nil)
	if err == nil {
		t.Error("SwitchChain() should fail with empty headers")
	}
}

// 测试替代链不连续时返回错误
func TestBlockchain_SwitchChain_NonSequential(t *testing.T) {
	bc := newTestBlockchain(t)

	// 构建主链 height 1-3
	for i := int32(1); i <= 3; i++ {
		h := nextHeader(bc, i)
		bc.SubmitBlock(h)
	}

	// 构造不连续的替代链
	base, _ := bc.HeaderByHeight(1)
	h2 := &BlockHeader{
		Version: 1, PrevBlock: base.BlockID(),
		CheckRoot: types.Hash512{10}, Stakes: 10, Height: 2,
	}
	h4 := &BlockHeader{ // height=4，跳过了 3
		Version: 1, PrevBlock: h2.BlockID(),
		CheckRoot: types.Hash512{11}, Stakes: 11, Height: 4,
	}

	err := bc.SwitchChain(1, []*BlockHeader{h2, h4})
	if err == nil {
		t.Error("SwitchChain() should fail with non-sequential headers")
	}
}

// 测试替代链起始 PrevBlock 不匹配分叉点时返回错误
func TestBlockchain_SwitchChain_WrongForkPoint(t *testing.T) {
	bc := newTestBlockchain(t)

	for i := int32(1); i <= 3; i++ {
		h := nextHeader(bc, i)
		bc.SubmitBlock(h)
	}

	// 第一个替代链区块的 PrevBlock 不指向 forkHeight 的区块
	h := &BlockHeader{
		Version: 1, PrevBlock: types.Hash512{0xFF},
		CheckRoot: types.Hash512{10}, Stakes: 10, Height: 2,
	}

	err := bc.SwitchChain(1, []*BlockHeader{h})
	if err == nil {
		t.Error("SwitchChain() should fail when first alt header PrevBlock doesn't match fork point")
	}
}

// 测试分叉高度大于当前链高度时返回错误
func TestBlockchain_SwitchChain_ForkTooHigh(t *testing.T) {
	bc := newTestBlockchain(t)

	h := &BlockHeader{
		Version: 1, PrevBlock: types.Hash512{1},
		CheckRoot: types.Hash512{10}, Stakes: 10, Height: 100,
	}

	err := bc.SwitchChain(99, []*BlockHeader{h})
	if err == nil {
		t.Error("SwitchChain() should fail when fork height exceeds chain height")
	}
}

// 测试 ForkInfo 结构基本使用
func TestForkInfo(t *testing.T) {
	local := &BlockHeader{Height: 10}
	remote := &BlockHeader{Height: 15}
	info := ForkInfo{
		ForkHeight: 5,
		LocalTip:   local,
		RemoteTip:  remote,
		Length:     10,
	}

	if info.ForkHeight != 5 {
		t.Errorf("ForkInfo.ForkHeight = %d, want 5", info.ForkHeight)
	}
	if info.Length != 10 {
		t.Errorf("ForkInfo.Length = %d, want 10", info.Length)
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/blockchain/ -run "TestBlockchain_SwitchChain|TestForkInfo"
```

预期输出：编译失败，`SwitchChain`、`ForkInfo` 等未定义。

### Step 3: 写最小实现

创建 `internal/blockchain/fork.go`：

```go
package blockchain

import (
	"errors"
	"fmt"
)

// 分叉相关错误
var (
	ErrEmptyAltChain     = errors.New("alternative chain is empty")
	ErrForkTooHigh       = errors.New("fork height exceeds current chain height")
	ErrAltChainBroken    = errors.New("alternative chain is not sequential")
	ErrAltChainForkPoint = errors.New("alternative chain does not connect to fork point")
)

// ForkInfo 分叉信息。
type ForkInfo struct {
	ForkHeight int          // 分叉高度
	LocalTip   *BlockHeader // 本地链顶
	RemoteTip  *BlockHeader // 远程链顶
	Length     int          // 远程链长度（分叉点之后）
}

// SwitchChain 切换到替代链。
// forkHeight 为分叉点高度（保留该高度及之前的区块），
// headers 为替代链的区块头列表（从 forkHeight+1 开始的连续区块头）。
//
// 操作步骤：
// 1. 验证替代链的连续性（Height 连续、PrevBlock 链接）
// 2. 回滚到分叉高度（删除 forkHeight+1 之后的所有区块）
// 3. 逐一提交替代链区块头
func (bc *Blockchain) SwitchChain(forkHeight int, headers []*BlockHeader) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 基本检查
	if len(headers) == 0 {
		return ErrEmptyAltChain
	}

	// 获取当前链顶高度
	tip, err := bc.store.Tip()
	if err != nil {
		return fmt.Errorf("get chain tip: %w", err)
	}
	currentHeight := int(tip.Height)

	if forkHeight > currentHeight {
		return ErrForkTooHigh
	}

	// 检查分叉点区块是否存在
	forkBlock, err := bc.store.Get(forkHeight)
	if err != nil {
		return fmt.Errorf("get fork point block: %w", err)
	}

	// 验证替代链：首个区块的 PrevBlock 应指向分叉点
	forkBlockID := forkBlock.BlockID()
	if headers[0].PrevBlock != forkBlockID {
		return ErrAltChainForkPoint
	}

	// 验证替代链：Height 连续性和 PrevBlock 链接
	expectedHeight := int32(forkHeight + 1)
	for i, h := range headers {
		if h.Height != expectedHeight {
			return fmt.Errorf("%w: header[%d] height=%d, want %d",
				ErrAltChainBroken, i, h.Height, expectedHeight)
		}
		if i > 0 && h.PrevBlock != headers[i-1].BlockID() {
			return fmt.Errorf("%w: header[%d] PrevBlock mismatch", ErrAltChainBroken, i)
		}
		// 基本字段验证
		if err := h.Validate(); err != nil {
			return fmt.Errorf("validate alt header[%d]: %w", i, err)
		}
		expectedHeight++
	}

	// 回滚：删除 forkHeight+1 到当前链顶的所有区块
	for h := currentHeight; h > forkHeight; h-- {
		if bc.store.Has(h) {
			if err := bc.store.Delete(h); err != nil {
				return fmt.Errorf("rollback height %d: %w", h, err)
			}
		}
	}

	// 提交替代链区块头
	for i, h := range headers {
		if err := bc.store.Put(h); err != nil {
			return fmt.Errorf("store alt header[%d]: %w", i, err)
		}
		// 通知订阅者
		bc.notifySubscribers(h)
	}

	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/blockchain/ -run "TestBlockchain_SwitchChain|TestForkInfo"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/blockchain/fork.go internal/blockchain/fork_test.go
git commit -m "feat(blockchain): add SwitchChain for fork switching with rollback and validation"
```

---

## Task 7: 初始验证框架

**Files:**
- Create: `internal/blockchain/bootstrap.go`
- Create: `internal/blockchain/bootstrap_test.go`

本 Task 定义初始主链验证的接口与本地验证逻辑框架。暂不包含网络交互实现。

### Step 1: 写失败测试

创建 `internal/blockchain/bootstrap_test.go`：

```go
package blockchain

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 测试 BootstrapResult 结构
func TestBootstrapResult(t *testing.T) {
	r := &BootstrapResult{
		ChainTip:   &BlockHeader{Height: 100},
		Height:     100,
		Confidence: 0.95,
		Sources:    3,
	}

	if r.Height != 100 {
		t.Errorf("Height = %d, want 100", r.Height)
	}
	if r.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want 0.95", r.Confidence)
	}
	if r.Sources != 3 {
		t.Errorf("Sources = %d, want 3", r.Sources)
	}
}

// 测试 ValidateHeaderChain：验证一组连续的区块头
func TestValidateHeaderChain(t *testing.T) {
	// 构建一条短链
	genesis := &GenesisBlock
	headers := []*BlockHeader{genesis}

	prev := genesis
	for i := int32(1); i <= 5; i++ {
		var checkRoot types.Hash512
		checkRoot[0] = byte(i)
		h := &BlockHeader{
			Version:   1,
			PrevBlock: prev.BlockID(),
			CheckRoot: checkRoot,
			Stakes:    i * 10,
			Height:    i,
		}
		headers = append(headers, h)
		prev = h
	}

	err := ValidateHeaderChain(headers)
	if err != nil {
		t.Fatalf("ValidateHeaderChain() error = %v", err)
	}
}

// 测试 ValidateHeaderChain：空链表
func TestValidateHeaderChain_Empty(t *testing.T) {
	err := ValidateHeaderChain(nil)
	if err == nil {
		t.Error("ValidateHeaderChain(nil) should return error")
	}
}

// 测试 ValidateHeaderChain：不连续的链
func TestValidateHeaderChain_Broken(t *testing.T) {
	genesis := &GenesisBlock
	h2 := &BlockHeader{
		Version:   1,
		PrevBlock: types.Hash512{0xFF}, // 不指向 genesis
		CheckRoot: types.Hash512{1},
		Stakes:    10,
		Height:    1,
	}

	err := ValidateHeaderChain([]*BlockHeader{genesis, h2})
	if err == nil {
		t.Error("ValidateHeaderChain() should fail with broken chain")
	}
}

// 测试 ValidateHeaderChain：高度不连续
func TestValidateHeaderChain_HeightGap(t *testing.T) {
	genesis := &GenesisBlock
	h := &BlockHeader{
		Version:   1,
		PrevBlock: genesis.BlockID(),
		CheckRoot: types.Hash512{1},
		Stakes:    10,
		Height:    5, // 应该是 1
	}

	err := ValidateHeaderChain([]*BlockHeader{genesis, h})
	if err == nil {
		t.Error("ValidateHeaderChain() should fail with height gap")
	}
}

// 测试 BlockqsClient 接口可编译
func TestBlockqsClient_Interface(t *testing.T) {
	// 仅验证接口定义可编译
	var _ BlockqsClient = (*mockBlockqsClient)(nil)
}

// 模拟 BlockqsClient 实现
type mockBlockqsClient struct{}

func (m *mockBlockqsClient) FetchHeader(height int) (*BlockHeader, error) {
	return nil, nil
}

func (m *mockBlockqsClient) FetchHeaders(from, to int) ([]*BlockHeader, error) {
	return nil, nil
}

func (m *mockBlockqsClient) FetchHeaderByHash(hash types.Hash512) (*BlockHeader, error) {
	return nil, nil
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/blockchain/ -run "TestBootstrap|TestValidateHeaderChain|TestBlockqsClient"
```

预期输出：编译失败，`BootstrapResult`、`ValidateHeaderChain`、`BlockqsClient` 等未定义。

### Step 3: 写最小实现

创建 `internal/blockchain/bootstrap.go`：

```go
package blockchain

import (
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// 引导验证相关错误
var (
	ErrEmptyHeaderChain  = errors.New("header chain is empty")
	ErrChainLinkBroken   = errors.New("header chain link is broken")
	ErrChainHeightBroken = errors.New("header chain height is not sequential")
)

// BootstrapResult 初始验证结果。
type BootstrapResult struct {
	ChainTip   *BlockHeader // 验证通过的链顶区块头
	Height     int          // 当前链高度
	Confidence float64      // 置信度（基于多源一致性，0.0-1.0）
	Sources    int          // 参与验证的数据源数量
}

// BlockqsClient 区块查询服务客户端接口。
// 由外部实现，用于从远程服务获取区块头数据。
type BlockqsClient interface {
	// FetchHeader 从远程服务获取指定高度的区块头。
	FetchHeader(height int) (*BlockHeader, error)

	// FetchHeaders 批量获取区块头（用于同步）。
	FetchHeaders(from, to int) ([]*BlockHeader, error)

	// FetchHeaderByHash 按哈希获取区块头。
	FetchHeaderByHash(hash types.Hash512) (*BlockHeader, error)
}

// ValidateHeaderChain 验证一组区块头的连续性。
// 检查：
//   - 每个区块的 PrevBlock 指向前一个区块的 BlockID
//   - Height 连续递增
//   - 基本字段合法性
//
// headers 应按高度升序排列。
func ValidateHeaderChain(headers []*BlockHeader) error {
	if len(headers) == 0 {
		return ErrEmptyHeaderChain
	}

	// 验证第一个区块的基本合法性
	if err := headers[0].Validate(); err != nil {
		return fmt.Errorf("header[0]: %w", err)
	}

	// 逐一验证后续区块
	for i := 1; i < len(headers); i++ {
		prev := headers[i-1]
		curr := headers[i]

		// 基本字段验证
		if err := curr.Validate(); err != nil {
			return fmt.Errorf("header[%d]: %w", i, err)
		}

		// 高度连续性
		if curr.Height != prev.Height+1 {
			return fmt.Errorf("%w: header[%d] height=%d, want %d",
				ErrChainHeightBroken, i, curr.Height, prev.Height+1)
		}

		// PrevBlock 链接
		if curr.PrevBlock != prev.BlockID() {
			return fmt.Errorf("%w: header[%d] PrevBlock mismatch", ErrChainLinkBroken, i)
		}
	}

	return nil
}

// BootstrapVerify 初始主链验证。
// 从多个数据源获取主链信息并交叉验证，确认目标主链的合法性。
//
// 当前实现为框架性代码，仅定义接口和本地验证逻辑。
// 网络交互部分（从 Blockqs 获取数据）待后续实现。
//
// sources 为 Blockqs 节点地址列表。
func (bc *Blockchain) BootstrapVerify(sources []string) (*BootstrapResult, error) {
	// 框架实现：返回当前本地链状态
	tip, err := bc.ChainTip()
	if err != nil {
		return nil, fmt.Errorf("get chain tip: %w", err)
	}

	return &BootstrapResult{
		ChainTip:   tip,
		Height:     int(tip.Height),
		Confidence: 1.0, // 本地验证，置信度 100%
		Sources:    0,    // 尚未从远程获取
	}, nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/blockchain/ -run "TestBootstrap|TestValidateHeaderChain|TestBlockqsClient"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/blockchain/bootstrap.go internal/blockchain/bootstrap_test.go
git commit -m "feat(blockchain): add bootstrap verification framework and BlockqsClient interface"
```

---

## 全量测试验证

完成所有 Task 后，运行完整测试套件：

```bash
go test -v -count=1 ./internal/blockchain/...
```

预期：全部 PASS。

运行覆盖率检查：

```bash
go test -cover ./internal/blockchain/...
```

预期：覆盖率 ≥ 80%。

运行格式化检查：

```bash
go fmt ./internal/blockchain/...
```

预期：无变更。

运行编译检查：

```bash
go build ./...
```

预期：无错误。

---

## 验收标准

1. **文件结构**：以下文件全部存在
   - `internal/blockchain/header.go` — BlockHeader 结构、序列化、BlockID、Validate
   - `internal/blockchain/genesis.go` — 创世块常量
   - `internal/blockchain/store.go` — HeaderStore 接口、MemoryHeaderStore 实现
   - `internal/blockchain/blockchain.go` — Blockchain 核心结构、SubmitBlock、ReplaceBlock、查询、订阅
   - `internal/blockchain/yearblock.go` — 年块辅助函数
   - `internal/blockchain/fork.go` — SwitchChain 分叉切换
   - `internal/blockchain/bootstrap.go` — 初始验证框架、BlockqsClient 接口
   - `internal/blockchain/header_test.go`
   - `internal/blockchain/store_test.go`
   - `internal/blockchain/blockchain_test.go`
   - `internal/blockchain/yearblock_test.go`
   - `internal/blockchain/subscribe_test.go`
   - `internal/blockchain/fork_test.go`
   - `internal/blockchain/bootstrap_test.go`

2. **编译通过**：`go build ./...` 无错误

3. **全量测试通过**：`go test ./internal/blockchain/...` 全部 PASS

4. **测试覆盖率**：核心逻辑 ≥ 80%

5. **功能完整性**：
   - BlockHeader 可序列化为 140/204 字节二进制格式
   - BlockID = SHA-512(Serialize())
   - MemoryHeaderStore 支持 Put/Get/GetByHash/Has/Delete/Tip/YearBlock
   - SubmitBlock 执行 5 项结构性验证
   - ReplaceBlock 仅允许替换链顶
   - 年块在 Height % 87661 == 0 时正确验证 YearBlock 引用
   - Subscribe/Unsubscribe 正确通知新区块事件
   - SwitchChain 完成回滚+替代链提交
   - ValidateHeaderChain 验证区块头链连续性
   - BlockqsClient 接口已定义

6. **设计约束**：
   - Core 不包含共识逻辑或交易校验
   - 仅依赖 `pkg/types` 和 `pkg/crypto`
   - 所有公共 API 并发安全（sync.RWMutex）
   - 错误信息英文小写、无标点
   - 注释使用中文

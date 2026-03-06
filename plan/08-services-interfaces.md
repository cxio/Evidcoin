# Phase 8：公共服务接口 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现与第三方公共服务交互的接口与客户端逻辑——包括 Blockqs（区块查询）、Depots（数据驿站）、STUN（NAT穿透）的客户端接口定义、服务质量评估框架、奖励地址缓存管理、以及奖励确认位图操作。

**Architecture:** `internal/services` 包，定义与三类外部公共服务交互的客户端接口和本地管理逻辑。实际网络通信由外部 P2P 库处理，本模块专注于接口协议、评估逻辑和缓存管理。

**Tech Stack:** Go 1.25+, pkg/types, pkg/crypto

---

## 前置依赖

本 Phase 假设 Phase 1 和 Phase 2 已完成，以下类型和函数已可用：

```go
// pkg/types
type Hash512 [64]byte              // SHA-512（64 字节）
type PubKeyHash [48]byte           // 公钥哈希（SHA3-384）
const HashLen = 64
const PubKeyHashLen = 48
const BlocksPerYear = 87661        // 每年区块数

// Hash512 方法
func (h Hash512) IsZero() bool
func (h Hash512) String() string
func (h Hash512) Equal(other Hash512) bool

// PubKeyHash 方法
func (p PubKeyHash) IsZero() bool
func (p PubKeyHash) String() string

// pkg/crypto
func SHA512Sum(data []byte) types.Hash512
func SHA3_512Sum(data []byte) types.Hash512
```

> **注意：** 如果 Phase 1/2 的具体 API 与以上描述有差异，请在实现时以 `pkg/types` 和 `pkg/crypto` 的实际导出符号为准，适当调整本计划中的调用方式。

---

## Task 1: 服务类型与通用接口 (internal/services/types.go)

**Files:**
- Create: `internal/services/types.go`
- Test: `internal/services/types_test.go`

本 Task 定义服务类型枚举 `ServiceType`、节点标识 `NodeID`、服务节点结构体 `ServiceNode`、通用客户端接口 `ServiceClient`、以及基础错误变量。

### Step 1: 写失败测试

创建 `internal/services/types_test.go`：

```go
package services

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- ServiceType 枚举测试 ---

func TestServiceType_Values(t *testing.T) {
	// 确认三种服务类型的枚举值互不相同
	types_set := map[ServiceType]bool{
		ServiceDepots:  true,
		ServiceBlockqs: true,
		ServiceStun:    true,
	}
	if len(types_set) != 3 {
		t.Errorf("expected 3 distinct ServiceType values, got %d", len(types_set))
	}
}

func TestServiceType_String(t *testing.T) {
	tests := []struct {
		name string
		st   ServiceType
		want string
	}{
		{"depots", ServiceDepots, "depots"},
		{"blockqs", ServiceBlockqs, "blockqs"},
		{"stun", ServiceStun, "stun"},
		{"unknown", ServiceType(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.st.String()
			if got != tt.want {
				t.Errorf("ServiceType(%d).String() = %q, want %q", tt.st, got, tt.want)
			}
		})
	}
}

func TestServiceType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		st   ServiceType
		want bool
	}{
		{"depots valid", ServiceDepots, true},
		{"blockqs valid", ServiceBlockqs, true},
		{"stun valid", ServiceStun, true},
		{"zero invalid", ServiceType(0), false},
		{"high value invalid", ServiceType(99), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.st.IsValid(); got != tt.want {
				t.Errorf("ServiceType(%d).IsValid() = %v, want %v", tt.st, got, tt.want)
			}
		})
	}
}

// --- ServiceTypeCount 常量测试 ---

func TestServiceTypeCount(t *testing.T) {
	if ServiceTypeCount != 3 {
		t.Errorf("ServiceTypeCount = %d, want 3", ServiceTypeCount)
	}
}

// --- NodeID 测试 ---

func TestNodeID_IsZero(t *testing.T) {
	var zeroID NodeID
	if !zeroID.IsZero() {
		t.Error("zero NodeID.IsZero() should return true")
	}

	nonZeroID := NodeID{0x01}
	if nonZeroID.IsZero() {
		t.Error("non-zero NodeID.IsZero() should return false")
	}
}

func TestNodeID_String(t *testing.T) {
	var id NodeID
	id[0] = 0xab
	id[1] = 0xcd

	s := id.String()
	// 应返回 64 字符的十六进制字符串
	if len(s) != 64 {
		t.Errorf("NodeID.String() length = %d, want 64", len(s))
	}
	if s[:4] != "abcd" {
		t.Errorf("NodeID.String() prefix = %q, want %q", s[:4], "abcd")
	}
}

func TestNodeID_Different(t *testing.T) {
	id1 := NodeID{0x01}
	id2 := NodeID{0x02}

	if id1.String() == id2.String() {
		t.Error("different NodeIDs should produce different String()")
	}
}

// --- ServiceNode 测试 ---

func TestNewServiceNode(t *testing.T) {
	id := NodeID{0x01}
	addr := types.PubKeyHash{0x02}
	fingerprint := [32]byte{0x03}

	node := NewServiceNode(id, ServiceBlockqs, addr, fingerprint)

	if node.ID != id {
		t.Errorf("NewServiceNode().ID = %v, want %v", node.ID, id)
	}
	if node.Type != ServiceBlockqs {
		t.Errorf("NewServiceNode().Type = %v, want %v", node.Type, ServiceBlockqs)
	}
	if node.Address != addr {
		t.Errorf("NewServiceNode().Address = %v, want %v", node.Address, addr)
	}
	if node.SPKIFingerprint != fingerprint {
		t.Errorf("NewServiceNode().SPKIFingerprint mismatch")
	}
	if node.QualityScore != 0.0 {
		t.Errorf("NewServiceNode().QualityScore = %f, want 0.0", node.QualityScore)
	}
	if node.LastSeen.IsZero() {
		t.Error("NewServiceNode().LastSeen should not be zero")
	}
}

func TestServiceNode_Validate(t *testing.T) {
	tests := []struct {
		name    string
		node    ServiceNode
		wantErr bool
	}{
		{
			name: "valid node",
			node: ServiceNode{
				ID:              NodeID{0x01},
				Type:            ServiceBlockqs,
				Address:         types.PubKeyHash{0x02},
				SPKIFingerprint: [32]byte{0x03},
				LastSeen:        time.Now(),
			},
			wantErr: false,
		},
		{
			name: "zero node id",
			node: ServiceNode{
				ID:       NodeID{},
				Type:     ServiceBlockqs,
				Address:  types.PubKeyHash{0x02},
				LastSeen: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid service type",
			node: ServiceNode{
				ID:       NodeID{0x01},
				Type:     ServiceType(99),
				Address:  types.PubKeyHash{0x02},
				LastSeen: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "zero address",
			node: ServiceNode{
				ID:       NodeID{0x01},
				Type:     ServiceDepots,
				Address:  types.PubKeyHash{},
				LastSeen: time.Now(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- 错误变量测试 ---

func TestErrors_NonNil(t *testing.T) {
	errors := []error{
		ErrNotConnected,
		ErrNodeNotFound,
		ErrServiceUnavailable,
	}
	for _, err := range errors {
		if err == nil {
			t.Error("error variable should not be nil")
		}
	}
}

func TestErrors_Distinct(t *testing.T) {
	errs := []error{
		ErrNotConnected,
		ErrNodeNotFound,
		ErrServiceUnavailable,
	}
	seen := make(map[string]bool)
	for _, err := range errs {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("duplicate error message: %q", msg)
		}
		seen[msg] = true
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/services/ -run "TestServiceType|TestServiceTypeCount|TestNodeID|TestServiceNode|TestErrors"
```

预期输出：编译失败，`ServiceType`、`NodeID`、`ServiceNode`、`NewServiceNode`、`ErrNotConnected` 等未定义。

### Step 3: 写最小实现

创建 `internal/services/types.go`：

```go
package services

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// ServiceType 表示公共服务的类型。
type ServiceType byte

const (
	// ServiceDepots 数据驿站服务。
	ServiceDepots ServiceType = iota + 1
	// ServiceBlockqs 区块查询服务。
	ServiceBlockqs
	// ServiceStun NAT 穿透服务。
	ServiceStun
)

// ServiceTypeCount 服务类型总数。
const ServiceTypeCount = 3

// String 返回服务类型的字符串表示。
func (st ServiceType) String() string {
	switch st {
	case ServiceDepots:
		return "depots"
	case ServiceBlockqs:
		return "blockqs"
	case ServiceStun:
		return "stun"
	default:
		return "unknown"
	}
}

// IsValid 检查服务类型是否有效。
func (st ServiceType) IsValid() bool {
	return st >= ServiceDepots && st <= ServiceStun
}

// NodeID 表示服务节点的唯一标识（32 字节）。
// 通常由节点的 SPKI 指纹或其他唯一标识派生。
type NodeID [32]byte

// IsZero 检查 NodeID 是否为全零值。
func (id NodeID) IsZero() bool {
	for _, b := range id {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回 NodeID 的十六进制字符串表示。
func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
}

// ServiceNode 表示一个公共服务节点的信息。
type ServiceNode struct {
	ID              NodeID           // 节点唯一标识
	Type            ServiceType      // 服务类型
	Address         types.PubKeyHash // 奖励接收地址
	SPKIFingerprint [32]byte         // SPKI 指纹，用于连接安全验证
	LastSeen        time.Time        // 最后一次通信时间
	QualityScore    float64          // 服务质量评分（0.0 ~ 1.0）
}

// NewServiceNode 创建一个新的服务节点信息。
// 初始评分为 0.0，LastSeen 设为当前时间。
func NewServiceNode(id NodeID, serviceType ServiceType, address types.PubKeyHash, spkiFingerprint [32]byte) *ServiceNode {
	return &ServiceNode{
		ID:              id,
		Type:            serviceType,
		Address:         address,
		SPKIFingerprint: spkiFingerprint,
		LastSeen:        time.Now(),
		QualityScore:    0.0,
	}
}

// Validate 对服务节点执行基本字段验证。
func (n *ServiceNode) Validate() error {
	if n.ID.IsZero() {
		return fmt.Errorf("service node: %w", ErrNodeIDZero)
	}
	if !n.Type.IsValid() {
		return fmt.Errorf("service node: %w", ErrInvalidServiceType)
	}
	if n.Address.IsZero() {
		return fmt.Errorf("service node: %w", ErrAddressZero)
	}
	return nil
}

// ServiceClient 公共服务客户端的通用接口。
// 所有三类服务客户端（Blockqs、Depots、STUN）都应实现此接口。
type ServiceClient interface {
	// Connect 连接到指定节点。
	Connect(nodeID NodeID) error
	// Disconnect 断开与指定节点的连接。
	Disconnect(nodeID NodeID) error
	// IsConnected 检查是否已连接到指定节点。
	IsConnected(nodeID NodeID) bool
	// RewardAddress 获取指定节点的奖励接收地址。
	RewardAddress(nodeID NodeID) (types.PubKeyHash, error)
}

// 错误定义
var (
	// ErrNotConnected 节点未连接。
	ErrNotConnected = errors.New("service node not connected")
	// ErrNodeNotFound 节点未找到。
	ErrNodeNotFound = errors.New("service node not found")
	// ErrServiceUnavailable 服务不可用。
	ErrServiceUnavailable = errors.New("service unavailable")
	// ErrNodeIDZero 节点 ID 为零。
	ErrNodeIDZero = errors.New("node id is zero")
	// ErrInvalidServiceType 无效的服务类型。
	ErrInvalidServiceType = errors.New("invalid service type")
	// ErrAddressZero 地址为零。
	ErrAddressZero = errors.New("reward address is zero")
)
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/services/ -run "TestServiceType|TestServiceTypeCount|TestNodeID|TestServiceNode|TestErrors"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/services/types.go internal/services/types_test.go
git commit -m "feat(services): add ServiceType, NodeID, ServiceNode and ServiceClient interface"
```

---

## Task 2: Blockqs 客户端接口 (internal/services/blockqs.go)

**Files:**
- Create: `internal/services/blockqs.go`
- Test: `internal/services/blockqs_test.go`

本 Task 定义 Blockqs 区块查询服务的客户端接口 `BlockqsClient`、查询类型枚举 `BlockqsQueryType`、内存 Mock 实现 `MemBlockqs`、以及多源验证结构体 `BlockqsMultiSource`。

### Step 1: 写失败测试

创建 `internal/services/blockqs_test.go`：

```go
package services

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- BlockqsQueryType 枚举测试 ---

func TestBlockqsQueryType_Values(t *testing.T) {
	// 确认查询类型枚举值互不相同
	qtSet := map[BlockqsQueryType]bool{
		QueryScript:      true,
		QueryScriptSet:   true,
		QueryTransaction: true,
		QueryUTXOData:    true,
		QueryUTCOData:    true,
		QueryTxIDSet:     true,
		QueryAttachment:  true,
	}
	if len(qtSet) != 7 {
		t.Errorf("expected 7 distinct BlockqsQueryType values, got %d", len(qtSet))
	}
}

func TestBlockqsQueryType_String(t *testing.T) {
	tests := []struct {
		name string
		qt   BlockqsQueryType
		want string
	}{
		{"script", QueryScript, "script"},
		{"script_set", QueryScriptSet, "script_set"},
		{"transaction", QueryTransaction, "transaction"},
		{"utxo_data", QueryUTXOData, "utxo_data"},
		{"utco_data", QueryUTCOData, "utco_data"},
		{"txid_set", QueryTxIDSet, "txid_set"},
		{"attachment", QueryAttachment, "attachment"},
		{"unknown", BlockqsQueryType(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.qt.String()
			if got != tt.want {
				t.Errorf("BlockqsQueryType(%d).String() = %q, want %q", tt.qt, got, tt.want)
			}
		})
	}
}

// --- BlockHeader 简化结构测试 ---

func TestBlockHeader_Basic(t *testing.T) {
	header := &BlockHeader{
		Version:   1,
		Height:    100,
		PrevBlock: types.Hash512{0x01},
		CheckRoot: types.Hash512{0x02},
		Stakes:    42,
	}

	if header.Version != 1 {
		t.Errorf("Version = %d, want 1", header.Version)
	}
	if header.Height != 100 {
		t.Errorf("Height = %d, want 100", header.Height)
	}
	if header.PrevBlock[0] != 0x01 {
		t.Error("PrevBlock mismatch")
	}
	if header.CheckRoot[0] != 0x02 {
		t.Error("CheckRoot mismatch")
	}
	if header.Stakes != 42 {
		t.Errorf("Stakes = %d, want 42", header.Stakes)
	}
}

// --- MemBlockqs Mock 测试 ---

func TestMemBlockqs_FetchHeader(t *testing.T) {
	mock := NewMemBlockqs()

	// 预置区块头
	h := &BlockHeader{Version: 1, Height: 10, Stakes: 5}
	mock.AddHeader(h)

	// 正常获取
	got, err := mock.FetchHeader(10)
	if err != nil {
		t.Fatalf("FetchHeader(10) error = %v", err)
	}
	if got.Height != 10 {
		t.Errorf("FetchHeader(10).Height = %d, want 10", got.Height)
	}
	if got.Stakes != 5 {
		t.Errorf("FetchHeader(10).Stakes = %d, want 5", got.Stakes)
	}

	// 不存在的高度
	_, err = mock.FetchHeader(999)
	if err == nil {
		t.Error("FetchHeader(999) should return error for non-existent height")
	}
}

func TestMemBlockqs_FetchHeaders(t *testing.T) {
	mock := NewMemBlockqs()

	for i := 0; i < 5; i++ {
		mock.AddHeader(&BlockHeader{Version: 1, Height: int32(i)})
	}

	// 获取范围 [1, 3]
	headers, err := mock.FetchHeaders(1, 3)
	if err != nil {
		t.Fatalf("FetchHeaders(1, 3) error = %v", err)
	}
	if len(headers) != 3 {
		t.Fatalf("FetchHeaders(1, 3) returned %d headers, want 3", len(headers))
	}
	for i, h := range headers {
		expected := int32(i + 1)
		if h.Height != expected {
			t.Errorf("headers[%d].Height = %d, want %d", i, h.Height, expected)
		}
	}

	// 无效范围
	_, err = mock.FetchHeaders(3, 1)
	if err == nil {
		t.Error("FetchHeaders(3, 1) should return error for invalid range")
	}
}

func TestMemBlockqs_FetchHeaderByHash(t *testing.T) {
	mock := NewMemBlockqs()

	hash := types.Hash512{0xaa, 0xbb}
	h := &BlockHeader{Version: 1, Height: 50, PrevBlock: hash}
	mock.AddHeaderWithHash(hash, h)

	// 正常获取
	got, err := mock.FetchHeaderByHash(hash)
	if err != nil {
		t.Fatalf("FetchHeaderByHash() error = %v", err)
	}
	if got.Height != 50 {
		t.Errorf("FetchHeaderByHash().Height = %d, want 50", got.Height)
	}

	// 不存在的哈希
	_, err = mock.FetchHeaderByHash(types.Hash512{0xff})
	if err == nil {
		t.Error("FetchHeaderByHash() should return error for non-existent hash")
	}
}

func TestMemBlockqs_FetchTransaction(t *testing.T) {
	mock := NewMemBlockqs()

	txID := types.Hash512{0x01}
	txData := []byte{0x10, 0x20, 0x30}
	mock.AddTransaction(txID, txData)

	got, err := mock.FetchTransaction(txID)
	if err != nil {
		t.Fatalf("FetchTransaction() error = %v", err)
	}
	if !bytes.Equal(got, txData) {
		t.Errorf("FetchTransaction() = %x, want %x", got, txData)
	}

	// 不存在的交易
	_, err = mock.FetchTransaction(types.Hash512{0xff})
	if err == nil {
		t.Error("FetchTransaction() should return error for non-existent txid")
	}
}

func TestMemBlockqs_FetchUTXOData(t *testing.T) {
	mock := NewMemBlockqs()

	data := []byte{0xaa, 0xbb}
	mock.AddUTXOData(2026, data)

	got, err := mock.FetchUTXOData(2026)
	if err != nil {
		t.Fatalf("FetchUTXOData(2026) error = %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("FetchUTXOData(2026) = %x, want %x", got, data)
	}

	_, err = mock.FetchUTXOData(1999)
	if err == nil {
		t.Error("FetchUTXOData(1999) should return error")
	}
}

func TestMemBlockqs_FetchUTCOData(t *testing.T) {
	mock := NewMemBlockqs()

	data := []byte{0xcc, 0xdd}
	mock.AddUTCOData(2026, data)

	got, err := mock.FetchUTCOData(2026)
	if err != nil {
		t.Fatalf("FetchUTCOData(2026) error = %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("FetchUTCOData(2026) = %x, want %x", got, data)
	}

	_, err = mock.FetchUTCOData(1999)
	if err == nil {
		t.Error("FetchUTCOData(1999) should return error")
	}
}

func TestMemBlockqs_FetchTxIDSet(t *testing.T) {
	mock := NewMemBlockqs()

	addr := types.PubKeyHash{0x01}
	txIDs := []types.Hash512{{0xaa}, {0xbb}}
	mock.AddTxIDSet(addr, txIDs)

	got, err := mock.FetchTxIDSet(addr)
	if err != nil {
		t.Fatalf("FetchTxIDSet() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("FetchTxIDSet() returned %d items, want 2", len(got))
	}
	if got[0] != txIDs[0] || got[1] != txIDs[1] {
		t.Error("FetchTxIDSet() returned unexpected txIDs")
	}

	_, err = mock.FetchTxIDSet(types.PubKeyHash{0xff})
	if err == nil {
		t.Error("FetchTxIDSet() should return error for unknown address")
	}
}

func TestMemBlockqs_FetchScript(t *testing.T) {
	mock := NewMemBlockqs()

	txID := types.Hash512{0x01}
	script := []byte{0x76, 0xa9, 0x14}
	mock.AddScript(txID, 0, script)

	got, err := mock.FetchScript(txID, 0)
	if err != nil {
		t.Fatalf("FetchScript() error = %v", err)
	}
	if !bytes.Equal(got, script) {
		t.Errorf("FetchScript() = %x, want %x", got, script)
	}

	_, err = mock.FetchScript(txID, 1)
	if err == nil {
		t.Error("FetchScript() should return error for non-existent output index")
	}
}

func TestMemBlockqs_FetchAttachment(t *testing.T) {
	mock := NewMemBlockqs()

	attachID := types.Hash512{0x42}
	data := []byte("attachment data")
	mock.AddAttachment(attachID, data)

	got, err := mock.FetchAttachment(attachID)
	if err != nil {
		t.Fatalf("FetchAttachment() error = %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("FetchAttachment() = %q, want %q", got, data)
	}

	_, err = mock.FetchAttachment(types.Hash512{0xff})
	if err == nil {
		t.Error("FetchAttachment() should return error for non-existent attachID")
	}
}

// --- BlockqsMultiSource 多源验证测试 ---

func TestBlockqsMultiSource_FetchHeaderVerified_Consistent(t *testing.T) {
	// 三个 Mock 返回相同区块头
	m1 := NewMemBlockqs()
	m2 := NewMemBlockqs()
	m3 := NewMemBlockqs()

	header := &BlockHeader{Version: 1, Height: 100, Stakes: 10, CheckRoot: types.Hash512{0x01}}
	m1.AddHeader(header)
	m2.AddHeader(header)
	m3.AddHeader(header)

	ms := NewBlockqsMultiSource([]BlockqsClient{m1, m2, m3})

	got, err := ms.FetchHeaderVerified(100)
	if err != nil {
		t.Fatalf("FetchHeaderVerified() error = %v", err)
	}
	if got.Height != 100 {
		t.Errorf("FetchHeaderVerified().Height = %d, want 100", got.Height)
	}
	if got.Stakes != 10 {
		t.Errorf("FetchHeaderVerified().Stakes = %d, want 10", got.Stakes)
	}
}

func TestBlockqsMultiSource_FetchHeaderVerified_Inconsistent(t *testing.T) {
	// 两个 Mock 返回不同的 CheckRoot
	m1 := NewMemBlockqs()
	m2 := NewMemBlockqs()

	h1 := &BlockHeader{Version: 1, Height: 100, CheckRoot: types.Hash512{0x01}}
	h2 := &BlockHeader{Version: 1, Height: 100, CheckRoot: types.Hash512{0x02}}
	m1.AddHeader(h1)
	m2.AddHeader(h2)

	ms := NewBlockqsMultiSource([]BlockqsClient{m1, m2})

	_, err := ms.FetchHeaderVerified(100)
	if err == nil {
		t.Error("FetchHeaderVerified() should return error for inconsistent headers")
	}
	if !errors.Is(err, ErrInconsistentData) {
		t.Errorf("FetchHeaderVerified() error = %v, want ErrInconsistentData", err)
	}
}

func TestBlockqsMultiSource_FetchHeaderVerified_PartialFailure(t *testing.T) {
	// 一个 Mock 有数据，另一个没有——应返回错误
	m1 := NewMemBlockqs()
	m2 := NewMemBlockqs()

	m1.AddHeader(&BlockHeader{Version: 1, Height: 100, CheckRoot: types.Hash512{0x01}})
	// m2 没有高度 100 的数据

	ms := NewBlockqsMultiSource([]BlockqsClient{m1, m2})

	_, err := ms.FetchHeaderVerified(100)
	if err == nil {
		t.Error("FetchHeaderVerified() should return error when some sources fail")
	}
}

func TestBlockqsMultiSource_Empty(t *testing.T) {
	ms := NewBlockqsMultiSource(nil)

	_, err := ms.FetchHeaderVerified(100)
	if err == nil {
		t.Error("FetchHeaderVerified() should return error with no clients")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/services/ -run "TestBlockqsQueryType|TestBlockHeader|TestMemBlockqs|TestBlockqsMultiSource"
```

预期输出：编译失败，`BlockqsClient`、`MemBlockqs`、`BlockqsMultiSource` 等未定义。

### Step 3: 写最小实现

创建 `internal/services/blockqs.go`：

```go
package services

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cxio/evidcoin/pkg/types"
)

// BlockqsQueryType 表示 Blockqs 查询的类型。
type BlockqsQueryType byte

const (
	// QueryScript 通用检索交易及其输出。
	QueryScript BlockqsQueryType = iota + 1
	// QueryScriptSet 获取目标交易的全部输出。
	QueryScriptSet
	// QueryTransaction 获取完整交易数据。
	QueryTransaction
	// QueryUTXOData 获取特定年度的 UTXO 集数据。
	QueryUTXOData
	// QueryUTCOData 获取特定年度的 UTCO 集数据。
	QueryUTCOData
	// QueryTxIDSet 获取与某地址相关的交易 ID 集合。
	QueryTxIDSet
	// QueryAttachment 获取小附件或分片索引文件。
	QueryAttachment
)

// String 返回查询类型的字符串表示。
func (qt BlockqsQueryType) String() string {
	switch qt {
	case QueryScript:
		return "script"
	case QueryScriptSet:
		return "script_set"
	case QueryTransaction:
		return "transaction"
	case QueryUTXOData:
		return "utxo_data"
	case QueryUTCOData:
		return "utco_data"
	case QueryTxIDSet:
		return "txid_set"
	case QueryAttachment:
		return "attachment"
	default:
		return "unknown"
	}
}

// BlockHeader 区块头简化结构。
// 用于在 services 包内传递区块头数据。
// 完整结构定义参见 internal/blockchain 包。
type BlockHeader struct {
	Version   int32         // 协议版本号
	PrevBlock types.Hash512 // 前一区块哈希
	CheckRoot types.Hash512 // 校验根
	Stakes    int32         // 币权销毁量
	Height    int32         // 区块高度
}

// headerEqual 比较两个区块头是否一致（所有字段相同）。
func headerEqual(a, b *BlockHeader) bool {
	return a.Version == b.Version &&
		a.PrevBlock == b.PrevBlock &&
		a.CheckRoot == b.CheckRoot &&
		a.Stakes == b.Stakes &&
		a.Height == b.Height
}

// BlockqsClient 区块查询服务客户端接口。
// 定义与 Blockqs 服务交互的全部查询操作。
type BlockqsClient interface {
	// FetchHeader 获取指定高度的区块头。
	FetchHeader(height int) (*BlockHeader, error)
	// FetchHeaders 批量获取区块头，范围为 [from, to]（含两端）。
	FetchHeaders(from, to int) ([]*BlockHeader, error)
	// FetchHeaderByHash 按哈希获取区块头。
	FetchHeaderByHash(hash types.Hash512) (*BlockHeader, error)
	// FetchTransaction 获取完整交易的原始字节。
	FetchTransaction(txID types.Hash512) ([]byte, error)
	// FetchUTXOData 获取指定年度的 UTXO 集数据。
	FetchUTXOData(year int) ([]byte, error)
	// FetchUTCOData 获取指定年度的 UTCO 集数据。
	FetchUTCOData(year int) ([]byte, error)
	// FetchTxIDSet 获取与指定地址相关的交易 ID 列表。
	FetchTxIDSet(address types.PubKeyHash) ([]types.Hash512, error)
	// FetchScript 获取指定交易输出的锁定脚本。
	FetchScript(txID types.Hash512, outputIndex int) ([]byte, error)
	// FetchAttachment 获取小附件数据（< 10MB）。
	FetchAttachment(attachID types.Hash512) ([]byte, error)
}

// ErrInconsistentData 多源验证时数据不一致。
var ErrInconsistentData = errors.New("inconsistent data from multiple sources")

// scriptKey 用于构造 FetchScript 的 map 键。
type scriptKey struct {
	txID        types.Hash512
	outputIndex int
}

// MemBlockqs 是 BlockqsClient 的内存 Mock 实现，用于测试。
type MemBlockqs struct {
	mu          sync.RWMutex
	headers     map[int]*BlockHeader          // 按高度索引
	headerHash  map[types.Hash512]*BlockHeader // 按哈希索引
	txs         map[types.Hash512][]byte       // 交易数据
	utxoData    map[int][]byte                 // UTXO 年度数据
	utcoData    map[int][]byte                 // UTCO 年度数据
	txIDSets    map[types.PubKeyHash][]types.Hash512 // 地址关联交易
	scripts     map[scriptKey][]byte           // 脚本数据
	attachments map[types.Hash512][]byte       // 附件数据
}

// NewMemBlockqs 创建一个空的内存 Mock Blockqs 客户端。
func NewMemBlockqs() *MemBlockqs {
	return &MemBlockqs{
		headers:     make(map[int]*BlockHeader),
		headerHash:  make(map[types.Hash512]*BlockHeader),
		txs:         make(map[types.Hash512][]byte),
		utxoData:    make(map[int][]byte),
		utcoData:    make(map[int][]byte),
		txIDSets:    make(map[types.PubKeyHash][]types.Hash512),
		scripts:     make(map[scriptKey][]byte),
		attachments: make(map[types.Hash512][]byte),
	}
}

// --- Mock 数据添加方法 ---

// AddHeader 向 Mock 中添加一个区块头（按高度索引）。
func (m *MemBlockqs) AddHeader(h *BlockHeader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headers[int(h.Height)] = h
}

// AddHeaderWithHash 向 Mock 中添加一个区块头（同时按哈希索引）。
func (m *MemBlockqs) AddHeaderWithHash(hash types.Hash512, h *BlockHeader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headers[int(h.Height)] = h
	m.headerHash[hash] = h
}

// AddTransaction 向 Mock 中添加一笔交易数据。
func (m *MemBlockqs) AddTransaction(txID types.Hash512, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txs[txID] = data
}

// AddUTXOData 向 Mock 中添加指定年度的 UTXO 数据。
func (m *MemBlockqs) AddUTXOData(year int, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.utxoData[year] = data
}

// AddUTCOData 向 Mock 中添加指定年度的 UTCO 数据。
func (m *MemBlockqs) AddUTCOData(year int, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.utcoData[year] = data
}

// AddTxIDSet 向 Mock 中添加地址关联的交易 ID 集合。
func (m *MemBlockqs) AddTxIDSet(addr types.PubKeyHash, txIDs []types.Hash512) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txIDSets[addr] = txIDs
}

// AddScript 向 Mock 中添加脚本数据。
func (m *MemBlockqs) AddScript(txID types.Hash512, outputIndex int, script []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scripts[scriptKey{txID: txID, outputIndex: outputIndex}] = script
}

// AddAttachment 向 Mock 中添加附件数据。
func (m *MemBlockqs) AddAttachment(attachID types.Hash512, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachments[attachID] = data
}

// --- BlockqsClient 接口实现 ---

// FetchHeader 获取指定高度的区块头。
func (m *MemBlockqs) FetchHeader(height int) (*BlockHeader, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.headers[height]
	if !ok {
		return nil, fmt.Errorf("fetch header: height %d: %w", height, ErrNodeNotFound)
	}
	return h, nil
}

// FetchHeaders 批量获取区块头，范围为 [from, to]（含两端）。
func (m *MemBlockqs) FetchHeaders(from, to int) ([]*BlockHeader, error) {
	if from > to {
		return nil, fmt.Errorf("fetch headers: invalid range [%d, %d]", from, to)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*BlockHeader, 0, to-from+1)
	for i := from; i <= to; i++ {
		h, ok := m.headers[i]
		if !ok {
			return nil, fmt.Errorf("fetch headers: height %d: %w", i, ErrNodeNotFound)
		}
		result = append(result, h)
	}
	return result, nil
}

// FetchHeaderByHash 按哈希获取区块头。
func (m *MemBlockqs) FetchHeaderByHash(hash types.Hash512) (*BlockHeader, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.headerHash[hash]
	if !ok {
		return nil, fmt.Errorf("fetch header by hash: %w", ErrNodeNotFound)
	}
	return h, nil
}

// FetchTransaction 获取完整交易的原始字节。
func (m *MemBlockqs) FetchTransaction(txID types.Hash512) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.txs[txID]
	if !ok {
		return nil, fmt.Errorf("fetch transaction: %w", ErrNodeNotFound)
	}
	return data, nil
}

// FetchUTXOData 获取指定年度的 UTXO 集数据。
func (m *MemBlockqs) FetchUTXOData(year int) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.utxoData[year]
	if !ok {
		return nil, fmt.Errorf("fetch utxo data: year %d: %w", year, ErrNodeNotFound)
	}
	return data, nil
}

// FetchUTCOData 获取指定年度的 UTCO 集数据。
func (m *MemBlockqs) FetchUTCOData(year int) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.utcoData[year]
	if !ok {
		return nil, fmt.Errorf("fetch utco data: year %d: %w", year, ErrNodeNotFound)
	}
	return data, nil
}

// FetchTxIDSet 获取与指定地址相关的交易 ID 列表。
func (m *MemBlockqs) FetchTxIDSet(address types.PubKeyHash) ([]types.Hash512, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids, ok := m.txIDSets[address]
	if !ok {
		return nil, fmt.Errorf("fetch txid set: %w", ErrNodeNotFound)
	}
	return ids, nil
}

// FetchScript 获取指定交易输出的锁定脚本。
func (m *MemBlockqs) FetchScript(txID types.Hash512, outputIndex int) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := scriptKey{txID: txID, outputIndex: outputIndex}
	data, ok := m.scripts[key]
	if !ok {
		return nil, fmt.Errorf("fetch script: txid %s index %d: %w", txID.String(), outputIndex, ErrNodeNotFound)
	}
	return data, nil
}

// FetchAttachment 获取小附件数据。
func (m *MemBlockqs) FetchAttachment(attachID types.Hash512) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.attachments[attachID]
	if !ok {
		return nil, fmt.Errorf("fetch attachment: %w", ErrNodeNotFound)
	}
	return data, nil
}

// BlockqsMultiSource 多源验证客户端。
// 向多个 Blockqs 节点请求相同数据，比较结果一致性。
// 不一致时返回错误并标记异常。
type BlockqsMultiSource struct {
	clients []BlockqsClient
}

// NewBlockqsMultiSource 创建一个多源验证客户端。
func NewBlockqsMultiSource(clients []BlockqsClient) *BlockqsMultiSource {
	return &BlockqsMultiSource{clients: clients}
}

// FetchHeaderVerified 从多个源获取同一区块头并验证一致性。
// 所有源必须返回相同的区块头，否则返回 ErrInconsistentData。
// 如果任何源返回错误，整体也返回错误。
func (ms *BlockqsMultiSource) FetchHeaderVerified(height int) (*BlockHeader, error) {
	if len(ms.clients) == 0 {
		return nil, fmt.Errorf("multi-source fetch: no clients available")
	}

	// 从各源并发获取结果
	type result struct {
		header *BlockHeader
		err    error
	}
	results := make([]result, len(ms.clients))
	var wg sync.WaitGroup

	for i, client := range ms.clients {
		wg.Add(1)
		go func(idx int, c BlockqsClient) {
			defer wg.Done()
			h, err := c.FetchHeader(height)
			results[idx] = result{header: h, err: err}
		}(i, client)
	}
	wg.Wait()

	// 检查是否有错误
	for i, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("multi-source fetch: source %d: %w", i, r.err)
		}
	}

	// 验证一致性：所有结果必须与第一个相同
	ref := results[0].header
	for i := 1; i < len(results); i++ {
		if !headerEqual(ref, results[i].header) {
			return nil, fmt.Errorf("multi-source fetch: header mismatch at source %d: %w", i, ErrInconsistentData)
		}
	}

	return ref, nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/services/ -run "TestBlockqsQueryType|TestBlockHeader|TestMemBlockqs|TestBlockqsMultiSource"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/services/blockqs.go internal/services/blockqs_test.go
git commit -m "feat(services): add BlockqsClient interface, MemBlockqs mock and multi-source verification"
```

---

## Task 3: Depots 客户端接口 (internal/services/depots.go)

**Files:**
- Create: `internal/services/depots.go`
- Test: `internal/services/depots_test.go`

本 Task 定义 Depots 数据驿站的客户端接口 `DepotsClient`、稀缺性信息 `ScarcityInfo`、数据心跳 `DataHeartbeat`、以及内存 Mock 实现 `MemDepots`。

### Step 1: 写失败测试

创建 `internal/services/depots_test.go`：

```go
package services

import (
	"bytes"
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- MemDepots Mock 基础测试 ---

func TestMemDepots_FetchBlock(t *testing.T) {
	mock := NewMemDepots()

	blockHash := types.Hash512{0x01}
	blockData := []byte("block data content")
	mock.AddBlock(blockHash, blockData)

	got, err := mock.FetchBlock(blockHash)
	if err != nil {
		t.Fatalf("FetchBlock() error = %v", err)
	}
	if !bytes.Equal(got, blockData) {
		t.Errorf("FetchBlock() = %q, want %q", got, blockData)
	}

	_, err = mock.FetchBlock(types.Hash512{0xff})
	if err == nil {
		t.Error("FetchBlock() should return error for non-existent hash")
	}
}

func TestMemDepots_FetchAttachment(t *testing.T) {
	mock := NewMemDepots()

	attachID := types.Hash512{0x42}
	data := []byte("large attachment data")
	mock.AddAttachment(attachID, data)

	got, err := mock.FetchAttachment(attachID)
	if err != nil {
		t.Fatalf("FetchAttachment() error = %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("FetchAttachment() = %q, want %q", got, data)
	}

	_, err = mock.FetchAttachment(types.Hash512{0xff})
	if err == nil {
		t.Error("FetchAttachment() should return error for non-existent attachID")
	}
}

func TestMemDepots_FetchShard(t *testing.T) {
	mock := NewMemDepots()

	attachID := types.Hash512{0x01}
	mock.AddShard(attachID, 0, []byte("shard-0"))
	mock.AddShard(attachID, 1, []byte("shard-1"))

	got, err := mock.FetchShard(attachID, 0)
	if err != nil {
		t.Fatalf("FetchShard(0) error = %v", err)
	}
	if !bytes.Equal(got, []byte("shard-0")) {
		t.Errorf("FetchShard(0) = %q, want %q", got, "shard-0")
	}

	got, err = mock.FetchShard(attachID, 1)
	if err != nil {
		t.Fatalf("FetchShard(1) error = %v", err)
	}
	if !bytes.Equal(got, []byte("shard-1")) {
		t.Errorf("FetchShard(1) = %q, want %q", got, "shard-1")
	}

	_, err = mock.FetchShard(attachID, 999)
	if err == nil {
		t.Error("FetchShard() should return error for non-existent shard index")
	}
}

func TestMemDepots_ProbeAvailability(t *testing.T) {
	mock := NewMemDepots()

	// 设置数据可用，跳数为 3
	dataID := types.Hash512{0x01}
	mock.SetAvailability(dataID, 3)

	hops, err := mock.ProbeAvailability(dataID)
	if err != nil {
		t.Fatalf("ProbeAvailability() error = %v", err)
	}
	if hops != 3 {
		t.Errorf("ProbeAvailability() hops = %d, want 3", hops)
	}

	// 不存在的数据返回错误
	_, err = mock.ProbeAvailability(types.Hash512{0xff})
	if err == nil {
		t.Error("ProbeAvailability() should return error for unavailable data")
	}
}

func TestMemDepots_Upload(t *testing.T) {
	mock := NewMemDepots()

	dataID := types.Hash512{0x01}
	data := []byte("uploaded data")

	err := mock.Upload(dataID, data)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	// 上传后应该能获取到
	got, err := mock.FetchAttachment(dataID)
	if err != nil {
		t.Fatalf("FetchAttachment() after Upload() error = %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("FetchAttachment() after Upload() = %q, want %q", got, data)
	}
}

// --- ScarcityInfo 测试 ---

func TestScarcityInfo_Fields(t *testing.T) {
	info := ScarcityInfo{
		DataID:    types.Hash512{0x01},
		HopCount:  5,
		Available: true,
	}

	if info.DataID[0] != 0x01 {
		t.Error("ScarcityInfo.DataID mismatch")
	}
	if info.HopCount != 5 {
		t.Errorf("ScarcityInfo.HopCount = %d, want 5", info.HopCount)
	}
	if !info.Available {
		t.Error("ScarcityInfo.Available should be true")
	}
}

func TestScarcityInfo_IsScarce(t *testing.T) {
	tests := []struct {
		name      string
		info      ScarcityInfo
		threshold int
		want      bool
	}{
		{
			name:      "hop count above threshold",
			info:      ScarcityInfo{HopCount: 10, Available: true},
			threshold: 5,
			want:      true,
		},
		{
			name:      "hop count below threshold",
			info:      ScarcityInfo{HopCount: 2, Available: true},
			threshold: 5,
			want:      false,
		},
		{
			name:      "hop count equal threshold",
			info:      ScarcityInfo{HopCount: 5, Available: true},
			threshold: 5,
			want:      false,
		},
		{
			name:      "unavailable always scarce",
			info:      ScarcityInfo{HopCount: 0, Available: false},
			threshold: 5,
			want:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.IsScarce(tt.threshold)
			if got != tt.want {
				t.Errorf("IsScarce(%d) = %v, want %v", tt.threshold, got, tt.want)
			}
		})
	}
}

// --- DataHeartbeat 测试 ---

func TestDataHeartbeat_ShouldProbe(t *testing.T) {
	hb := NewDataHeartbeat(1 * time.Hour)

	dataID := types.Hash512{0x01}

	// 从未探测过——应该探测
	if !hb.ShouldProbe(dataID) {
		t.Error("ShouldProbe() should return true for never-probed data")
	}

	// 标记为已探测
	hb.MarkProbed(dataID)

	// 刚探测过——不应该再探测
	if hb.ShouldProbe(dataID) {
		t.Error("ShouldProbe() should return false for recently probed data")
	}
}

func TestDataHeartbeat_ShouldProbe_Expired(t *testing.T) {
	// 极短的周期以便测试过期
	hb := NewDataHeartbeat(1 * time.Millisecond)

	dataID := types.Hash512{0x02}
	hb.MarkProbed(dataID)

	// 等待超过周期
	time.Sleep(5 * time.Millisecond)

	if !hb.ShouldProbe(dataID) {
		t.Error("ShouldProbe() should return true after probe interval expired")
	}
}

func TestDataHeartbeat_Probe(t *testing.T) {
	hb := NewDataHeartbeat(1 * time.Hour)
	mock := NewMemDepots()

	dataID := types.Hash512{0x01}
	mock.SetAvailability(dataID, 3)

	info, err := hb.Probe(mock, dataID)
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if info.HopCount != 3 {
		t.Errorf("Probe().HopCount = %d, want 3", info.HopCount)
	}
	if !info.Available {
		t.Error("Probe().Available should be true")
	}

	// 探测后应标记为已探测
	if hb.ShouldProbe(dataID) {
		t.Error("ShouldProbe() should return false after Probe()")
	}
}

func TestDataHeartbeat_Probe_Unavailable(t *testing.T) {
	hb := NewDataHeartbeat(1 * time.Hour)
	mock := NewMemDepots()

	dataID := types.Hash512{0xff}

	info, err := hb.Probe(mock, dataID)
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if info.Available {
		t.Error("Probe().Available should be false for unavailable data")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/services/ -run "TestMemDepots|TestScarcityInfo|TestDataHeartbeat"
```

预期输出：编译失败，`DepotsClient`、`MemDepots`、`ScarcityInfo`、`DataHeartbeat` 等未定义。

### Step 3: 写最小实现

创建 `internal/services/depots.go`：

```go
package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// DepotsClient 数据驿站客户端接口。
// 定义与 Depots 服务交互的全部数据操作。
// Depots 负责区块/交易附件的持久化存储与 P2P 文件共享。
type DepotsClient interface {
	// FetchBlock 获取完整区块的原始字节。
	FetchBlock(blockHash types.Hash512) ([]byte, error)
	// FetchAttachment 获取附件数据。
	FetchAttachment(attachID types.Hash512) ([]byte, error)
	// FetchShard 获取附件的指定分片。
	FetchShard(attachID types.Hash512, shardIndex int) ([]byte, error)
	// ProbeAvailability 探测数据可用性，返回跳数。
	// 跳数越高表示数据越稀缺。
	ProbeAvailability(dataID types.Hash512) (hopCount int, err error)
	// Upload 上传数据到网络。
	// 触发附近节点的补充存储行为。
	Upload(dataID types.Hash512, data []byte) error
}

// ScarcityInfo 数据稀缺性信息。
// 通过广播探测获取，跳数可粗略衡量数据的稀缺程度。
type ScarcityInfo struct {
	DataID    types.Hash512 // 数据标识
	HopCount  int           // 跳数（距最近持有者的网络距离）
	Available bool          // 数据是否可用
}

// IsScarce 判断数据是否稀缺。
// 跳数超过阈值或数据不可用时视为稀缺。
func (s *ScarcityInfo) IsScarce(hopThreshold int) bool {
	if !s.Available {
		return true
	}
	return s.HopCount > hopThreshold
}

// DataHeartbeat 数据心跳管理器。
// 公益节点使用心跳周期性探测冷数据或低需求数据，
// 以维持其在网络中的可用性。
type DataHeartbeat struct {
	interval  time.Duration           // 探测周期
	lastProbe map[types.Hash512]time.Time // 各数据的最后探测时间
	mu        sync.RWMutex
}

// NewDataHeartbeat 创建一个数据心跳管理器。
// interval 为探测周期。
func NewDataHeartbeat(interval time.Duration) *DataHeartbeat {
	return &DataHeartbeat{
		interval:  interval,
		lastProbe: make(map[types.Hash512]time.Time),
	}
}

// ShouldProbe 检查是否应该对指定数据进行探测。
// 如果从未探测过，或距上次探测已超过设定周期，返回 true。
func (dh *DataHeartbeat) ShouldProbe(dataID types.Hash512) bool {
	dh.mu.RLock()
	defer dh.mu.RUnlock()

	last, ok := dh.lastProbe[dataID]
	if !ok {
		return true
	}
	return time.Since(last) >= dh.interval
}

// MarkProbed 标记指定数据已被探测。
func (dh *DataHeartbeat) MarkProbed(dataID types.Hash512) {
	dh.mu.Lock()
	defer dh.mu.Unlock()
	dh.lastProbe[dataID] = time.Now()
}

// Probe 对指定数据执行一次探测。
// 使用 DepotsClient 探测数据可用性，并更新心跳记录。
// 如果 ProbeAvailability 返回错误，说明数据不可用（返回 Available=false）。
func (dh *DataHeartbeat) Probe(client DepotsClient, dataID types.Hash512) (*ScarcityInfo, error) {
	hopCount, err := client.ProbeAvailability(dataID)

	// 无论探测结果如何，都标记为已探测
	dh.MarkProbed(dataID)

	if err != nil {
		// 探测失败表示数据不可用
		return &ScarcityInfo{
			DataID:    dataID,
			HopCount:  0,
			Available: false,
		}, nil
	}

	return &ScarcityInfo{
		DataID:    dataID,
		HopCount:  hopCount,
		Available: true,
	}, nil
}

// shardKey 用于构造分片存储的 map 键。
type shardKey struct {
	attachID   types.Hash512
	shardIndex int
}

// MemDepots 是 DepotsClient 的内存 Mock 实现，用于测试。
type MemDepots struct {
	mu           sync.RWMutex
	blocks       map[types.Hash512][]byte // 区块数据
	attachments  map[types.Hash512][]byte // 附件数据
	shards       map[shardKey][]byte      // 分片数据
	availability map[types.Hash512]int    // 可用性信息（跳数）
}

// NewMemDepots 创建一个空的内存 Mock Depots 客户端。
func NewMemDepots() *MemDepots {
	return &MemDepots{
		blocks:       make(map[types.Hash512][]byte),
		attachments:  make(map[types.Hash512][]byte),
		shards:       make(map[shardKey][]byte),
		availability: make(map[types.Hash512]int),
	}
}

// --- Mock 数据添加方法 ---

// AddBlock 向 Mock 中添加区块数据。
func (m *MemDepots) AddBlock(hash types.Hash512, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocks[hash] = data
}

// AddAttachment 向 Mock 中添加附件数据。
func (m *MemDepots) AddAttachment(attachID types.Hash512, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachments[attachID] = data
}

// AddShard 向 Mock 中添加分片数据。
func (m *MemDepots) AddShard(attachID types.Hash512, shardIndex int, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shards[shardKey{attachID: attachID, shardIndex: shardIndex}] = data
}

// SetAvailability 设置数据的可用性跳数。
func (m *MemDepots) SetAvailability(dataID types.Hash512, hopCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.availability[dataID] = hopCount
}

// --- DepotsClient 接口实现 ---

// FetchBlock 获取完整区块的原始字节。
func (m *MemDepots) FetchBlock(blockHash types.Hash512) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.blocks[blockHash]
	if !ok {
		return nil, fmt.Errorf("fetch block: %w", ErrNodeNotFound)
	}
	return data, nil
}

// FetchAttachment 获取附件数据。
func (m *MemDepots) FetchAttachment(attachID types.Hash512) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.attachments[attachID]
	if !ok {
		return nil, fmt.Errorf("fetch attachment: %w", ErrNodeNotFound)
	}
	return data, nil
}

// FetchShard 获取附件的指定分片。
func (m *MemDepots) FetchShard(attachID types.Hash512, shardIndex int) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := shardKey{attachID: attachID, shardIndex: shardIndex}
	data, ok := m.shards[key]
	if !ok {
		return nil, fmt.Errorf("fetch shard: index %d: %w", shardIndex, ErrNodeNotFound)
	}
	return data, nil
}

// ProbeAvailability 探测数据可用性，返回跳数。
func (m *MemDepots) ProbeAvailability(dataID types.Hash512) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hops, ok := m.availability[dataID]
	if !ok {
		return 0, fmt.Errorf("probe availability: %w", ErrServiceUnavailable)
	}
	return hops, nil
}

// Upload 上传数据到网络（Mock 将数据存入附件 map）。
func (m *MemDepots) Upload(dataID types.Hash512, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachments[dataID] = data
	// 上传后设置可用性（跳数为 0，表示本地持有）
	m.availability[dataID] = 0
	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/services/ -run "TestMemDepots|TestScarcityInfo|TestDataHeartbeat"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/services/depots.go internal/services/depots_test.go
git commit -m "feat(services): add DepotsClient interface, ScarcityInfo, DataHeartbeat and MemDepots mock"
```

---

## Task 4: STUN 客户端接口 (internal/services/stun.go)

**Files:**
- Create: `internal/services/stun.go`
- Test: `internal/services/stun_test.go`

本 Task 定义 STUN NAT 穿透服务的客户端接口 `StunClient`、NAT 类型枚举 `NATType`、以及内存 Mock 实现 `MemStun`。

### Step 1: 写失败测试

创建 `internal/services/stun_test.go`：

```go
package services

import (
	"testing"
)

// --- NATType 枚举测试 ---

func TestNATType_Values(t *testing.T) {
	// 确认 NAT 类型枚举值互不相同
	natSet := map[NATType]bool{
		NATNone:           true,
		NATFullCone:       true,
		NATRestrictedCone: true,
		NATPortRestricted: true,
		NATSymmetric:      true,
		NATUnknown:        true,
	}
	if len(natSet) != 6 {
		t.Errorf("expected 6 distinct NATType values, got %d", len(natSet))
	}
}

func TestNATType_String(t *testing.T) {
	tests := []struct {
		name string
		nt   NATType
		want string
	}{
		{"none", NATNone, "none"},
		{"full_cone", NATFullCone, "full_cone"},
		{"restricted_cone", NATRestrictedCone, "restricted_cone"},
		{"port_restricted", NATPortRestricted, "port_restricted"},
		{"symmetric", NATSymmetric, "symmetric"},
		{"unknown", NATUnknown, "unknown"},
		{"invalid", NATType(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.nt.String()
			if got != tt.want {
				t.Errorf("NATType(%d).String() = %q, want %q", tt.nt, got, tt.want)
			}
		})
	}
}

func TestNATType_IsTraversable(t *testing.T) {
	tests := []struct {
		name string
		nt   NATType
		want bool
	}{
		{"none is traversable", NATNone, true},
		{"full cone is traversable", NATFullCone, true},
		{"restricted cone is traversable", NATRestrictedCone, true},
		{"port restricted is traversable", NATPortRestricted, true},
		{"symmetric is not traversable", NATSymmetric, false},
		{"unknown is not traversable", NATUnknown, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.nt.IsTraversable()
			if got != tt.want {
				t.Errorf("NATType(%d).IsTraversable() = %v, want %v", tt.nt, got, tt.want)
			}
		})
	}
}

// --- MemStun Mock 测试 ---

func TestMemStun_DetectNATType(t *testing.T) {
	mock := NewMemStun(NATFullCone, "1.2.3.4:5678")

	natType, err := mock.DetectNATType()
	if err != nil {
		t.Fatalf("DetectNATType() error = %v", err)
	}
	if natType != NATFullCone {
		t.Errorf("DetectNATType() = %v, want %v", natType, NATFullCone)
	}
}

func TestMemStun_GetExternalAddr(t *testing.T) {
	mock := NewMemStun(NATNone, "203.0.113.1:12345")

	addr, err := mock.GetExternalAddr()
	if err != nil {
		t.Fatalf("GetExternalAddr() error = %v", err)
	}
	if addr != "203.0.113.1:12345" {
		t.Errorf("GetExternalAddr() = %q, want %q", addr, "203.0.113.1:12345")
	}
}

func TestMemStun_RequestHolePunch(t *testing.T) {
	mock := NewMemStun(NATFullCone, "1.2.3.4:5678")

	// 可穿透的 NAT 类型应成功
	err := mock.RequestHolePunch("10.0.0.1:8080")
	if err != nil {
		t.Errorf("RequestHolePunch() error = %v", err)
	}
}

func TestMemStun_RequestHolePunch_Symmetric(t *testing.T) {
	mock := NewMemStun(NATSymmetric, "1.2.3.4:5678")

	// 对称 NAT 打洞应失败
	err := mock.RequestHolePunch("10.0.0.1:8080")
	if err == nil {
		t.Error("RequestHolePunch() should fail for symmetric NAT")
	}
}

func TestMemStun_RequestHolePunch_EmptyAddr(t *testing.T) {
	mock := NewMemStun(NATFullCone, "1.2.3.4:5678")

	err := mock.RequestHolePunch("")
	if err == nil {
		t.Error("RequestHolePunch() should fail for empty address")
	}
}

func TestMemStun_Unavailable(t *testing.T) {
	// 模拟不可用的 STUN 服务
	mock := NewMemStunUnavailable()

	_, err := mock.DetectNATType()
	if err == nil {
		t.Error("DetectNATType() should fail when service is unavailable")
	}

	_, err = mock.GetExternalAddr()
	if err == nil {
		t.Error("GetExternalAddr() should fail when service is unavailable")
	}

	err = mock.RequestHolePunch("10.0.0.1:8080")
	if err == nil {
		t.Error("RequestHolePunch() should fail when service is unavailable")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/services/ -run "TestNATType|TestMemStun"
```

预期输出：编译失败，`NATType`、`StunClient`、`MemStun` 等未定义。

### Step 3: 写最小实现

创建 `internal/services/stun.go`：

```go
package services

import (
	"errors"
	"fmt"
)

// NATType 表示 NAT 类型。
type NATType byte

const (
	// NATNone 无 NAT（公网直连）。
	NATNone NATType = iota
	// NATFullCone 完全锥形 NAT。
	NATFullCone
	// NATRestrictedCone 受限锥形 NAT。
	NATRestrictedCone
	// NATPortRestricted 端口受限 NAT。
	NATPortRestricted
	// NATSymmetric 对称 NAT（最难穿透）。
	NATSymmetric
	// NATUnknown 未知 NAT 类型。
	NATUnknown
)

// String 返回 NAT 类型的字符串表示。
func (nt NATType) String() string {
	switch nt {
	case NATNone:
		return "none"
	case NATFullCone:
		return "full_cone"
	case NATRestrictedCone:
		return "restricted_cone"
	case NATPortRestricted:
		return "port_restricted"
	case NATSymmetric:
		return "symmetric"
	case NATUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// IsTraversable 检查该 NAT 类型是否支持穿透。
// 对称 NAT 和未知类型不支持穿透。
func (nt NATType) IsTraversable() bool {
	return nt >= NATNone && nt <= NATPortRestricted
}

// StunClient STUN NAT 穿透服务客户端接口。
// 提供 NAT 类型检测、打洞协助和外部地址获取功能。
// 这是通用服务，不限于区块链使用。
type StunClient interface {
	// DetectNATType 检测当前节点的 NAT 类型。
	DetectNATType() (NATType, error)
	// RequestHolePunch 请求向目标地址打洞。
	// targetAddr 格式为 "ip:port"。
	RequestHolePunch(targetAddr string) error
	// GetExternalAddr 获取当前节点的外部地址。
	// 返回格式为 "ip:port"。
	GetExternalAddr() (string, error)
}

// ErrHolePunchFailed 打洞失败。
var ErrHolePunchFailed = errors.New("hole punch failed")

// ErrEmptyAddress 目标地址为空。
var ErrEmptyAddress = errors.New("target address is empty")

// MemStun 是 StunClient 的内存 Mock 实现，用于测试。
type MemStun struct {
	natType      NATType // 模拟的 NAT 类型
	externalAddr string  // 模拟的外部地址
	available    bool    // 服务是否可用
}

// NewMemStun 创建一个可用的 Mock STUN 客户端。
func NewMemStun(natType NATType, externalAddr string) *MemStun {
	return &MemStun{
		natType:      natType,
		externalAddr: externalAddr,
		available:    true,
	}
}

// NewMemStunUnavailable 创建一个不可用的 Mock STUN 客户端。
func NewMemStunUnavailable() *MemStun {
	return &MemStun{
		natType:   NATUnknown,
		available: false,
	}
}

// DetectNATType 检测 NAT 类型。
func (m *MemStun) DetectNATType() (NATType, error) {
	if !m.available {
		return NATUnknown, fmt.Errorf("detect nat type: %w", ErrServiceUnavailable)
	}
	return m.natType, nil
}

// RequestHolePunch 请求打洞。
// 对称 NAT 下打洞一定失败。
func (m *MemStun) RequestHolePunch(targetAddr string) error {
	if !m.available {
		return fmt.Errorf("request hole punch: %w", ErrServiceUnavailable)
	}
	if targetAddr == "" {
		return fmt.Errorf("request hole punch: %w", ErrEmptyAddress)
	}
	if !m.natType.IsTraversable() {
		return fmt.Errorf("request hole punch: nat type %s: %w", m.natType, ErrHolePunchFailed)
	}
	return nil
}

// GetExternalAddr 获取外部地址。
func (m *MemStun) GetExternalAddr() (string, error) {
	if !m.available {
		return "", fmt.Errorf("get external addr: %w", ErrServiceUnavailable)
	}
	return m.externalAddr, nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/services/ -run "TestNATType|TestMemStun"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/services/stun.go internal/services/stun_test.go
git commit -m "feat(services): add StunClient interface, NATType enum and MemStun mock"
```

---

## Task 5: 服务质量评估框架 (internal/services/evaluator.go)

**Files:**
- Create: `internal/services/evaluator.go`
- Test: `internal/services/evaluator_test.go`

本 Task 实现服务质量评估框架，包括通用评估接口 `ServiceEvaluator`、评估结果 `EvalResult`、以及三类服务各自的评估器实现 `StunEvaluator`、`DepotsEvaluator`、`BlockqsEvaluator`。

### Step 1: 写失败测试

创建 `internal/services/evaluator_test.go`：

```go
package services

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- EvalResult 测试 ---

func TestEvalResult_Fields(t *testing.T) {
	r := EvalResult{
		Score:        0.85,
		Available:    true,
		ResponseTime: 50 * time.Millisecond,
		Details:      "test evaluation",
	}

	if r.Score != 0.85 {
		t.Errorf("Score = %f, want 0.85", r.Score)
	}
	if !r.Available {
		t.Error("Available should be true")
	}
	if r.ResponseTime != 50*time.Millisecond {
		t.Errorf("ResponseTime = %v, want 50ms", r.ResponseTime)
	}
	if r.Details != "test evaluation" {
		t.Errorf("Details = %q, want %q", r.Details, "test evaluation")
	}
}

func TestEvalResult_IsPassing(t *testing.T) {
	tests := []struct {
		name      string
		result    EvalResult
		threshold float64
		want      bool
	}{
		{
			name:      "above threshold",
			result:    EvalResult{Score: 0.8, Available: true},
			threshold: 0.5,
			want:      true,
		},
		{
			name:      "below threshold",
			result:    EvalResult{Score: 0.3, Available: true},
			threshold: 0.5,
			want:      false,
		},
		{
			name:      "equal threshold",
			result:    EvalResult{Score: 0.5, Available: true},
			threshold: 0.5,
			want:      true,
		},
		{
			name:      "unavailable always fails",
			result:    EvalResult{Score: 1.0, Available: false},
			threshold: 0.0,
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.IsPassing(tt.threshold)
			if got != tt.want {
				t.Errorf("IsPassing(%f) = %v, want %v", tt.threshold, got, tt.want)
			}
		})
	}
}

// --- StunEvaluator 测试 ---

func TestStunEvaluator_Evaluate_Available(t *testing.T) {
	mock := NewMemStun(NATFullCone, "1.2.3.4:5678")
	evaluator := NewStunEvaluator(mock)

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceStun,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !result.Available {
		t.Error("Evaluate().Available should be true for available STUN")
	}
	if result.Score <= 0.0 || result.Score > 1.0 {
		t.Errorf("Evaluate().Score = %f, want (0.0, 1.0]", result.Score)
	}
	if result.ResponseTime < 0 {
		t.Error("Evaluate().ResponseTime should be non-negative")
	}
}

func TestStunEvaluator_Evaluate_Unavailable(t *testing.T) {
	mock := NewMemStunUnavailable()
	evaluator := NewStunEvaluator(mock)

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceStun,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Available {
		t.Error("Evaluate().Available should be false for unavailable STUN")
	}
	if result.Score != 0.0 {
		t.Errorf("Evaluate().Score = %f, want 0.0 for unavailable", result.Score)
	}
}

func TestStunEvaluator_Evaluate_SymmetricNAT(t *testing.T) {
	// 对称 NAT——NAT 检测成功但穿透质量差
	mock := NewMemStun(NATSymmetric, "1.2.3.4:5678")
	evaluator := NewStunEvaluator(mock)

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceStun,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !result.Available {
		t.Error("Evaluate().Available should be true (service reachable)")
	}
	// 对称 NAT 下分数应偏低但不为零（检测本身是成功的）
	if result.Score <= 0.0 {
		t.Errorf("Evaluate().Score = %f, should be > 0 for reachable service", result.Score)
	}
}

// --- DepotsEvaluator 测试 ---

func TestDepotsEvaluator_Evaluate_Available(t *testing.T) {
	mock := NewMemDepots()

	// 预设一些可探测的数据
	probeID := types.Hash512{0xaa}
	mock.SetAvailability(probeID, 2)
	mock.AddAttachment(probeID, []byte("test data"))

	evaluator := NewDepotsEvaluator(mock, []types.Hash512{probeID})

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceDepots,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !result.Available {
		t.Error("Evaluate().Available should be true")
	}
	if result.Score <= 0.0 || result.Score > 1.0 {
		t.Errorf("Evaluate().Score = %f, want (0.0, 1.0]", result.Score)
	}
}

func TestDepotsEvaluator_Evaluate_NoData(t *testing.T) {
	mock := NewMemDepots()

	// 没有任何数据
	probeID := types.Hash512{0xff}
	evaluator := NewDepotsEvaluator(mock, []types.Hash512{probeID})

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceDepots,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Score != 0.0 {
		t.Errorf("Evaluate().Score = %f, want 0.0 for node with no data", result.Score)
	}
}

func TestDepotsEvaluator_Evaluate_EmptyProbeIDs(t *testing.T) {
	mock := NewMemDepots()
	evaluator := NewDepotsEvaluator(mock, nil)

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceDepots,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	// 没有探测目标时无法评估，分数为 0
	if result.Score != 0.0 {
		t.Errorf("Evaluate().Score = %f, want 0.0 with no probe targets", result.Score)
	}
}

// --- BlockqsEvaluator 测试 ---

func TestBlockqsEvaluator_Evaluate_Available(t *testing.T) {
	mock := NewMemBlockqs()

	// 预设可查询的区块头
	mock.AddHeader(&BlockHeader{Version: 1, Height: 100, CheckRoot: types.Hash512{0x01}})

	evaluator := NewBlockqsEvaluator(mock, []int{100})

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceBlockqs,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !result.Available {
		t.Error("Evaluate().Available should be true")
	}
	if result.Score <= 0.0 || result.Score > 1.0 {
		t.Errorf("Evaluate().Score = %f, want (0.0, 1.0]", result.Score)
	}
}

func TestBlockqsEvaluator_Evaluate_MissingData(t *testing.T) {
	mock := NewMemBlockqs()

	// 不预设数据
	evaluator := NewBlockqsEvaluator(mock, []int{100, 200})

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceBlockqs,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Score != 0.0 {
		t.Errorf("Evaluate().Score = %f, want 0.0 for node with no data", result.Score)
	}
}

func TestBlockqsEvaluator_Evaluate_PartialData(t *testing.T) {
	mock := NewMemBlockqs()

	// 只有部分区块头
	mock.AddHeader(&BlockHeader{Version: 1, Height: 100, CheckRoot: types.Hash512{0x01}})
	// 高度 200 缺失

	evaluator := NewBlockqsEvaluator(mock, []int{100, 200})

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceBlockqs,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	// 部分数据：分数应在 0 和 1 之间
	if result.Score <= 0.0 || result.Score >= 1.0 {
		t.Errorf("Evaluate().Score = %f, want (0.0, 1.0) for partial data", result.Score)
	}
}

func TestBlockqsEvaluator_Evaluate_EmptyProbeHeights(t *testing.T) {
	mock := NewMemBlockqs()
	evaluator := NewBlockqsEvaluator(mock, nil)

	node := ServiceNode{
		ID:   NodeID{0x01},
		Type: ServiceBlockqs,
	}

	result, err := evaluator.Evaluate(node)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Score != 0.0 {
		t.Errorf("Evaluate().Score = %f, want 0.0 with no probe targets", result.Score)
	}
}

// --- 评分范围验证 ---

func TestEvaluator_ScoreRange(t *testing.T) {
	// 验证所有评估器的评分在 [0.0, 1.0] 范围内

	// STUN 评估
	stunMock := NewMemStun(NATFullCone, "1.2.3.4:5678")
	stunEval := NewStunEvaluator(stunMock)
	node := ServiceNode{ID: NodeID{0x01}, Type: ServiceStun}
	r, err := stunEval.Evaluate(node)
	if err != nil {
		t.Fatalf("StunEvaluator.Evaluate() error = %v", err)
	}
	if r.Score < 0.0 || r.Score > 1.0 {
		t.Errorf("StunEvaluator score %f out of range [0.0, 1.0]", r.Score)
	}

	// Depots 评估
	depotsMock := NewMemDepots()
	probeID := types.Hash512{0x01}
	depotsMock.SetAvailability(probeID, 1)
	depotsMock.AddAttachment(probeID, []byte("data"))
	depotsEval := NewDepotsEvaluator(depotsMock, []types.Hash512{probeID})
	node.Type = ServiceDepots
	r, err = depotsEval.Evaluate(node)
	if err != nil {
		t.Fatalf("DepotsEvaluator.Evaluate() error = %v", err)
	}
	if r.Score < 0.0 || r.Score > 1.0 {
		t.Errorf("DepotsEvaluator score %f out of range [0.0, 1.0]", r.Score)
	}

	// Blockqs 评估
	bqMock := NewMemBlockqs()
	bqMock.AddHeader(&BlockHeader{Version: 1, Height: 10})
	bqEval := NewBlockqsEvaluator(bqMock, []int{10})
	node.Type = ServiceBlockqs
	r, err = bqEval.Evaluate(node)
	if err != nil {
		t.Fatalf("BlockqsEvaluator.Evaluate() error = %v", err)
	}
	if r.Score < 0.0 || r.Score > 1.0 {
		t.Errorf("BlockqsEvaluator score %f out of range [0.0, 1.0]", r.Score)
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/services/ -run "TestEvalResult|TestStunEvaluator|TestDepotsEvaluator|TestBlockqsEvaluator|TestEvaluator_ScoreRange"
```

预期输出：编译失败，`ServiceEvaluator`、`EvalResult`、`StunEvaluator`、`DepotsEvaluator`、`BlockqsEvaluator` 等未定义。

### Step 3: 写最小实现

创建 `internal/services/evaluator.go`：

```go
package services

import (
	"fmt"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// ServiceEvaluator 服务质量评估接口。
// 各类服务评估器都应实现此接口。
type ServiceEvaluator interface {
	// Evaluate 评估单个服务节点的质量。
	// 返回评估结果，包含评分、可用性、响应时间等信息。
	Evaluate(node ServiceNode) (*EvalResult, error)
}

// EvalResult 服务评估结果。
type EvalResult struct {
	Score        float64       // 评分（0.0 ~ 1.0）
	Available    bool          // 服务是否可达
	ResponseTime time.Duration // 响应时间
	Details      string        // 评估详情描述
}

// IsPassing 检查评估结果是否达到指定阈值。
// 不可达的服务始终不通过。
func (r *EvalResult) IsPassing(threshold float64) bool {
	if !r.Available {
		return false
	}
	return r.Score >= threshold
}

// --- STUN 评估器 ---

// StunEvaluator 评估 STUN NAT 穿透服务的质量。
// 评估标准：
//   - 连通性：能否成功执行 NAT 类型检测
//   - 准确性：NAT 类型检测结果是否合理
//   - 响应时间：检测所花费的时间
type StunEvaluator struct {
	client StunClient
}

// NewStunEvaluator 创建一个 STUN 评估器。
func NewStunEvaluator(client StunClient) *StunEvaluator {
	return &StunEvaluator{client: client}
}

// Evaluate 评估 STUN 服务节点。
func (e *StunEvaluator) Evaluate(node ServiceNode) (*EvalResult, error) {
	start := time.Now()

	// 步骤 1：检测 NAT 类型
	natType, err := e.client.DetectNATType()
	elapsed := time.Since(start)

	if err != nil {
		// 服务不可达
		return &EvalResult{
			Score:        0.0,
			Available:    false,
			ResponseTime: elapsed,
			Details:      fmt.Sprintf("nat detection failed: %v", err),
		}, nil
	}

	// 步骤 2：获取外部地址
	addr, addrErr := e.client.GetExternalAddr()

	// 步骤 3：计算评分
	score := calcStunScore(natType, addr, addrErr, elapsed)

	details := fmt.Sprintf("nat_type=%s, external_addr=%s, response=%v", natType, addr, elapsed)

	return &EvalResult{
		Score:        score,
		Available:    true,
		ResponseTime: elapsed,
		Details:      details,
	}, nil
}

// calcStunScore 计算 STUN 服务评分。
// 评分维度：
//   - 连通性（40%）：NAT 检测是否成功 → 已通过（进入此函数说明已成功）
//   - 准确性（30%）：NAT 类型是否可穿透 + 是否能获取外部地址
//   - 响应时间（30%）：响应越快评分越高
func calcStunScore(natType NATType, addr string, addrErr error, responseTime time.Duration) float64 {
	var score float64

	// 连通性：40%（能到达此处说明已连通）
	score += 0.4

	// 准确性：30%
	accuracyScore := 0.0
	if natType != NATUnknown {
		accuracyScore += 0.5 // NAT 类型有效
	}
	if addrErr == nil && addr != "" {
		accuracyScore += 0.5 // 能获取外部地址
	}
	score += 0.3 * accuracyScore

	// 响应时间：30%
	// 基准：<100ms 满分，>2s 零分，线性插值
	timeScore := calcTimeScore(responseTime, 100*time.Millisecond, 2*time.Second)
	score += 0.3 * timeScore

	// 确保在 [0.0, 1.0] 范围内
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// --- Depots 评估器 ---

// DepotsEvaluator 评估 Depots 数据驿站服务的质量。
// 评估标准：
//   - 请求随机链上数据，检查可用性
//   - 数据完整性：能否正确返回数据
//   - 响应质量：响应时间与数据正确性
type DepotsEvaluator struct {
	client   DepotsClient
	probeIDs []types.Hash512 // 用于探测的数据 ID 列表
}

// NewDepotsEvaluator 创建一个 Depots 评估器。
// probeIDs 是用于探测的数据标识列表（通常从链上随机选取）。
func NewDepotsEvaluator(client DepotsClient, probeIDs []types.Hash512) *DepotsEvaluator {
	return &DepotsEvaluator{
		client:   client,
		probeIDs: probeIDs,
	}
}

// Evaluate 评估 Depots 服务节点。
func (e *DepotsEvaluator) Evaluate(node ServiceNode) (*EvalResult, error) {
	if len(e.probeIDs) == 0 {
		return &EvalResult{
			Score:        0.0,
			Available:    false,
			ResponseTime: 0,
			Details:      "no probe targets available",
		}, nil
	}

	start := time.Now()

	// 对每个探测目标进行可用性检查和数据获取
	successCount := 0
	totalTime := time.Duration(0)

	for _, probeID := range e.probeIDs {
		probeStart := time.Now()

		// 探测可用性
		_, probeErr := e.client.ProbeAvailability(probeID)
		if probeErr != nil {
			continue
		}

		// 尝试获取数据
		data, fetchErr := e.client.FetchAttachment(probeID)
		probeElapsed := time.Since(probeStart)
		totalTime += probeElapsed

		if fetchErr == nil && len(data) > 0 {
			successCount++
		}
	}

	elapsed := time.Since(start)

	if successCount == 0 {
		return &EvalResult{
			Score:        0.0,
			Available:    false,
			ResponseTime: elapsed,
			Details:      fmt.Sprintf("all %d probes failed", len(e.probeIDs)),
		}, nil
	}

	// 计算评分
	score := calcDepotsScore(successCount, len(e.probeIDs), totalTime)

	return &EvalResult{
		Score:        score,
		Available:    true,
		ResponseTime: elapsed,
		Details:      fmt.Sprintf("probes: %d/%d successful, total_time=%v", successCount, len(e.probeIDs), totalTime),
	}, nil
}

// calcDepotsScore 计算 Depots 服务评分。
// 评分维度：
//   - 数据完整性（60%）：成功探测的比例
//   - 响应质量（40%）：平均响应时间
func calcDepotsScore(successCount, totalProbes int, totalTime time.Duration) float64 {
	// 数据完整性：60%
	completenessScore := float64(successCount) / float64(totalProbes)

	// 响应质量：40%
	avgTime := totalTime / time.Duration(totalProbes)
	timeScore := calcTimeScore(avgTime, 200*time.Millisecond, 5*time.Second)

	score := 0.6*completenessScore + 0.4*timeScore

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// --- Blockqs 评估器 ---

// BlockqsEvaluator 评估 Blockqs 区块查询服务的质量。
// 评估标准：
//   - 常规可达性：能否成功查询区块头
//   - 性能：响应时间
//   - 数据正确性：返回的区块头是否与预期一致
type BlockqsEvaluator struct {
	client       BlockqsClient
	probeHeights []int // 用于探测的区块高度列表
}

// NewBlockqsEvaluator 创建一个 Blockqs 评估器。
// probeHeights 是用于探测的区块高度列表。
func NewBlockqsEvaluator(client BlockqsClient, probeHeights []int) *BlockqsEvaluator {
	return &BlockqsEvaluator{
		client:       client,
		probeHeights: probeHeights,
	}
}

// Evaluate 评估 Blockqs 服务节点。
func (e *BlockqsEvaluator) Evaluate(node ServiceNode) (*EvalResult, error) {
	if len(e.probeHeights) == 0 {
		return &EvalResult{
			Score:        0.0,
			Available:    false,
			ResponseTime: 0,
			Details:      "no probe heights available",
		}, nil
	}

	start := time.Now()

	successCount := 0
	totalTime := time.Duration(0)

	for _, height := range e.probeHeights {
		probeStart := time.Now()

		header, err := e.client.FetchHeader(height)
		probeElapsed := time.Since(probeStart)
		totalTime += probeElapsed

		if err == nil && header != nil && header.Height == int32(height) {
			successCount++
		}
	}

	elapsed := time.Since(start)

	if successCount == 0 {
		return &EvalResult{
			Score:        0.0,
			Available:    false,
			ResponseTime: elapsed,
			Details:      fmt.Sprintf("all %d probes failed", len(e.probeHeights)),
		}, nil
	}

	// 计算评分
	score := calcBlockqsScore(successCount, len(e.probeHeights), totalTime)

	return &EvalResult{
		Score:        score,
		Available:    true,
		ResponseTime: elapsed,
		Details:      fmt.Sprintf("probes: %d/%d successful, total_time=%v", successCount, len(e.probeHeights), totalTime),
	}, nil
}

// calcBlockqsScore 计算 Blockqs 服务评分。
// 评分维度：
//   - 数据正确性（50%）：成功查询的比例
//   - 响应时间（50%）：平均响应时间
func calcBlockqsScore(successCount, totalProbes int, totalTime time.Duration) float64 {
	// 数据正确性：50%
	correctnessScore := float64(successCount) / float64(totalProbes)

	// 响应时间：50%
	avgTime := totalTime / time.Duration(totalProbes)
	timeScore := calcTimeScore(avgTime, 50*time.Millisecond, 2*time.Second)

	score := 0.5*correctnessScore + 0.5*timeScore

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// --- 通用辅助函数 ---

// calcTimeScore 根据响应时间计算时间评分。
// 在 [bestTime, worstTime] 之间线性插值：
//   - 响应时间 <= bestTime → 1.0
//   - 响应时间 >= worstTime → 0.0
//   - 中间线性插值
func calcTimeScore(actual, bestTime, worstTime time.Duration) float64 {
	if actual <= bestTime {
		return 1.0
	}
	if actual >= worstTime {
		return 0.0
	}
	// 线性插值
	totalRange := float64(worstTime - bestTime)
	elapsed := float64(actual - bestTime)
	return 1.0 - (elapsed / totalRange)
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/services/ -run "TestEvalResult|TestStunEvaluator|TestDepotsEvaluator|TestBlockqsEvaluator|TestEvaluator_ScoreRange"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/services/evaluator.go internal/services/evaluator_test.go
git commit -m "feat(services): add ServiceEvaluator interface with Stun, Depots and Blockqs evaluators"
```

---

## Task 6: 奖励地址缓存与确认位图 (internal/services/reward.go)

**Files:**
- Create: `internal/services/reward.go`
- Test: `internal/services/reward_test.go`

本 Task 实现奖励地址缓存 `RewardAddressCache` 和确认位图 `ConfirmationBitmap`，包括缓存的增删改查、评分更新、地址选择策略（最高分/加权随机）、位图的各位操作、奖励状态计算和回收条件判断。

### Step 1: 写失败测试

创建 `internal/services/reward_test.go`：

```go
package services

import (
	"math"
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- 常量测试 ---

func TestConfirmationConstants(t *testing.T) {
	if ConfirmationWindowSize != 48 {
		t.Errorf("ConfirmationWindowSize = %d, want 48", ConfirmationWindowSize)
	}
	if RequiredConfirmations != 2 {
		t.Errorf("RequiredConfirmations = %d, want 2", RequiredConfirmations)
	}
	if BitmapSize != 18 {
		t.Errorf("BitmapSize = %d, want 18", BitmapSize)
	}
}

// --- RewardAddressCache 基本操作测试 ---

func TestRewardAddressCache_NewCache(t *testing.T) {
	cache := NewRewardAddressCache(500)
	if cache == nil {
		t.Fatal("NewRewardAddressCache() returned nil")
	}
}

func TestRewardAddressCache_Add(t *testing.T) {
	cache := NewRewardAddressCache(100)

	addr := types.PubKeyHash{0x01}
	cache.Add(ServiceBlockqs, addr, 0.8)

	// 应能通过 SelectForReward 获取到
	selected, err := cache.SelectForReward(ServiceBlockqs)
	if err != nil {
		t.Fatalf("SelectForReward() error = %v", err)
	}
	if selected != addr {
		t.Errorf("SelectForReward() = %v, want %v", selected, addr)
	}
}

func TestRewardAddressCache_Add_Multiple(t *testing.T) {
	cache := NewRewardAddressCache(100)

	addr1 := types.PubKeyHash{0x01}
	addr2 := types.PubKeyHash{0x02}
	addr3 := types.PubKeyHash{0x03}

	cache.Add(ServiceDepots, addr1, 0.5)
	cache.Add(ServiceDepots, addr2, 0.9)
	cache.Add(ServiceDepots, addr3, 0.3)

	// 最高分应该是 addr2
	selected, err := cache.SelectForReward(ServiceDepots)
	if err != nil {
		t.Fatalf("SelectForReward() error = %v", err)
	}
	if selected != addr2 {
		t.Errorf("SelectForReward() = %v, want %v (highest score)", selected, addr2)
	}
}

func TestRewardAddressCache_Add_Duplicate(t *testing.T) {
	cache := NewRewardAddressCache(100)

	addr := types.PubKeyHash{0x01}

	// 第一次添加评分 0.5
	cache.Add(ServiceBlockqs, addr, 0.5)
	// 第二次以更高评分添加——应更新而非重复
	cache.Add(ServiceBlockqs, addr, 0.9)

	selected, err := cache.SelectForReward(ServiceBlockqs)
	if err != nil {
		t.Fatalf("SelectForReward() error = %v", err)
	}
	if selected != addr {
		t.Errorf("SelectForReward() = %v, want %v", selected, addr)
	}
}

func TestRewardAddressCache_Remove(t *testing.T) {
	cache := NewRewardAddressCache(100)

	addr := types.PubKeyHash{0x01}
	cache.Add(ServiceBlockqs, addr, 0.8)
	cache.Remove(ServiceBlockqs, addr)

	_, err := cache.SelectForReward(ServiceBlockqs)
	if err == nil {
		t.Error("SelectForReward() should return error after Remove()")
	}
}

func TestRewardAddressCache_Remove_NonExistent(t *testing.T) {
	cache := NewRewardAddressCache(100)

	// 删除不存在的地址不应 panic
	cache.Remove(ServiceBlockqs, types.PubKeyHash{0xff})
}

func TestRewardAddressCache_SelectForReward_Empty(t *testing.T) {
	cache := NewRewardAddressCache(100)

	_, err := cache.SelectForReward(ServiceBlockqs)
	if err == nil {
		t.Error("SelectForReward() should return error for empty cache")
	}
}

func TestRewardAddressCache_SelectForReward_DifferentTypes(t *testing.T) {
	cache := NewRewardAddressCache(100)

	addrBQ := types.PubKeyHash{0x01}
	addrDP := types.PubKeyHash{0x02}

	cache.Add(ServiceBlockqs, addrBQ, 0.9)
	cache.Add(ServiceDepots, addrDP, 0.8)

	// Blockqs 应选择 addrBQ
	got, err := cache.SelectForReward(ServiceBlockqs)
	if err != nil {
		t.Fatalf("SelectForReward(Blockqs) error = %v", err)
	}
	if got != addrBQ {
		t.Errorf("SelectForReward(Blockqs) = %v, want %v", got, addrBQ)
	}

	// Depots 应选择 addrDP
	got, err = cache.SelectForReward(ServiceDepots)
	if err != nil {
		t.Fatalf("SelectForReward(Depots) error = %v", err)
	}
	if got != addrDP {
		t.Errorf("SelectForReward(Depots) = %v, want %v", got, addrDP)
	}
}

func TestRewardAddressCache_UpdateScore(t *testing.T) {
	cache := NewRewardAddressCache(100)

	addr1 := types.PubKeyHash{0x01}
	addr2 := types.PubKeyHash{0x02}

	cache.Add(ServiceStun, addr1, 0.9)
	cache.Add(ServiceStun, addr2, 0.5)

	// 更新 addr2 的评分使其超过 addr1
	cache.UpdateScore(ServiceStun, addr2, 0.95)

	selected, err := cache.SelectForReward(ServiceStun)
	if err != nil {
		t.Fatalf("SelectForReward() error = %v", err)
	}
	if selected != addr2 {
		t.Errorf("SelectForReward() = %v, want %v (updated score)", selected, addr2)
	}
}

func TestRewardAddressCache_UpdateScore_NonExistent(t *testing.T) {
	cache := NewRewardAddressCache(100)

	// 更新不存在的地址不应 panic
	cache.UpdateScore(ServiceStun, types.PubKeyHash{0xff}, 0.5)
}

// --- SelectRandom 加权随机选择测试 ---

func TestRewardAddressCache_SelectRandom(t *testing.T) {
	cache := NewRewardAddressCache(100)

	addr1 := types.PubKeyHash{0x01}
	addr2 := types.PubKeyHash{0x02}

	cache.Add(ServiceDepots, addr1, 0.8)
	cache.Add(ServiceDepots, addr2, 0.2)

	// 多次随机选择，验证分布合理
	counts := make(map[types.PubKeyHash]int)
	iterations := 10000
	for i := 0; i < iterations; i++ {
		selected, err := cache.SelectRandom(ServiceDepots)
		if err != nil {
			t.Fatalf("SelectRandom() error = %v", err)
		}
		counts[selected]++
	}

	// addr1 评分 0.8，addr2 评分 0.2
	// 理论比例 80:20，允许误差范围
	ratio1 := float64(counts[addr1]) / float64(iterations)
	ratio2 := float64(counts[addr2]) / float64(iterations)

	if ratio1 < 0.6 || ratio1 > 0.95 {
		t.Errorf("SelectRandom() addr1 ratio = %.2f, expected ~0.80", ratio1)
	}
	if ratio2 < 0.05 || ratio2 > 0.4 {
		t.Errorf("SelectRandom() addr2 ratio = %.2f, expected ~0.20", ratio2)
	}
}

func TestRewardAddressCache_SelectRandom_Empty(t *testing.T) {
	cache := NewRewardAddressCache(100)

	_, err := cache.SelectRandom(ServiceDepots)
	if err == nil {
		t.Error("SelectRandom() should return error for empty cache")
	}
}

func TestRewardAddressCache_SelectRandom_SingleEntry(t *testing.T) {
	cache := NewRewardAddressCache(100)

	addr := types.PubKeyHash{0x01}
	cache.Add(ServiceStun, addr, 0.5)

	selected, err := cache.SelectRandom(ServiceStun)
	if err != nil {
		t.Fatalf("SelectRandom() error = %v", err)
	}
	if selected != addr {
		t.Errorf("SelectRandom() = %v, want %v (only entry)", selected, addr)
	}
}

// --- Prune 清理测试 ---

func TestRewardAddressCache_Prune(t *testing.T) {
	cache := NewRewardAddressCache(100)

	// 添加一个正常条目和一个过期条目
	addrGood := types.PubKeyHash{0x01}
	addrBad := types.PubKeyHash{0x02}

	cache.Add(ServiceBlockqs, addrGood, 0.8)
	cache.Add(ServiceBlockqs, addrBad, 0.01) // 非常低的评分

	// 执行清理（应移除低分条目）
	cache.Prune()

	// 正常条目应仍然存在
	selected, err := cache.SelectForReward(ServiceBlockqs)
	if err != nil {
		t.Fatalf("SelectForReward() error after Prune = %v", err)
	}
	if selected != addrGood {
		t.Errorf("SelectForReward() = %v, want %v after Prune", selected, addrGood)
	}
}

func TestRewardAddressCache_MaxSize(t *testing.T) {
	cache := NewRewardAddressCache(3)

	// 添加 4 个条目，超过最大容量
	for i := byte(1); i <= 4; i++ {
		addr := types.PubKeyHash{i}
		cache.Add(ServiceBlockqs, addr, float64(i)*0.2)
	}

	// 应该只保留评分最高的 3 个
	// 评分：0.2, 0.4, 0.6, 0.8 → 最低的 0.2 应被淘汰

	// 添加后自动淘汰，验证剩余数量
	count := cache.Count(ServiceBlockqs)
	if count > 3 {
		t.Errorf("Count() = %d, want <= 3 (maxSize)", count)
	}
}

// --- ConfirmationBitmap 测试 ---

func TestConfirmationBitmap_SetAndGetBit(t *testing.T) {
	var bm ConfirmationBitmap

	tests := []struct {
		name        string
		serviceType ServiceType
		blockOffset int
		wantErr     bool
	}{
		{"depots offset 0", ServiceDepots, 0, false},
		{"depots offset 47", ServiceDepots, 47, false},
		{"blockqs offset 0", ServiceBlockqs, 0, false},
		{"blockqs offset 23", ServiceBlockqs, 23, false},
		{"stun offset 0", ServiceStun, 0, false},
		{"stun offset 47", ServiceStun, 47, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bm.SetBit(tt.serviceType, tt.blockOffset)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetBit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				got, err := bm.GetBit(tt.serviceType, tt.blockOffset)
				if err != nil {
					t.Fatalf("GetBit() error = %v", err)
				}
				if !got {
					t.Error("GetBit() = false after SetBit()")
				}
			}
		})
	}
}

func TestConfirmationBitmap_SetBit_OutOfRange(t *testing.T) {
	var bm ConfirmationBitmap

	err := bm.SetBit(ServiceDepots, 48)
	if err == nil {
		t.Error("SetBit() should return error for blockOffset >= 48")
	}

	err = bm.SetBit(ServiceDepots, -1)
	if err == nil {
		t.Error("SetBit() should return error for negative blockOffset")
	}
}

func TestConfirmationBitmap_SetBit_InvalidServiceType(t *testing.T) {
	var bm ConfirmationBitmap

	err := bm.SetBit(ServiceType(0), 0)
	if err == nil {
		t.Error("SetBit() should return error for invalid service type")
	}

	err = bm.SetBit(ServiceType(99), 0)
	if err == nil {
		t.Error("SetBit() should return error for invalid service type")
	}
}

func TestConfirmationBitmap_GetBit_Unset(t *testing.T) {
	var bm ConfirmationBitmap

	got, err := bm.GetBit(ServiceDepots, 10)
	if err != nil {
		t.Fatalf("GetBit() error = %v", err)
	}
	if got {
		t.Error("GetBit() = true for unset bit")
	}
}

func TestConfirmationBitmap_IndependentServiceTypes(t *testing.T) {
	var bm ConfirmationBitmap

	// 设置 Depots 的 offset 5
	_ = bm.SetBit(ServiceDepots, 5)

	// Blockqs 和 STUN 的 offset 5 应该是未设置的
	gotBQ, _ := bm.GetBit(ServiceBlockqs, 5)
	gotST, _ := bm.GetBit(ServiceStun, 5)

	if gotBQ {
		t.Error("Blockqs bit should not be set when only Depots was set")
	}
	if gotST {
		t.Error("Stun bit should not be set when only Depots was set")
	}
}

func TestConfirmationBitmap_AllBits(t *testing.T) {
	var bm ConfirmationBitmap

	// 设置所有位
	for st := ServiceDepots; st <= ServiceStun; st++ {
		for offset := 0; offset < ConfirmationWindowSize; offset++ {
			if err := bm.SetBit(st, offset); err != nil {
				t.Fatalf("SetBit(%v, %d) error = %v", st, offset, err)
			}
		}
	}

	// 验证所有位都已设置
	for st := ServiceDepots; st <= ServiceStun; st++ {
		for offset := 0; offset < ConfirmationWindowSize; offset++ {
			got, err := bm.GetBit(st, offset)
			if err != nil {
				t.Fatalf("GetBit(%v, %d) error = %v", st, offset, err)
			}
			if !got {
				t.Errorf("GetBit(%v, %d) = false, want true", st, offset)
			}
		}
	}

	// 验证所有字节都是 0xFF
	for i, b := range bm {
		if b != 0xFF {
			t.Errorf("bm[%d] = 0x%02x, want 0xFF", i, b)
		}
	}
}

// --- CountConfirmations 测试 ---

func TestCountConfirmations(t *testing.T) {
	// 创建多个位图模拟多个铸造者的确认
	bitmaps := []ConfirmationBitmap{}

	// 铸造者 1：确认 Depots offset 5
	var bm1 ConfirmationBitmap
	_ = bm1.SetBit(ServiceDepots, 5)
	bitmaps = append(bitmaps, bm1)

	// 铸造者 2：确认 Depots offset 5
	var bm2 ConfirmationBitmap
	_ = bm2.SetBit(ServiceDepots, 5)
	bitmaps = append(bitmaps, bm2)

	// 铸造者 3：不确认
	var bm3 ConfirmationBitmap
	bitmaps = append(bitmaps, bm3)

	count := CountConfirmations(bitmaps, ServiceDepots, 5)
	if count != 2 {
		t.Errorf("CountConfirmations() = %d, want 2", count)
	}

	// 未被确认的 offset
	count = CountConfirmations(bitmaps, ServiceDepots, 10)
	if count != 0 {
		t.Errorf("CountConfirmations(offset=10) = %d, want 0", count)
	}
}

func TestCountConfirmations_Empty(t *testing.T) {
	count := CountConfirmations(nil, ServiceDepots, 0)
	if count != 0 {
		t.Errorf("CountConfirmations(nil) = %d, want 0", count)
	}
}

// --- RewardStatus 测试 ---

func TestRewardStatus(t *testing.T) {
	tests := []struct {
		name          string
		confirmations int
		wantPortion   float64
	}{
		{"zero confirmations", 0, 0.0},
		{"one confirmation", 1, 0.5},
		{"two confirmations", 2, 1.0},
		{"three confirmations", 3, 1.0},
		{"ten confirmations", 10, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewardStatus(tt.confirmations)
			if math.Abs(got-tt.wantPortion) > 0.001 {
				t.Errorf("RewardStatus(%d) = %f, want %f", tt.confirmations, got, tt.wantPortion)
			}
		})
	}
}

// --- IsReclaimable 测试 ---

func TestIsReclaimable(t *testing.T) {
	tests := []struct {
		name          string
		blocksSince   int
		confirmations int
		want          bool
	}{
		{"within window, no confirmations", 10, 0, false},
		{"within window, 1 confirmation", 10, 1, false},
		{"within window, 2 confirmations", 10, 2, false},
		{"past window, no confirmations", 49, 0, true},
		{"past window, 1 confirmation", 49, 1, true},
		{"past window, 2 confirmations", 49, 2, false},
		{"exactly at boundary, no confirmations", 48, 0, false},
		{"one past boundary, no confirmations", 49, 0, true},
		{"far past window, 1 confirmation", 100, 1, true},
		{"far past window, 2 confirmations", 100, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsReclaimable(tt.blocksSince, tt.confirmations)
			if got != tt.want {
				t.Errorf("IsReclaimable(%d, %d) = %v, want %v",
					tt.blocksSince, tt.confirmations, got, tt.want)
			}
		})
	}
}

// --- CacheEntry 测试 ---

func TestCacheEntry_Fields(t *testing.T) {
	now := time.Now()
	entry := CacheEntry{
		Address:       types.PubKeyHash{0x01},
		Score:         0.75,
		LastEvaluated: now,
		EvalCount:     3,
	}

	if entry.Address[0] != 0x01 {
		t.Error("CacheEntry.Address mismatch")
	}
	if entry.Score != 0.75 {
		t.Errorf("CacheEntry.Score = %f, want 0.75", entry.Score)
	}
	if !entry.LastEvaluated.Equal(now) {
		t.Error("CacheEntry.LastEvaluated mismatch")
	}
	if entry.EvalCount != 3 {
		t.Errorf("CacheEntry.EvalCount = %d, want 3", entry.EvalCount)
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/services/ -run "TestConfirmationConstants|TestRewardAddressCache|TestCountConfirmations|TestRewardStatus|TestIsReclaimable|TestCacheEntry"
```

预期输出：编译失败，`RewardAddressCache`、`ConfirmationBitmap`、`CountConfirmations`、`RewardStatus`、`IsReclaimable` 等未定义。

### Step 3: 写最小实现

创建 `internal/services/reward.go`：

```go
package services

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// 奖励确认相关常量
const (
	// ConfirmationWindowSize 确认窗口大小（区块数）。
	// 奖励发放后 48 个区块内需要获得确认。
	ConfirmationWindowSize = 48

	// RequiredConfirmations 完全兑换所需的确认次数。
	RequiredConfirmations = 2

	// BitmapSize 确认位图字节数。
	// 3 类服务 × 48 个区块 = 144 位 = 18 字节。
	BitmapSize = 18

	// PruneScoreThreshold 清理时的评分阈值。
	// 评分低于此值的条目将被移除。
	PruneScoreThreshold = 0.05
)

// CacheEntry 奖励地址缓存条目。
type CacheEntry struct {
	Address       types.PubKeyHash // 奖励接收地址
	Score         float64          // 服务质量评分（0.0 ~ 1.0）
	LastEvaluated time.Time        // 最后评估时间
	EvalCount     int              // 累计评估次数
}

// RewardAddressCache 奖励地址缓存。
// 维护各类服务的高质量节点地址池，供区块铸造时选择奖励接收者。
// 缓存规模约 ~500 个地址（对应约 10000 节点规模的网络）。
type RewardAddressCache struct {
	mu      sync.RWMutex
	entries map[ServiceType][]CacheEntry
	maxSize int // 每种服务类型的最大缓存条目数
}

// NewRewardAddressCache 创建一个新的奖励地址缓存。
// maxSize 为每种服务类型允许的最大条目数。
func NewRewardAddressCache(maxSize int) *RewardAddressCache {
	return &RewardAddressCache{
		entries: make(map[ServiceType][]CacheEntry),
		maxSize: maxSize,
	}
}

// Add 向缓存中添加或更新一个地址。
// 如果地址已存在则更新评分和时间。
// 如果超出最大容量则淘汰评分最低的条目。
func (c *RewardAddressCache) Add(serviceType ServiceType, address types.PubKeyHash, score float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries := c.entries[serviceType]

	// 检查是否已存在
	for i := range entries {
		if entries[i].Address == address {
			entries[i].Score = score
			entries[i].LastEvaluated = time.Now()
			entries[i].EvalCount++
			c.entries[serviceType] = entries
			return
		}
	}

	// 新增条目
	entry := CacheEntry{
		Address:       address,
		Score:         score,
		LastEvaluated: time.Now(),
		EvalCount:     1,
	}
	entries = append(entries, entry)

	// 如果超出容量，淘汰评分最低的条目
	if len(entries) > c.maxSize {
		minIdx := 0
		for i := 1; i < len(entries); i++ {
			if entries[i].Score < entries[minIdx].Score {
				minIdx = i
			}
		}
		// 用末尾元素覆盖最低分条目，然后截断
		entries[minIdx] = entries[len(entries)-1]
		entries = entries[:len(entries)-1]
	}

	c.entries[serviceType] = entries
}

// Remove 从缓存中移除指定地址。
func (c *RewardAddressCache) Remove(serviceType ServiceType, address types.PubKeyHash) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries := c.entries[serviceType]
	for i := range entries {
		if entries[i].Address == address {
			// 用末尾元素覆盖，然后截断
			entries[i] = entries[len(entries)-1]
			c.entries[serviceType] = entries[:len(entries)-1]
			return
		}
	}
}

// SelectForReward 选择评分最高的地址作为奖励接收者。
func (c *RewardAddressCache) SelectForReward(serviceType ServiceType) (types.PubKeyHash, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := c.entries[serviceType]
	if len(entries) == 0 {
		return types.PubKeyHash{}, fmt.Errorf("select for reward: %s: %w", serviceType, ErrCacheEmpty)
	}

	bestIdx := 0
	for i := 1; i < len(entries); i++ {
		if entries[i].Score > entries[bestIdx].Score {
			bestIdx = i
		}
	}

	return entries[bestIdx].Address, nil
}

// SelectRandom 以评分为权重进行加权随机选择。
// 评分越高的地址被选中的概率越大。
func (c *RewardAddressCache) SelectRandom(serviceType ServiceType) (types.PubKeyHash, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := c.entries[serviceType]
	if len(entries) == 0 {
		return types.PubKeyHash{}, fmt.Errorf("select random: %s: %w", serviceType, ErrCacheEmpty)
	}

	if len(entries) == 1 {
		return entries[0].Address, nil
	}

	// 计算总权重
	totalWeight := 0.0
	for _, e := range entries {
		totalWeight += e.Score
	}

	if totalWeight <= 0 {
		// 所有评分为零——等概率随机
		idx := rand.Intn(len(entries))
		return entries[idx].Address, nil
	}

	// 加权随机选择
	r := rand.Float64() * totalWeight
	cumulative := 0.0
	for _, e := range entries {
		cumulative += e.Score
		if r <= cumulative {
			return e.Address, nil
		}
	}

	// 兜底（浮点精度问题）
	return entries[len(entries)-1].Address, nil
}

// UpdateScore 更新指定地址的评分。
// 如果地址不存在则忽略。
func (c *RewardAddressCache) UpdateScore(serviceType ServiceType, address types.PubKeyHash, score float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries := c.entries[serviceType]
	for i := range entries {
		if entries[i].Address == address {
			entries[i].Score = score
			entries[i].LastEvaluated = time.Now()
			entries[i].EvalCount++
			return
		}
	}
}

// Prune 清理过期或低评分的缓存条目。
// 评分低于 PruneScoreThreshold 的条目将被移除。
func (c *RewardAddressCache) Prune() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for st, entries := range c.entries {
		kept := make([]CacheEntry, 0, len(entries))
		for _, e := range entries {
			if e.Score >= PruneScoreThreshold {
				kept = append(kept, e)
			}
		}
		c.entries[st] = kept
	}
}

// Count 返回指定服务类型的缓存条目数量。
func (c *RewardAddressCache) Count(serviceType ServiceType) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries[serviceType])
}

// ErrCacheEmpty 缓存为空。
var ErrCacheEmpty = errors.New("reward address cache is empty")

// --- 确认位图 ---

// ConfirmationBitmap 奖励确认位图。
// 144 位（18 字节）= 3 类服务 × 48 个区块。
//
// 位布局：
//
//	字节 [0..5]   → ServiceDepots  (48 位)
//	字节 [6..11]  → ServiceBlockqs (48 位)
//	字节 [12..17] → ServiceStun    (48 位)
//
// 每个服务类型内，bit 0 对应 blockOffset 0，bit 47 对应 blockOffset 47。
type ConfirmationBitmap [BitmapSize]byte

// serviceTypeOffset 计算指定服务类型在位图中的字节起始偏移。
// Depots=0, Blockqs=6, Stun=12
func serviceTypeOffset(st ServiceType) (int, error) {
	if !st.IsValid() {
		return 0, fmt.Errorf("bitmap: %w", ErrInvalidServiceType)
	}
	// ServiceDepots=1 → offset=0, ServiceBlockqs=2 → offset=6, ServiceStun=3 → offset=12
	return int(st-1) * 6, nil
}

// SetBit 设置确认位。
// serviceType 为服务类型，blockOffset 为区块偏移（0~47）。
func (bm *ConfirmationBitmap) SetBit(serviceType ServiceType, blockOffset int) error {
	if blockOffset < 0 || blockOffset >= ConfirmationWindowSize {
		return fmt.Errorf("bitmap set: block offset %d out of range [0, %d)", blockOffset, ConfirmationWindowSize)
	}
	baseOffset, err := serviceTypeOffset(serviceType)
	if err != nil {
		return err
	}

	byteIdx := baseOffset + blockOffset/8
	bitIdx := uint(blockOffset % 8)
	bm[byteIdx] |= 1 << bitIdx
	return nil
}

// GetBit 读取确认位。
func (bm *ConfirmationBitmap) GetBit(serviceType ServiceType, blockOffset int) (bool, error) {
	if blockOffset < 0 || blockOffset >= ConfirmationWindowSize {
		return false, fmt.Errorf("bitmap get: block offset %d out of range [0, %d)", blockOffset, ConfirmationWindowSize)
	}
	baseOffset, err := serviceTypeOffset(serviceType)
	if err != nil {
		return false, err
	}

	byteIdx := baseOffset + blockOffset/8
	bitIdx := uint(blockOffset % 8)
	return bm[byteIdx]&(1<<bitIdx) != 0, nil
}

// CountConfirmations 统计某个奖励在多个铸造者位图中获得的确认次数。
// 遍历所有位图，计算指定服务类型和区块偏移处被置位的总数。
func CountConfirmations(bitmaps []ConfirmationBitmap, serviceType ServiceType, blockOffset int) int {
	count := 0
	for _, bm := range bitmaps {
		got, err := bm.GetBit(serviceType, blockOffset)
		if err == nil && got {
			count++
		}
	}
	return count
}

// RewardStatus 根据确认次数计算可兑现比例。
//   - 0 次确认 → 0%（不可兑现）
//   - 1 次确认 → 50%
//   - 2+ 次确认 → 100%（完全可兑现）
func RewardStatus(confirmations int) float64 {
	switch {
	case confirmations <= 0:
		return 0.0
	case confirmations == 1:
		return 0.5
	default:
		return 1.0
	}
}

// IsReclaimable 判断某笔奖励是否可被回收。
// 条件：超过 48 个区块且确认不足 2 次。
// blocksSince 为自奖励发放以来经过的区块数。
func IsReclaimable(blocksSince int, confirmations int) bool {
	if blocksSince <= ConfirmationWindowSize {
		return false
	}
	return confirmations < RequiredConfirmations
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/services/ -run "TestConfirmationConstants|TestRewardAddressCache|TestCountConfirmations|TestRewardStatus|TestIsReclaimable|TestCacheEntry"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/services/reward.go internal/services/reward_test.go
git commit -m "feat(services): add RewardAddressCache, ConfirmationBitmap and reward status functions"
```

---

## 完整测试验证

所有 Task 完成后，运行完整的包测试：

```bash
go test -v ./internal/services/...
```

预期输出：全部 PASS。

检查测试覆盖率：

```bash
go test -cover ./internal/services/...
```

代码格式化：

```bash
go fmt ./internal/services/...
```

最终确认编译通过：

```bash
go build ./...
```

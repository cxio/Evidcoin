# Phase 8: Third-Party Service Interfaces（第三方服务接口）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `internal/services` 包，定义 Evidcoin 与三类第三方公共服务网络（Depots 数据驿站、Blockqs 区块查询、STUN NAT穿透）的交互接口，以及铸造时间表（Minting Schedule）相关常量与计算函数。

**Architecture:** Layer 5 集成层（服务接口层）。依赖 `pkg/types`（Layer 0），`internal/tx`（Layer 1），`internal/utxo`/`internal/utco`（Layer 2），`internal/consensus`（Layer 4）。本包**不实现**网络传输，仅定义 Go 接口与数据结构；实际服务节点由独立项目（`cxio/depots`、`cxio/blockqs`、`cxio/stun2p`）实现。

**Tech Stack:** Go 1.25+，`context.Context` 超时管理。

---

## 设计边界说明

- **基网（Base Network）** 由外部库 `cxio/p2p` 实现，本包不涉及。
- **服务节点发现** 基于基网广播机制，本包仅定义连接后的查询/响应接口。
- **连接安全**（SPKI 指纹验证）由传输层处理，接口层不感知。
- **奖励确认槽（144 位位图）** 作为 Coinbase 交易数据的一部分在 `internal/tx` 中定义，本包提供辅助计算函数。
- **铸造时间表** 是纯函数（无副作用），适合在本包中集中定义。

---

## 目录结构（预期）

```
internal/services/
  interfaces.go     # 三类服务接口定义（Depots, Blockqs, STUN）
  query.go          # Blockqs 查询请求/响应结构
  attachment.go     # Depots 附件操作结构（AttachmentQuery 等）
  reward.go         # 奖励确认槽辅助函数（RewardSlot 位图操作）
  schedule.go       # 铸造时间表（MintingSchedule / BlockReward）
  services_test.go  # 单元测试
```

---

## Task 1: 服务接口定义（internal/services/interfaces.go）

**Files:**
- Create: `internal/services/interfaces.go`

**Step 1: 编写接口文件**

```go
// internal/services/interfaces.go
// Package services 定义 Evidcoin 与第三方公共服务网络的交互接口。
// 服务节点实现由独立外部项目提供；本包仅声明契约与数据结构。
package services

import (
    "context"

    "github.com/cxio/evidcoin/pkg/types"
)

// DepotService 数据驿站服务接口（§2.1）。
// 负责区块链数据持久化存储与 P2P 文件共享，包括完整区块与交易附件。
type DepotService interface {
    // QueryData 按数据 ID 查询数据是否存在，并获取下载信息。
    QueryData(ctx context.Context, req *AttachmentQuery) (*AttachmentQueryResult, error)

    // FetchShard 获取附件的单个分片数据（用于大文件局部验证）。
    FetchShard(ctx context.Context, req *ShardRequest) (*ShardResponse, error)

    // StoreData 向 Depot 节点推送数据（应用节点作为初始数据源）。
    StoreData(ctx context.Context, data []byte, attachID []byte) error

    // NodeAddress 返回此 Depot 节点的区块链地址（用于奖励分配）。
    NodeAddress() [32]byte
}

// BlockqsService 区块查询服务接口（§2.2）。
// 面向区块链交易数据的高性能查询服务。
type BlockqsService interface {
    // QueryTransaction 按年度+TxID 获取完整交易数据。
    QueryTransaction(ctx context.Context, req *TxQueryRequest) (*TxQueryResponse, error)

    // QueryUTXO 获取特定年度或当前 UTXO 集数据。
    QueryUTXO(ctx context.Context, year int) (*UTXOQueryResponse, error)

    // QueryUTCO 获取特定年度或当前 UTCO 集数据。
    QueryUTCO(ctx context.Context, year int) (*UTCOQueryResponse, error)

    // QueryTxIDsByAddress 获取与指定地址相关的交易 ID 集合。
    QueryTxIDsByAddress(ctx context.Context, address [32]byte) ([]types.Hash384, error)

    // QueryBlockSummary 获取区块概要（TxID 列表）及哈希树验证路径。
    QueryBlockSummary(ctx context.Context, height int32) (*BlockSummaryResponse, error)

    // NodeAddress 返回此 Blockqs 节点的区块链地址。
    NodeAddress() [32]byte
}

// STUNService NAT 穿透服务接口（§2.3）。
// 提供 NAT 类型探测与打洞协助，使 NAT 后节点可建立 P2P 连接。
type STUNService interface {
    // ProbNAT 探测本节点的 NAT 类型与外部地址。
    ProbNAT(ctx context.Context) (*NATProbeResult, error)

    // Punch 协助两个节点之间的 UDP 打洞。
    Punch(ctx context.Context, req *PunchRequest) (*PunchResult, error)

    // NodeAddress 返回此 STUN 节点的区块链地址。
    NodeAddress() [32]byte
}
```

**Step 2: 构建验证**

```bash
go build ./internal/services/...
```

---

## Task 2: Blockqs 查询结构（internal/services/query.go）

**Files:**
- Create: `internal/services/query.go`

**Step 1: 编写查询结构**

```go
// internal/services/query.go
package services

import "github.com/cxio/evidcoin/pkg/types"

// TxQueryRequest 交易查询请求。
type TxQueryRequest struct {
    TxYear int            // 交易所在年度（由交易时间戳计算）
    TxID   types.Hash384  // 完整交易 ID（48 字节）
}

// TxQueryResponse 交易查询响应。
type TxQueryResponse struct {
    RawTx        []byte         // 序列化交易数据
    BlockHeight  int32          // 交易所在区块高度
    TxIndex      int            // 区块内交易序位
    MerklePath   []types.Hash384 // 哈希树验证路径（至 TxTreeRoot）
}

// UTXOQueryResponse UTXO 查询响应。
type UTXOQueryResponse struct {
    Year        int            // 年度（0 = 当前所有年度）
    Fingerprint types.Hash256  // UTXO 指纹
    RawData     []byte         // 序列化 UTXO 集数据（可选，按需提供）
}

// UTCOQueryResponse UTCO 查询响应（结构与 UTXO 对称）。
type UTCOQueryResponse struct {
    Year        int
    Fingerprint types.Hash256
    RawData     []byte
}

// BlockSummaryResponse 区块概要查询响应。
type BlockSummaryResponse struct {
    BlockHeight    int32          // 区块高度
    TxIDPrefixes   [][16]byte     // 每笔交易 TxID 前 16 字节
    TxTreeRoot     types.Hash384  // 交易哈希树根
    UTXOFingerprint types.Hash256 // UTXO 指纹
    UTCOFingerprint types.Hash256 // UTCO 指纹
    CheckRoot      types.Hash384  // CheckRoot
}
```

**Step 2: 构建验证**

```bash
go build ./internal/services/...
```

---

## Task 3: Depots 附件结构（internal/services/attachment.go）

**Files:**
- Create: `internal/services/attachment.go`

**Step 1: 编写附件结构**

```go
// internal/services/attachment.go
package services

// AttachmentQuery 附件查询请求。
type AttachmentQuery struct {
    AttachID []byte // 附件 ID（来自交易输出的 AttachID 字段）
}

// AttachmentQueryResult 附件查询结果。
type AttachmentQueryResult struct {
    Found      bool    // 是否找到数据
    DataSize   int64   // 附件总大小（字节）
    ShardCount uint16  // 分片数量
    HopCount   int     // 跳数（粗略衡量稀缺程度，跳数越高越稀缺）
    SourceAddr string  // 持有数据的节点地址（可直连获取）
}

// ShardRequest 单个分片请求。
type ShardRequest struct {
    AttachID    []byte // 附件 ID
    ShardIndex  uint16 // 分片序号（从 0 开始）
}

// ShardResponse 单个分片响应。
type ShardResponse struct {
    ShardIndex uint16 // 分片序号
    Data       []byte // 分片数据
    ShardHash  []byte // 分片哈希（48 字节 SHA3-384），由验证方前置序号后验证
}

// NATProbeResult NAT 探测结果（STUN 服务）。
type NATProbeResult struct {
    NATType     string // NAT 类型描述（如 "full-cone", "symmetric"）
    ExternalIP  string // 外部 IP
    ExternalPort int   // 外部端口
}

// PunchRequest UDP 打洞请求。
type PunchRequest struct {
    TargetAddr string // 目标节点外部地址（IP:Port）
}

// PunchResult 打洞结果。
type PunchResult struct {
    Success     bool
    LocalAddr   string // 本地用于打洞的地址
    RemoteAddr  string // 成功建立连接的对端地址
}
```

**Step 2: 构建验证**

```bash
go build ./internal/services/...
```

**Step 3: 提交（Tasks 1-3）**

```bash
git add internal/services/
git commit -m "feat(services): add third-party service interfaces and query structures"
```

---

## Task 4: 奖励确认槽辅助函数（internal/services/reward.go）

**Files:**
- Create: `internal/services/reward.go`
- Modify: `internal/services/services_test.go`

**背景：**

奖励确认槽是 Coinbase 交易中的 144 位位图，覆盖 3 类服务 × 48 个区块：
- 位 `[0, 47]`：Depots 奖励确认
- 位 `[48, 95]`：Blockqs 奖励确认
- 位 `[96, 143]`：STUN 奖励确认

总共 18 字节（144 位）。

**Step 1: 编写失败测试**

```go
// internal/services/services_test.go
package services_test

import (
    "testing"
    "github.com/cxio/evidcoin/internal/services"
)

func TestRewardSlot_SetAndGet(t *testing.T) {
    slot := services.NewRewardSlot()
    // 确认第 0 个区块的 Depots 奖励
    slot.Confirm(services.ServiceDepots, 0)
    if !slot.IsConfirmed(services.ServiceDepots, 0) {
        t.Error("Depots slot 0 should be confirmed")
    }
    if slot.IsConfirmed(services.ServiceDepots, 1) {
        t.Error("Depots slot 1 should not be confirmed")
    }
}

func TestRewardSlot_CountConfirmations(t *testing.T) {
    slot := services.NewRewardSlot()
    slot.Confirm(services.ServiceBlockqs, 5)
    slot.Confirm(services.ServiceBlockqs, 10)
    count := slot.CountConfirmations(services.ServiceBlockqs)
    if count != 2 {
        t.Errorf("count = %d, want 2", count)
    }
}

func TestRewardSlot_AllServices(t *testing.T) {
    slot := services.NewRewardSlot()
    services_ := []services.ServiceType{
        services.ServiceDepots,
        services.ServiceBlockqs,
        services.ServiceSTUN,
    }
    for _, svc := range services_ {
        slot.Confirm(svc, 47) // 最后一个槽
        if !slot.IsConfirmed(svc, 47) {
            t.Errorf("service %d slot 47 should be confirmed", svc)
        }
    }
}

func TestRewardSlot_Bytes_RoundTrip(t *testing.T) {
    slot := services.NewRewardSlot()
    slot.Confirm(services.ServiceDepots, 3)
    slot.Confirm(services.ServiceSTUN, 40)
    raw := slot.Bytes()
    if len(raw) != 18 {
        t.Errorf("raw len = %d, want 18", len(raw))
    }
    slot2, err := services.ParseRewardSlot(raw)
    if err != nil {
        t.Fatalf("parse: %v", err)
    }
    if !slot2.IsConfirmed(services.ServiceDepots, 3) {
        t.Error("round-trip: Depots slot 3 should be confirmed")
    }
    if !slot2.IsConfirmed(services.ServiceSTUN, 40) {
        t.Error("round-trip: STUN slot 40 should be confirmed")
    }
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/services/... -v -run TestRewardSlot
```

**Step 3: 实现 reward.go**

```go
// internal/services/reward.go
package services

import (
    "errors"
)

// ServiceType 服务类型枚举（对应 Coinbase 奖励槽分区）。
type ServiceType int

const (
    ServiceDepots  ServiceType = 0  // 数据驿站（位 0–47）
    ServiceBlockqs ServiceType = 1  // 区块查询（位 48–95）
    ServiceSTUN    ServiceType = 2  // NAT 穿透（位 96–143）
)

const (
    rewardWindowBlocks = 48  // 奖励确认窗口（区块数）
    rewardSlotBytes    = 18  // 144 位 / 8 = 18 字节
    serviceCount       = 3   // 服务类型数量
)

// RewardSlot 奖励确认槽（144 位位图，18 字节）。
// 每个服务类型占 48 位，对应 48 个后续区块的确认标记。
type RewardSlot struct {
    data [rewardSlotBytes]byte
}

// NewRewardSlot 创建空奖励确认槽。
func NewRewardSlot() *RewardSlot {
    return &RewardSlot{}
}

// ParseRewardSlot 从 18 字节数据解析奖励确认槽。
func ParseRewardSlot(raw []byte) (*RewardSlot, error) {
    if len(raw) != rewardSlotBytes {
        return nil, errors.New("reward slot: invalid length, expected 18 bytes")
    }
    slot := &RewardSlot{}
    copy(slot.data[:], raw)
    return slot, nil
}

// bitIndex 计算指定服务类型与区块偏移对应的位索引。
func bitIndex(svc ServiceType, blockOffset int) int {
    return int(svc)*rewardWindowBlocks + blockOffset
}

// Confirm 置位指定服务、指定区块偏移的确认标记。
func (s *RewardSlot) Confirm(svc ServiceType, blockOffset int) {
    idx := bitIndex(svc, blockOffset)
    s.data[idx/8] |= 1 << (uint(idx) % 8)
}

// IsConfirmed 查询指定位是否已置位。
func (s *RewardSlot) IsConfirmed(svc ServiceType, blockOffset int) bool {
    idx := bitIndex(svc, blockOffset)
    return s.data[idx/8]&(1<<(uint(idx)%8)) != 0
}

// CountConfirmations 统计指定服务的确认次数。
func (s *RewardSlot) CountConfirmations(svc ServiceType) int {
    count := 0
    for i := 0; i < rewardWindowBlocks; i++ {
        if s.IsConfirmed(svc, i) {
            count++
        }
    }
    return count
}

// Bytes 返回 18 字节的原始位图数据。
func (s *RewardSlot) Bytes() []byte {
    result := make([]byte, rewardSlotBytes)
    copy(result, s.data[:])
    return result
}
```

**Step 4: 运行测试确认通过**

```bash
go test ./internal/services/... -v -run TestRewardSlot
```

**Step 5: 提交**

```bash
git add internal/services/reward.go internal/services/services_test.go
git commit -m "feat(services): implement RewardSlot 144-bit confirmation bitmap"
```

---

## Task 5: 铸造时间表（internal/services/schedule.go）

**Files:**
- Create: `internal/services/schedule.go`
- Modify: `internal/services/services_test.go`

**Step 1: 编写失败测试**

```go
// 追加到 services_test.go

func TestBlockReward_PreRelease(t *testing.T) {
    tests := []struct {
        blockHeight int
        want        int64
    }{
        {0, 10},     // 第 1 年（Year 1 = blocks 0–87660）
        {87660, 10},
        {87661, 20}, // 第 2 年
        {175321, 30}, // 第 3 年
    }
    for _, tt := range tests {
        got := services.BlockReward(tt.blockHeight)
        if got != tt.want {
            t.Errorf("BlockReward(%d) = %d, want %d", tt.blockHeight, got, tt.want)
        }
    }
}

func TestBlockReward_FormalPhase(t *testing.T) {
    // 第 4 年起步 40 coins/block
    block4thYear := 3 * 87661
    got := services.BlockReward(block4thYear)
    if got != 40 {
        t.Errorf("BlockReward(year4_start) = %d, want 40", got)
    }
}

func TestBlockReward_LongTermMinimum(t *testing.T) {
    // 超过 25 年后应维持最低 3 coins/block
    blockFarFuture := 26 * 87661
    got := services.BlockReward(blockFarFuture)
    if got != 3 {
        t.Errorf("BlockReward(far_future) = %d, want 3", got)
    }
}

func TestBlockYear(t *testing.T) {
    tests := []struct {
        blockHeight int
        want        int
    }{
        {0, 1},
        {87660, 1},
        {87661, 2},
        {175321, 3},
    }
    for _, tt := range tests {
        got := services.BlockYear(tt.blockHeight)
        if got != tt.want {
            t.Errorf("BlockYear(%d) = %d, want %d", tt.blockHeight, got, tt.want)
        }
    }
}
```

**Step 2: 实现 schedule.go**

```go
// internal/services/schedule.go
package services

const blocksPerYear = 87661 // ≈ 365.25636 天 × 24h × 60min / 6min

// BlockYear 根据区块高度计算所在年份（从 1 开始）。
func BlockYear(blockHeight int) int {
    return blockHeight/blocksPerYear + 1
}

// BlockReward 根据区块高度计算该区块的铸造奖励（整币数量）。
//
// 铸造时间表（§4）：
//   - 预发布阶段（Year 1–3）：10 / 20 / 30 coins/block
//   - 正式发行阶段（Year 4 起）：从 40 开始，每 2 年下降 20%，整币计算
//   - 长期阶段：最低 3 coins/block（永久维持）
func BlockReward(blockHeight int) int64 {
    year := BlockYear(blockHeight)

    // 预发布阶段
    switch {
    case year <= 1:
        return 10
    case year == 2:
        return 20
    case year == 3:
        return 30
    }

    // 正式发行阶段：从第 4 年开始，每 2 年 × 0.8（整币）
    rate := int64(40)
    formalYear := year - 4 // 从 0 开始计的正式年份
    periods := formalYear / 2
    for i := 0; i < periods; i++ {
        rate = rate * 8 / 10 // × 80%，整币截断
        if rate <= 3 {
            return 3
        }
    }
    if rate < 3 {
        return 3
    }
    return rate
}

// AnnualMint 计算指定年份的年度铸币总量。
func AnnualMint(year int) int64 {
    // 以该年度起始区块高度为参考
    startBlock := (year - 1) * blocksPerYear
    reward := BlockReward(startBlock)
    return reward * blocksPerYear
}
```

**Step 3: 运行测试确认通过**

```bash
go test ./internal/services/... -v -run TestBlockReward
go test ./internal/services/... -v -run TestBlockYear
```

**Step 4: 提交**

```bash
git add internal/services/schedule.go internal/services/services_test.go
git commit -m "feat(services): implement minting schedule BlockReward and BlockYear"
```

---

## Task 6: 完整构建与测试

**Step 1: 完整构建**

```bash
go build ./...
```

**Step 2: 运行所有测试**

```bash
go test ./... -cover
```

预期：`internal/services` 核心逻辑覆盖率 ≥80%，全量测试通过。

**Step 3: 格式检查**

```bash
go fmt ./... && gofmt -s -w .
```

**Step 4: 最终提交**

```bash
git add .
git commit -m "feat(services): complete Phase 8 third-party service interfaces"
```

---

## 验收标准

| 标准 | 命令 |
|------|------|
| 编译通过 | `go build ./...` |
| 测试全部通过 | `go test ./...` |
| 核心逻辑覆盖率 ≥80% | `go test -cover ./internal/services/...` |
| 格式无变更 | `go fmt ./...` |
| 无 lint 警告 | `golangci-lint run` |

---

## 已知设计边界

1. **服务节点连接管理**（建立连接、断开重连、SPKI 指纹校验）由传输层（`cxio/p2p`）处理，本包不涉及。
2. **数据稀缺性探测**（Depots Scarcity sensing，§2.1）的具体广播逻辑在 Depots 服务项目中实现，本包提供查询接口。
3. **奖励确认评估缓存池**（§7.5）的大小与更新逻辑由管理层（`internal/checkteam`）运行时维护，本包仅提供 `RewardSlot` 位图工具。
4. **铸造时间表中的余数处理**（§6.2）：分配比例按除法整数截断，余数归最后一位接收者，实际分配逻辑在 `internal/tx` 的 Coinbase 构造中处理。

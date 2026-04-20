# Phase 7: Team-Based Verification（组队校验）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `internal/checkteam` 包，定义组队校验的角色接口（管理层、守卫者、校验员、UTXO/UTCO 缓存器）、首领校验逻辑、黑名单机制、冗余校验决策、铸造集成工作流接口，以及区块发布的三阶段结构定义。

**Architecture:** Layer 5 集成层（位于共识层与服务接口之间）。依赖 `pkg/types`、`pkg/crypto`（Layer 0），`internal/tx`（Layer 1），`internal/utxo`/`internal/utco`（Layer 2），`internal/consensus`（Layer 4）。各角色（管理层、守卫者、校验员）在生产中运行为独立进程，通过连接通信；本包仅定义接口、数据结构与核心算法，不含网络层实现（网络连接由外部 P2P 库提供）。

**Tech Stack:** Go 1.25+，`context.Context` 生命周期管理，channel 优先于 mutex。

---

## 设计边界说明

组队校验的各角色（管理层、守卫者、校验员）在真实部署中是**独立运行的应用程序**，通过 P2P 连接通信。本包的职责：

1. **接口定义**：声明各角色必须实现的 Go 接口。
2. **核心算法**：首领校验逻辑、黑名单、冗余决策、交易收录优先级。
3. **数据结构**：铸造流程消息、区块发布三阶段数据结构。
4. **不实现**：网络传输、实际脚本执行、存储层（这些由其他包提供）。

---

## 目录结构（预期）

```
internal/checkteam/
  roles.go          # 各角色接口定义（Management, Guard, Verifier, Cache）
  leader.go         # 首领校验逻辑（LeaderVerification）
  blacklist.go      # 黑名单机制（Blacklist，冻结 24 小时）
  redundancy.go     # 冗余校验决策（VerifyResult, RedundancyDecision）
  priority.go       # 交易收录优先级计算（TxPriority）
  minting.go        # 铸造集成消息结构（MintRequest, MintResponse 等）
  publication.go    # 区块发布三阶段结构（BlockProof, BlockSummary）
  checkteam_test.go # 单元测试
```

---

## Task 1: 角色接口定义（internal/checkteam/roles.go）

**Files:**
- Create: `internal/checkteam/roles.go`

**Step 1: 编写角色接口**

```go
// internal/checkteam/roles.go
// Package checkteam 定义 Evidcoin 组队校验架构的接口、数据结构与核心算法。
// 各角色（Management、Guard、Verifier）在生产中为独立应用进程；本包仅声明契约。
package checkteam

import (
    "context"

    "github.com/cxio/evidcoin/internal/tx"
    "github.com/cxio/evidcoin/internal/utco"
    "github.com/cxio/evidcoin/internal/utxo"
    "github.com/cxio/evidcoin/pkg/types"
)

// VerifyStatus 表示校验员对一笔交易的校验结果。
type VerifyStatus int

const (
    VerifyValid     VerifyStatus = iota // 校验通过
    VerifyInvalid                       // 校验失败
    VerifyOverloaded                    // 校验员过载，暂时无法处理
)

// VerifyResult 校验员返回给调度员的结果。
type VerifyResult struct {
    TxID   types.Hash384
    Status VerifyStatus
    Reason string // 失败原因（仅 VerifyInvalid 时有意义）
}

// Management 管理层（广播者+调度员）对外暴露的接口。
// 实现方负责交易打包、区块铸造流程协调、跨团队区块交换。
type Management interface {
    // SubmitVerified 接收来自调度员已验证的交易 ID，存入已验证集合。
    SubmitVerified(ctx context.Context, txID types.Hash384) error

    // NotifyBlock 通知管理层接收到来自其他团队发布的区块。
    NotifyBlock(ctx context.Context, proof *BlockProof) error

    // RequestMintInfo 铸造者向管理层请求铸造必要信息。
    RequestMintInfo(ctx context.Context, req *MintInfoRequest) (*MintInfoResponse, error)

    // SubmitCoinbase 铸造者提交 Coinbase 交易。
    SubmitCoinbase(ctx context.Context, coinbase *tx.CoinbaseTx) (*CoinbaseAck, error)

    // SubmitBlockSignature 铸造者提交 CheckRoot 签名。
    SubmitBlockSignature(ctx context.Context, sig *BlockSignature) error
}

// Guard 守卫者接口。
// 守卫者是交易的唯一外部入口，负责首领校验与转播。
type Guard interface {
    // ReceiveTx 接收来自外部（其他团队 Guard/Verifier）的交易。
    // 50% 概率先执行首领校验再转发，50% 直接转发。
    ReceiveTx(ctx context.Context, rawTx []byte) error

    // IsBlacklisted 检查指定首领输入是否在黑名单中。
    IsBlacklisted(op utxo.Outpoint) bool

    // AddToBlacklist 将通过首领校验但最终失败的首领输入加入黑名单。
    AddToBlacklist(op utxo.Outpoint)
}

// Verifier 校验员接口。
// 校验员向调度员请求任务并执行完整校验。
type Verifier interface {
    // RequestTask 向调度员请求一个待校验的交易。
    RequestTask(ctx context.Context) (*tx.Transaction, error)

    // ReportResult 向调度员汇报校验结果。
    ReportResult(ctx context.Context, result VerifyResult) error
}

// UTXOCacheService UTXO 缓存服务接口（§2.4）。
type UTXOCacheService interface {
    // Lookup 查询指定输出点是否未花费，返回条目信息。
    Lookup(op utxo.Outpoint) (*utxo.UTXOEntry, error)

    // Update 根据已打包区块的交易更新缓存（标记花费、添加新输出）。
    Update(transaction *tx.Transaction) error
}

// UTCOCacheService UTCO 缓存服务接口（§2.4）。
type UTCOCacheService interface {
    Lookup(op utco.Outpoint) (*utco.UTCOEntry, error)
    Update(transaction *tx.Transaction) error
}
```

**Step 2: 构建验证**

```bash
go build ./internal/checkteam/...
```

---

## Task 2: 首领校验（internal/checkteam/leader.go）

**Files:**
- Create: `internal/checkteam/leader.go`
- Modify: `internal/checkteam/checkteam_test.go`

**背景：**
首领校验仅校验交易的第一个输入（首领输入）：
1. 首领输入必须是 Coin 类型。
2. 首领输入必须在所有 Coin 输入中具有最高的币权（CoinAge × Amount）。
3. 首领输入不在黑名单中。

**Step 1: 编写失败测试**

```go
// internal/checkteam/checkteam_test.go
package checkteam_test

import (
    "testing"
    "github.com/cxio/evidcoin/internal/checkteam"
    "github.com/cxio/evidcoin/internal/utxo"
    "github.com/cxio/evidcoin/pkg/types"
)

func TestLeaderVerification_Valid(t *testing.T) {
    // 首领输入：Coin 类型，币权最高
    leader := checkteam.LeaderInput{
        TxID:     types.Hash384{0x01},
        OutIndex: 0,
        IsCoin:   true,
        CoinAge:  1000, // 最高
    }
    others := []checkteam.LeaderInput{
        {IsCoin: true, CoinAge: 500},
        {IsCoin: true, CoinAge: 200},
    }
    bl := checkteam.NewBlacklist()
    if err := checkteam.VerifyLeader(leader, others, bl); err != nil {
        t.Errorf("expected valid leader, got: %v", err)
    }
}

func TestLeaderVerification_NotCoin(t *testing.T) {
    leader := checkteam.LeaderInput{IsCoin: false, CoinAge: 1000}
    bl := checkteam.NewBlacklist()
    if err := checkteam.VerifyLeader(leader, nil, bl); err == nil {
        t.Error("non-coin leader should fail")
    }
}

func TestLeaderVerification_NotHighestCoinAge(t *testing.T) {
    leader := checkteam.LeaderInput{IsCoin: true, CoinAge: 100}
    others := []checkteam.LeaderInput{
        {IsCoin: true, CoinAge: 500}, // 比 leader 更高
    }
    bl := checkteam.NewBlacklist()
    if err := checkteam.VerifyLeader(leader, others, bl); err == nil {
        t.Error("leader without highest coin-age should fail")
    }
}

func TestLeaderVerification_Blacklisted(t *testing.T) {
    op := utxo.Outpoint{TxID: types.Hash384{0x02}}
    leader := checkteam.LeaderInput{
        TxID:     op.TxID,
        OutIndex: op.OutIndex,
        IsCoin:   true,
        CoinAge:  9999,
    }
    bl := checkteam.NewBlacklist()
    bl.Add(op)
    if err := checkteam.VerifyLeader(leader, nil, bl); err == nil {
        t.Error("blacklisted leader should fail")
    }
}
```

**Step 2: 实现 leader.go**

```go
// internal/checkteam/leader.go
package checkteam

import (
    "errors"

    "github.com/cxio/evidcoin/internal/utxo"
    "github.com/cxio/evidcoin/pkg/types"
)

// LeaderInput 首领输入的摘要信息（用于首领校验）。
type LeaderInput struct {
    TxID     types.Hash384 // 被引用交易 ID
    OutIndex int           // 输出序位
    IsCoin   bool          // 是否为 Coin 类型输出
    CoinAge  int64         // 币权值（Amount × HoldHours）
}

// VerifyLeader 执行首领校验。
//   leader：交易的首领输入；
//   others：同一交易的其他 Coin 输入（不含首领）；
//   bl：黑名单（禁止通过首领校验但最终失败的首领输入）。
//
// 规则：
//  1. 首领输入必须为 Coin 类型。
//  2. 首领输入的币权必须在所有 Coin 输入中最高。
//  3. 首领输入不得在黑名单中。
func VerifyLeader(leader LeaderInput, others []LeaderInput, bl *Blacklist) error {
    if !leader.IsCoin {
        return errors.New("leader input must be a Coin type")
    }
    op := utxo.Outpoint{TxID: leader.TxID, OutIndex: leader.OutIndex}
    if bl.Contains(op) {
        return errors.New("leader input is blacklisted")
    }
    for _, other := range others {
        if other.IsCoin && other.CoinAge > leader.CoinAge {
            return errors.New("leader input does not have highest coin-age among Coin inputs")
        }
    }
    return nil
}
```

**Step 3: 运行测试**

```bash
go test ./internal/checkteam/... -v -run TestLeaderVerification
```

**Step 4: 提交**

```bash
git add internal/checkteam/roles.go internal/checkteam/leader.go internal/checkteam/checkteam_test.go
git commit -m "feat(checkteam): add role interfaces and leader verification"
```

---

## Task 3: 黑名单机制（internal/checkteam/blacklist.go）

**Files:**
- Create: `internal/checkteam/blacklist.go`
- Modify: `internal/checkteam/checkteam_test.go`

**Step 1: 编写失败测试**

```go
// 追加到 checkteam_test.go

import "time"

func TestBlacklist_AddAndContains(t *testing.T) {
    bl := checkteam.NewBlacklist()
    op := utxo.Outpoint{TxID: types.Hash384{0x10}}
    bl.Add(op)
    if !bl.Contains(op) {
        t.Error("blacklisted op should be contained")
    }
}

func TestBlacklist_NotContained(t *testing.T) {
    bl := checkteam.NewBlacklist()
    op := utxo.Outpoint{TxID: types.Hash384{0x20}}
    if bl.Contains(op) {
        t.Error("op not added should not be contained")
    }
}

func TestBlacklist_ExpireAfterDuration(t *testing.T) {
    bl := checkteam.NewBlacklistWithClock(func() time.Time {
        return time.Unix(0, 0)
    })
    op := utxo.Outpoint{TxID: types.Hash384{0x30}}
    bl.Add(op)
    // 24 小时后应过期
    bl.SetClock(func() time.Time {
        return time.Unix(0, 0).Add(25 * time.Hour)
    })
    bl.Expire()
    if bl.Contains(op) {
        t.Error("blacklist entry should expire after 24 hours")
    }
}
```

**Step 2: 实现 blacklist.go**

```go
// internal/checkteam/blacklist.go
package checkteam

import (
    "sync"
    "time"

    "github.com/cxio/evidcoin/internal/utxo"
)

const blacklistFreezeDuration = 24 * time.Hour

// blacklistEntry 黑名单条目。
type blacklistEntry struct {
    addedAt time.Time
}

// Blacklist 维护通过首领校验但最终未通过完整校验的首领输入黑名单。
// 冻结时长为 24 小时。
type Blacklist struct {
    mu      sync.RWMutex
    entries map[utxo.Outpoint]blacklistEntry
    clock   func() time.Time
}

// NewBlacklist 创建黑名单（使用系统时钟）。
func NewBlacklist() *Blacklist {
    return &Blacklist{
        entries: make(map[utxo.Outpoint]blacklistEntry),
        clock:   time.Now,
    }
}

// NewBlacklistWithClock 创建黑名单（使用自定义时钟，用于测试）。
func NewBlacklistWithClock(clock func() time.Time) *Blacklist {
    return &Blacklist{
        entries: make(map[utxo.Outpoint]blacklistEntry),
        clock:   clock,
    }
}

// SetClock 替换时钟函数（仅用于测试）。
func (bl *Blacklist) SetClock(clock func() time.Time) {
    bl.mu.Lock()
    bl.clock = clock
    bl.mu.Unlock()
}

// Add 将指定输出点加入黑名单。
func (bl *Blacklist) Add(op utxo.Outpoint) {
    bl.mu.Lock()
    bl.entries[op] = blacklistEntry{addedAt: bl.clock()}
    bl.mu.Unlock()
}

// Contains 检查指定输出点是否在有效黑名单中（未过期）。
func (bl *Blacklist) Contains(op utxo.Outpoint) bool {
    bl.mu.RLock()
    entry, ok := bl.entries[op]
    clock := bl.clock
    bl.mu.RUnlock()
    if !ok {
        return false
    }
    return clock().Sub(entry.addedAt) < blacklistFreezeDuration
}

// Expire 清理所有已过期的黑名单条目（建议周期性调用）。
func (bl *Blacklist) Expire() {
    bl.mu.Lock()
    defer bl.mu.Unlock()
    now := bl.clock()
    for op, entry := range bl.entries {
        if now.Sub(entry.addedAt) >= blacklistFreezeDuration {
            delete(bl.entries, op)
        }
    }
}
```

**Step 3: 运行测试**

```bash
go test ./internal/checkteam/... -v -run TestBlacklist
```

**Step 4: 提交**

```bash
git add internal/checkteam/blacklist.go internal/checkteam/checkteam_test.go
git commit -m "feat(checkteam): implement Blacklist with 24h freeze"
```

---

## Task 4: 冗余校验决策（internal/checkteam/redundancy.go）

**Files:**
- Create: `internal/checkteam/redundancy.go`
- Modify: `internal/checkteam/checkteam_test.go`

**Step 1: 编写失败测试**

```go
// 追加到 checkteam_test.go

func TestRedundancyDecision_AllValid(t *testing.T) {
    results := []checkteam.VerifyStatus{
        checkteam.VerifyValid, checkteam.VerifyValid,
    }
    dec := checkteam.MakeDecision(results)
    if dec != checkteam.DecisionAccept {
        t.Errorf("all valid => Accept, got %v", dec)
    }
}

func TestRedundancyDecision_AnyInvalid(t *testing.T) {
    results := []checkteam.VerifyStatus{
        checkteam.VerifyValid, checkteam.VerifyInvalid,
    }
    dec := checkteam.MakeDecision(results)
    if dec != checkteam.DecisionExtendedReview {
        t.Errorf("any invalid => ExtendedReview, got %v", dec)
    }
}

func TestLevel1Review_AllValid(t *testing.T) {
    results := []checkteam.VerifyStatus{
        checkteam.VerifyValid, checkteam.VerifyValid, checkteam.VerifyValid,
    }
    dec := checkteam.Level1Review(results)
    if dec != checkteam.DecisionAccept {
        t.Errorf("L1 zero errors => Accept, got %v", dec)
    }
}

func TestLevel1Review_MoreThanHalfInvalid(t *testing.T) {
    results := []checkteam.VerifyStatus{
        checkteam.VerifyInvalid, checkteam.VerifyInvalid, checkteam.VerifyValid,
    }
    dec := checkteam.Level1Review(results)
    if dec != checkteam.DecisionReject {
        t.Errorf("L1 majority invalid => Reject, got %v", dec)
    }
}

func TestLevel2Review_AnyInvalid(t *testing.T) {
    results := []checkteam.VerifyStatus{
        checkteam.VerifyValid, checkteam.VerifyInvalid,
    }
    dec := checkteam.Level2Review(results)
    if dec != checkteam.DecisionReject {
        t.Errorf("L2 any error => Reject, got %v", dec)
    }
}

func TestLevel2Review_AllValid(t *testing.T) {
    results := []checkteam.VerifyStatus{
        checkteam.VerifyValid, checkteam.VerifyValid,
    }
    dec := checkteam.Level2Review(results)
    if dec != checkteam.DecisionAccept {
        t.Errorf("L2 all valid => Accept, got %v", dec)
    }
}
```

**Step 2: 实现 redundancy.go**

```go
// internal/checkteam/redundancy.go
package checkteam

// Decision 调度员对交易的最终决策。
type Decision int

const (
    DecisionAccept         Decision = iota // 接受为有效
    DecisionReject                         // 拒绝为无效
    DecisionExtendedReview                 // 升级扩展复核
    DecisionLevel2Review                   // 升级二级复核
)

// MakeDecision 初次冗余校验决策（§4.1）。
// 若所有结果均为 Valid，则 Accept；否则进入扩展复核。
func MakeDecision(results []VerifyStatus) Decision {
    for _, r := range results {
        if r == VerifyInvalid {
            return DecisionExtendedReview
        }
    }
    return DecisionAccept
}

// Level1Review 一级扩展复核决策（§4.2）。
//   - 零个无效报告 → Accept。
//   - 超过半数无效 → Reject。
//   - 其他（少数无效）→ Level2Review。
func Level1Review(results []VerifyStatus) Decision {
    invalid := 0
    for _, r := range results {
        if r == VerifyInvalid {
            invalid++
        }
    }
    if invalid == 0 {
        return DecisionAccept
    }
    if invalid*2 > len(results) {
        return DecisionReject
    }
    return DecisionLevel2Review
}

// Level2Review 二级扩展复核决策（§4.2）。
// 任何无效报告 → Reject；全部有效 → Accept。
func Level2Review(results []VerifyStatus) Decision {
    for _, r := range results {
        if r == VerifyInvalid {
            return DecisionReject
        }
    }
    return DecisionAccept
}
```

**Step 3: 运行测试**

```bash
go test ./internal/checkteam/... -v -run TestRedundancy
go test ./internal/checkteam/... -v -run TestLevel
```

**Step 4: 提交**

```bash
git add internal/checkteam/redundancy.go internal/checkteam/checkteam_test.go
git commit -m "feat(checkteam): implement redundancy and extended review decision logic"
```

---

## Task 5: 交易收录优先级（internal/checkteam/priority.go）

**Files:**
- Create: `internal/checkteam/priority.go`
- Modify: `internal/checkteam/checkteam_test.go`

**Step 1: 编写失败测试**

```go
// 追加到 checkteam_test.go

func TestTxPriority_HigherStakesFirst(t *testing.T) {
    p1 := checkteam.TxPriority{StakesBurned: 1000, Fee: 100, HasCredBurn: false}
    p2 := checkteam.TxPriority{StakesBurned: 500, Fee: 200, HasCredBurn: true}
    if !p1.Higher(p2) {
        t.Error("higher stakes should win")
    }
}

func TestTxPriority_SameStakes_HigherFeeFirst(t *testing.T) {
    p1 := checkteam.TxPriority{StakesBurned: 100, Fee: 200, HasCredBurn: false}
    p2 := checkteam.TxPriority{StakesBurned: 100, Fee: 100, HasCredBurn: true}
    if !p1.Higher(p2) {
        t.Error("same stakes, higher fee should win")
    }
}

func TestTxPriority_SameStakesAndFee_CredBurnFirst(t *testing.T) {
    p1 := checkteam.TxPriority{StakesBurned: 100, Fee: 100, HasCredBurn: true}
    p2 := checkteam.TxPriority{StakesBurned: 100, Fee: 100, HasCredBurn: false}
    if !p1.Higher(p2) {
        t.Error("same stakes and fee, cred burn should win")
    }
}
```

**Step 2: 实现 priority.go**

```go
// internal/checkteam/priority.go
package checkteam

// TxPriority 交易收录优先级指标（§5）。
// 优先级规则（共约）：
//  1. 最高：币权销毁更多。
//  2. 次级：手续费更高。
//  3. 第三：有凭信提前销毁。
type TxPriority struct {
    StakesBurned int64 // 总币权销毁量（聪时）
    Fee          int64 // 交易手续费（聪）
    HasCredBurn  bool  // 是否包含凭信提前销毁
}

// Higher 判断 p 是否比 other 具有更高的收录优先级。
func (p TxPriority) Higher(other TxPriority) bool {
    if p.StakesBurned != other.StakesBurned {
        return p.StakesBurned > other.StakesBurned
    }
    if p.Fee != other.Fee {
        return p.Fee > other.Fee
    }
    return p.HasCredBurn && !other.HasCredBurn
}
```

**Step 3: 运行测试**

```bash
go test ./internal/checkteam/... -v -run TestTxPriority
```

**Step 4: 提交**

```bash
git add internal/checkteam/priority.go internal/checkteam/checkteam_test.go
git commit -m "feat(checkteam): add TxPriority for block inclusion ordering"
```

---

## Task 6: 铸造集成消息结构（internal/checkteam/minting.go）

**Files:**
- Create: `internal/checkteam/minting.go`

**Step 1: 编写消息结构**

```go
// internal/checkteam/minting.go
package checkteam

import (
    "github.com/cxio/evidcoin/internal/consensus"
    "github.com/cxio/evidcoin/internal/tx"
    "github.com/cxio/evidcoin/internal/utco"
    "github.com/cxio/evidcoin/internal/utxo"
    "github.com/cxio/evidcoin/pkg/types"
)

// MintInfoRequest 铸造者向管理层发送的铸造申请（步骤 1）。
type MintInfoRequest struct {
    BestPoolProof consensus.MintCandidate // 择优凭证
}

// MintInfoResponse 管理层返回给铸造者的铸造信息（步骤 2）。
type MintInfoResponse struct {
    TotalFee         int64          // 当前区块总手续费（用于构造 Coinbase）
    MintingAmount    int64          // 本区块铸币奖励
    ValidatorAddress [32]byte       // 校验组奖励地址
    BlockqsAddress   [32]byte       // Blockqs 奖励地址
    DepotsAddress    [32]byte       // Depots 奖励地址
    StunAddress      [32]byte       // STUN 奖励地址
    LotterySlotData  []byte         // 兑奖槽数据（可选，用于 SYS_AWARD）
}

// CoinbaseAck 管理层打包 Coinbase 后返回给铸造者的确认（步骤 4）。
type CoinbaseAck struct {
    CoinbaseMerklePath []types.Hash384 // Coinbase 的 Merkle 验证路径（连接 Coinbase 至 TxTreeRoot）
    TxTreeRoot         types.Hash384   // 交易哈希树根
    UTXOFingerprint    types.Hash256   // UTXO 指纹
    UTCOFingerprint    types.Hash256   // UTCO 指纹
    CheckRoot          types.Hash384   // CheckRoot = SHA3-384(TxTreeRoot ‖ UTXOFp ‖ UTCOFp)
}

// BlockSignature 铸造者对 CheckRoot 的签名（步骤 5）。
type BlockSignature struct {
    MinterPub types.PublicKey // 铸造者公钥
    Signature []byte          // 对 CheckRoot 的 ML-DSA-65 签名
    CheckRoot types.Hash384   // 被签名的 CheckRoot
}

// 以下为 UTXO/UTCO 缓存服务中使用的 Outpoint 类型别名（方便外部调用）
var _ = utxo.Outpoint{}
var _ = utco.Outpoint{}
var _ = tx.Transaction{}
```

**Step 2: 构建验证**

```bash
go build ./internal/checkteam/...
```

**Step 3: 提交**

```bash
git add internal/checkteam/minting.go
git commit -m "feat(checkteam): add minting workflow message structures"
```

---

## Task 7: 区块发布三阶段结构（internal/checkteam/publication.go）

**Files:**
- Create: `internal/checkteam/publication.go`

**Step 1: 编写发布结构**

```go
// internal/checkteam/publication.go
package checkteam

import "github.com/cxio/evidcoin/pkg/types"

// BlockProof 阶段一：区块证明广播（§7.1）。
// 使接收团队无需完整交易即可验证区块真实性。
type BlockProof struct {
    BlockHeader        []byte          // 序列化区块头
    CoinbaseTx         []byte          // 序列化 Coinbase 交易
    CoinbaseMerklePath []types.Hash384 // Coinbase 的 Merkle 验证路径
    MinterSignature    []byte          // 铸造者对 CheckRoot 的签名
    MinterPubKey       types.PublicKey // 铸造者公钥
}

// BlockSummary 阶段二：区块概要（§7.2）。
// 每个 TxID 截断为前 16 字节，接收方与本地已验证集对比并请求缺失交易。
type BlockSummary struct {
    BlockHeight   int32       // 区块高度
    TxIDPrefixes  [][16]byte  // 每笔交易 TxID 的前 16 字节（按区块内顺序）
}

// TxRequest 接收方请求缺失交易（阶段二响应）。
type TxRequest struct {
    BlockHeight int32
    Positions   []int // 缺失交易在区块内的位置索引
}

// IsValidBlockProof 检查 BlockProof 的基本完整性（非密码学校验）。
func IsValidBlockProof(bp *BlockProof) bool {
    return len(bp.BlockHeader) > 0 &&
        len(bp.CoinbaseTx) > 0 &&
        len(bp.MinterSignature) > 0
}
```

**Step 2: 构建验证**

```bash
go build ./internal/checkteam/...
```

**Step 3: 提交**

```bash
git add internal/checkteam/publication.go
git commit -m "feat(checkteam): add 3-phase block publication data structures"
```

---

## Task 8: 完整构建与测试

**Step 1: 完整构建**

```bash
go build ./...
```

**Step 2: 运行所有测试**

```bash
go test ./... -cover
```

预期：`internal/checkteam` 核心逻辑覆盖率 ≥80%。

**Step 3: 格式检查**

```bash
go fmt ./... && gofmt -s -w .
```

**Step 4: 最终提交**

```bash
git add .
git commit -m "feat(checkteam): complete Phase 7 team-based verification implementation"
```

---

## 验收标准

| 标准 | 命令 |
|------|------|
| 编译通过 | `go build ./...` |
| 测试全部通过 | `go test ./...` |
| 核心逻辑覆盖率 ≥80% | `go test -cover ./internal/checkteam/...` |
| 格式无变更 | `go fmt ./...` |
| 无 lint 警告 | `golangci-lint run` |

---

## 已知设计边界

1. **网络传输不在本包范围内。** 各角色之间的消息传递（Guard → Guard、Verifier → Guard、Management ↔ Minter）通过连接通信，连接层由外部 P2P 库（`cxio/p2p`）提供。
2. **脚本执行不在本包范围内。** 校验员的完整校验（`UnlockScript + LockScript → Execute`）调用 `internal/script` 包（Phase 5，后续实现）；本包仅声明接口。
3. **UTXO/UTCO 缓存底层存储** 由 `internal/utxo`/`internal/utco` 的 `Store` 提供（Phase 4）；本包仅定义缓存服务接口。
4. **冗余度配置（每笔交易分配几名校验员）** 由调度员在运行时决定，本包提供决策算法，不硬编码并发数量。

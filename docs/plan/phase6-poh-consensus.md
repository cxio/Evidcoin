# Phase 6: PoH Consensus（历史证明共识）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `internal/consensus` 包，涵盖铸凭哈希计算、择优池管理（BestPool）、分叉解决、区块时间计算与出块参数校验。

**Architecture:** Layer 4 共识层。依赖 `pkg/types`、`pkg/crypto`（Layer 0），`internal/blockchain`（Layer 1），`internal/tx`（Layer 1），`internal/utxo`/`internal/utco`（Layer 2）。**不依赖** Layer 3（脚本层），共识层不执行脚本，仅校验结构完整性与哈希。

**Tech Stack:** Go 1.25+，`lukechampine.com/blake3`（BLAKE3-512），ML-DSA-65 签名（`pkg/crypto`），`golang.org/x/crypto`（SHA3-384）。

---

## 目录结构（预期）

```
internal/consensus/
  params.go         # 设计参数常量
  blocktime.go      # 区块时间戳计算（BlockTime / RefBlockHeight）
  minting.go        # 铸凭哈希计算（MintingHash）
  eligibility.go    # 铸凭交易合法性检查（IsMintTxEligible）
  bestpool.go       # 择优池（BestPool / MintCandidate）
  fork.go           # 分叉解决（ResolveFork）
  stakes.go         # 币权计算（CoinStakes）
  consensus_test.go # 单元测试
```

---

## Task 1: 设计参数常量（internal/consensus/params.go）

**Files:**
- Create: `internal/consensus/params.go`

**Step 1: 编写参数文件**

```go
// internal/consensus/params.go
// Package consensus 实现 Evidcoin 历史证明（PoH）共识机制。
package consensus

import "time"

const (
    // BlockInterval 出块时间间隔（6 分钟）。
    BlockInterval = 6 * time.Minute

    // BlocksPerYear 每年区块数（≈365.25636 天 × 24h × 60min / 6min）。
    BlocksPerYear = 87661

    // MintTxMinDepth 铸凭交易距当前高度的最小深度（排除尾部，防塑造）。
    MintTxMinDepth = 28

    // MintTxMaxDepth 铸凭交易距当前高度的最大深度（约 11 个月）。
    MintTxMaxDepth = 80000

    // RefBlockOffset 评参区块相对当前高度的偏移（取 Height-9）。
    RefBlockOffset = 9

    // StakeBlockOffset 币权销毁来源区块的偏移（取 Height-27）。
    StakeBlockOffset = 27

    // BestPoolCapacity 择优池最大容量。
    BestPoolCapacity = 20

    // SyncAuthorizedStart 可发起同步的池成员起始排名（从 0 计，第 6–20 名 = 索引 5–19）。
    SyncAuthorizedStart = 5

    // ForkWindowSize 分叉竞争窗口（区块数）。
    ForkWindowSize = 29

    // ForkMajority 分叉胜出所需最小胜场数。
    ForkMajority = ForkWindowSize/2 + 1

    // TxExpiryBlocks 未确认交易过期区块数（24 小时）。
    TxExpiryBlocks = 240

    // FeeRecalcPeriod 最低手续费重算周期（约 25 天）。
    FeeRecalcPeriod = 6000

    // BlockPublishDelay1st 排名第 1 候选者的出块延迟（秒）。
    BlockPublishDelay1st = 30

    // BlockPublishDelayStep 每个后续候选者的额外延迟（秒）。
    BlockPublishDelayStep = 15

    // MintingMixConst 铸凭哈希混合常数。
    MintingMixConst = uint64(0x517cc1b727220a95)

    // CoinbaseConfirmDepth 新币花费所需的区块确认数。
    CoinbaseConfirmDepth = 29

    // RewardWindowBlocks 公共服务奖励兑换窗口（区块数）。
    RewardWindowBlocks = 48

    // RewardMinConfirmations 奖励全额兑换所需最少确认次数。
    RewardMinConfirmations = 2
)
```

**Step 2: 构建验证**

```bash
go build ./internal/consensus/...
```

---

## Task 2: 区块时间戳计算（internal/consensus/blocktime.go）

**Files:**
- Create: `internal/consensus/blocktime.go`
- Modify: `internal/consensus/consensus_test.go`

**Step 1: 编写失败测试**

```go
// internal/consensus/consensus_test.go
package consensus_test

import (
    "testing"
    "time"
    "github.com/cxio/evidcoin/internal/consensus"
)

func TestBlockTime(t *testing.T) {
    genesis := int64(1_700_000_000) // 固定创世时间戳（秒）
    tests := []struct {
        height int
        want   int64
    }{
        {0, genesis},
        {1, genesis + int64(consensus.BlockInterval/time.Second)},
        {10, genesis + 10*int64(consensus.BlockInterval/time.Second)},
    }
    for _, tt := range tests {
        got := consensus.BlockTime(genesis, tt.height)
        if got != tt.want {
            t.Errorf("BlockTime(%d) = %d, want %d", tt.height, got, tt.want)
        }
    }
}

func TestRefBlockHeight(t *testing.T) {
    tests := []struct {
        current int
        want    int
    }{
        {0, 0},
        {5, 0},  // height < 9，使用创世块
        {9, 0},
        {10, 1},
        {100, 91},
    }
    for _, tt := range tests {
        got := consensus.RefBlockHeight(tt.current)
        if got != tt.want {
            t.Errorf("RefBlockHeight(%d) = %d, want %d", tt.current, got, tt.want)
        }
    }
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/consensus/... -v -run TestBlockTime
```

**Step 3: 实现 blocktime.go**

```go
// internal/consensus/blocktime.go
package consensus

import "time"

// BlockTime 根据创世时间戳与区块高度计算区块时间戳（Unix 秒）。
// 时间戳由算法确定，不存储在区块头中。
func BlockTime(genesisTimestamp int64, height int) int64 {
    return genesisTimestamp + int64(height)*int64(BlockInterval/time.Second)
}

// RefBlockHeight 返回当前高度对应的评参区块高度。
// 链初始阶段（height < RefBlockOffset）使用创世块（高度 0）作为评参区块。
func RefBlockHeight(currentHeight int) int {
    if currentHeight < RefBlockOffset {
        return 0
    }
    return currentHeight - RefBlockOffset
}
```

**Step 4: 运行测试确认通过**

```bash
go test ./internal/consensus/... -v -run TestBlockTime
go test ./internal/consensus/... -v -run TestRefBlockHeight
```

**Step 5: 提交**

```bash
git add internal/consensus/params.go internal/consensus/blocktime.go internal/consensus/consensus_test.go
git commit -m "feat(consensus): add design params and BlockTime/RefBlockHeight"
```

---

## Task 3: 铸凭交易合法性（internal/consensus/eligibility.go）

**Files:**
- Create: `internal/consensus/eligibility.go`
- Modify: `internal/consensus/consensus_test.go`

**Step 1: 编写失败测试**

```go
// 追加到 consensus_test.go

func TestIsMintTxEligible(t *testing.T) {
    tests := []struct {
        name          string
        currentHeight int
        txHeight      int
        want          bool
    }{
        {"初段：height<28 均合法", 10, 5, true},
        {"初段：height<28 均合法2", 27, 0, true},
        {"尾部排除（depth<=27）", 100, 73, false},  // depth=27，不合法
        {"尾部排除（depth<28）", 100, 74, false},
        {"depth==28（合法下界）", 100, 72, true},
        {"depth==80000（合法上界）", 90000, 10000, true},
        {"depth>80000（超出）", 90001, 1, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := consensus.IsMintTxEligible(tt.currentHeight, tt.txHeight)
            if got != tt.want {
                t.Errorf("IsMintTxEligible(%d, %d) = %v, want %v",
                    tt.currentHeight, tt.txHeight, got, tt.want)
            }
        })
    }
}
```

**Step 2: 实现 eligibility.go**

```go
// internal/consensus/eligibility.go
package consensus

// IsMintTxEligible 判断铸凭交易是否在合法区块深度范围内。
// currentHeight：当前区块高度；txHeight：铸凭交易所在区块高度。
// 合法范围：depth ∈ (MintTxMinDepth, MintTxMaxDepth]，即 (27, 80000]。
func IsMintTxEligible(currentHeight, txHeight int) bool {
    if currentHeight < MintTxMinDepth {
        // 链初始阶段：所有已确认交易均可参与
        return true
    }
    depth := currentHeight - txHeight
    return depth > MintTxMinDepth-1 && depth <= MintTxMaxDepth
}
```

**Step 3: 运行测试**

```bash
go test ./internal/consensus/... -v -run TestIsMintTxEligible
```

**Step 4: 提交**

```bash
git add internal/consensus/eligibility.go internal/consensus/consensus_test.go
git commit -m "feat(consensus): add IsMintTxEligible for minting tx depth check"
```

---

## Task 4: 币权计算（internal/consensus/stakes.go）

**Files:**
- Create: `internal/consensus/stakes.go`
- Modify: `internal/consensus/consensus_test.go`

**Step 1: 编写失败测试**

```go
// 追加到 consensus_test.go

func TestCoinStakes(t *testing.T) {
    tests := []struct {
        amount    int64
        holdHours int
        want      int64
    }{
        {1000, 10, 10000},
        {500, 0, 0},   // 不足1小时记为零
        {100, 1, 100},
        {0, 100, 0},
    }
    for _, tt := range tests {
        got := consensus.CoinStakes(tt.amount, tt.holdHours)
        if got != tt.want {
            t.Errorf("CoinStakes(%d, %d) = %d, want %d", tt.amount, tt.holdHours, got, tt.want)
        }
    }
}

func TestMinTransactionFee(t *testing.T) {
    got := consensus.MinTransactionFee(1000)
    if got != 250 {
        t.Errorf("MinTransactionFee(1000) = %d, want 250", got)
    }
}
```

**Step 2: 实现 stakes.go**

```go
// internal/consensus/stakes.go
package consensus

// CoinStakes 计算币权（聪时 = 聪 × 持有整小时数）。
// amount：未花费输出金额（聪）；holdHours：持有整小时数（不足1小时截断为零）。
// 花费后币权归零——调用方负责在花费时清零。
func CoinStakes(amount int64, holdHours int) int64 {
    if holdHours <= 0 {
        return 0
    }
    return amount * int64(holdHours)
}

// MinTransactionFee 计算最低交易手续费（共约，非协议强制）。
// avgFeeLastPeriod：过去 FeeRecalcPeriod 个区块的平均手续费。
func MinTransactionFee(avgFeeLastPeriod int64) int64 {
    return avgFeeLastPeriod / 4
}
```

**Step 3: 运行测试**

```bash
go test ./internal/consensus/... -v -run TestCoinStakes
go test ./internal/consensus/... -v -run TestMinTransactionFee
```

**Step 4: 提交**

```bash
git add internal/consensus/stakes.go internal/consensus/consensus_test.go
git commit -m "feat(consensus): add CoinStakes and MinTransactionFee"
```

---

## Task 5: 铸凭哈希计算（internal/consensus/minting.go）

**Files:**
- Create: `internal/consensus/minting.go`
- Modify: `internal/consensus/consensus_test.go`

**背景：**

```
X        = encode_int64( blockTimestamp × stakes × MintingMixConst )
hashData = BLAKE3-512( txID ‖ refMintHash ‖ X )
signData = Sign( privateKey, hashData )
MintHash = BLAKE3-512( signData )
```

选择铸凭哈希**最小**值的候选者获胜。

**Step 1: 编写失败测试**

```go
// 追加到 consensus_test.go

import (
    "github.com/cxio/evidcoin/pkg/crypto"
    "github.com/cxio/evidcoin/pkg/types"
)

func TestMintingHash_Deterministic(t *testing.T) {
    // 使用测试用私钥（从 crypto 包获取）
    priv, pub, err := crypto.GenerateKeyPair()
    if err != nil {
        t.Fatal(err)
    }
    _ = pub

    txID := types.Hash384{0x01, 0x02}
    refHash := types.Hash512{0xAA}
    var stakes uint64 = 10000
    ts := int64(1_700_000_000)

    h1, err := consensus.MintingHash(txID, refHash, stakes, ts, priv)
    if err != nil {
        t.Fatal(err)
    }
    h2, err := consensus.MintingHash(txID, refHash, stakes, ts, priv)
    if err != nil {
        t.Fatal(err)
    }
    // ML-DSA 是确定性签名（无随机数），结果应相同
    if h1 != h2 {
        t.Error("MintingHash must be deterministic for same inputs")
    }
}

func TestMintingHash_DiffersOnDiffInput(t *testing.T) {
    priv, _, _ := crypto.GenerateKeyPair()
    txID1 := types.Hash384{0x01}
    txID2 := types.Hash384{0x02}
    refHash := types.Hash512{}
    var stakes uint64 = 1

    h1, _ := consensus.MintingHash(txID1, refHash, stakes, 0, priv)
    h2, _ := consensus.MintingHash(txID2, refHash, stakes, 0, priv)
    if h1 == h2 {
        t.Error("MintingHash should differ for different txIDs")
    }
}
```

**Step 2: 实现 minting.go**

```go
// internal/consensus/minting.go
package consensus

import (
    "encoding/binary"

    "github.com/cxio/evidcoin/pkg/crypto"
    "github.com/cxio/evidcoin/pkg/types"
)

// MintingHash 计算铸凭哈希。
//
// 算法（三阶段）：
//  1. 构造动态因子 X = encode_int64(blockTimestamp × stakes × MixConst)。
//  2. 铸造者私钥对中间哈希签名（签名结果对外不可预测）。
//  3. 对签名结果再次 BLAKE3-512 得到铸凭哈希。
//
// 选择铸凭哈希最小值（字节序列比较）的候选者获得铸造权。
func MintingHash(
    txID types.Hash384,
    refMintHash types.Hash512,
    stakes uint64,
    blockTimestamp int64,
    privateKey crypto.PrivateKey,
) (types.Hash512, error) {
    // 阶段一：构造动态因子 X
    x := encodeInt64(blockTimestamp * int64(stakes) * int64(MintingMixConst))
    var src []byte
    src = append(src, txID[:]...)
    src = append(src, refMintHash[:]...)
    src = append(src, x...)
    hashData := crypto.BLAKE3512(src)

    // 阶段二：私钥签名
    signData, err := crypto.Sign(privateKey, hashData[:])
    if err != nil {
        return types.Hash512{}, err
    }

    // 阶段三：对签名结果哈希，得到铸凭哈希
    return crypto.BLAKE3512(signData), nil
}

// encodeInt64 将 int64 编码为 8 字节大端序（用于铸凭哈希混合因子）。
func encodeInt64(v int64) []byte {
    b := make([]byte, 8)
    binary.BigEndian.PutUint64(b, uint64(v))
    return b
}

// CompareMintHash 比较两个铸凭哈希；返回 -1/0/1（字节序列比较）。
// 铸凭哈希最小者获得铸造权。
func CompareMintHash(a, b types.Hash512) int {
    for i := range a {
        if a[i] < b[i] {
            return -1
        }
        if a[i] > b[i] {
            return 1
        }
    }
    return 0
}
```

> **注意：** `crypto.BLAKE3512`、`crypto.Sign`、`crypto.GenerateKeyPair` 和 `crypto.PrivateKey` 需在 `pkg/crypto` 中定义（Phase 1 应已包含）。若 `types.Hash512` 尚未定义需在 `pkg/types` 补充。

**Step 3: 运行测试**

```bash
go test ./internal/consensus/... -v -run TestMintingHash
```

**Step 4: 提交**

```bash
git add internal/consensus/minting.go internal/consensus/consensus_test.go
git commit -m "feat(consensus): implement MintingHash (3-phase PoH algorithm)"
```

---

## Task 6: 择优池（internal/consensus/bestpool.go）

**Files:**
- Create: `internal/consensus/bestpool.go`
- Modify: `internal/consensus/consensus_test.go`

**Step 1: 编写失败测试**

```go
// 追加到 consensus_test.go

func TestBestPool_Insert(t *testing.T) {
    pool := consensus.NewBestPool(10)
    c := consensus.MintCandidate{
        TxYear:   2026,
        TxID:     types.Hash384{0x01},
        MintHash: types.Hash512{0xFF}, // 大值（排名靠后）
    }
    inserted := pool.Insert(c)
    if !inserted {
        t.Error("should insert into empty pool")
    }
    if pool.Size() != 1 {
        t.Errorf("pool size = %d, want 1", pool.Size())
    }
}

func TestBestPool_CapacityEviction(t *testing.T) {
    pool := consensus.NewBestPool(2)
    // 插入 3 个候选者，第 3 个铸凭哈希最小（最优），应留下
    c1 := consensus.MintCandidate{MintHash: types.Hash512{0x50}}
    c2 := consensus.MintCandidate{MintHash: types.Hash512{0x30}}
    c3 := consensus.MintCandidate{MintHash: types.Hash512{0x10}} // 最优

    pool.Insert(c1)
    pool.Insert(c2)
    inserted := pool.Insert(c3) // 应替换 c1（最差）
    if !inserted {
        t.Error("better candidate should be inserted")
    }
    if pool.Size() != 2 {
        t.Errorf("pool size = %d, want 2", pool.Size())
    }
    best := pool.Best()
    if best.MintHash != c3.MintHash {
        t.Error("best candidate should have smallest hash")
    }
}

func TestBestPool_RejectWorse(t *testing.T) {
    pool := consensus.NewBestPool(2)
    pool.Insert(consensus.MintCandidate{MintHash: types.Hash512{0x10}})
    pool.Insert(consensus.MintCandidate{MintHash: types.Hash512{0x20}})

    // 新候选者更差，应被拒绝
    inserted := pool.Insert(consensus.MintCandidate{MintHash: types.Hash512{0xFF}})
    if inserted {
        t.Error("worse candidate should not be inserted into full pool")
    }
}

func TestBestPool_SyncAuthorized(t *testing.T) {
    pool := consensus.NewBestPool(consensus.BestPoolCapacity)
    // 插入 20 个候选者（排名 0–19）
    for i := 0; i < consensus.BestPoolCapacity; i++ {
        c := consensus.MintCandidate{MintHash: types.Hash512{byte(i)}}
        pool.Insert(c)
    }
    // 排名 5–19（索引 5–19）应是授权同步节点
    for i := 0; i < consensus.BestPoolCapacity; i++ {
        authorized := pool.IsSyncAuthorized(i)
        want := i >= consensus.SyncAuthorizedStart
        if authorized != want {
            t.Errorf("rank %d: authorized=%v, want=%v", i, authorized, want)
        }
    }
}
```

**Step 2: 实现 bestpool.go**

```go
// internal/consensus/bestpool.go
package consensus

import (
    "sort"
    "github.com/cxio/evidcoin/pkg/types"
)

// MintCandidate 铸造候选者的择优凭证信息。
type MintCandidate struct {
    TxYear    int            // 铸凭交易所在年度
    TxID      types.Hash384  // 铸凭交易 ID（48 字节）
    MinterPub types.PublicKey // 铸造者公钥（首笔输入来源输出接收者）
    SignData  []byte         // 铸造者对铸凭哈希源数据的签名
    MintHash  types.Hash512  // 铸凭哈希（BLAKE3-512，64 字节）
}

// BestPool 择优池：按铸凭哈希升序排列的铸造候选者集合。
type BestPool struct {
    capacity   int
    candidates []MintCandidate // 按 MintHash 升序
}

// NewBestPool 创建指定容量的空择优池。
func NewBestPool(capacity int) *BestPool {
    return &BestPool{capacity: capacity}
}

// Size 返回当前候选者数量。
func (p *BestPool) Size() int {
    return len(p.candidates)
}

// Best 返回排名第一（铸凭哈希最小）的候选者。
func (p *BestPool) Best() MintCandidate {
    return p.candidates[0]
}

// Insert 尝试插入候选者。若池未满直接插入；若池满则与最差（最大）比较，更优则替换。
// 返回是否成功插入。
func (p *BestPool) Insert(c MintCandidate) bool {
    if len(p.candidates) < p.capacity {
        p.candidates = append(p.candidates, c)
        p.sort()
        return true
    }
    // 池满：与最差比较
    worst := p.candidates[len(p.candidates)-1]
    if CompareMintHash(c.MintHash, worst.MintHash) < 0 {
        p.candidates[len(p.candidates)-1] = c
        p.sort()
        return true
    }
    return false
}

// IsSyncAuthorized 判断指定排名（0-indexed）是否为授权同步节点。
// 授权范围：排名 SyncAuthorizedStart 至 capacity-1（即后 15 名）。
func (p *BestPool) IsSyncAuthorized(rank int) bool {
    return rank >= SyncAuthorizedStart && rank < len(p.candidates)
}

// Candidates 返回当前候选者列表（按铸凭哈希升序，只读副本）。
func (p *BestPool) Candidates() []MintCandidate {
    result := make([]MintCandidate, len(p.candidates))
    copy(result, p.candidates)
    return result
}

// sort 对内部列表按铸凭哈希升序排序。
func (p *BestPool) sort() {
    sort.Slice(p.candidates, func(i, j int) bool {
        return CompareMintHash(p.candidates[i].MintHash, p.candidates[j].MintHash) < 0
    })
}
```

> **注意：** `types.PublicKey` 需在 `pkg/types` 中定义（Phase 1 应有，若为别名可调整）。

**Step 3: 运行测试**

```bash
go test ./internal/consensus/... -v -run TestBestPool
```

**Step 4: 提交**

```bash
git add internal/consensus/bestpool.go internal/consensus/consensus_test.go
git commit -m "feat(consensus): implement BestPool with capacity and sync authorization"
```

---

## Task 7: 分叉解决（internal/consensus/fork.go）

**Files:**
- Create: `internal/consensus/fork.go`
- Modify: `internal/consensus/consensus_test.go`

**Step 1: 编写失败测试**

```go
// 追加到 consensus_test.go

func TestResolveFork_ChallengerWins(t *testing.T) {
    var main [consensus.ForkWindowSize]types.Hash512
    var challenger [consensus.ForkWindowSize]types.Hash512
    // challenger 在所有位置都更小
    for i := range main {
        main[i] = types.Hash512{0xFF}
        challenger[i] = types.Hash512{0x01}
    }
    if !consensus.ResolveFork(main, challenger) {
        t.Error("challenger with all smaller hashes should win")
    }
}

func TestResolveFork_MainWins(t *testing.T) {
    var main [consensus.ForkWindowSize]types.Hash512
    var challenger [consensus.ForkWindowSize]types.Hash512
    // main 更小
    for i := range main {
        main[i] = types.Hash512{0x01}
        challenger[i] = types.Hash512{0xFF}
    }
    if consensus.ResolveFork(main, challenger) {
        t.Error("main chain with all smaller hashes should win")
    }
}

func TestResolveFork_EarlyTermination(t *testing.T) {
    // challenger 前 15 胜出，无需全部比较即可提前终止
    var main [consensus.ForkWindowSize]types.Hash512
    var challenger [consensus.ForkWindowSize]types.Hash512
    for i := 0; i < 15; i++ {
        challenger[i] = types.Hash512{0x01}
        main[i] = types.Hash512{0xFF}
    }
    if !consensus.ResolveFork(main, challenger) {
        t.Error("challenger winning 15/29 should win")
    }
}
```

**Step 2: 实现 fork.go**

```go
// internal/consensus/fork.go
package consensus

import "github.com/cxio/evidcoin/pkg/types"

// ResolveFork 比较主链与挑战链的铸凭哈希序列；返回 true 表示挑战链胜出。
// 逐区块比较铸凭哈希，胜场过半（≥ ForkMajority）则胜出；一旦超过半数可提前终止。
//
// 前提：挑战链的铸凭哈希合法性已由调用方验证。
func ResolveFork(main, challenger [ForkWindowSize]types.Hash512) bool {
    wins := 0
    losses := 0
    remaining := ForkWindowSize
    for i := 0; i < ForkWindowSize; i++ {
        remaining--
        if CompareMintHash(challenger[i], main[i]) < 0 {
            wins++
        } else {
            losses++
        }
        // 提前终止：已不可能翻盘
        if wins >= ForkMajority {
            return true
        }
        if losses > ForkWindowSize-ForkMajority {
            return false
        }
        _ = remaining
    }
    return wins >= ForkMajority
}
```

**Step 3: 运行测试**

```bash
go test ./internal/consensus/... -v -run TestResolveFork
```

**Step 4: 提交**

```bash
git add internal/consensus/fork.go internal/consensus/consensus_test.go
git commit -m "feat(consensus): implement ResolveFork with early termination"
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

预期：所有测试通过，`internal/consensus` 核心逻辑覆盖率 ≥80%。

**Step 3: 格式检查**

```bash
go fmt ./... && gofmt -s -w .
```

**Step 4: 最终提交**

```bash
git add .
git commit -m "feat(consensus): complete Phase 6 PoH consensus implementation"
```

---

## 验收标准

| 标准 | 命令 |
|------|------|
| 编译通过 | `go build ./...` |
| 测试全部通过 | `go test ./...` |
| 核心逻辑覆盖率 ≥80% | `go test -cover ./internal/consensus/...` |
| 格式无变更 | `go fmt ./...` |
| 无 lint 警告 | `golangci-lint run` |

---

## 已知依赖与前提条件

1. **`pkg/types` 必须已定义：** `Hash384`、`Hash512`、`Hash256`、`PublicKey`。
2. **`pkg/crypto` 必须已提供：** `BLAKE3512([]byte) types.Hash512`、`BLAKE3256([]byte) types.Hash256`、`SHA3384([]byte) types.Hash384`、`Sign(PrivateKey, []byte) ([]byte, error)`、`GenerateKeyPair() (PrivateKey, PublicKey, error)`。
3. **ML-DSA-65 确定性签名：** 若使用 Go 1.25 标准库 `crypto/mlkem65` / `crypto/mldsa65` 的 ML-DSA-65，确认其 Sign 为确定性（无随机数）——如有随机数，铸凭哈希不可重现，需调整测试策略（只验证不等性与结构正确性）。
4. **分叉处理的完整流程**（多区块缓存、链切换、mempool 回收）属于更高层的节点协调逻辑，不在本包范围内；本包仅提供 `ResolveFork` 决策函数。

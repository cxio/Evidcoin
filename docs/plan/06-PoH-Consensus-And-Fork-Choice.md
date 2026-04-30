# PoH Consensus And Fork Choice Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现历史证明共识的 MintHash、铸凭资格验证、择优池、同步池、确定性区块时间、快速转播状态和分叉选择规则。

**Architecture:** `internal/consensus/` 只处理共识验证和本地协作策略边界，通过接口读取区块头、交易片段、状态证明和签名验证结果。公共服务返回的数据必须先被验证，不能作为可信输入。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、`internal/blockchain`、`internal/tx` 接口、表驱动测试。

---

## 来源提案

- `docs/proposal/10.PoH-Consensus.md`
- `docs/proposal/11.Endpoint-Conventions-And-Fork-Choice.md`
- 依赖 `docs/proposal/06.Transaction-Model.md`
- 依赖 `docs/proposal/08.UTXO-UTCO-State.md`

## 非目标

- 不实现 P2P 网络。
- 不实现交易池调度。
- 不直接查询外部 Blockqs。
- 不自动解决长度超过 20 才发现的长期分叉。
- 不把端点共约误写成协议合法性规则。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/consensus/mint_hash.go` | MintHash 输入和排序 |
| `internal/consensus/mint_proof.go` | 铸凭资格证明和验证接口 |
| `internal/consensus/best_pool.go` | Best Pool 排序和容量 |
| `internal/consensus/sync_pool.go` | Sync Pool 签名、合并、防重放 |
| `internal/consensus/block_time.go` | 确定性区块时间 |
| `internal/consensus/relay.go` | 快速转播三阶段状态 |
| `internal/consensus/block_competition.go` | 同高区块竞争规则 |
| `internal/consensus/fork_choice.go` | 29 区块竞争与 15 胜规则 |
| `internal/consensus/decision.go` | 长度 20 临界裁决消息 |
| `internal/consensus/errors.go` | 错误定义 |

## Task 1: MintHash 类型、输入和排序

**Files:**
- Create: `internal/consensus/mint_hash.go`
- Create: `internal/consensus/mint_hash_test.go`
- Create: `internal/consensus/errors.go`

**Step 1: 写失败测试**

测试：

- `MintHash` 输出 64B。
- 排序按 unsigned lexicographic byte order。
- 不能按 hex 字符串排序。
- 相同 hash 的 tie-breaker 未固定时返回 `ErrTieBreakerUnspecified`。
- `timeStamp` 来自 `GenesisTime + height * BlockInterval`，不读取本机时间。

**Step 2: 处理未决项**

PoH 内层 `Hash(...)` 算法、domain tag、`X` 整数宽度仍未固定时，先实现 `MintHashInput.CanonicalBytes()` 和 `RankMintHashes()`，`ComputeMintHash()` 对未决参数返回明确错误，或要求调用方注入 `InnerHasher` 和 `XEncoder`。

**Step 3: 验证并提交**

```bash
go test ./internal/consensus -run 'TestMintHash' -v
git add internal/consensus/mint_hash.go internal/consensus/mint_hash_test.go internal/consensus/errors.go
git commit -m "feat: add mint hash ordering"
```

## Task 2: 铸凭资格验证接口

**Files:**
- Create: `internal/consensus/mint_proof.go`
- Create: `internal/consensus/mint_proof_test.go`

**Step 1: 写失败测试**

测试：

- 高度窗口 `28 <= currentHeight - txHeight <= 80000`。
- 首领输入必须是 Coin 输入。
- 首领输入必须引用完整 48B `TxID`。
- 来源输出接收者地址哈希必须匹配铸造者公钥材料。
- 签名验证必须通过 `pkg/crypto` 抽象。
- 币权销毁门槛未固定时返回可区分的未决错误或配置化参数。

**Step 2: 实现接口化 verifier**

定义：

```go
type MintProofDataSource interface {
    MintTransaction(id types.TxID) (...)
    SourceOutput(...) (...)
    InclusionProof(...) (...)
}
```

不要直接 import Blockqs 客户端。

**Step 3: 验证并提交**

```bash
go test ./internal/consensus -run 'TestMintProof' -v
git add internal/consensus/mint_proof.go internal/consensus/mint_proof_test.go
git commit -m "feat: verify mint proof boundaries"
```

## Task 3: Best Pool

**Files:**
- Create: `internal/consensus/best_pool.go`
- Create: `internal/consensus/best_pool_test.go`

**Step 1: 写失败测试**

测试：

- 容量最多 20。
- 按 `MintHash` 升序。
- 新候选更优时进入池并挤出最差。
- 重复候选去重。
- 前 5 名不具备同步发起权。
- ranks 6..20 具备同步发起权。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'TestBestPool' -v
git add internal/consensus/best_pool.go internal/consensus/best_pool_test.go
git commit -m "feat: add poh best pool"
```

## Task 4: Sync Pool 签名、合并和防重放

**Files:**
- Create: `internal/consensus/sync_pool.go`
- Create: `internal/consensus/sync_pool_test.go`

**Step 1: 写失败测试**

测试：

- 只有 ranks 6..20 授权节点可发起同步。
- 每个授权节点对同一目标池只有一次同步权。
- 签名消息绑定目标评参区块、池内容、防重放字段。
- 多个同步池合并后仍按 `MintHash` 排序并截断到 20。
- 签名错误拒绝。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'TestSyncPool' -v
git add internal/consensus/sync_pool.go internal/consensus/sync_pool_test.go
git commit -m "feat: add poh sync pool"
```

## Task 5: 确定性区块时间与端点共约参数

**Files:**
- Create: `internal/consensus/block_time.go`
- Create: `internal/consensus/block_time_test.go`

**Step 1: 写失败测试**

测试：

- `BlockTime(height) = GenesisTime + height * BlockInterval`。
- 交易 `timestamp <= block.time`。
- 第一候选额外延迟 30 秒。
- 后续候选广播间隔 15 秒。
- 这些出块延迟标记为 convention，不作为区块头合法性必要条件。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'TestBlockTime' -v
git add internal/consensus/block_time.go internal/consensus/block_time_test.go
git commit -m "feat: add deterministic block time"
```

## Task 6: 快速转播三阶段状态

**Files:**
- Create: `internal/consensus/relay.go`
- Create: `internal/consensus/relay_test.go`

**Step 1: 写失败测试**

测试：

- 阶段 1 只验证 Coinbase、Coinbase 纳入证明、区块头、铸造者签名。
- 阶段 2 同步交易 ID 序列并识别缺失交易。
- 阶段 3 补齐交易后重算交易树和 `CheckRoot`。
- 阶段 1 通过不能标记为最终验证通过。
- 阶段 3 失败必须回退临时转播状态。

**Step 2: 实现状态机**

只实现状态和接口，不实现网络传输。

**Step 3: 验证并提交**

```bash
go test ./internal/consensus -run 'TestRelay' -v
git add internal/consensus/relay.go internal/consensus/relay_test.go
git commit -m "feat: add fast relay states"
```

## Task 7: 同高区块竞争

**Files:**
- Create: `internal/consensus/block_competition.go`
- Create: `internal/consensus/block_competition_test.go`

**Step 1: 写失败测试**

测试：

- 同一铸造者签署多个区块时，交易费收益较低者胜出。
- 候选区块币权销毁量达到主区块 3 倍时胜出。
- 交易费收益或币权销毁量定义未注入时返回未决错误。
- 3 倍边界是否包含等于未固定时测试标注未决。

**Step 2: 实现**

通过接口注入收益和币权数据，避免在共识包里重复交易/状态计算。

**Step 3: 验证并提交**

```bash
go test ./internal/consensus -run 'TestBlockCompetition' -v
git add internal/consensus/block_competition.go internal/consensus/block_competition_test.go
git commit -m "feat: add block competition policy"
```

## Task 8: 分叉选择和临界裁决

**Files:**
- Create: `internal/consensus/fork_choice.go`
- Create: `internal/consensus/decision.go`
- Create: `internal/consensus/fork_choice_test.go`
- Create: `internal/consensus/decision_test.go`

**Step 1: 写失败测试**

测试：

- 分叉长度 `<= 20` 可进入自动评比。
- 分叉长度 `> 20` 不进入自动竞争。
- 29 区块竞争窗口。
- 某链先达 15 wins 提前胜出。
- 逐块比较 `MintHash`。
- `MintHash` 相等处理未固定时返回未决错误。
- 长度 20 临界裁决消息绑定分叉点、本链末端、支链末端、当前高度、目标择优池引用、domain tag、防重放字段。
- 前 5 名裁决签名按排名选择最靠前有效签名。
- 前 5 名均无有效签名时默认拒绝。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'Test(ForkChoice|Decision)' -v
git add internal/consensus/fork_choice.go internal/consensus/decision.go internal/consensus/fork_choice_test.go internal/consensus/decision_test.go
git commit -m "feat: add fork choice rules"
```

## 阶段验收

运行：

```bash
go fmt ./...
go test ./internal/consensus
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- `MintHash` 排序跨平台稳定。
- 端点共约和协议规则在类型/注释/测试中明确区分。
- 快速转播临时状态不等于最终验证。
- 分叉选择不会自动处理 Proposal 明确排除的长期分叉。
- 未决 PoH 细节不会被伪装为最终规则。

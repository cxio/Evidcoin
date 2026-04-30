# Team Validation Services Incentives Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 定义校验组协作接口、公共服务可验证数据接口、Coinbase 奖励计算、公共服务兑奖和成熟期规则。

**Architecture:** 校验组和公共服务不是新的共识信任源，只提供接口、任务状态和可验证数据边界。激励逻辑依赖交易、状态和共识层已验证的数据，Coinbase 奖励输出必须进入状态层成熟期控制。

**Tech Stack:** Go 1.26.2、接口驱动设计、`internal/tx`、`internal/utxo`、`internal/consensus`、表驱动测试。

---

## 来源提案

- `docs/proposal/12.Team-Validation.md`
- `docs/proposal/13.Public-Service-Interfaces.md`
- `docs/proposal/14.Incentives-And-Coinbase-Rewards.md`

## 非目标

- 不实现组内 RPC。
- 不实现 P2P 拓扑。
- 不实现外部 Depots、Blockqs、stun2p 服务。
- 不实现校验员信誉系统。
- 不让公共服务返回值直接改变区块合法性。

## 建议包与文件

| 包/文件 | 内容 |
|---------|------|
| `internal/validation/task.go` | 校验任务接口 |
| `internal/validation/result.go` | 校验结果类型 |
| `internal/validation/leader_check.go` | 首领校验 |
| `internal/validation/review.go` | 冗余和复核流程 |
| `internal/validation/block_building.go` | 铸造协作接口 |
| `internal/services/depots.go` | Depots 可验证数据接口 |
| `internal/services/blockqs.go` | Blockqs 查询接口 |
| `internal/services/stun2p.go` | stun2p 观察接口 |
| `internal/services/verifiable_data.go` | 证明与验证结果结构 |
| `internal/rewards/subsidy.go` | 发行曲线 |
| `internal/rewards/fees.go` | 50% 回收 / 50% 销毁 |
| `internal/rewards/distribution.go` | 40/10/20/20/10 分配 |
| `internal/rewards/redemption.go` | 兑奖窗口和确认 |
| `internal/rewards/maturity.go` | Coinbase 成熟期 |
| `internal/rewards/slots.go` | 兑奖槽 bitset，未决时仅骨架 |

这些包属于 Layer 5 集成层：它们可以依赖 `pkg/*`、`internal/blockchain`、`internal/tx`、`internal/utxo`、`internal/utco`、`internal/script` 和 `internal/consensus` 的接口或稳定类型，但 Layer 0-4 不得反向 import 它们。如果项目希望减少包数量，可将 `internal/rewards` 合并进 `internal/tx` 或 `internal/consensus`，但推荐单独包以避免交易包承担经济策略。

## Task 1: 校验任务和结果模型

**Files:**
- Create: `internal/validation/task.go`
- Create: `internal/validation/result.go`
- Create: `internal/validation/task_test.go`
- Create: `internal/validation/result_test.go`

**Step 1: 写失败测试**

测试：

- 任务包含交易 ID、任务类型、分配时间、候选验证上下文。
- 结果区分合法、非法、拒绝任务、验证错误。
- 复核任务不暴露复核身份。
- 结果必须可追溯到校验员标识，但不定义信誉系统。

**Step 2: 实现并提交**

```bash
go test ./internal/validation -run 'Test(Task|Result)' -v
git add internal/validation/task.go internal/validation/result.go internal/validation/task_test.go internal/validation/result_test.go
git commit -m "feat: define validation tasks"
```

## Task 2: 首领校验

**Files:**
- Create: `internal/validation/leader_check.go`
- Create: `internal/validation/leader_check_test.go`

**Step 1: 写失败测试**

测试：

- 首笔输入必须是 Coin 输入。
- 首领校验只验证首笔输入。
- 首领校验通过只代表准合法。
- 完整验证失败后首领输入进入临时黑名单。
- 黑名单冻结时长默认配置化，不作为协议规则硬编码。
- “首笔输入应为全部 Coin 输入中币权最大者”如未确定协议地位，作为 convention 检查而非合法性检查。

**Step 2: 实现并提交**

```bash
go test ./internal/validation -run 'TestLeaderCheck' -v
git add internal/validation/leader_check.go internal/validation/leader_check_test.go
git commit -m "feat: add leader input checks"
```

## Task 3: 冗余与复核流程

**Files:**
- Create: `internal/validation/review.go`
- Create: `internal/validation/review_test.go`

**Step 1: 写失败测试**

测试：

- `MinValidationRedundancy = 2`。
- 两个结果均合法才进入合法池。
- 任一非法进入扩展复核。
- 一级复核零报错为合法。
- 一级复核超过半数报错为非法。
- 一级复核低于半数报错进入二级复核。
- 二级复核只要有报错即非法。

**Step 2: 实现并提交**

```bash
go test ./internal/validation -run 'TestReview' -v
git add internal/validation/review.go internal/validation/review_test.go
git commit -m "feat: add validation review flow"
```

## Task 4: 铸造协作接口

**Files:**
- Create: `internal/validation/block_building.go`
- Create: `internal/validation/block_building_test.go`

**Step 1: 写失败测试**

测试：

- 铸造候选者提交择优证明。
- 管理层返回交易费、校验组收益地址、公共服务推荐地址、铸币量、兑奖截留、兑奖槽推荐。
- 铸造者必须能验证 Coinbase 未被篡改。
- 管理层不能伪造铸造者签名。
- Coinbase 纳入证明缺失时拒绝进入区块证明阶段。

**Step 2: 实现接口和数据结构**

不实现网络 RPC，只定义协作消息和验证钩子。

**Step 3: 验证并提交**

```bash
go test ./internal/validation -run 'TestBlockBuilding' -v
git add internal/validation/block_building.go internal/validation/block_building_test.go
git commit -m "feat: define validation block building"
```

## Task 5: 公共服务可验证数据接口

**Files:**
- Create: `internal/services/depots.go`
- Create: `internal/services/blockqs.go`
- Create: `internal/services/stun2p.go`
- Create: `internal/services/verifiable_data.go`
- Create: `internal/services/services_test.go`

**Step 1: 写失败测试**

测试：

- Depots 返回附件必须用 `SHA3-512` 指纹验证。
- 分片必须用片组 Hash 和证明路径验证。
- 完整区块数据必须通过区块头、交易树和 `CheckRoot` 验证。
- Blockqs 返回 PoH 所需交易片段和证明路径，但结果默认不可信。
- stun2p 不参与区块、交易、PoH 或脚本验证。
- 服务不可达不改变区块合法性。

**Step 2: 实现接口**

接口返回 `Data + Proof`，验证函数在本地执行。不要把 HTTP、RPC、P2P 客户端写进核心接口。

**Step 3: 验证并提交**

```bash
go test ./internal/services -v
git add internal/services/depots.go internal/services/blockqs.go internal/services/stun2p.go internal/services/verifiable_data.go internal/services/services_test.go
git commit -m "feat: define public service interfaces"
```

## Task 6: 发行曲线和交易费回收

**Files:**
- Create: `internal/rewards/subsidy.go`
- Create: `internal/rewards/fees.go`
- Create: `internal/rewards/subsidy_test.go`
- Create: `internal/rewards/fees_test.go`

**Step 1: 写失败测试**

测试：

- 第 1 年 `10 coins/block`。
- 第 2 年 `20 coins/block`。
- 第 3 年 `30 coins/block`。
- 正式期从 `40 coins/block` 开始，每 2 年乘 80%。
- 长期低通胀 `3 coins/block`。
- 取整规则未固定时，正式递减测试只覆盖边界和返回未决错误。
- 交易费 50% 回收、50% 销毁。
- 奇数最小单位余数未固定时返回未决错误或配置化策略。

**Step 2: 实现并提交**

```bash
go test ./internal/rewards -run 'Test(Subsidy|Fees)' -v
git add internal/rewards/subsidy.go internal/rewards/fees.go internal/rewards/subsidy_test.go internal/rewards/fees_test.go
git commit -m "feat: add reward subsidy rules"
```

## Task 7: 奖励分配

**Files:**
- Create: `internal/rewards/distribution.go`
- Create: `internal/rewards/distribution_test.go`

**Step 1: 写失败测试**

测试：

- `RewardTotal = MintSubsidy + RecoveredTransactionFees + ReclaimedUnredeemedRewards`。
- 校验组 40%。
- 铸凭者/铸造者 10%。
- Depots 20%。
- Blockqs 20%。
- stun2p 10%。
- 除法余数未固定时返回未决错误或要求策略参数。

**Step 2: 实现并提交**

```bash
go test ./internal/rewards -run 'TestDistribution' -v
git add internal/rewards/distribution.go internal/rewards/distribution_test.go
git commit -m "feat: distribute block rewards"
```

## Task 8: 公共服务兑奖和成熟期

**Files:**
- Create: `internal/rewards/redemption.go`
- Create: `internal/rewards/maturity.go`
- Create: `internal/rewards/slots.go`
- Create: `internal/rewards/redemption_test.go`
- Create: `internal/rewards/maturity_test.go`
- Create: `internal/rewards/slots_test.go`

**Step 1: 写失败测试**

测试：

- `RedemptionWindow = 48 blocks`。
- `RequiredConfirmations = 2`。
- 1 次确认兑奖 50%。
- 2 次确认兑奖 100%。
- 即使提前 2 次确认，也需等待 `CoinbaseMaturity = 29 blocks`。
- 第 49 个区块回收未兑奖部分。
- 兑奖槽总计 144 bits = 18 bytes。
- bit 顺序未固定时，slot 编解码返回未决错误或只支持显式策略。

**Step 2: 实现并提交**

```bash
go test ./internal/rewards -run 'Test(Redemption|Maturity|Slots)' -v
git add internal/rewards/redemption.go internal/rewards/maturity.go internal/rewards/slots.go internal/rewards/redemption_test.go internal/rewards/maturity_test.go internal/rewards/slots_test.go
git commit -m "feat: add reward redemption rules"
```

## 阶段验收

运行：

```bash
go fmt ./...
go test ./internal/validation ./internal/services ./internal/rewards
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- 校验组不是共识权主体。
- 公共服务返回数据必须本地验证。
- 服务失败不改变区块合法性。
- Coinbase 成熟期和兑奖窗口测试覆盖。
- 未决奖励余数和 bit 顺序不被默认固化。

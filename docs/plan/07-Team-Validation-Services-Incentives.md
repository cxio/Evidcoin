# Team Validation Services Incentives Implementation Plan

**Goal:** 定义校验组协作接口、公共服务可验证数据接口、Coinbase 奖励计算、公共服务兑奖和成熟期规则。

**Architecture:** 校验组和公共服务不是新的共识信任源，只提供接口、任务状态和可验证数据边界。激励逻辑依赖交易、状态和共识层已验证的数据，Coinbase 奖励输出必须进入状态层成熟期控制。

**Tech Stack:** Go 1.26.2、接口驱动设计、`internal/tx`、`internal/utxo`、`internal/consensus`、表驱动测试。

---

## 来源提案

- `docs/proposal/12.Team-Validation.md`
- `docs/proposal/13.Public-Service-Interfaces.md`
- `docs/proposal/14.Incentives-And-Coinbase-Rewards.md`
- 依赖 ADR-0009：奖励分配和交易费分配使用整数顺序公式，奖励余数归 `stun2p`，交易费余数归 `destroyed`。
- 依赖 ADR-0012：铸造者签名消息为 `domainTag || chainIdentityBytes || CheckRoot`，签名不进入 `BlockID`。
- 依赖 ADR-0017：公共服务质量评估保持铸造者主观确认，不引入链上客观质量证明。
- 依赖 ADR-0026：首领黑名单冻结为本地共约，只影响入池和传播。
- 依赖 ADR-0028：全网通告通过普通 Proof 交易承载，授权公钥过滤属于客户端或服务展示层。
- 依赖 ADR-0029：百日扩张为客户端运行策略，不参与区块验证或奖励计算。
- 依赖 ADR-0031：协议层金额统一使用 `chx`，`1 Coin = 100,000,000 chx`。

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
- 黑名单冻结时长默认配置化，是本地入池和传播策略，不作为协议规则硬编码；区块验证不得因首领输入处于本地黑名单冻结期而拒绝区块。
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
- 铸造者签名消息构造固定为 `domainTag || chainIdentityBytes || CheckRoot`。
- 铸造者签名验证不把签名数据纳入 `BlockID` 或区块头哈希输入。
- 相同 `CheckRoot` 在不同 `chainIdentityBytes` 下签名消息不同，防止跨链重放。
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
- 公共服务质量评估只记录铸造者主观确认，不生成链上客观质量证明。
- 全网通告检索返回普通 Proof 交易、纳入证明和附件证明，客户端本地验证后再做授权展示过滤。
- 未授权公钥发布的 Announcement Proof 仍可作为普通 Proof 被检索，但默认不进入可信通告展示列表。
- 授权公钥列表支持初始公钥加载、公钥更新通告轮换、撤销通告失效处理和按初始授权区块高度进行权威级排序。

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

- 第 1 年 `1,000,000,000 chx/block`。
- 第 2 年 `2,000,000,000 chx/block`。
- 第 3 年 `3,000,000,000 chx/block`。
- 正式期从 `4,000,000,000 chx/block` 开始，每 2 年乘 80%。
- 长期低通胀 `300,000,000 chx/block`。
- 发行递减取整规则属于未被 ADR-0001 至 ADR-0031 覆盖的剩余开放问题，正式递减测试只覆盖边界和返回开放规格错误。
- 交易费 50% 回收、50% 销毁。
- 奇数交易费余数归 `destroyed`，例如 `TxFee = 101 chx` 时 `recovered = 50 chx`、`destroyed = 51 chx`。
- 交易费分配禁止使用浮点中间计算。
- 百日扩张不参与奖励计算、交易费回收或销毁计算；如实现提示，仅放在客户端、钱包或交易构造器策略层。

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
- 所有断言使用 `chx`，不得使用小数 Coin 或浮点金额。
- 分配公式为 `validation=R*40/100`、`minter=R*10/100`、`depots=R*20/100`、`blockqs=R*20/100`、`stun2p=R-validation-minter-depots-blockqs`。
- 奖励余数归 `stun2p`，例如 `RewardTotal = 101 chx` 时 `validation = 40 chx`、`minter = 10 chx`、`depots = 20 chx`、`blockqs = 20 chx`、`stun2p = 11 chx`。
- 奖励分配禁止使用浮点中间计算。

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
- 兑奖 slot bit 顺序属于未被 ADR-0001 至 ADR-0031 覆盖的剩余开放问题，slot 编解码返回开放规格错误或只支持显式策略。

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
- 首领黑名单和百日扩张均不得写入区块合法性或奖励计算路径，分别只作为本地策略和客户端策略。
- Coinbase 成熟期和兑奖窗口测试覆盖。
- 奖励余数归 `stun2p`、交易费余数归 `destroyed` 必须被测试覆盖；兑奖槽 bit 顺序待 ADR 固定前不被默认固化。

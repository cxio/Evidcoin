# Incentives & Coinbase Implementation Plan

**Goal:** 实现激励机制与 Coinbase 结算：铸币发行曲线、交易费 50% 销毁、Coinbase 收益结构与奖励分配、百日前/后输出 profile、金额取整与余数承接、兑奖槽编码与确认/回收、`reclaimed_award` 隐含计算、`BurnCoin` 非负语义。金额一律以 **chx** 整数承载。

**Architecture:** `internal/rewards/`（Layer 5）承载经济结算逻辑，Coinbase 序列化口径与 `internal/tx` 一致（输出配置值升序影响 TxID）。奖励金额计算禁止浮点中间值，全部整数除法；`reclaimed_award` 不显式编码但影响 Coinbase 金额与 TxID，验证器必须重算。

**Tech Stack:** Go 1.26.2、`pkg/types`（chx 单位、规范编码）、`internal/tx`（Coinbase 头与输出）、`internal/consensus`（铸造者/铸凭者）、表驱动测试。

---

## 来源提案

- `docs/proposal/14.Incentives-And-Coinbase.md`（主源）
- 依赖 `docs/proposal/01.Types-And-Encoding.md`（编码、chx 单位）、`03.Identifiers-And-Constants.md`（87661、单位）、`06.Transaction-Model.md`（Coinbase 头与输出、币金销毁单点化）、`11.PoH-Consensus.md`（铸造者/铸凭者）、`13.Team-Validation.md`（铸造协作）；`10.Script-System.md`（`SYS_AWARD` 仅公共服务奖励输出）。
- **DEC-0401**：Coinbase 输出顺序与配置值、百日前/后 profile、金额取整与余数承接、`BurnCoin` 非负、兑奖槽编码、`reclaimed_award` 隐含计算。

## 包边界

| 包 | 职责 | 禁止事项 |
|----|------|----------|
| `internal/rewards` | 发行曲线、交易费销毁、奖励分配、Coinbase 序列化口径、兑奖槽、回收、`reclaimed_award` | 不使用浮点、不实现公共服务内部逻辑、不让服务可达性改变区块合法性 |

`internal/rewards` 属 Layer 5，可依赖 Layer 0-4 接口/稳定类型，不被反向 import。如需减少包数，可并入 `internal/tx`/`internal/consensus`，但推荐独立以隔离经济策略。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/rewards/subsidy.go` | 发行曲线（三阶段） |
| `internal/rewards/fees.go` | 交易费 50% 销毁、奇数归属、`BurnCoin` |
| `internal/rewards/distribution.go` | 奖励分配（配置值升序、百日前/后比例） |
| `internal/rewards/coinbase.go` | Coinbase 输出序列化口径（百日前 2/百日后 5） |
| `internal/rewards/slots.go` | 兑奖槽 `[18]byte` 编解码、确认/回收 |
| `internal/rewards/reclaim.go` | `reclaimed_award` 隐含计算 |
| `internal/rewards/errors.go` | 错误定义 |

## Task 1: 发行曲线

**Files:**
- Create: `internal/rewards/subsidy.go`
- Create: `internal/rewards/subsidy_test.go`
- Create: `internal/rewards/errors.go`

**Step 1: 写失败测试**（proposal 14 §1）

- 每年 **87661** 块（恒星年 6min 间隔）；`1 币 = 10^8 chx`。
- 第一阶段（前三年）：`10/20/30` 币/块逐年增加（即 `1_000_000_000 / 2_000_000_000 / 3_000_000_000 chx`）。
- 第二阶段（正式发行期）：自 `40` 币/块起，每 **2 年**递减 **20%**，精确到 chx（整数除法）；递减到 `< 3` 币/块（`< 300_000_000 chx`）时初期铸币结束。
- 长期微通胀：之后固定 `300_000_000 chx/块`（3 币/块）。
- 边界测试点：第 1/2/3/5/7/…/25 年与转入长期微通胀的分界年。

**Step 2: 实现并提交**

整数除法，精确到 chx，禁止浮点。

```bash
go test ./internal/rewards -run 'TestSubsidy' -v
git add internal/rewards/subsidy.go internal/rewards/subsidy_test.go internal/rewards/errors.go
git commit -m "feat: add issuance subsidy curve"
```

## Task 2: 交易费销毁与 BurnCoin

**Files:**
- Create: `internal/rewards/fees.go`
- Create: `internal/rewards/fees_test.go`

**Step 1: 写失败测试**（proposal 14 §2 / DEC-0401）

- `burned_tx_fee = total_tx_fee / 2`（整数除法向下取整）；`unburned_tx_fee = total_tx_fee - burned_tx_fee`。
- 交易费为奇数 chx 时，多出的 1 chx 归 `unburned_tx_fee`（如 `total=101 → burned=50, unburned=51`）。
- 销毁单点化于 Coinbase（普通交易不可销毁币金，见 06 章 §3）。
- `BurnCoin` 记录**非负**的 `burned_tx_fee`（chx）；负值奇偶/余数编码已废弃（06 章字段类型 `int64`，本章约束恒非负）。
- 禁止浮点中间计算。

**Step 2: 实现并提交**

```bash
go test ./internal/rewards -run 'TestFees' -v
git add internal/rewards/fees.go internal/rewards/fees_test.go
git commit -m "feat: add tx fee burn rule"
```

## Task 3: 奖励分配与 Coinbase 输出 profile

**Files:**
- Create: `internal/rewards/distribution.go`
- Create: `internal/rewards/coinbase.go`
- Create: `internal/rewards/distribution_test.go`
- Create: `internal/rewards/coinbase_test.go`

**Step 1: 写失败测试**（proposal 14 §3·§4·§5 / DEC-0401）

- `RewardBase = issuance + unburned_tx_fee + reclaimed_award`。
- 百日后（`height >= 24001`）5 输出，按**配置值升序**排列（影响 TxID）：

| 配置值 | 受奖方 | 百日后比例 |
|--------|--------|-----------|
| 1 | 铸凭者 | 10% |
| 2 | 校验组 | 40% |
| 3 | Blockqs | 20% |
| 4 | Depots | 20% |
| 5 | STUN | 10% |

- 百日前（`height <= 24000`）2 输出：铸凭者 20%、校验组 80%（重标定）；无公共服务奖励、不使用 `SYS_AWARD`，Coinbase 头仍含 `AwardSlots [18]byte` 但值恒为全零。
- 金额取整：前 N-1 项按 `RewardBase × percent / 100` 向下取整；**最后一项**承接全部余数（百日前=校验组、百日后=STUN）。
- 输出顺序固定为配置值升序，不得调换。
- 边界测试点：高度 0/24000/24001、`RewardBase` 含余数、各项取整与最后一项承接。

**Step 2: 实现并提交**

```bash
go test ./internal/rewards -run 'Test(Distribution|Coinbase)' -v
git add internal/rewards/distribution.go internal/rewards/coinbase.go internal/rewards/distribution_test.go internal/rewards/coinbase_test.go
git commit -m "feat: add reward distribution and coinbase profile"
```

## Task 4: 兑奖槽编码与确认/回收

**Files:**
- Create: `internal/rewards/slots.go`
- Create: `internal/rewards/slots_test.go`

**Step 1: 写失败测试**（proposal 14 §6 / DEC-0401）

- `AwardSlots [18]byte` 不作为输出项，创世与百日前 Coinbase 该字段值恒为全零。Blockqs/Depots/STUN 各 **6** 字节，顺序为 Blockqs 6B、Depots 6B、STUN 6B。
- 每槽覆盖前 48 块每块 1 bit：`bit0` 对应 `H-1`，`bit47` 对应 `H-48`。
- 确认窗口：区块 `K` 的公共服务奖励在 `K+1..K+48` 被后续 Coinbase 对应槽确认；1 次确认兑 50%，2 次确认兑 100%。
- 提取时机：花费公共服务奖励输出必须至少在 `K+31` 之后。
- 回收：到 `K+49`，未被确认可兑的剩余部分进入该块 Coinbase 的 `reclaimed_award`。
- 边界测试点：兑 0%/50%/100%、`bit0`/`bit47` 映射、第 49 块回收。

**Step 2: 实现并提交**

DEC-0401 已固定兑奖槽 bit 顺序，按定值实现，不作为待决占位。

```bash
go test ./internal/rewards -run 'TestSlots' -v
git add internal/rewards/slots.go internal/rewards/slots_test.go
git commit -m "feat: add award slots encoding"
```

## Task 5: reclaimed_award 隐含计算

**Files:**
- Create: `internal/rewards/reclaim.go`
- Create: `internal/rewards/reclaim_test.go`

**Step 1: 写失败测试**（proposal 14 §7 / DEC-0401）

- `reclaimed_award` 不新增单独输出项、不作为 Coinbase 头字段。
- 作为 `RewardBase` 的隐含输入项，由验证器根据 `H-49` 区块公共服务输出金额与后续 48 块兑奖槽计算得出。
- 当前块 Coinbase 金额校验时必须重算 `reclaimed_award` 并纳入 `RewardBase`。
- 边界测试点：`H-49` 全兑/部分兑/未兑情形下的回收额与 `RewardBase` 一致性。

**Step 2: 实现并提交**

```bash
go test ./internal/rewards -run 'TestReclaim' -v
git add internal/rewards/reclaim.go internal/rewards/reclaim_test.go
git commit -m "feat: add reclaimed award computation"
```

## 待决问题

- 本篇无 conception/decision 未覆盖的待决项（DEC-0401 已冻结序列化、取整、兑奖槽、回收与百日前 profile）。货币单位口径（C-8）已裁决（`1 Bi = 10^8 chx`）并由 01 章承载，本篇发行/销毁口径一律 chx。
- 创世 Coinbase 具体工件取值依赖 **C-9**（见 02、07 章），与本篇结算口径无冲突；占位不影响本篇算法实现。

## 阶段门禁/验收

进入条件：区块与共识（02、07、08）稳定，Coinbase 头与输出（03）可引用。

运行：

```bash
go fmt ./...
go test ./internal/rewards
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- 发行曲线精确到 chx 整数除法，第二阶段每 2 年递减 20% 至 `<3` 币/块转长期微通胀（恒定 `300_000_000 chx/块`）。
- 交易费 `burned=total/2`、奇数余 1 归未销毁、`BurnCoin` 恒非负；禁止浮点。
- Coinbase 输出配置值升序排列，百日前 2 输出（无 `SYS_AWARD`）、百日后 5 输出，分界 24000/24001；前 N-1 向下取整、最后一项承接余数。
- 兑奖槽 18 字节、三服务各 6 字节、`bit0=H-1`/`bit47=H-48`；提取须 `>=K+31`；`K+49` 回收。
- `reclaimed_award` 隐含计算并纳入 `RewardBase`，金额校验时重算。
- 测试覆盖高度 0/24000/24001、奇数交易费、输出余数归属、兑奖 0/50/100%、第 49 块回收（DEC-0401）。
- 金额一律 chx；`internal/rewards` 不被 Layer 0-4 反向 import。

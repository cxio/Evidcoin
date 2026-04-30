# Transaction And Units Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现交易头、输入模型、输出 envelope、Coin/Credit/Proof/Mediator/Custom payload、签名消息和 Coinbase 初始边界。

**Architecture:** `internal/tx/` 只表达交易数据、规范化编码、Hash 和本地可判定的结构规则；状态可用性、脚本执行、PoH 资格和完整 Coinbase 奖励结算通过接口或后续层处理。交易包可依赖 `pkg/types`、`pkg/crypto`、`pkg/hashtree` 和 `internal/blockchain` 的链身份类型，但不能依赖状态、脚本或共识具体实现。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、`pkg/hashtree`、表驱动测试。

---

## 来源提案

- `docs/proposal/06.Transaction-Model.md`
- `docs/proposal/07.Coin-Credit-Proof-Units.md`
- `docs/proposal/14.Incentives-And-Coinbase-Rewards.md` 的 Coinbase 和交易费边界
- 依赖 `docs/proposal/01.Types-And-Encoding.md`
- 依赖 `docs/proposal/04.Hash-Trees.md`

## 非目标

- 不检查输入是否存在于 UTXO/UTCO。
- 不执行锁定脚本或解锁脚本。
- 不验证 PoH 铸凭资格。
- 不实现完整公共服务兑奖。
- 不选择交易池策略。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/tx/header.go` | `TxHeader`、规范化编码、`TxID` |
| `internal/tx/input.go` | `LeadInput`、`RestInput`、局部引用结构 |
| `internal/tx/input_hash.go` | `LeadHash`、`RestHash`、`HashInputs` |
| `internal/tx/output.go` | 输出 envelope、类型与标记 |
| `internal/tx/coin.go` | Coin payload |
| `internal/tx/credit.go` | Credit payload 与配置 |
| `internal/tx/proof.go` | Proof payload |
| `internal/tx/attachment.go` | `AttachmentID`、片组树引用 |
| `internal/tx/mediator.go` | Mediator payload |
| `internal/tx/custom.go` | Custom payload 边界 |
| `internal/tx/signature_message.go` | 签名 flag 与签名消息构造 |
| `internal/tx/fees.go` | Coin 输入输出金额差额计算接口 |
| `internal/tx/coinbase.go` | Coinbase 结构骨架和位置规则 |
| `internal/tx/validate.go` | 本地结构验证 |
| `internal/tx/errors.go` | 错误定义 |

## Task 1: 交易头与 TxID

**Files:**
- Create: `internal/tx/header.go`
- Create: `internal/tx/header_test.go`

**Step 1: 写失败测试**

测试：

- `TxHeader` 字段顺序为 `Version, HashInputs, HashOutputs, Timestamp`。
- 相同交易头得到相同 `TxID`。
- 修改任一字段会改变 `TxID`。
- `TxID` 输出 48B。

**Step 2: 运行测试确认失败**

```bash
go test ./internal/tx -run 'TestTxHeader' -v
```

**Step 3: 最小实现并提交**

```bash
go test ./internal/tx -run 'TestTxHeader' -v
git add internal/tx/header.go internal/tx/header_test.go
git commit -m "feat: add transaction header hashing"
```

## Task 2: 输入模型与 HashInputs

**Files:**
- Create: `internal/tx/input.go`
- Create: `internal/tx/input_hash.go`
- Create: `internal/tx/input_test.go`
- Create: `internal/tx/input_hash_test.go`

**Step 1: 写失败测试**

测试：

- `LeadInput` 必须包含完整 48B `TxID`。
- `LeadInput` 必须标记为 Coin 输入。
- `RestInput` 使用 `TxIDPart` 前 20B、`Year`、`OutIndex`。
- Proof 输入类型被结构验证拒绝。
- `HashInputs = BLAKE3-256(LeadHash || RestHash)`。
- Rest inputs 顺序变化导致 `HashInputs` 变化。

**Step 2: 实现**

定义输入类型常量，显式建模 `TransferIndex` 只适用于 Credit。

**Step 3: 验证并提交**

```bash
go test ./internal/tx -run 'Test(Input|HashInputs)' -v
git add internal/tx/input.go internal/tx/input_hash.go internal/tx/input_test.go internal/tx/input_hash_test.go
git commit -m "feat: add transaction input hashing"
```

## Task 3: 输出 envelope

**Files:**
- Create: `internal/tx/output.go`
- Create: `internal/tx/output_test.go`

**Step 1: 写失败测试**

测试：

- 配置字节高 4 位为 flags，低 4 位为 type。
- 类型 `1` Coin、`2` Credit、`3` Proof、`4` Mediator。
- `Serial` 从 0 开始，必须等于输出列表位置。
- 未知公共输出类型拒绝。
- 销毁 flag 可解析，但状态处理延后。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestOutputEnvelope' -v
git add internal/tx/output.go internal/tx/output_test.go
git commit -m "feat: add transaction output envelope"
```

## Task 4: Coin payload

**Files:**
- Create: `internal/tx/coin.go`
- Create: `internal/tx/coin_test.go`

**Step 1: 写失败测试**

测试：

- Coin payload 编码包含 `Receiver`、`Amount`、`Memo optional Bytes`、`LockScript`。
- `Amount == 0` 是否允许必须按 Proposal 标注；未决时测试拒绝 0 并加 `TODO(spec)`。
- `LockScript` 超过 `MaxLockScript` 拒绝。
- Coin 不接受 AttachmentID。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestCoin' -v
git add internal/tx/coin.go internal/tx/coin_test.go
git commit -m "feat: add coin payload"
```

## Task 5: Credit payload

**Files:**
- Create: `internal/tx/credit.go`
- Create: `internal/tx/credit_test.go`

**Step 1: 写失败测试**

测试：

- Credit payload 编码包含 receiver、creator、config、title、description、optional attachment、lock script。
- 可修改性只能降级，不能恢复。
- 创建者、标题、附件 ID 作为不可变字段参与转移比较。
- 高度截止超过相对 100 年拒绝。
- 无期限 Credit 激活规则先以接口暴露，不在 payload 验证中硬编码。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestCredit' -v
git add internal/tx/credit.go internal/tx/credit_test.go
git commit -m "feat: add credit payload"
```

## Task 6: Proof、Attachment、Mediator、Custom

**Files:**
- Create: `internal/tx/proof.go`
- Create: `internal/tx/attachment.go`
- Create: `internal/tx/mediator.go`
- Create: `internal/tx/custom.go`
- Create: `internal/tx/proof_test.go`
- Create: `internal/tx/attachment_test.go`
- Create: `internal/tx/mediator_test.go`
- Create: `internal/tx/custom_test.go`

**Step 1: 写失败测试**

测试：

- Proof payload 编码包含 creator、title、content、optional attachment、identify script。
- Proof 不可作为输入。
- AttachmentID 必须 64B。
- 片组树引用必须是 32B `TreeHash`。
- Mediator 不可作为输入。
- Custom 默认不能作为公共输入源。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'Test(Proof|Attachment|Mediator|Custom)' -v
git add internal/tx/proof.go internal/tx/attachment.go internal/tx/mediator.go internal/tx/custom.go internal/tx/proof_test.go internal/tx/attachment_test.go internal/tx/mediator_test.go internal/tx/custom_test.go
git commit -m "feat: add non-coin transaction units"
```

## Task 7: 输出树与 HashOutputs

**Files:**
- Create: `internal/tx/output_hash.go`
- Create: `internal/tx/output_hash_test.go`

**Step 1: 写失败测试**

测试：

- 输出列表顺序变化导致 `HashOutputs` 变化。
- `Serial` 不匹配输出位置时拒绝。
- 单输出、多个输出路径可计算。
- 空输出普通交易拒绝。

**Step 2: 实现**

使用 `pkg/hashtree`，但空树、单叶、奇数叶按 Proposal 未决项显式选择策略。

**Step 3: 验证并提交**

```bash
go test ./internal/tx -run 'TestHashOutputs' -v
git add internal/tx/output_hash.go internal/tx/output_hash_test.go
git commit -m "feat: add transaction output hashing"
```

## Task 8: 签名消息

**Files:**
- Create: `internal/tx/signature_message.go`
- Create: `internal/tx/signature_message_test.go`

**Step 1: 写失败测试**

测试：

- 默认 flag 为 `SIGIN_ALL | SIGOUT_ALL | SIGOUTPUT`。
- 非法 flag 组合拒绝。
- 签名消息包含链识别信息。
- 修改链身份、输入范围或输出范围会改变签名消息。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestSignatureMessage' -v
git add internal/tx/signature_message.go internal/tx/signature_message_test.go
git commit -m "feat: add transaction signature messages"
```

## Task 9: Coinbase 骨架与交易费接口

**Files:**
- Create: `internal/tx/fees.go`
- Create: `internal/tx/coinbase.go`
- Create: `internal/tx/fees_test.go`
- Create: `internal/tx/coinbase_test.go`

**Step 1: 写失败测试**

测试：

- 普通交易费 = Coin 输入总额 - Coin 输出总额。
- 输出总额大于输入总额拒绝。
- Coinbase 无输入。
- Coinbase 必须位于区块交易序列第 0 项。
- Coinbase 字段未完全固定时，编码函数返回明确未实现错误或只编码已固定字段。

**Step 2: 实现**

交易包只定义 Coinbase 结构边界和位置验证。奖励计算放到 `07-Team-Validation-Services-Incentives.md` 对应任务。

**Step 3: 验证并提交**

```bash
go test ./internal/tx -run 'Test(Fees|Coinbase)' -v
git add internal/tx/fees.go internal/tx/coinbase.go internal/tx/fees_test.go internal/tx/coinbase_test.go
git commit -m "feat: add coinbase transaction boundaries"
```

## 阶段验收

运行：

```bash
go fmt ./...
go test ./internal/tx ./pkg/types ./pkg/crypto ./pkg/hashtree
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- 普通交易必须至少有 Coin lead input。
- Proof、Mediator、Custom 默认不能作为公共输入源。
- 输出 envelope、payload 和签名消息均有表驱动测试。
- 交易包不 import `internal/utxo`、`internal/utco`、`internal/script`、`internal/consensus`。

# Transaction And Units Implementation Plan

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
- 依赖 ADR-0010：Coinbase `HashInputs` 使用 `BLAKE3-256(DomainTag(CoinbaseInputs) || bigEndianUint64(blockHeight))`。
- 依赖 ADR-0023：Coinbase 独立处理，只允许 Coin 输出。
- 依赖 ADR-0024：`SYS_CHKPASS` 标准验证签名由 Witness 注入；定制验证签名可由 UnlockScript 普通数据提供；UnlockScript 参与输入 Hash 和 `TxID`，Witness 不参与 `TxID`。
- 依赖 ADR-0021：Credit `Config[9:0]` 约束 `len(description)` 最大值。
- 依赖 ADR-0031：Coin `Amount` 以 `chx` 为单位。
- 依赖 ADR-0029：百日扩张为客户端运行策略，核心交易验证不检查输入输出数量比例。

## 非目标

- 不检查输入是否存在于 UTXO/UTCO。
- 不执行锁定脚本或解锁脚本。
- 不验证 PoH 铸凭资格。
- 不实现完整公共服务兑奖。
- 不选择交易池策略。
- 不在核心交易验证中检查百日扩张的输入输出数量比例；该检查仅可作为交易构造器或钱包的可选客户端策略。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/tx/header.go` | `TxHeader`、规范化编码、`TxID` |
| `internal/tx/input.go` | `LeadInput`、`RestInput`、局部引用结构、UnlockScript 输入字段 |
| `internal/tx/input_hash.go` | `LeadHash`、`RestHash`、`HashInputs` |
| `internal/tx/size.go` | 不含 Witness 的 `MaxTxSize` 检查 |
| `internal/tx/output.go` | 输出 envelope、类型与标记 |
| `internal/tx/coin.go` | Coin payload |
| `internal/tx/credit.go` | Credit payload 与配置 |
| `internal/tx/proof.go` | Proof payload |
| `internal/tx/attachment.go` | `AttachmentID`、片组树引用 |
| `internal/tx/mediator.go` | Mediator payload |
| `internal/tx/custom.go` | Custom payload 边界 |
| `internal/tx/signature_message.go` | 签名 flag 与签名消息构造 |
| `internal/tx/witness.go` | Witness 签名附件、完整交易编码和 TxID 排除规则 |
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
- `LeadInput` 和 `RestInput` 均包含 `UnlockScript`。
- 定制验证签名字节可位于 `UnlockScript`，并作为普通输入字节参与规范编码。
- `RestInput` 使用 `TxIDPart` 前 20B、`Year`、`OutIndex`。
- Proof 输入类型被结构验证拒绝。
- `HashInputs = BLAKE3-256(LeadHash || RestHash)`。
- 修改任一输入的 `UnlockScript` 会改变 `LeadHash` 或 `RestHash`，继而改变 `HashInputs` 和 `TxID`。
- Rest inputs 顺序变化导致 `HashInputs` 变化。

**Step 2: 实现**

定义输入类型常量，显式建模 `TransferIndex` 只适用于 Credit。输入规范编码必须包含 `UnlockScript`，并应用 `MaxUnlockScript` 长度检查；Witness 字段不得进入输入规范编码。

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
- `Amount` 类型和测试向量明确以 `chx` 为单位，使用 `uint64` 表示；不得用小数 Coin 参与协议编码。
- `Amount == 0` 是否允许必须按 Proposal 标注；开放规格项阶段测试拒绝 0，并说明该规则未被 ADR-0001 至 ADR-0031 覆盖。
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
- `len(description) <= config & 0x03FF`；覆盖 0 表示必须为空、边界等于上限接受、超过上限拒绝。
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

使用 `pkg/hashtree`，空树、单叶、奇数叶按 ADR-0013 的已决策略实现。

**Step 3: 验证并提交**

```bash
go test ./internal/tx -run 'TestHashOutputs' -v
git add internal/tx/output_hash.go internal/tx/output_hash_test.go
git commit -m "feat: add transaction output hashing"
```

## Task 8: 签名消息

**Files:**
- Create: `internal/tx/signature_message.go`
- Create: `internal/tx/witness.go`
- Create: `internal/tx/size.go`
- Create: `internal/tx/signature_message_test.go`
- Create: `internal/tx/witness_test.go`
- Create: `internal/tx/size_test.go`

**Step 1: 写失败测试**

测试：

- 默认 flag 为 `SIGIN_ALL | SIGOUT_ALL | SIGOUTPUT`。
- 非法 flag 组合拒绝。
- 签名消息包含链识别信息。
- 修改链身份、输入范围或输出范围会改变签名消息。
- 交易包含 Witness 字段和完整交易编码，但 `TxID` 编码必须排除 Witness。
- `MaxTxSize` 检查基于不含 Witness 的规范交易编码，包含 Header、Inputs、Outputs；Inputs 中的 UnlockScript 必须计入大小。
- 修改 Witness 签名字节不得改变 `MaxTxSize` 检查结果；修改 UnlockScript 长度必须影响 `MaxTxSize` 检查结果。
- 修改 Witness 签名字节不得改变 `TxID`，但应改变完整交易编码或签名附件摘要。
- 修改 UnlockScript 必须改变 `TxID`；签名消息中的输入范围覆盖 UnlockScript 但不覆盖 Witness。
- Witness 可为空且交易结构仍合法；是否因缺失 Witness 失败由脚本执行 `SYS_CHKPASS` 时判定。
- `SYS_CHKPASS` 标准验证签名由 Witness 传递给脚本环境；定制验证签名可位于 UnlockScript 并经 `FN_CHECKSIG` / `FN_MCHECKSIG` 验证。
- UnlockScript 内签名字节经 `FN_CHECKSIG` / `FN_MCHECKSIG` 验证时，签名消息自排除或占位规则仍为 OQ-011A；规则固定前不得生成最终签名测试向量。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'Test(SignatureMessage|Witness|TxSize)' -v
git add internal/tx/signature_message.go internal/tx/witness.go internal/tx/size.go internal/tx/signature_message_test.go internal/tx/witness_test.go internal/tx/size_test.go
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
- 不因输出项数量超过输入项数量 2 倍而拒绝普通交易；百日扩张比例检查只属于交易构造器或钱包的可选客户端策略（ADR-0029）。
- Coinbase 无输入。
- Coinbase 必须位于区块交易序列第 0 项。
- Coinbase 使用独立 parser，不套用普通输出 envelope 的低 4 位类型字段。
- Coinbase 输出只能是 Coin；出现 Credit、Proof、Mediator 或 Custom 输出必须拒绝。
- Coinbase `HashInputs` 测试覆盖高度 `0`、`1`、`math.MaxUint64`。
- Coinbase 已决字段必须规范编码；奖励分配余数按 ADR-0009 由 `stun2p` 吸收，金额单位按 ADR-0031 使用 `chx`。发行递减取整等剩余开放问题返回明确未实现错误或只编码已决字段。

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
- 输入规范编码必须包含 UnlockScript，且修改 UnlockScript 会改变 `TxID`；修改 Witness 不得改变 `TxID`。
- `MaxTxSize` 按不含 Witness 的规范交易编码检查，包含 UnlockScript；超过 65535 bytes 拒绝。
- Witness 为空的交易结构合法；仅在脚本实际执行 `SYS_CHKPASS` 且缺失对应 Witness 时失败。
- Proof、Mediator、Custom 默认不能作为公共输入源。
- 输出 envelope、payload 和签名消息均有表驱动测试。
- 交易包不 import `internal/utxo`、`internal/utco`、`internal/script`、`internal/consensus`。

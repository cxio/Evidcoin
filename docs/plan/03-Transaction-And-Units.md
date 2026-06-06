# Transaction And Units Implementation Plan

**Goal:** 实现交易头（普通/Coinbase）、输入模型、输出 envelope、Coin/Credit/Proof/Mediator/Custom payload、交易输入/输出哈希与 Coinbase 结构边界（签名消息与见证容器见第 04 章）。

**Architecture:** `internal/tx/` 只表达交易数据、规范化编码、Hash 和本地可判定的结构规则；状态可用性、脚本执行、PoH 资格和完整 Coinbase 奖励结算通过接口或后续层处理。交易包可依赖 `pkg/types`、`pkg/crypto`、`pkg/hashtree` 和 `internal/blockchain` 的链身份类型，但不能依赖状态、脚本或共识具体实现。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、`pkg/hashtree`、表驱动测试。

---

## 来源提案

- `docs/proposal/06.Transaction-Model.md`
- `docs/proposal/07.Coin-Credit-Proof-Units.md`
- `docs/proposal/14.Incentives-And-Coinbase.md` 的 Coinbase 头与交易费边界
- 依赖 `docs/proposal/01.Types-And-Encoding.md`、`04.Hash-Trees.md`
- DEC-0003：普通/Coinbase 交易头字段顺序与 `MintPKHash` 编码差异；Coinbase **无 `HashInputs` 字段**。
- DEC-0101：交易体编码、输入项编码、输出公共头（Config 字节、**无销毁位**）、三类 payload、可选字段 `varint(0)`、自定义类不进 UTXO/UTCO、重复输入非法、Credit 31 年过期边界。
- DEC-0401（引用）：Coinbase 只允许 Coin 输出、奖励余数与 `BurnCoin` 销毁，金额单位 `chx`（详见第 10 章）。
- 单位 C-8 已裁决：Coin `Amount` 以 `chx` 为协议单位（`1 Bi = 10^8 chx`，见第 01 章）。
- 签名消息与见证容器移至第 04 章（DEC-0102/0103/0104）。
- 百日扩张比例为客户端构造策略，核心交易验证不检查输入输出数量比例（proposal 05/14）。

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
| `internal/tx/input_hash.go` | `ListHash`、`LeadPKHash`、`HashInputs` |
| `internal/tx/size.go` | 不含见证的 `MaxTxSize` 检查（见证定义见第 04 章） |
| `internal/tx/output.go` | 输出 envelope、类型与标记 |
| `internal/tx/coin.go` | Coin payload |
| `internal/tx/credit.go` | Credit payload 与配置 |
| `internal/tx/proof.go` | Proof payload |
| `internal/tx/attachment.go` | `AttachmentID`、片组树引用 |
| `internal/tx/mediator.go` | Mediator payload |
| `internal/tx/custom.go` | Custom payload 边界 |
| `internal/tx/coinbase.go` | Coinbase 结构骨架和位置规则 |
| `internal/tx/validate.go` | 本地结构验证 |
| `internal/tx/errors.go` | 错误定义 |

## Task 1: 交易头与 TxID

**Files:**
- Create: `internal/tx/header.go`
- Create: `internal/tx/header_test.go`

**Step 1: 写失败测试**

测试：

- 普通 `TxHeader` 字段顺序为 `Version(uint16), HashInputs[32], HashOutputs[32], Timestamp(int64), MintPKHash`。
- `MintPKHash` 为可选 `varint(len) || bytes`，`len∈{0,32}`，`len==0` 表未设置，其它 len 非法（DEC-0003）。
- 相同交易头得到相同 `TxID`；修改任一字段会改变 `TxID`；`TxID` 输出 48B，域标签 `tx.header` + SHA3-384。

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

- `LeadInput` 必须标记为 Coin 输入。
- `LeadInput` 和 `RestInput` 均包含 `UnlockScript`。
- 定制验证签名字节可位于 `UnlockScript`，并作为普通输入字节参与规范编码。
- 输入项使用 `TxIDPart`，长度 `>=16`。
- Proof 输入类型被结构验证拒绝。
- `HashInputs = BLAKE3-256(ListHash || LeadPKHash)`。
- 修改任一输入的 `UnlockScript` 会改变 `ListHash`，继而改变 `HashInputs` 和 `TxID`。
- List inputs 顺序变化导致 `HashInputs` 变化。

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

测试（DEC-0101，Config 字节）：

- `bit7` 自定义类：置位时余下低 7 位为类 ID 长度计数（≤127）。
- `bit6` 包含附件（若 `bit7` 置位则此位变义）。
- `bit[3:0]` 类型值：`0=预留 / 1=币金 / 2=凭信 / 3=存证`；介管脚本属存证（类型 3）。
- **无销毁位**：普通交易不可销毁币金（销毁仅由 Coinbase `BurnCoin` 表达）；以空 `Receiver` 表达销毁的交易必须拒绝。
- 自定义类（`bit7=1`）输出按公共头编码参与区块哈希，但**不进入 UTXO/UTCO**，不能作为后续输入源。
- 输出项按创建者给定顺序编码，位置下标即序位；未知类型值（预留 0 等非法位置）拒绝。

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

测试（DEC-0101 / proposal 07 §2，payload 字段顺序）：

- Coin payload 编码为 `Amount(varint) || Receiver(varint(len)||bytes, len<256) || Memo(varint(len)||bytes, len<256)`。
- `Amount` 以 `chx` 为最小单位，`varint` 编码；不得用小数 Bi 参与协议编码（C-8）。
- `Receiver` 为接收者公钥哈希；若脚本自定义验证（不用 `SYS_CHKPASS`），`Receiver` 可空或任意 <256B 字节序列。
- `Memo` 可选，最多 255 字节；缺省以 `varint(0)` 表示并参与前像。
- `LockScript` 属输出公共头（Task 3），超过 `MaxLockScript` 拒绝；Coin payload 不含 AttachmentID。

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

测试（proposal 07 §2/§5）：

- Credit payload 编码为 `Receiver(<256) || Creator(<256) || Title(<256) || Description(≤2KB) || AttachmentID(optional)`，均 `varint(len)||bytes`。
- `Description` 最多 2KB，超过拒绝；`Title`/`Creator`/`Receiver` 均 <256B。
- `AttachmentID` 可选，缺省 `varint(0)` 并参与前像；结构见 Task 6。
- 每交易凭信输出 ≤2：第 3 笔 Credit 输出协议拒绝；第 2 笔触发交易费加倍规则（≥ 前一区块平均交易费 2 倍，共约，结算见状态/共识层）。
- 31 年过期边界：`age > 31 × 87661` 失效；`age == 31 × 87661` 仍可被引用花销（过期移除归第 05 章 UTCO）。

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

测试（proposal 07 §2/§4/§6）：

- Proof payload 编码为 `Creator(<256) || Title(<256) || Content(≤2KB) || AttachmentID(optional)`；无接收者字段，不可作为输入。
- AttachmentID 结构：`类型(2B) || 指纹(64B SHA3-512) || 分片数(2B) || 片组哈希(32B BLAKE3-256) || 大小(varint)`；编码长度由外层 varint(length) 表达，字节数须 <256；分片数 `0` 时片组哈希字段不编码，`1` 时计算但无树，`>1` 时为含序片组树根。
- 附件指纹 64B 用 `attachment.fingerprint` 域 + SHA3-512；片组哈希 32B 走免域标签 BLAKE3-256 路径（第 01 章 hashtree）。
- Mediator（介管脚本）属存证类（类型 3），不可作为输入项，由脚本 `GOTO/EMBED` 引用。
- Custom（`bit7=1`，≤127B 私有 ID）不进入 UTXO/UTCO，不能作为公共输入源；节点仅校验编码合法性。

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
- 单输出时，`HashOutputs` 等于单叶树根按 tree.branch profile 一元归一化后的 32B 根。
- 空输出普通交易拒绝。

**Step 2: 实现**

使用 `pkg/hashtree` 计算输出树根。`HashOutputs = Hash256:Tree<Outputs>` 中的 `Hash256` 仅表示采用 256 位树根 profile；空树、单叶、奇数层按 DEC-0004 的已决策略实现：单叶根为 `BLAKE3-256(DomainTag("tree.branch") || leafHash)`，不复制叶子，不构造 `leafHash || leafHash`；奇数层最后节点直接提升不复制。

**Step 3: 验证并提交**

```bash
go test ./internal/tx -run 'TestHashOutputs' -v
git add internal/tx/output_hash.go internal/tx/output_hash_test.go
git commit -m "feat: add transaction output hashing"
```

## Task 8: 交易尺寸检查（不含见证）

**Files:**
- Create: `internal/tx/size.go`
- Create: `internal/tx/size_test.go`

**Step 1: 写失败测试**

测试（proposal 06 §7）：

- `MaxTxSize = 65535` 基于不含见证的规范交易编码（Header + Inputs + Outputs），其中 Inputs 的 `UnlockScript` 必须计入大小。
- 修改 `UnlockScript` 长度影响尺寸检查结果。
- 超过 65535 字节拒绝。
- 见证容器不计入交易尺寸（见证定义与剪枝见第 04 章）。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestTxSize' -v
git add internal/tx/size.go internal/tx/size_test.go
git commit -m "feat: add transaction size limit"
```

## Task 9: Coinbase 骨架与交易费接口

**Files:**
- Create: `internal/tx/fees.go`
- Create: `internal/tx/coinbase.go`
- Create: `internal/tx/fees_test.go`
- Create: `internal/tx/coinbase_test.go`

**Step 1: 写失败测试**

测试：

- 普通交易费 = Coin 输入总额 - Coin 输出总额；输出总额大于输入总额拒绝。
- 不因输出项数量超过输入项数量 2 倍而拒绝普通交易；百日扩张比例检查仅属客户端构造策略（proposal 05/14）。
- Coinbase 头字段顺序（DEC-0003/DEC-0401）：`Version(uint16) || HashOutputs[32] || Timestamp(int64) || MintPKHash[32]定长 || BlockHeight(uint32) || Minter(MintProof) || FreeData(varint(len)||bytes, len≤255) || BurnCoin(int64) || AwardSlots[18]byte`；`AwardSlots` 对所有 Coinbase（含创世）始终存在，创世与百日前其值恒为全零；**无 `HashInputs` 字段**。
- `Minter`（择优凭证）**当且仅当 `BlockHeight == 0`（创世）时省略**，无额外 presence 标识（结构见第 07 章 PoH）。
- Coinbase 必须位于区块交易序列第 0 项；使用独立 parser，TxID 用 `tx.header` 域但前像为 Coinbase 字段集。
- Coinbase 输出只能是 Coin；出现 Credit/Proof/Mediator/Custom 输出必须拒绝（DEC-0401）。
- 奖励分配、`BurnCoin` 销毁与兑奖槽（`AwardSlots`）金额单位 `chx`，结算逻辑放第 10 章；本层只固定 Coinbase 结构编码与位置规则。

**Step 2: 实现**

交易包只定义 Coinbase 结构边界和位置验证。奖励计算放到 `10-Incentives-And-Coinbase.md` 对应任务。

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
- 输入规范编码必须包含 UnlockScript，且修改 UnlockScript 会改变 `TxID`（见证不参与 TxID，见第 04 章）。
- `MaxTxSize` 按不含见证的规范交易编码检查，包含 UnlockScript；超过 65535 字节拒绝。
- Coinbase 无 `HashInputs`、位于第 0 项、仅 Coin 输出；`Minter` 仅创世省略。
- Proof、Mediator、Custom 默认不能作为公共输入源。
- 三类 payload、输出 envelope、附件 ID 结构均有表驱动测试。
- 交易包不 import `internal/utxo`、`internal/utco`、`internal/script`、`internal/consensus`。

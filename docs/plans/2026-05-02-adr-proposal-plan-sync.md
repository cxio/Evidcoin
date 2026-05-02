# ADR Proposal/Plan Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `docs/adr/ADR-0001` 至 `ADR-0031` 的已决架构结论全量、精准同步到相关 Proposal 与 Plan 文档。

**Architecture:** ADR 是最高优先级决策源；Proposal 固化协议/技术规格，Plan 固化实现任务、边界测试和验收要求。采用定点编辑，不重构现有文档体系，不修改 `docs/conception/`。

**Tech Stack:** Markdown 文档、ripgrep/grep 文本核验、Git 分步提交。

---

## 执行总则

- 每个任务只修改列出的文件。
- 每次修改后运行对应 `rg` 检查，确认不再保留已被 ADR 关闭的“未决/待确认”冲突表述。
- 新增或修订的规则尽量写明对应 ADR 编号，例如“由 ADR-0003 固定”。
- 不改 `docs/conception/`。
- 若发现 ADR 原文与本计划摘要不一致，以 ADR 原文为准，并暂停请求复核。

---

### Task 1: 同步基础编码、Domain Tag、地址与金额单位

**Files:**
- Modify: `docs/proposal/01.Types-And-Encoding.md`
- Modify: `docs/proposal/02.Cryptography-And-Hashing.md`
- Modify: `docs/proposal/03.Identifiers-And-Constants.md`
- Modify: `docs/proposal/Instruction/0.Base-Constraints.md`
- Modify: `docs/proposal/Instruction/16.Function-Instructions.md`
- Modify: `docs/plan/01-Foundation-Types-Crypto.md`

**Step 1: 读取 ADR 原文**

Read:
- `docs/adr/ADR-0003-varint-encoding.md`
- `docs/adr/ADR-0004-domain-tag-format.md`
- `docs/adr/ADR-0014-maxstackheight-boundary.md`
- `docs/adr/ADR-0018-ml-dsa-65-integration.md`
- `docs/adr/ADR-0020-address-text-encoding.md`
- `docs/adr/ADR-0030-genesis-year-boundary.md`
- `docs/adr/ADR-0031-coin-chx-ratio.md`

**Step 2: 更新 `01.Types-And-Encoding.md`**

- 将 `canonical unsigned varint` 明确为 unsigned LEB128 / Protocol Buffers varint。
- 写明必须拒绝非最短编码，`uint64` 最大 10 字节。
- 加入示例：
  - `0 -> 0x00`
  - `127 -> 0x7F`
  - `128 -> 0x80 0x01`
  - `16383 -> 0xFF 0x7F`
  - `16384 -> 0x80 0x80 0x01`
- 删除“varint 算法未固定/不得假定”的表述。

**Step 3: 更新 `02.Cryptography-And-Hashing.md`**

- 增加 Domain Tag 格式：`"Evidcoin:" + Purpose + ":v" + version_number + "\\x00"`，其中 `version_number` 使用 ADR-0003 的 LEB128，当前为 `0x01`。
- 增加或同步 Purpose 表：`BlockHeader`、`TxHeader`、`CheckRoot`、`TreeBranch`、`TreeLeaf`、`StateLeaf`、`StateData`、`Attachment`、`AddressHash`、`MintInner`、`MintHash`、`CoinbaseInputs`、`SignMessage`、`SyncPool`、`ForkDecision`。
- 在 Hash 算法表中补充 `MintInner = BLAKE3-256, 32B`。
- 补充 ML-DSA-65 实现选择优先级：Go 标准库优先，其次 `github.com/cloudflare/circl`，备选 `filippo.io/mldsa`；上层只依赖 `pkg/crypto` 接口。
- 删除“domain tag 字节表未固定”“varint 算法未确认”等冲突表述。

**Step 4: 更新 `03.Identifiers-And-Constants.md`**

- 将脚本边界常量改为：
  - `MaxStackHeight = 255`
  - `MaxStackItem = 1023`
  - `MaxLockScript = 1023`
  - `MaxUnlockScript = 4095`
  - 拒绝条件统一为 `> 常量值`。
- 固定地址文本编码：`"Cx" + Base58Check(AddressHash)`；checksum 为 `SHA3-256(SHA3-256(AddressHash))[0:4]`。
- 增加年度公式：`Year(blockHeight) = blockHeight / BlocksPerYear`；`Year 0 = [0, 87660]`，`Year 1 = [87661, 175321]`。
- 增加金额常量：`ChxPerCoin = 100_000_000`，协议层金额为 `uint64 chx`。

**Step 5: 更新指令相关 Proposal**

- `Instruction/0.Base-Constraints.md`：删除 varint/domain tag/初始 pass 状态仍未固定的冲突表述，引用对应 ADR。
- `Instruction/16.Function-Instructions.md`：将 Address 函数依赖的地址文本编码固定为 ADR-0020 规则。

**Step 6: 更新 `plan/01-Foundation-Types-Crypto.md`**

- Task 3 改为实现 LEB128 unsigned varint，并测试非最短编码拒绝。
- Task 4 改为实现固定 Domain Tag，不允许调用方传入协议 tag。
- 增加 `EncodeAddress` / `DecodeAddress` 任务与 checksum 失败测试。
- 增加 `type Amount uint64`、`ChxPerCoin`、显示层转换辅助函数测试。
- 将 `IsYearBoundary(0)` 预期固定为 `true`，增加 `YearOfHeight(0)=0`、`YearOfHeight(87660)=0`、`YearOfHeight(87661)=1`。
- ML-DSA-65 Task 增加库选择优先级和用户确认步骤。

**Step 7: 验证**

Run:

```bash
rg -n "varint.*未|domain tag.*未|地址文本.*未|MaxStackHeight = 256|MaxStackItem = 1024|MaxLockScript = 1024|MaxUnlockScript = 4096|Coin 与 chx.*未" docs/proposal docs/plan
```

Expected: 不再出现本任务涉及的冲突表述；若出现，只能是历史说明并明确标记“已由 ADR-xxxx 关闭”。

**Step 8: Commit**

```bash
git add docs/proposal/01.Types-And-Encoding.md docs/proposal/02.Cryptography-And-Hashing.md docs/proposal/03.Identifiers-And-Constants.md docs/proposal/Instruction/0.Base-Constraints.md docs/proposal/Instruction/16.Function-Instructions.md docs/plan/01-Foundation-Types-Crypto.md
git commit -m "docs: sync foundation specs with ADRs"
```

---

### Task 2: 同步哈希树、区块核心与状态时间点

**Files:**
- Modify: `docs/proposal/04.Hash-Trees.md`
- Modify: `docs/proposal/05.Blockchain-Core.md`
- Modify: `docs/proposal/08.UTXO-UTCO-State.md`
- Modify: `docs/proposal/12.Team-Validation.md`
- Modify: `docs/plan/01-Foundation-Types-Crypto.md`
- Modify: `docs/plan/02-Blockchain-Core.md`
- Modify: `docs/plan/04-UTXO-UTCO-State.md`

**Step 1: 读取 ADR 原文**

Read:
- `docs/adr/ADR-0008-utxo-txid-byte-index.md`
- `docs/adr/ADR-0013-hash-tree-edge-cases.md`
- `docs/adr/ADR-0015-intra-block-chained-spending.md`
- `docs/adr/ADR-0022-checkroot-utxo-state-timing.md`

**Step 2: 更新 `04.Hash-Trees.md`**

- 固定空树根为对应 Hash 长度全零值。
- 固定单叶树根等于叶子 Hash。
- 固定奇数叶复制最后一个叶子后配对计算。
- UTXO/UTCO 指纹树 `[8,13,18]` 明确为 0-based 字节索引。
- 删除“空树/单叶/奇数叶/domain tag 未固定”的冲突表述。

**Step 3: 更新 `05.Blockchain-Core.md`**

- 将 `CheckRoot[H]` 的 `UTXORoot/UTCORoot` 定义为高度 `H-1` 执行后的前置状态指纹。
- 创世块使用空状态指纹。
- 删除“倾向区块后状态/状态时间点未决”的冲突表述。
- 保留区块执行后产生的新状态 root，说明它供 `CheckRoot[H+1]` 使用。

**Step 4: 更新 `08.UTXO-UTCO-State.md`**

- 在状态转移流程中区分：验证当前区块使用前置状态 root；执行后生成下一高度前置 root。
- 明确禁止同一区块内交易消费本区块前序交易输出。
- 明确输入只能引用已确认历史区块的 UTXO/UTCO。
- 指纹树 TxID 字节索引写为 `TxID[8]`、`TxID[13]`、`TxID[18]`，均为 0-based。

**Step 5: 更新 `12.Team-Validation.md`**

- 在组队校验/管理层返回数据处说明 UTXO/UTCO 指纹是前置状态指纹。
- 补充禁止同区块链式消费可支持交易并行验证。

**Step 6: 更新 Plan**

- `plan/01-Foundation-Types-Crypto.md`：哈希树任务不再使用“未指定策略返回错误”作为协议默认；测试空树、单叶、奇数叶复制。
- `plan/02-Blockchain-Core.md`：CheckRoot 任务增加 `h == 0` 空状态指纹、普通区块读取上一高度状态指纹测试。
- `plan/04-UTXO-UTCO-State.md`：增加 TxID 0-based 路由测试；增加同区块 A 输出被 B 输入引用时拒绝的测试。

**Step 7: 验证**

Run:

```bash
rg -n "空树.*未|单叶.*未|奇数叶.*未|区块后状态|状态时间点.*未|字节索引是否|同区块.*链式" docs/proposal docs/plan
```

Expected: 冲突表述已移除或改为 ADR-0013/0008/0015/0022 的已决规则。

**Step 8: Commit**

```bash
git add docs/proposal/04.Hash-Trees.md docs/proposal/05.Blockchain-Core.md docs/proposal/08.UTXO-UTCO-State.md docs/proposal/12.Team-Validation.md docs/plan/01-Foundation-Types-Crypto.md docs/plan/02-Blockchain-Core.md docs/plan/04-UTXO-UTCO-State.md
git commit -m "docs: sync hash tree and state timing ADRs"
```

---

### Task 3: 同步交易模型、Coin/Credit/Proof、Coinbase 与 Witness

**Files:**
- Modify: `docs/proposal/06.Transaction-Model.md`
- Modify: `docs/proposal/07.Coin-Credit-Proof-Units.md`
- Modify: `docs/proposal/09.Script-System.md`
- Modify: `docs/proposal/Instruction/15.System-Instructions.md`
- Modify: `docs/plan/03-Transaction-And-Units.md`
- Modify: `docs/plan/04-UTXO-UTCO-State.md`
- Modify: `docs/plan/05-Script-System.md`

**Step 1: 读取 ADR 原文**

Read:
- `docs/adr/ADR-0010-coinbase-hashinputs.md`
- `docs/adr/ADR-0016-txidpart-collision.md`
- `docs/adr/ADR-0021-credit-config-description-length.md`
- `docs/adr/ADR-0023-coinbase-independent-processing.md`
- `docs/adr/ADR-0024-signature-witness-separation.md`
- `docs/adr/ADR-0028-announcement-authority-pubkey.md`
- `docs/adr/ADR-0031-coin-chx-ratio.md`

**Step 2: 更新 `06.Transaction-Model.md`**

- Coinbase HashInputs 固定为 `BLAKE3-256(DomainTag(CoinbaseInputs) || bigEndianUint64(blockHeight))`。
- 增加 Witness / Signature Attachment 小节：Witness 不参与 TxID；`UnlockScript` 不含 ML-DSA 签名字节。
- 将“解锁数据提供签名”改为“Witness 提供签名，UnlockScript 提供控制逻辑/公钥/辅助数据”。
- 增加 Coinbase 独立格式说明：Coinbase 输出只能是 Coin；不使用普通输出 envelope 的低 4 位类型字段。
- 将 TxIDPart 碰撞处理固定为歧义拒绝，不扩展长度，不预留升级路径。

**Step 3: 更新 `07.Coin-Credit-Proof-Units.md`**

- `CoinOutput.Amount` 明确为 `chx` 单位的 `uint64`。
- `Credit.Config[9:0]` 明确为描述字段最大允许字节长度；验证 `len(description) <= Config[9:0]`。
- 明确 `Config[9:0] = 0` 表示描述必须为空，`1023` 表示最多 1023 字节。
- 增加 Announcement Proof 约定：`Title = "Announcement:<Level>: <title>"`；内容放在 Proof Body / Attachment。

**Step 4: 更新脚本文档**

- `09.Script-System.md`：解锁脚本不承载签名字节；签名由 Witness 注入系统环境。
- `Instruction/15.System-Instructions.md`：`SYS_CHKPASS` 从系统环境读取签名，不从普通栈或 UnlockScript 字节流读取。

**Step 5: 更新 Plan**

- `plan/03-Transaction-And-Units.md`：
  - 增加 Witness 字段、编码和 TxID 排除测试。
  - Coinbase Task 增加独立 parser、只允许 Coin 输出、拒绝 Credit/Proof/Mediator 输出测试。
  - Coinbase HashInputs 测试高度 `0`、`1`、`math.MaxUint64`。
  - Credit 验证增加 `len(description) <= config & 0x03FF` 测试。
  - Coin payload 测试明确 `Amount` 单位为 chx。
- `plan/04-UTXO-UTCO-State.md`：TxIDPart 歧义拒绝保持，并补充概率说明引用 ADR-0016。
- `plan/05-Script-System.md`：`SYS_CHKPASS` 测试补充签名只能来自环境。

**Step 6: 验证**

Run:

```bash
rg -n "Coinbase.*低 4 位|HashInputs.*特殊值.*必须|解锁脚本.*签名数据|TxIDPart.*未决|Description.*未决|Witness.*TxID" docs/proposal docs/plan
```

Expected: Coinbase、Witness、TxIDPart、Credit 描述长度不再保留冲突或未决表述。

**Step 7: Commit**

```bash
git add docs/proposal/06.Transaction-Model.md docs/proposal/07.Coin-Credit-Proof-Units.md docs/proposal/09.Script-System.md docs/proposal/Instruction/15.System-Instructions.md docs/plan/03-Transaction-And-Units.md docs/plan/04-UTXO-UTCO-State.md docs/plan/05-Script-System.md
git commit -m "docs: sync transaction and witness ADRs"
```

---

### Task 4: 同步脚本 VM 运行语义与边界

**Files:**
- Modify: `docs/proposal/09.Script-System.md`
- Modify: `docs/proposal/Instruction/1.Value-Instructions.md`
- Modify: `docs/proposal/Instruction/6.Result-Instructions.md`
- Modify: `docs/proposal/Instruction/8.Conversion-Instructions.md`
- Modify: `docs/proposal/Instruction/9.Arithmetic-Instructions.md`
- Modify: `docs/proposal/Instruction/10.Comparison-Instructions.md`
- Modify: `docs/proposal/Instruction/15.System-Instructions.md`
- Modify: `docs/plan/05-Script-System.md`

**Step 1: 读取 ADR 原文**

Read:
- `docs/adr/ADR-0005-float-determinism.md`
- `docs/adr/ADR-0006-script-initial-pass-state.md`
- `docs/adr/ADR-0007-check-override-semantics.md`
- `docs/adr/ADR-0014-maxstackheight-boundary.md`
- `docs/adr/ADR-0019-sys-null-unlock-exception.md`

**Step 2: 更新 `09.Script-System.md`**

- 初始 pass 状态固定为 `true`；空脚本通过；无 `CHECK`/`PASS` 的脚本默认通过。
- `CHECK(x)` 可双向覆盖 pass 状态，最终执行到 `END` 或等效终止点时以当前 pass 状态为最终结果。
- Float 使用 IEEE 754 binary64 / Go `float64`，round-to-nearest-even；NaN 统一规范化为 quiet NaN `0x7FF8000000000000`；Infinity 保留。
- 公共验证路径中 Float 不得直接作为 `CHECK`/`PASS` 参数。
- 更新脚本资源常量为 ADR-0014 数值。
- 解锁脚本白名单固定为 `op <= 50 || op == SYS_NULL`；`SYS_NULL` 不执行计算、不访问状态、不影响栈。

**Step 3: 更新 Instruction 文档**

- `1.Value-Instructions.md`：Float 字面量入栈前必须规范化 NaN。
- `6.Result-Instructions.md`：`CHECK(x)` 等价于 `passState = bool(x)`，不做防覆盖保护。
- `8.Conversion-Instructions.md`：补充 NaN/Inf 转换规则，公共验证中 Float 到布尔结果不能直接用于 `CHECK`/`PASS`。
- `9.Arithmetic-Instructions.md`：补充 binary64、舍入和 NaN 规范化。
- `10.Comparison-Instructions.md`：补充 NaN 比较和 `ISNAN` 规则。
- `15.System-Instructions.md`：补充 `SYS_NULL` 解锁段例外和安全边界。

**Step 4: 更新 `plan/05-Script-System.md`**

- 增加测试：空脚本通过；无结果指令脚本通过；仅 `CHECK(false)` 拒绝。
- 增加测试：`CHECK(true) -> CHECK(false)` 最终拒绝；`CHECK(false) -> CHECK(true)` 最终通过。
- 增加 Float NaN 规范化、Infinity 保留、Float-as-CHECK-arg 拒绝测试。
- 边界测试改为：255 合法、256 拒绝；1023 合法、1024 拒绝；4095 合法、4096 拒绝。
- 增加 unlock script 测试：`SYS_NULL` 合法，其他系统指令如 `SYS_TIME` 非法。

**Step 5: 验证**

Run:

```bash
rg -n "初始 pass.*false|初始 pass.*未|CHECK.*未|Float.*未决|NaN/Inf.*未|MaxStackHeight = 256|SYS_NULL.*未" docs/proposal/09.Script-System.md docs/proposal/Instruction docs/plan/05-Script-System.md
```

Expected: 脚本初始状态、CHECK、Float、资源边界和 SYS_NULL 均已改为 ADR 已决规则。

**Step 6: Commit**

```bash
git add docs/proposal/09.Script-System.md docs/proposal/Instruction/1.Value-Instructions.md docs/proposal/Instruction/6.Result-Instructions.md docs/proposal/Instruction/8.Conversion-Instructions.md docs/proposal/Instruction/9.Arithmetic-Instructions.md docs/proposal/Instruction/10.Comparison-Instructions.md docs/proposal/Instruction/15.System-Instructions.md docs/plan/05-Script-System.md
git commit -m "docs: sync script VM ADRs"
```

---

### Task 5: 同步 PoH 共识、MintHash、分叉选择与客户端策略

**Files:**
- Modify: `docs/proposal/10.PoH-Consensus.md`
- Modify: `docs/proposal/11.Endpoint-Conventions-And-Fork-Choice.md`
- Modify: `docs/proposal/14.Incentives-And-Coinbase-Rewards.md`
- Modify: `docs/plan/03-Transaction-And-Units.md`
- Modify: `docs/plan/06-PoH-Consensus-And-Fork-Choice.md`
- Modify: `docs/plan/07-Team-Validation-Services-Incentives.md`

**Step 1: 读取 ADR 原文**

Read:
- `docs/adr/ADR-0001-poh-inner-hash-algorithm.md`
- `docs/adr/ADR-0002-poh-x-encoding.md`
- `docs/adr/ADR-0011-minthash-tie-breaker.md`
- `docs/adr/ADR-0025-minttxid-eligibility-by-block-height.md`
- `docs/adr/ADR-0026-leader-blacklist-convention.md`
- `docs/adr/ADR-0027-same-height-minter-tiebreaker.md`
- `docs/adr/ADR-0029-expansion-client-convention.md`
- `docs/adr/ADR-0031-coin-chx-ratio.md`

**Step 2: 更新 `10.PoH-Consensus.md`**

- `hashData = BLAKE3-256(mintTxID || referenceBlockMintHash || X)`。
- `MintHash = BLAKE3-512-XOF(Sign(hashData), 64)`。
- `timeStamp` 使用 Unix epoch 毫秒数，按 `GenesisTimestampMs + height * 360000` 推导。
- `X_BigInt = timeStamp * Stakes * Mix`，使用无截断 BigInt。
- `X = BigIntToMinimalBigEndianBytes(X_BigInt)`，零值为 `0x00`。
- 相同 `MintHash` 候选入择优池时只保留先到者。
- `mintTxID` 资格窗口按 `blockHeight(mintTxID)` 判定，交易自身 `Timestamp` 不参与。
- 将“百日扩张”标为客户端运行策略，不是协议级共识规则。
- 将 `聪时` 术语改为 `chx时` 或 `chx-time`，并引用 ADR-0031。

**Step 3: 更新 `11.Endpoint-Conventions-And-Fork-Choice.md`**

- 增加首领黑名单冻结为本地共约：影响入池/传播，不影响区块合法性。
- 增加分叉平局规则：比较分叉点后第一个区块 `BlockID`，字典序更小者胜出。
- 保留同一高度、同一铸造者多签多个区块时“交易费总和更低者优先”。

**Step 4: 更新 `14.Incentives-And-Coinbase-Rewards.md` 中百日扩张位置**

- 将百日扩张从奖励/结算未决项移出。
- 说明其为客户端/钱包/交易构造器建议，不进入区块验证或奖励计算。

**Step 5: 更新 Plan**

- `plan/06-PoH-Consensus-And-Fork-Choice.md`：
  - 删除 `ErrTieBreakerUnspecified` 作为 ADR 已决场景的要求。
  - 增加 BigInt X 编码测试向量。
  - 增加同 `MintHash` 先到者保留、后到者拒绝测试。
  - 增加交易 Timestamp 变化不影响 mintTxID 资格判定测试。
  - 增加 BlockID 字典序分叉平局测试。
- `plan/03-Transaction-And-Units.md`：若提及百日扩张，明确仅为交易构造器/钱包可选检查，核心交易验证不检查输入输出比例。
- `plan/07-Team-Validation-Services-Incentives.md`：若提及首领黑名单/百日扩张，标为本地策略或客户端策略。

**Step 6: 验证**

Run:

```bash
rg -n "内层 Hash.*未|X 整数宽度.*未|MintHash.*tie-breaker.*未|ErrTieBreakerUnspecified|交易自身 Timestamp|百日扩张.*未|协议规则.*百日扩张|聪时" docs/proposal docs/plan
```

Expected: PoH、分叉选择、客户端策略相关未决项已由 ADR 规则替换。

**Step 7: Commit**

```bash
git add docs/proposal/10.PoH-Consensus.md docs/proposal/11.Endpoint-Conventions-And-Fork-Choice.md docs/proposal/14.Incentives-And-Coinbase-Rewards.md docs/plan/03-Transaction-And-Units.md docs/plan/06-PoH-Consensus-And-Fork-Choice.md docs/plan/07-Team-Validation-Services-Incentives.md
git commit -m "docs: sync PoH and fork-choice ADRs"
```

---

### Task 6: 同步团队校验、公共服务、全网通告与激励

**Files:**
- Modify: `docs/proposal/07.Coin-Credit-Proof-Units.md`
- Modify: `docs/proposal/11.Endpoint-Conventions-And-Fork-Choice.md`
- Modify: `docs/proposal/12.Team-Validation.md`
- Modify: `docs/proposal/13.Public-Service-Interfaces.md`
- Modify: `docs/proposal/14.Incentives-And-Coinbase-Rewards.md`
- Modify: `docs/plan/07-Team-Validation-Services-Incentives.md`

**Step 1: 读取 ADR 原文**

Read:
- `docs/adr/ADR-0009-reward-rounding.md`
- `docs/adr/ADR-0012-block-signature-scope.md`
- `docs/adr/ADR-0017-public-service-evaluation.md`
- `docs/adr/ADR-0026-leader-blacklist-convention.md`
- `docs/adr/ADR-0028-announcement-authority-pubkey.md`
- `docs/adr/ADR-0031-coin-chx-ratio.md`

**Step 2: 更新 `12.Team-Validation.md`**

- 铸造者签名消息固定为 `domainTag || chainIdentityBytes || CheckRoot`。
- 明确签名不进入 `BlockID`。
- 首领黑名单冻结标注为 Local Convention / Node Policy，不作为区块合法性规则。

**Step 3: 更新 `13.Public-Service-Interfaces.md`**

- 增加客户端/公共服务检索全网通告的接口边界。
- 明确通告作为普通 Proof 交易进入链；可信展示是客户端层授权过滤，不新增共识规则。
- 公共服务质量评估保持主观确认，不引入链上客观质量证明。

**Step 4: 更新 `14.Incentives-And-Coinbase-Rewards.md`**

- 奖励分配使用整数顺序公式：
  - `validation = R * 40 / 100`
  - `minter = R * 10 / 100`
  - `depots = R * 20 / 100`
  - `blockqs = R * 20 / 100`
  - `stun2p = R - validation - minter - depots - blockqs`
- 交易费公式：
  - `recovered = TxFee * 50 / 100`
  - `destroyed = TxFee - recovered`
- 明确禁止浮点中间计算。
- 将发行示例补充 chx 等价值：
  - `10 Coin/block = 1,000,000,000 chx/block`
  - `20 Coin/block = 2,000,000,000 chx/block`
  - `30 Coin/block = 3,000,000,000 chx/block`
  - `40 Coin/block = 4,000,000,000 chx/block`
  - `3 Coin/block = 300,000,000 chx/block`

**Step 5: 更新全网通告相关 Proposal**

- `07.Coin-Credit-Proof-Units.md`：Announcement Proof 标题格式与 Body/Attachment 规则。
- `11.Endpoint-Conventions-And-Fork-Choice.md`：修正“通告不进入共识输入”为“不新增额外共识规则；作为普通 Proof 交易仍进入区块与交易树”。

**Step 6: 更新 `plan/07-Team-Validation-Services-Incentives.md`**

- 增加铸造者签名消息构造测试。
- 增加奖励余数归 `stun2p`、奇数交易费余数归 `destroyed` 测试。
- 增加授权公钥列表、轮换、撤销、权威级排序的客户端/服务层测试任务。
- 奖励计算断言统一使用 chx。

**Step 7: 验证**

Run:

```bash
rg -n "签完整区块头|只签 CheckRoot.*未|余数.*未|浮点.*奖励|服务评估.*全球可验证|通告不进入共识输入|Coin/block" docs/proposal docs/plan
```

Expected: 签名范围、奖励取整、公共服务评估、通告机制和 chx 单位均已同步 ADR。

**Step 8: Commit**

```bash
git add docs/proposal/07.Coin-Credit-Proof-Units.md docs/proposal/11.Endpoint-Conventions-And-Fork-Choice.md docs/proposal/12.Team-Validation.md docs/proposal/13.Public-Service-Interfaces.md docs/proposal/14.Incentives-And-Coinbase-Rewards.md docs/plan/07-Team-Validation-Services-Incentives.md
git commit -m "docs: sync validation service and incentive ADRs"
```

---

### Task 7: 关闭 Open Questions 并补充 ADR 追溯

**Files:**
- Modify: `docs/plan/08-Open-Questions-And-Acceptance.md`
- Modify: `docs/AGENTS.md`

**Step 1: 更新 Open Questions**

在 `docs/plan/08-Open-Questions-And-Acceptance.md` 中标记：

- `OQ-001`：ADR-0003 已关闭。
- `OQ-003`：ADR-0004 已关闭。
- `OQ-004`、`OQ-005`、`OQ-006`：ADR-0013 已关闭。
- `OQ-008` 或 CheckRoot 状态时间点相关项：ADR-0022 已关闭。
- `OQ-009`：ADR-0010 已关闭。
- `OQ-012` 字节索引部分：ADR-0008 已关闭；若还有年度分区之外的问题，只保留未被 ADR 覆盖的部分。
- `OQ-013`：初始 pass 状态由 ADR-0006 关闭，CHECK 覆盖由 ADR-0007 关闭。
- `OQ-015`：ADR-0020 已关闭。
- `OQ-016`：ADR-0001 已关闭。
- `OQ-017`：ADR-0002 已关闭。
- `OQ-018`：仅保留 `Stakes` 精确定义/单位问题，不再保留 X 编码问题。
- `OQ-019`：ADR-0011/ADR-0027 已关闭。
- `OQ-021`：仅保留发行递减取整规则；奖励分配余数不再未决。
- `OQ-022`、`OQ-023`：ADR-0009 已关闭。
- `OQ-026`：ADR-0026 已关闭。

**Step 2: 更新 `docs/AGENTS.md`**

- 在 ADR 章节补充说明：ADR 决策优先于 Proposal/Plan 中的旧未决表述。
- 增加“修订 Proposal/Plan 时应检查 ADR 是否已关闭对应 OQ”的维护规则。

**Step 3: 验证**

Run:

```bash
rg -n "OQ-00[1345689]|OQ-01[235679]|OQ-02[1236]|未决|待确认" docs/plan/08-Open-Questions-And-Acceptance.md docs/AGENTS.md
```

Expected: 被 ADR 关闭的 OQ 均明确标为已关闭；仍保留的未决项必须说明未被 ADR 覆盖的剩余问题。

**Step 4: Commit**

```bash
git add docs/plan/08-Open-Questions-And-Acceptance.md docs/AGENTS.md
git commit -m "docs: close ADR-resolved open questions"
```

---

### Task 8: 全局一致性核验

**Files:**
- Inspect only: `docs/adr/*.md`
- Inspect only: `docs/proposal/**/*.md`
- Inspect only: `docs/plan/*.md`
- Inspect only: `docs/plans/2026-05-02-adr-proposal-plan-sync-design.md`

**Step 1: 搜索已知冲突关键词**

Run:

```bash
rg -n "未固定|未确认|仍未|待确认|ErrTieBreakerUnspecified|MaxStackHeight = 256|MaxStackItem = 1024|MaxLockScript = 1024|MaxUnlockScript = 4096|区块后状态|签完整区块头|Coinbase 输出配置低 4 位|解锁脚本.*签名数据|聪时" docs/proposal docs/plan
```

Expected: 没有与 ADR-0001 至 ADR-0031 冲突的命中。允许存在未被 ADR 覆盖的真实剩余问题，但必须在上下文中明确不是已决 ADR 范围。

**Step 2: 核对 ADR 编号覆盖**

Run:

```bash
for n in $(seq -f "%04g" 1 31); do rg -n "ADR-$n" docs/proposal docs/plan docs/AGENTS.md >/dev/null || echo "missing ADR-$n"; done
```

Expected: 每个 ADR 至少在相关 Proposal 或 Plan 中有追溯引用；如某 ADR 已完全由原文覆盖且无需修改，需在 `08-Open-Questions-And-Acceptance.md` 或相关章节说明。

**Step 3: Markdown 基础检查**

Run:

```bash
rg -n "TODO|FIXME|TBD" docs/proposal docs/plan docs/plans
```

Expected: 不出现新增的占位符；已有占位符若与本任务无关，记录在最终汇报中。

**Step 4: 最终提交**

```bash
git status --short
git log --oneline -8
```

Expected: 工作区干净；最近提交包含本计划各任务提交。

**Step 5: 最终汇报**

汇报内容包括：

- 已修改的 Proposal 文件列表。
- 已修改的 Plan 文件列表。
- 已关闭的 OQ 列表。
- 仍保留的真实开放问题。
- 全局一致性核验命令及结果。

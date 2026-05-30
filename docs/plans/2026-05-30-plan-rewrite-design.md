# 实施方案重写设计（Plan Rewrite Design）

> 日期：2026-05-30
> 状态：待作者确认（确认后转入 `2026-05-30-plan-rewrite.md` 执行）

## 1. 背景与目标

`docs/proposal/` 已于 2026-05-29 依最新 `docs/AGENTS.md` 两层结构（conception + decision）整体重写为 **16 篇主文（00~15）+ Instruction 子目录**，决策层亦重写为 **21 个 `DEC-NNNN`（全部 Accepted）**，并裁定了若干旧待决项（如 C-8 单位体系 `1 Bi = 10^8 chx`、Coinbase 省略 `HashInputs`、全网通告取消）。

但现有 `docs/plan/`（8 篇）仍**停留在旧基线**：

- 引用旧版 proposal 编号（`00~14`，含已不存在的 `14.Incentives-And-Coinbase-Rewards.md`、`09.UTXO`、`Instruction 0-18` 等旧映射）。
- 全程使用旧 `ADR-xxxx` 决策命名（`ADR-0003/0004/0013/0030/0031` 等），与新 `DEC-NNNN` 命名空间脱节。
- 未反映 proposal 重写后的**重排与新增章节**：签名见证独立成章（08）、端点约定与分叉选择合并独立（12）、组队校验（13）、公共服务接口（15）。
- 仍引用旧的待决项口径（如「兑奖槽 bit 顺序未决」「Stakes 未定义」「单位未定」），与新 proposal 已裁决/已收敛的状态矛盾。

**目标**：以重写后的 `proposal/`（追溯自 `conception/` + `decision/`）为唯一依据，**整体重写 `docs/plan/`**，使其成为与 proposal 16 篇章节、DEC 编号空间、代码包、实施阶段**四方对齐**的、可被 Go 编码直接执行的阶段化实施计划。

## 2. 约束（已与作者确认）

1. **工作流**：先出本设计文档并经作者确认，再转入执行重写。
2. **章节结构**：重排为与新 proposal 更接近的分篇——签名见证、端点分叉、组队校验、公共服务各自独立成阶段（详见第 4 节）。
3. **内容深度**：保持现有 TDD 任务式深度——含包边界表、建议文件清单、API 草图、表驱动测试步骤（写失败测试→确认失败→最小实现→验证→可选提交）、阶段门禁与验收标准。

## 3. 不改动范围

`conception/`、`decision/`、`proposal/`、`plans/`（除本设计文档及后续执行文档）、所有 Go 代码。本任务仅产出/重写 `docs/plan/` 文档。

> 备注：实现层面仍**无生产代码**——本任务只重写「Proposal ⇒ Plan」层的阶段化计划文档。

## 4. 新章节划分（1 篇路线图 + 11 篇阶段 + 1 篇验收 = 13 篇）

与旧 8 篇相比，按用户选择将签名见证、端点分叉、组队校验、服务拆为独立阶段，实现与 proposal 16 篇近 1:1 对齐。

| 新文件 | 覆盖 proposal | 对应层/包 | 主要 DEC |
|--------|---------------|-----------|----------|
| `00-Implementation-Roadmap.md` | 00 | 全局/索引 | — |
| `01-Foundation-Types-Crypto.md` | 01·02·03·04 | `pkg/types`·`pkg/crypto`·`pkg/hashtree`（L0） | DEC-0001·0002·0004·0104 |
| `02-Blockchain-Core.md` | 05 | `internal/blockchain`（L1） | DEC-0003·0302 |
| `03-Transaction-And-Units.md` | 06·07 | `internal/tx`（L1） | DEC-0101·0003 |
| `04-Signatures-And-Witness.md` | 08 | `internal/tx`（L1） | DEC-0102·0103·0104 |
| `05-UTXO-UTCO-State.md` | 09 | `internal/utxo`·`internal/utco`（L2） | DEC-0201·0002 |
| `06-Script-System.md` | 10 + Instruction/ | `internal/script`（L3） | DEC-0501~0505 |
| `07-PoH-Consensus.md` | 11 | `internal/consensus`（L4） | DEC-0301·0302 |
| `08-Endpoint-And-Fork-Choice.md` | 12 | `internal/consensus`（L4） | DEC-0303 |
| `09-Team-Validation.md` | 13 | `internal/validation`（接口，L5） | DEC-0601 |
| `10-Incentives-And-Coinbase.md` | 14 | `internal/rewards`（L5） | DEC-0401 |
| `11-Public-Service-Interfaces.md` | 15 | `internal/services`（接口，L5） | DEC-0602·0603 |
| `12-Open-Questions-And-Acceptance.md` | 全部 | 全局 | C-6·C-7·C-9·C-10 |

**与旧 8 篇的关键调整：**

- 旧 `03-Transaction-And-Units`（混入 Coinbase 骨架）→ 拆分：交易模型与信元（03）/ 签名见证（04）/ 激励与 Coinbase（10）各自独立，消除一篇塞三层的过载。
- 旧 `06-PoH-Consensus-And-Fork-Choice` → 拆为 `07-PoH-Consensus`（铸凭/择优池/铸造者验证/创世初段窗口）与 `08-Endpoint-And-Fork-Choice`（出块时序/区块竞争归一化/RandomX 平局），与 proposal 11/12 对齐。
- 旧 `07-Team-Validation-Services-Incentives`（三合一）→ 拆为 `09-Team-Validation`、`10-Incentives-And-Coinbase`、`11-Public-Service-Interfaces` 三篇。
- 旧 `08-Open-Questions-And-Acceptance` → 编号顺延为 `12`，内容按新待决集（C-6/C-7/C-9/C-10）重写。

**包命名约定（L5 接口层）：** 采用 `internal/validation`（组队校验接口）、`internal/services`（公共服务接口）、`internal/rewards`（激励与 Coinbase 结算）、`cmd/evidcoin`、`test/`，与旧 `00-Implementation-Roadmap` 既有命名一致。

## 5. 单篇统一模板（沿用现有 plan 风格）

```
# <Title> Implementation Plan
**Goal:** —— 本阶段产出目标
**Architecture:** —— 包边界与依赖方向、追溯约束
**Tech Stack:** —— Go 版本与依赖

## 来源提案        —— 列出覆盖的 proposal 章（新编号）
## 包边界          —— 允许/禁止依赖表
## 建议文件        —— 文件清单与职责
## Task N: <名>    —— TDD 五步（失败测试→确认失败→最小实现→验证→可选提交）
## 阶段门禁/验收    —— 进入/退出条件、验收命令
```

**追溯标注规则（关键改动）：**

- 所有 proposal 引用改用**新编号与文件名**（如 `docs/proposal/08.Signatures-And-Witness.md`）。
- 所有决策引用从 `ADR-xxxx` **全量改写为 `DEC-NNNN`**，并按下表语义重映射（见第 6 节）。
- 每个 Task 的关键规格就近标注 proposal 章节或 DEC 编号。
- 待决项（C-6/C-7/C-9/C-10）对应的 Task 必须显式标注为**阻塞/占位**，不得在计划中默认选值固化。

## 6. ADR → DEC 重映射与待决项更新清单

### A. 决策命名重映射（旧 plan 措辞 → 新 DEC）

| 旧 plan 引用 | 主题 | 新依据 |
|--------------|------|--------|
| ADR-0003（canonical varint/LEB128） | 规范整数与字节编码 | **DEC-0001** |
| ADR-0004（hash domain tag） | 域标签与哈希 profile | **DEC-0002** |
| ADR-0013（哈希树空根/单叶/奇数叶） | 哈希树与证明边界 | **DEC-0004** |
| ADR-0020（地址文本编码 Encode/DecodeAddress） | 地址与 ML-DSA profile | **DEC-0104** |
| ADR-0030（年度公式 height/BlocksPerYear） | 年度=自然年/编码 | **DEC-0001** |
| ADR-0031（Coin/chx 金额单位） | 单位体系（已裁决 chx/Bi） | proposal 01·03（C-8 已裁决） |
| ADR-0001（PoH 内层 Hash=BLAKE3-256） | 铸凭哈希 | **DEC-0301** |
| ADR-0002（X 整数宽度/编码） | 铸凭前像 | **DEC-0301** |
| ADR-0009/0010（奖励余数/HashInputs） | Coinbase 序列化与奖励 | **DEC-0401**（HashInputs 省略已由 conception 明确） |
| ADR-0011/0027（入池规则/分叉平局） | 择优池/RandomX 平局 | **DEC-0301**/**DEC-0303** |

### B. 待决项更新（旧口径 → 新口径）

| 旧 plan 待决表述 | 新状态 | 处理 |
|------------------|--------|------|
| 「单位金额单位由 ADR-0031 关闭」 | 已裁决 `1 Bi = 10^8 chx`，`chx` 最小承载单位 | 改为「已定」，引 proposal 01·03，移出待决 |
| 「兑奖槽 bit 顺序属剩余开放问题」 | DEC-0401 已固定 Coinbase 序列化/兑奖槽 | 改为「已定」，引 DEC-0401，移出待决 |
| 「Stakes 精确定义仍需后续固定」 | proposal 03/11/12 已锚定三种语义（B-5） | 改为「已定」，分别引 03·11·12 |
| 「证明路径编码待决」 | DEC-0004 已固定哈希树与证明边界 | 改为「已定」，引 DEC-0004 |
| —— | 新增 C-6 脚本成本数值（DEC-0504 开放） | 入 `06`·`12` 待决，相关 Task 阻塞 |
| —— | 新增 C-7 禁用指令解除方式（DEC-0505 开放） | 入 `06`·`12` 待决，相关 Task 阻塞 |
| 创世具体参数 | C-9（创世时间戳、mainnet Genesis-ID 占位）仍开放 | 入 `02`·`07`·`12` 待决，创世硬编码 Task 阻塞 |
| P2P 线格式/治理/子链 | C-10 实现边界 | 入 `09`·`11`·`12` 边界声明，不规格化 |

> 创世边界遵循仓库既有约定（repo memory）：Plan 只固定创世块实现边界与验证规则，不伪造创世时间戳/初始输出/启动期归属；这些待 C-9 单独创世规范给出。

## 7. 交付与执行流程（分 3 批，每批审阅）

1. **批次 1**：`00`（路线图）+ `01`（基础层 L0）→ ⏸ 审阅文风、追溯标注密度、DEC 引用正确性，校准后再继续。
2. **批次 2**：`02`~`06`（区块链/交易/签名见证/状态/脚本，L1~L3）。
3. **批次 3**：`07`~`11`（共识/端点分叉/组队校验/激励/服务，L4~L5）+ `12`（待决与验收）。
4. 全部完成后做一致性自检：proposal 引用编号正确、无 `ADR-xxxx` 残留、DEC 编号与主题匹配、阶段门禁前后衔接、待决项与 proposal 00 汇总一致。
5. 同步更新 `00-Implementation-Roadmap.md` 中「方案文件」表、「主要风险」「阶段门禁」为新 13 篇结构与新 DEC 口径。

> 旧 8 篇的处理：逐篇重写覆盖（保留可复用的 TDD 任务骨架与文件清单），**删除已不存在的旧文件**（如旧 `06`/`07`/`08` 编号语义变化的部分），最终 `docs/plan/` 只保留新 13 篇。

## 8. 验收标准

- `docs/plan/` 下存在 1 篇路线图（00）+ 11 篇阶段（01~11）+ 1 篇待决验收（12），共 13 篇。
- 每篇含「来源提案（新编号）」「包边界」「建议文件」「TDD Task」「阶段门禁/验收」。
- 全程无 `ADR-xxxx` 残留；所有决策引用为 `DEC-NNNN` 且主题匹配第 6 节 A 表。
- 待决项严格限于 C-6/C-7/C-9/C-10，相关 Task 显式标注阻塞/占位；已裁决项（单位、兑奖槽、Stakes、证明路径）不再列为待决。
- `00` 路线图的方案文件表、风险表、门禁表与新 13 篇一致；与 proposal 00 的层/包/待决汇总无矛盾。

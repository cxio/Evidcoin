# docs 文件结构 & 关联指南

本文档在 `docs` 目录之下，因此文档文件相对于此目录，不再从项目根目录计算路径。

## 目录结构概览

Evidcoin 文档分为四个层级：2个核心层级，由用户设计，2个辅助层级，由 AI 生成。权威性自上而下递减：

| 层级 | 目录 | 作者 | 说明 |
|------|------|------|------|
| Conception（构想层） | `conception/` | 人工 + AI 辅助 | 设计构想，作者对协议、系统和应用边界的原始设计。 |
| Decision（决策层） | `decision/` | 人与 AI 互助思考 | 架构决策，记录 Conception 尚未明确或澄清模糊的补充决策。 |
| Proposal（提案层） | `proposal/` | AI 生成 + 人工审阅 | 详细技术规格，追溯自 Conception + Decision。 |
| Plan（方案层） | `plan/` | AI 生成 + 人工浏览 | 按阶段的实施计划（TDD 任务、包边界、文件清单），追溯自 Proposal。 |

### 权重关系

**权威顺序**：`Conception` + `Decision` > `Proposal` > `Plan`。

如遇冲突以更上层为准，最终以 `Conception` 和 `Decision` 为准。`Decision` 可以是 `Conception` 的细化或模糊 => 明确，但两者不得冲突，否则必须人工介入裁决。

若发现 `Proposal` 或 `Plan` 与 `Conception` 或 `Decision` 不一致，先修改受影响的 `Conception` 或 `Decision` 文档，再重新生成对应的 `Proposal` 和 `Plan` 文件；不要只修改一个文件而不同步其下游内容。

### 排除目录

`plans/`（带 s）用于 AI Agent 工作过程中的临时实施计划，不作为正式文档的一部分；正式方案在 `plan/`。

另外，上级目录中的 `working/` 属于工作间目录，存放临时杂物，也不在正式文档序列里。


## 维护总则（Agent）

以上章节和本章节不可修改，这是文档结构的基础设计和关系逻辑。

下面四个章节记录了项目的设计构想、决策、技术提案和实施计划部分，需要根据实际情况更新。

## 设计构想（Conception）

位于 `conception/` 目录下，包含以下内容：

| 功能 | 设计构想文件 |
|------|-------------|
| 共识 | `1.共识-历史证明（PoH）.md` |
| 共识 | `2.共识-端点约定.md` |
| 服务 | `3.公共服务.md` |
| 经济 | `4.激励机制.md` |
| 信用 | `5.信用结构.md` |
| 脚本 | `6.脚本系统.md` |
| 交易 | `附.交易.md` |
| 校验 | `附.组队校验.md` |
| 总体 | `blockchain.md` |
| 图示 | `images/*.svg` |
| 脚本指令集 | `Instruction/*.md` |
| 示例 | `examples/*.md` |


## 架构决策（Decision）

位于 `decision/` 目录。Decision 的作用是补充 Conception 未直接固定的必要规则，如方向选择、边界划分、实现路径等关键决策。
不涉及详细的技术规格，其地位/范围是：如果没有这些决策，技术规格部分（Proposal）将无法进行。

文档内容基本上遵循如下结构：

- 背景 （Context）
- 决策 （Decision）
- 理由 （Rationale）
- 影响 （Consequences）
- 构想层依据 （Conception References）
- 开放问题 （Open Questions）

|    编号    |    文件    |  覆盖主题  |
|------------|------------|------------|
| （待更新） | （待更新） | （待更新） |

维护规则：

- 新增 Decision 前必须先检查 `conception/` 是否已经明确该规则。
- 若 Conception 已明确，直接引用 Conception，不新增 Decision。
- 若后续 Conception 修订吸收了某个 Decision，应删除或标记该 Decision 已被吸收。
- Decision 文件命名为 `DEC-<NNNN>-<short-description>.md`。其中 `<NNNN>` 为序号，可以有类别区分（如 `0201`，第2类第1项），也可没有类别（如 `0001`）。


## 技术提案（Proposal）

位于 `proposal/` 目录，由 Conception + Decision 生成的可实施技术规格（Spec），如字节编码、字段宽度、实现的详细规则等。

文档内容大致遵循如下结构：

- 来源追溯
- 概述
- 规格正文
- 边界与限制
- 待决问题
- 对 Plan 的约束

| 编号 | 文件 | 覆盖主题 |
|------|------|---------|
| 00 | `00.Project-Scope.md` | 项目范围、全局待决问题汇总 |
| 01~04 | `01.Types-And-Encoding.md` … `04.Hash-Trees.md` | 基础类型/编码、密码学与哈希、标识符与常量、哈希树 |
| 05 | `05.Blockchain-Core.md` | 区块链核心 |
| 06~07 | `06.Transaction-Model.md`、`07.Coin-Credit-Proof-Units.md` | 交易模型、信元 |
| 08 | `08.Signatures-And-Witness.md` | 签名与见证 |
| 09 | `09.UTXO-UTCO-State.md` | UTXO/UTCO 状态 |
| 10 | `10.Script-System.md`（+ `Instruction/`） | 脚本系统 |
| 11~12 | `11.PoH-Consensus.md`、`12.Endpoint-And-Fork-Choice.md` | PoH 共识、端点与分叉选择 |
| 13~15 | `13.Team-Validation.md`、`14.Incentives-And-Coinbase.md`、`15.Public-Service-Interfaces.md` | 组队校验、激励与 Coinbase、公共服务接口 |

维护规则：

- Proposal 的权威性低于 Conception 与 Decision。
- 每篇「来源追溯」必须可回溯到具体 Conception 章节与 `DEC-<NNNN>`。
- 待决项严格限于全局待决集，相关规格须显式标注，不得默认选值固化。


## 实施方案（Plan）

位于 `plan/` 目录，由 Proposal 转化的阶段化实施计划。与 Proposal 章节、代码包、实施阶段对齐。

文档内容大致遵循如下结构：

- 来源提案
- 包边界/非目标
- 建议文件
- TDD Task
- 阶段验收/门禁

| 文件 | 覆盖 Proposal | 对应包（层） |
|------|---------------|-------------|
| `00-Implementation-Roadmap.md` | 00 | 全局/索引 |
| `01-Foundation-Types-Crypto.md` | 01·02·03·04 | `pkg/types`·`pkg/crypto`·`pkg/hashtree`（L0） |
| `02-Blockchain-Core.md` | 05 | `internal/blockchain`（L1） |
| `03-Transaction-And-Units.md` | 06·07 | `internal/tx`（L1） |
| `04-Signatures-And-Witness.md` | 08 | `internal/tx`（L1） |
| `05-UTXO-UTCO-State.md` | 09 | `internal/utxo`·`internal/utco`（L2） |
| `06-Script-System.md` | 10 + Instruction/ | `internal/script`（L3） |
| `07-PoH-Consensus.md` | 11 | `internal/consensus`（L4） |
| `08-Endpoint-And-Fork-Choice.md` | 12 | `internal/consensus`（L4） |
| `09-Team-Validation.md` | 13 | `internal/validation`（L5，接口） |
| `10-Incentives-And-Coinbase.md` | 14 | `internal/rewards`（L5） |
| `11-Public-Service-Interfaces.md` | 15 | `internal/services`（L5，接口） |
| `12-Open-Questions-And-Acceptance.md` | 全部 | 全局待决与验收 |

维护规则：

- 决策引用为 `DEC-<NNNN>` 且主题匹配。
- 待决项对应的 Task 显式标注阻塞/占位。
- 实现任何功能前，应先读对应 `plan/` 文件，再回溯 `proposal/`，如有疑问查 `decision/` 与 `conception/`。

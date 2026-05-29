# Proposal 重写设计（Proposal Rewrite Design）

> 日期：2026-05-29
> 状态：已与作者确认，待转入实施计划（writing-plans）

## 1. 背景与目标

`docs/AGENTS.md`（当前最权威）已将 Evidcoin 文档正式结构收敛为**两层**：`conception/`（构想）+ `decision/`（决策）；并明确 `proposal/` 与 `plan/` 为"待重构材料"，后续应**从 `conception/` 与 `decision/` 重新生成**。

现有 `docs/proposal/`（2026-05-03 生成）基于旧版构想，且仍按已废弃的"三层体系（Conception→Proposal→Plan）"措辞组织；而 `conception/` 在 2026-05-28~29 大幅演进（新增 `附.交易.md`、`附.组队校验.md`、`6.脚本系统.md`、`examples/` 等），`decision/` 新增 21 个 `DEC-*`（全部 Accepted）。两者已显著领先于旧 proposal。

**目标**：以演进后的 `conception/` + `decision/` 为唯一依据，重写整个 `docs/proposal/`，使其成为完整、自洽、可被 Plan/代码直接引用的**可实施技术规格**。

## 2. 约束（已确认）

1. **目标**：基于新 conception + decision 重生提案（非增量、非仅结构调整）。
2. **定位**：完整可实施技术规格，可被 Plan/代码直接引用。
3. **章节组织**：方案 A — 按实现分层（Layer 0→5），与 decision 编号空间 / 代码包 / Plan 阶段四方对齐；每章开头仍列 conception 来源。
4. **追溯**：严格逐条标注来源（conception 文件 / DEC-NNNN）；conception 与 decision 冲突时**以 conception 为准**并加注。

## 3. 不改动范围

`conception/`、`decision/`、`plan/`、`plans/`（除本设计文档）、所有 Go 代码。本任务仅产出 `docs/proposal/` 文档。

## 4. 章节划分（16 篇主文 + Instruction 子目录）

| 序号 | 文件 | 对应层/包 | 主要 conception 来源 | 主要 DEC 来源 |
|------|------|-----------|---------------------|--------------|
| 00 | Project-Scope.md | 全局/索引 | README、blockchain、docs/AGENTS | — |
| 01 | Types-And-Encoding.md | pkg/types | 6.脚本、1.值、8.转换、附.交易(地址) | DEC-0001 |
| 02 | Cryptography-And-Hashing.md | pkg/crypto | blockchain(哈希表)、附.交易(签名) | DEC-0002, 0104 |
| 03 | Identifiers-And-Constants.md | pkg/types | blockchain、1.PoH、2.端点、4.激励 | DEC-0001(年度) |
| 04 | Hash-Trees.md | pkg/types | blockchain、附.交易、附.组队校验、5.信用 | DEC-0004, 0002 |
| 05 | Blockchain-Core.md | internal/blockchain | blockchain | DEC-0003 |
| 06 | Transaction-Model.md | internal/tx | 附.交易、5.信用 | DEC-0101, 0003 |
| 07 | Coin-Credit-Proof-Units.md | internal/tx | 5.信用结构、附.交易 | DEC-0101 |
| 08 | Signatures-And-Witness.md | internal/tx | 附.交易(签名/多签) | DEC-0102, 0103, 0104 |
| 09 | UTXO-UTCO-State.md | internal/utxo·utco | 附.组队校验、blockchain | DEC-0201 |
| 10 | Script-System.md | internal/script | 6.脚本系统 + Instruction/* | DEC-0501~0505 |
| 11 | PoH-Consensus.md | internal/consensus | 1.共识-PoH、附.组队校验 | DEC-0301, 0302 |
| 12 | Endpoint-And-Fork-Choice.md | internal/consensus | 2.共识-端点约定 | DEC-0303 |
| 13 | Team-Validation.md | 接口 | 附.组队校验 | DEC-0601 |
| 14 | Incentives-And-Coinbase.md | 经济 | 4.激励机制、blockchain | DEC-0401 |
| 15 | Public-Service-Interfaces.md | 外部接口 | 3.公共服务、README | DEC-0602, 0603 |
| — | Instruction/*.md | internal/script | conception/Instruction/* | DEC-0501~0505 |

相比旧 14 篇的关键调整：
- 签名与见证从交易模型独立成章（08）：DEC-0102/0103/0104 内容量大（签名消息布局、见证容器剪枝、多签三套排序），合并会过载。
- 端点约定与分叉选择合并（12）：conception 同属一文，DEC-0303 统一覆盖。
- 中间件/链外交互、形式化验证、附件网络、量子安全不单独成章，并入相关章节（交互→脚本章；附件→信元/服务章；量子→密码学章）。

## 5. 单篇统一模板

```
# <Title>（中文标题）
## 来源追溯       —— Conception 文件 + DEC 编号及各自贡献
## 概述           —— 架构位置、对应代码包、上下层依赖
## 规格正文       —— 按主题分小节；每条关键规格就近标注来源
## 边界与限制     —— 常量上限、禁止项、非目标
## 待决问题       —— conception/decision 未覆盖或冲突待裁决的点
## 对 Plan 的约束 —— 对实施 Plan 的硬约束与追溯要求
```

**追溯标注规则**：
- 每条关键规格就近标注来源（conception 文件名 / DEC-NNNN）。
- conception 与 decision 冲突 → 正文采用 conception 口径，加注「注：DEC-NNNN 原述 X，与 conception <文件> 冲突，本提案以 conception 为准」。
- 仅 decision 有 → 正常采用并标注。
- 二者均无 → 不臆造，写入该章「待决问题」。

## 6. 冲突与缺口处理清单

### A. 冲突 → 以 conception 为准并加注（落入正文）
1. **空状态树域标签缺口**：DEC-0201 用 `utxo.empty`/`utco.empty`，DEC-0002 的 12 项清单未含。→ 第 02、09 章补入全集并加注扩展。
2. **ML-DSA 实现选择**：DEC-0104 锁定 `cloudflare/circl`；根 AGENTS.md 说"优先标准库"。→ 第 02、08 章以 DEC-0104 为准并加注取舍，同时列入待决（是否随 Go 1.26 标准库成熟改用）。

### B. 易混淆但不冲突 → 集中澄清（落入正文）
3. **MintPKHash 三处编码不同**（普通交易头 varint封装 / Coinbase头定长32 / 签名消息 varint封装）→ 第 06、08 章标注 + 第 06 章对照表。
4. **三套多签排序规则**（见证容器字典序 / 签名消息提供顺序 / 地址派生字典序）→ 第 08 章对照表分别锚定。
5. **Stakes 三种语义**（区块头累计值 / 铸凭哈希取 H-32 / 分叉比较 winner.Stakes）→ 第 03、11、12 章分别明确口径。

### C. 二者均未覆盖 → 仅列待决，不臆造
6. **脚本成本数值**（DEC-0504 明确开放）：opcode base_cost、正则/随机最坏上界、三层上限数值 → 第 10 章待决。
7. **禁用指令解除方式**（DEC-0505 开放）：逐项 opcode 激活 vs 统一版本激活 → 第 10 章待决。
8. **单位体系混用**（chx / Bi / 毫币）→ 第 01 或 03 章待决 + 建议统一口径。
9. **创世具体参数**（创世时间戳、mainnet Genesis-ID 占位）→ 第 05/11 章待决。
10. **P2P 线格式 / 版本分叉治理 / 通用子链派生协议**：conception 外包或未抽象 → 第 00 章「非目标/边界」声明，不强行规格化。

## 7. 交付与执行流程（分 4 批，每批审阅）

1. **批次 1**：00（范围/索引）+ 01~04（基础层）→ 暂停审阅文风与追溯标注。
2. **批次 2**：05~09（区块链/交易/信元/签名见证/状态）。
3. **批次 3**：10 + Instruction/（脚本系统与指令集，最大块）。
4. **批次 4**：11~15（共识/端点分叉/组队校验/激励/服务）+ README 索引。
5. 全部完成后一致性自检（交叉引用、追溯编号、待决问题汇总）。

同时将 00.Project-Scope 等仍按"三层体系"措辞的内容，更新为与最新 `docs/AGENTS.md`（两层结构 + proposal 为重生产物）一致。

## 8. 验收标准

- 每篇都有「来源追溯」「待决问题」「对 Plan 的约束」三节。
- 每条关键规格可追溯到 conception 文件或 DEC 编号。
- 全局待决问题在 00 汇总，与各章一致。
- 与最新 `docs/AGENTS.md` 两层结构表述一致，无"三层体系"残留矛盾。

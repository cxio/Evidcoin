# 实施方案重写执行计划（Plan Rewrite Implementation Plan）

> 设计依据：`docs/plans/2026-05-30-plan-rewrite-design.md`（已确认）

**Goal:** 以重写后的 `docs/proposal/`（00~15 + Instruction，追溯自 conception+decision）为唯一依据，整体重写 `docs/plan/` 为 13 篇（1 路线图 + 11 阶段 + 1 验收），与 proposal 章节、DEC 编号、代码包、实施阶段四方对齐，保持 TDD 任务式深度。

**Architecture:** 按 Layer 0→5 单向依赖推进。每篇沿用现有 plan 风格（Goal/Architecture/Tech Stack + 来源提案 + 包边界 + 建议文件 + TDD Task + 阶段门禁）。所有 proposal 引用改新编号，所有决策引用从 `ADR-xxxx` 改为 `DEC-NNNN`。

**Tech Stack:** Markdown 文档；来源为 `docs/proposal/*` 与 `docs/decision/DEC-*`。

**说明（非代码任务）：** 本计划无 Go 代码，校验环节为文档校验：①proposal 引用为新编号、文件存在；②无 `ADR-xxxx` 残留、DEC 编号与主题匹配（设计文档第 6 节 A 表）；③待决项限于 C-6/C-7/C-9/C-10；④阶段门禁前后衔接。每批末尾设审阅检查点。

---

## 通用单篇工作流（每篇 plan 均按此 4 步）

**Step A — 重读源提案**：打开该篇覆盖的 proposal 章，提取「对 Plan 的约束」与关键规格、待决标注。
**Step B — 重写**：按统一模板写出该篇，proposal 引用用新编号，决策引用用 DEC-NNNN，每个 Task 就近标注来源；待决相关 Task 标注阻塞/占位。
**Step C — 文档校验**：核对模板齐全、proposal 编号正确、无 ADR 残留、DEC 主题匹配、门禁衔接。
**Step D — 暂存**：`git add` 该篇（提交在每批末尾统一进行，仅在用户要求提交时执行）。

---

## 批次 1：路线图 + 基础层（00 + 01）→ 审阅检查点

### Task 1: 00-Implementation-Roadmap.md
**覆盖：** proposal 00。**要点：** 范围声明（Proposal⇒Plan 层）；Layer 0→5 包边界表；新 13 篇方案文件表；推荐实施顺序；全局编码原则（DEC 口径）；全局验证命令；阶段门禁表（按新阶段）；主要风险（DEC 重映射后）；全局待决指针（C-6/C-7/C-9/C-10）。删除旧 proposal 00-14 映射与 ADR 措辞。

### Task 2: 01-Foundation-Types-Crypto.md
**覆盖：** proposal 01·02·03·04。**包：** `pkg/types`·`pkg/crypto`·`pkg/hashtree`。**要点：** 规范编码（DEC-0001 ULEB128 最短/定宽大端/字节序列/BigInt `slen||magnitude`/年度 UTC/chx 单位）、Hash 与域标签全集 14 项（DEC-0002）、地址与多签复合公钥哈希、ML-DSA-65 circl profile（DEC-0104）、通用二叉树与专用树（DEC-0004）、常量与 Stakes 三义登记（proposal 03）。待决：A-2（ML-DSA 标准库切换）以策略参数处理。

### Task 3: 批次 1 校验与提交
核对 00 方案文件表与实际生成一致；01 的 DEC 引用（0001/0002/0004/0104）主题匹配、无 ADR 残留。⏸ **审阅检查点** — 交作者审阅文风、追溯密度、DEC 引用正确性，校准后进入批次 2。

---

## 批次 2：核心层（02~06，L1~L3）

### Task 4: 02-Blockchain-Core.md
**覆盖：** proposal 05。**包：** `internal/blockchain`。**要点：** 区块头字段顺序/编码（常规 vs 年块、YearBlock 条件存在）、CheckRoot=Hash384(TreeRoot||UTXORoot||UTCORoot)、Stakes 字段、入块/年块存储/完整性/手动切链、区块限额曲线、创世工件（引 07）。待决：C-9 创世参数 → 创世硬编码 Task 阻塞。

### Task 5: 03-Transaction-And-Units.md
**覆盖：** proposal 06·07。**包：** `internal/tx`。**要点：** 普通/Coinbase 交易头字段顺序（MintPKHash 三处编码对照表）、交易体、输入项、输出公共头、Coin/Credit/Proof payload、附件 ID、Credit 限制；DEC-0101·0003。

### Task 6: 04-Signatures-And-Witness.md
**覆盖：** proposal 08。**包：** `internal/tx`。**要点：** 签名消息布局（ChainScope/SigScope/TxHeaderCore/Covered*）、授权种类 8 位、辅项冲突、Coinbase 签名消息、见证容器与剪枝、多签 M-of-N 与三套排序对照；DEC-0102·0103·0104。

### Task 7: 05-UTXO-UTCO-State.md
**覆盖：** proposal 09。**包：** `internal/utxo`·`internal/utco`。**要点：** FlagOutputs/Count 位语义、四层宽成员树 `[8,13,18]`、空根 `utxo.empty`/`utco.empty`、叶子前像、UTCO 过期、链式约束；DEC-0201·0002。

### Task 8: 06-Script-System.md
**覆盖：** proposal 10 + Instruction/。**包：** `internal/script`。**要点：** 栈/实参区/附参/局部域全局域、254 指令分段、5 前缀 18 类别 3 特例、字节码编码、浮点 profile、注册表与环境边界、公共/私有路径与禁用指令、成本模型框架、解锁段 opcode 限制；DEC-0501~0505。待决：C-6 成本数值、C-7 禁用解除 → 相关 Task 阻塞。

### Task 9: 批次 2 校验与提交
核对 02~06 章间交叉引用（创世↔07、片组树↔01、空根↔01、签名↔脚本 FN_CHECKSIG）、DEC 主题匹配、待决标注；提交 `docs(plan): rewrite core layer (02-06)`。

---

## 批次 3：共识与外围层（07~12，L4~L5）

### Task 10: 07-PoH-Consensus.md
**覆盖：** proposal 11。**包：** `internal/consensus`。**要点：** 铸凭哈希前像、铸凭交易窗口 `[-80000,-240]`、`-32` 币权、择优池、铸造者验证、MintProof、创世与初段窗口；DEC-0301·0302。待决：C-9 创世参数。

### Task 11: 08-Endpoint-And-Fork-Choice.md
**覆盖：** proposal 12。**包：** `internal/consensus`。**要点：** 出块时序（6min/15s/30s）、区块发布三段、分叉链段竞争（31/16/20）、同铸造者多签归一化、2 倍币权/交易量归一化、RandomX 平局、交易回收、零确认/最低费/过期；DEC-0303。

### Task 12: 09-Team-Validation.md
**覆盖：** proposal 13。**包：** `internal/validation`（接口）。**要点：** 角色分工接口、首领校验、安全保障（冗余/复核/反馈）、铸造协作信息分离、区块证明包（DEC-0601）与快速预验证、最优规模参考。待决：C-10 边界（P2P/治理外包）。

### Task 13: 10-Incentives-And-Coinbase.md
**覆盖：** proposal 14。**包：** `internal/rewards`。**要点：** 发行曲线、50% 销毁、Coinbase 序列化（省略 HashInputs）、奖励分配与余数、兑奖槽 bit 顺序（DEC-0401 已固定）；金额一律 chx。

### Task 14: 11-Public-Service-Interfaces.md
**覆盖：** proposal 15。**包：** `internal/services`（接口）。**要点：** Depots/Blockqs/STUN/基网边界、Blockqs 6 类响应、数据量边界（<10MB/>=10MB）、区块概要 TxID profile（DEC-0602）、响应验证与服务密钥（DEC-0603）。待决：C-10 边界。

### Task 15: 12-Open-Questions-And-Acceptance.md
**覆盖：** 全部。**要点：** 全局待决（C-6/C-7/C-9/C-10）逐条承载章与阻塞策略；全局验收标准；阶段门禁汇总；已裁决项清单（单位/兑奖槽/Stakes/证明路径，移出待决）。

### Task 16: 批次 3 校验与全局自检
核对 07~12 交叉引用与待决一致；全局自检：无 ADR 残留、proposal 编号正确、DEC 主题匹配、13 篇齐全；删除旧 8 篇中已被新编号取代的残留。提交 `docs(plan): rewrite consensus & periphery (07-12) and finalize`。

# Proposal 重写实施计划（Proposal Rewrite Implementation Plan）

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.
> 设计依据：`docs/plans/2026-05-29-proposal-rewrite-design.md`

**Goal:** 以演进后的 `conception/` 与 `decision/` 为唯一依据，重写 `docs/proposal/` 全部 16 篇主文与 Instruction 子目录，使其成为完整、自洽、可追溯、可被 Plan/代码直接引用的技术规格。

**Architecture:** 按实现分层（Layer 0→5）组织章节，与 decision 编号空间、代码包、Plan 阶段四方对齐。每篇用统一模板（来源追溯 / 概述 / 规格正文 / 边界 / 待决问题 / 对 Plan 的约束）。严格逐条追溯，conception 与 decision 冲突时以 conception 为准并加注。

**Tech Stack:** Markdown 文档；来源为 `docs/conception/*`（含 Instruction/、examples/）与 `docs/decision/DEC-*`。

**说明（非代码任务）：** 本计划无 Go 代码，故 TDD 的"测试"环节替换为**文档校验**：①追溯标注核对（每条关键规格有 conception 文件或 DEC 编号）；②与源材料事实比对（字段顺序/字节/常量与 DEC 一致）；③交叉引用核对（章间引用编号正确）；④模板完整性（六节齐全）。每批末尾有作者审阅检查点。

---

## 通用单篇工作流（每篇 proposal 均按此 5 步）

**Step A — 重读源材料**：打开该篇对应的 conception 文件与 DEC 文件，列出本篇需覆盖的规格要点清单。
**Step B — 起草**：按统一模板写出该篇，每条关键规格就近标注来源；冲突点按"以 conception 为准 + 加注"处理；缺口写入「待决问题」。
**Step C — 文档校验**：核对模板六节齐全、每条关键规格有来源标注、字段/常量与 DEC 事实一致、章间交叉引用编号正确。
**Step D — 自查清单比对**：对照 `2026-05-29-proposal-rewrite-design.md` 第 6 节冲突/缺口清单，确认本篇涉及条目已正确落地。
**Step E — 暂存**：`git add` 该篇（提交在每批末尾统一进行）。

---

## 批次 1：基础层（00 + 01~04）→ 审阅检查点

### Task 1: 00.Project-Scope.md（范围与全局索引）
**Files:** Create `docs/proposal/00.Project-Scope.md`
**要点：** 项目范围与文档定位（对齐最新 `docs/AGENTS.md` 两层结构，声明 proposal 为"由 conception+decision 重生的可实施规格"，删除旧"三层体系"措辞）；16 篇章节索引表 + conception/decision↔章节对照；全局「待决问题」汇总（设计文档第 6 节 C 类：脚本成本数值、禁用解除方式、单位体系、创世参数、P2P/治理/子链边界）；非目标与实现边界。
按通用工作流 A→E。

### Task 2: 01.Types-And-Encoding.md（基础类型与编码）
**Files:** Create `docs/proposal/01.Types-And-Encoding.md`
**来源：** conception 6.脚本、1.值指令、8.转换指令、附.交易(地址)；DEC-0001。
**要点：** ULEB128 最短编码、定宽大端、字节序列 `varint(len)||bytes`、定宽字段白名单、年度=UTC 自然年、BigInt 序列化 `slen||magnitude`；基础值类型（Byte/Rune/Int/BigInt/Float/String/Bytes）。
**冲突/缺口落地：** 单位体系（chx/Bi/毫币）混用 → 「待决问题」+建议统一口径。
按通用工作流 A→E。

### Task 3: 02.Cryptography-And-Hashing.md（密码学与哈希）
**Files:** Create `docs/proposal/02.Cryptography-And-Hashing.md`
**来源：** conception blockchain(哈希策略表)、附.交易(签名)；DEC-0002、DEC-0104。
**要点：** 域标签编码 `"Evidcoin/v1/"||name||0x00` 与 14 项标签全集（12 项 + 补 `utxo.empty`/`utco.empty`）；各用途算法 profile 表；附件片组树无域标签例外；地址编码与公钥哈希（单签/多签）；ML-DSA-65 profile。
**冲突/缺口落地：** ①空根域标签补入并加注扩展 DEC-0002；②ML-DSA 以 DEC-0104(circl) 为准并加注取舍 + 列待决。
按通用工作流 A→E。

### Task 4: 03.Identifiers-And-Constants.md（标识符与常量）
**Files:** Create `docs/proposal/03.Identifiers-And-Constants.md`
**来源：** conception blockchain、1.PoH、2.端点、4.激励；DEC-0001(年度)。
**要点：** Protocol/Chain/Genesis/Bound-ID、MixData 组成；核心常量（87661、6min、240、31、20、16、6000、MaxStack* 等）；chx 单位。
**冲突/缺口落地：** Stakes 三义之一（区块头累计值口径）在此明确；单位体系待决（与 01 协调，避免重复，择一承载）。
按通用工作流 A→E。

### Task 5: 04.Hash-Trees.md（哈希树）
**Files:** Create `docs/proposal/04.Hash-Trees.md`
**来源：** conception blockchain、附.交易、附.组队校验、5.信用；DEC-0004、DEC-0002。
**要点：** 通用二叉树（枝/叶前像、奇数层提升、单叶树根、验证路径编码）；专用树（区块交易树 `seq(3B)||TxID`、输入根 `BLAKE3-256(ListHash||LeadPKHash)`、附件片组树无域标签、UTXO/UTCO 宽成员树引向第 09 章）。
按通用工作流 A→E。

### Task 6: 批次 1 校验与提交
**Step 1:** 复核 01/03 单位体系待决只在一处承载、另一处引用，无重复矛盾。
**Step 2:** 复核 02 与 04 对 UTXO/UTCO 空根/域标签的引用一致，且都指向第 09 章承载细节。
**Step 3:** 复核 00 索引表与实际生成文件一致。
**Step 4:** 提交：`git add docs/proposal/00.* docs/proposal/01.* docs/proposal/02.* docs/proposal/03.* docs/proposal/04.* && git commit -m "docs(proposal): rewrite foundation layer (00-04) from conception+decision"`
**Step 5:** ⏸ **审阅检查点** — 交作者审阅文风、追溯标注密度、模板落地是否达标，校准后再进入批次 2。

---

## 批次 2：核心层（05~09）

### Task 7: 05.Blockchain-Core.md
**来源：** conception blockchain；DEC-0003。
**要点：** 区块头字段顺序与编码（常规 112B / 年块 160B、YearBlock 条件存在）、CheckRoot=Hash384(TreeRoot||UTXORoot||UTCORoot)、币权 Stakes、入块/年块存储/完整性/手动切链、区块限额曲线、创世区块工件（引向 11 章）。
**缺口落地：** 创世具体参数（时间戳/Genesis-ID 占位）→ 待决。
按通用工作流 A→E。

### Task 8: 06.Transaction-Model.md
**来源：** conception 附.交易、5.信用；DEC-0101、DEC-0003。
**要点：** 普通交易头/Coinbase 头字段顺序（含 MintPKHash 三处编码对照表）、交易体（Inputs/Outputs 编码）、输入项、输出公共头（Config 字节、无销毁位）、交易合法性。
**冲突/缺口落地：** MintPKHash 三处编码对照表（澄清 B-3）。
按通用工作流 A→E。

### Task 9: 07.Coin-Credit-Proof-Units.md
**来源：** conception 5.信用结构、附.交易；DEC-0101。
**要点：** Coin/Credit/Proof 三类 payload 字段与编码、可选字段 `varint(0)` 缺省、自定义类不进 UTXO/UTCO、介管脚本、附件 ID 结构、Credit 限制（单次寿命/每交易≤2笔/31年过期）。
按通用工作流 A→E。

### Task 10: 08.Signatures-And-Witness.md
**来源：** conception 附.交易(签名/多签)；DEC-0102、DEC-0103、DEC-0104。
**要点：** 签名消息布局（ChainScope/SigScope/TxHeaderCore/Covered*）、授权种类、辅项冲突规则、Coinbase 签名消息；见证容器格式与剪枝边界；多签 M-of-N 与复合公钥哈希。
**冲突/缺口落地：** 三套多签排序规则对照表（澄清 B-4）；ML-DSA 引用第 02 章。
按通用工作流 A→E。

### Task 11: 09.UTXO-UTCO-State.md
**来源：** conception 附.组队校验、blockchain；DEC-0201。
**要点：** FlagOutputs/Count 位语义、四层宽成员树分层（年度+TxID `[8,13,18]`）、空根 `utxo.empty`/`utco.empty`、叶子前像、UTCO 过期处理、链式约束（三路耦合）、缓存边界。
**冲突/缺口落地：** 空根域标签与第 02 章保持一致。
按通用工作流 A→E。

### Task 12: 批次 2 校验与提交
核对 05~09 章间交叉引用（创世↔11、片组树↔04、空根↔02）；MintPKHash 对照表与多签排序表无矛盾；提交 `docs(proposal): rewrite core layer (05-09)`。

---

## 批次 3：脚本系统（10 + Instruction/）

### Task 13: 10.Script-System.md
**来源：** conception 6.脚本系统；DEC-0501~0505。
**要点：** 栈/实参区/附参/关联数据/局部域全局域、254 指令分段（基础[0-169]/函数[170-224]/模块[225-250]/扩展[251-253]）、5 前缀、18 类别、3 特例（SYS_NULL/SYS_TIME/CALL）、字节码编码、浮点 profile、注册表与环境边界、公共/私有路径与禁用指令、成本模型框架、执行状态机、解锁脚本 opcode[0-50] 限制。
**冲突/缺口落地：** 脚本成本数值（C-6）、禁用解除方式（C-7）→ 「待决问题」。
按通用工作流 A→E。

### Task 14: Instruction/ 子目录重写
**来源：** conception/Instruction/*（AGENTS、0.基本约束、1~18 类）；DEC-0501~0505。
**要点：** 对齐最新指令集基线（254 位、未用保留位、前期禁用 SCRIPT/VALUE/EVAL/INOUT、解锁段限制）。重写 `docs/proposal/Instruction/` 下 README + 0~18 各类文件 + Base-Constraints + AGENTS。逐类核对指令码区间与数量（合计 254）。
按通用工作流 A→E（可按"每类一文件"拆为子任务执行）。

### Task 15: 批次 3 校验与提交
核对 10 章与 Instruction/ 指令码区间/禁用清单/成本待决一致；提交 `docs(proposal): rewrite script system (10 + Instruction)`。

---

## 批次 4：共识/服务/激励 + 索引（11~15 + README）

### Task 16: 11.PoH-Consensus.md
**来源：** conception 1.共识-PoH、附.组队校验；DEC-0301、DEC-0302。
**要点：** 铸凭哈希前像（含 X、Mix、Stakes 取 H-32 口径）、MintProof 字段、择优排序、铸造者身份、创世工件与初段窗口规则、高度边界测试点。
**冲突/缺口落地：** Stakes "H-32" 口径明确（B-5）；创世参数待决（与 05 协调）。
按通用工作流 A→E。

### Task 17: 12.Endpoint-And-Fork-Choice.md
**来源：** conception 2.共识-端点约定；DEC-0303。
**要点：** 协议vs共约、6min出块/15s冗余/30s首块、链段比较（31块/16分）、同高度同铸造者归一化、RandomX 平局 profile、2倍币权销毁+交易量替换、过期/错时/最低费共约。
**冲突/缺口落地：** Stakes "winner.Stakes" 口径明确（B-5）。
按通用工作流 A→E。

### Task 18: 13.Team-Validation.md
**来源：** conception 附.组队校验；DEC-0601。
**要点：** 3 角色分工、首领校验、冗余/扩展复核、交易收录优先级、交易量约束、区块证明包字段与快速预验证、最优规模。
按通用工作流 A→E。

### Task 19: 14.Incentives-And-Coinbase.md
**来源：** conception 4.激励机制、blockchain；DEC-0401。
**要点：** 铸币发行曲线、通缩 50%、百日前/后 Coinbase 输出 profile（2笔/5笔）、金额取整与余数承接、兑奖槽 18B/48块/确认规则、reclaimed_award、BurnCoin 销毁单点化（引用第 06 章）。
按通用工作流 A→E。

### Task 20: 15.Public-Service-Interfaces.md
**来源：** conception 3.公共服务、README；DEC-0602、DEC-0603。
**要点：** Depots/Blockqs/STUN/基网职责边界、区块概要 TxID 前16B profile 与碰撞回退、Blockqs 6 类响应、服务边界（<10MB / ≥10MB）、响应验证与服务密钥。
**缺口落地：** P2P 线格式外包 → 边界声明（呼应 00 章）。
按通用工作流 A→E。

### Task 21: README.md（提案层总索引）
**Files:** Create `docs/proposal/README.md`
**要点：** 提案层定位说明、16 篇章节导航、conception↔proposal↔decision 三向对照表、全局待决问题指针（指向 00 章）。

### Task 22: 批次 4 校验与全局一致性自检
**Step 1:** 全局交叉引用核对（各章 @ 引用编号正确）。
**Step 2:** 全局待决问题汇总核对（00 章汇总 == 各章「待决问题」并集）。
**Step 3:** grep 检查无"三层体系/Proposal=>Plan"等过时措辞残留。
**Step 4:** 核对每篇六节模板齐全。
**Step 5:** 提交 `docs(proposal): rewrite consensus/service/incentive layer (11-15) + index`。
**Step 6:** ⏸ 交作者最终审阅。

---

## 验收标准（全计划）

- 每篇含「来源追溯」「待决问题」「对 Plan 的约束」三节（六节模板齐全）。
- 每条关键规格可追溯到 conception 文件或 DEC 编号。
- 全局待决问题在 00 汇总且与各章一致。
- 与最新 `docs/AGENTS.md` 两层结构表述一致，无"三层体系"残留。
- conception 与 decision 冲突处均以 conception 为准并加注。

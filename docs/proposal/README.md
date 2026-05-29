# Evidcoin 提案层（Proposal Layer）总索引

## 本层定位

`docs/proposal/`（提案层 / 技术规格层）是依据最新 `docs/AGENTS.md` 两层文档结构，由 **`conception/`（构想层）+ `decision/`（决策层）重新生成**的可实施技术规格。

- 提案层**不是**正式结构中的独立一层，其权威性永远低于 conception 与 decision。
- 遇 conception 与 decision 冲突时，**以 conception 为准**并加注（见第 00 章追溯与冲突处理规则）。
- 仅 decision 覆盖的细节正常采用并标注 DEC 编号；二者均未覆盖则不臆造，写入对应章「待决问题」并在第 00 章汇总。

> 旧版「三层体系（Conception→Proposal→Plan）」措辞已废弃；本层定位为「由 conception+decision 重生的可实施规格」。

每篇主文采用统一六节模板：**来源追溯 / 概述 / 规格正文 / 边界与限制 / 待决问题 / 对 Plan 的约束**。

## 章节导航（16 篇主文 + Instruction 子目录）

按实现分层（Layer 0→5）组织，与 decision 编号空间 / 代码包 / Plan 阶段四方对齐。

| 序号 | 文件 | 对应层/包 | 主题 |
|------|------|-----------|------|
| 00 | [`00.Project-Scope.md`](00.Project-Scope.md) | 全局/索引 | 项目范围、全局索引与对照、待决问题汇总、实现边界 |
| 01 | [`01.Types-And-Encoding.md`](01.Types-And-Encoding.md) | `pkg/types` | 基础类型与规范编码（ULEB128、定宽、字节序列、BigInt） |
| 02 | [`02.Cryptography-And-Hashing.md`](02.Cryptography-And-Hashing.md) | `pkg/crypto` | 域标签全集、哈希 profile、地址、ML-DSA-65 |
| 03 | [`03.Identifiers-And-Constants.md`](03.Identifiers-And-Constants.md) | `pkg/types` | 标识符（Protocol/Chain/Genesis/Bound-ID）、核心常量 |
| 04 | [`04.Hash-Trees.md`](04.Hash-Trees.md) | `pkg/types` | 通用二叉树与专用树（交易树/输入根/片组树） |
| 05 | [`05.Blockchain-Core.md`](05.Blockchain-Core.md) | `internal/blockchain` | 区块头、CheckRoot、币权、限额曲线、创世工件 |
| 06 | [`06.Transaction-Model.md`](06.Transaction-Model.md) | `internal/tx` | 普通/Coinbase 交易头、交易体、输入输出、合法性 |
| 07 | [`07.Coin-Credit-Proof-Units.md`](07.Coin-Credit-Proof-Units.md) | `internal/tx` | Coin/Credit/Proof 三类 payload、附件 ID |
| 08 | [`08.Signatures-And-Witness.md`](08.Signatures-And-Witness.md) | `internal/tx` | 签名消息布局、见证容器与剪枝、多签 M-of-N |
| 09 | [`09.UTXO-UTCO-State.md`](09.UTXO-UTCO-State.md) | `internal/utxo`·`utco` | 宽成员树、空根、过期处理、链式约束 |
| 10 | [`10.Script-System.md`](10.Script-System.md) | `internal/script` | 栈式虚拟机、254 指令分段、字节码、成本模型 |
| 11 | [`11.PoH-Consensus.md`](11.PoH-Consensus.md) | `internal/consensus` | 铸凭哈希、择优池、铸造者验证、创世与初段窗口 |
| 12 | [`12.Endpoint-And-Fork-Choice.md`](12.Endpoint-And-Fork-Choice.md) | `internal/consensus` | 出块时序、分叉链段竞争、归一化、RandomX 平局 |
| 13 | [`13.Team-Validation.md`](13.Team-Validation.md) | 接口 | 角色分工、首领校验、安全保障、区块证明包 |
| 14 | [`14.Incentives-And-Coinbase.md`](14.Incentives-And-Coinbase.md) | 经济 | 发行曲线、50% 销毁、Coinbase 序列化、兑奖槽 |
| 15 | [`15.Public-Service-Interfaces.md`](15.Public-Service-Interfaces.md) | 外部接口 | Depots/Blockqs/STUN/基网边界、区块概要、响应验证 |
| — | [`Instruction/`](Instruction/) | `internal/script` | 脚本指令集（README + 0~18 各类 + Base-Constraints + AGENTS） |

## conception ↔ proposal ↔ decision 三向对照

| conception 文件 | 承载章节 | 主要 DEC |
|-----------------|---------|---------|
| `README.md` | 00、15 | DEC-0602、DEC-0603 |
| `blockchain.md` | 02、03、04、05、09、14 | DEC-0002、DEC-0003、DEC-0004、DEC-0201、DEC-0401 |
| `1.共识-历史证明（PoH）.md` | 03、11 | DEC-0301、DEC-0302 |
| `2.共识-端点约定.md` | 12 | DEC-0303 |
| `3.公共服务.md` | 15 | DEC-0602、DEC-0603 |
| `4.激励机制.md` | 03、14 | DEC-0401 |
| `5.信用结构.md` | 04、06、07 | DEC-0101 |
| `6.脚本系统.md` | 01、10、Instruction/ | DEC-0501~0505 |
| `附.交易.md` | 01、02、04、06、07、08 | DEC-0001、DEC-0101、DEC-0102、DEC-0103、DEC-0104 |
| `附.组队校验.md` | 04、09、11、13 | DEC-0201、DEC-0301、DEC-0601 |
| `Instruction/*` | 10、Instruction/* | DEC-0501~0505 |

| DEC | 主题 | 承载章节 |
|-----|------|---------|
| DEC-0001 | 规范整数与字节编码 | 01、03 |
| DEC-0002 | 域标签与哈希 profile | 02、04、09 |
| DEC-0003 | 区块与交易字段编码 | 05、06 |
| DEC-0004 | 哈希树与证明边界 | 04 |
| DEC-0101 | 交易体与输出 payload | 06、07 |
| DEC-0102 | 签名消息 profile | 08 |
| DEC-0103 | 见证容器与剪枝 | 08 |
| DEC-0104 | 地址与 ML-DSA profile | 02、08 |
| DEC-0201 | UTXO/UTCO 状态指纹 | 09 |
| DEC-0301 | PoH 铸凭哈希与 MintProof | 11 |
| DEC-0302 | 创世与初段窗口 | 11、05 |
| DEC-0303 | 分叉选择与 RandomX 平局 | 12 |
| DEC-0401 | Coinbase 序列化、奖励与兑奖槽 | 14 |
| DEC-0501~0505 | 脚本字节码/浮点/注册表/成本/失败 | 10、Instruction/ |
| DEC-0601 | 区块证明包 | 13 |
| DEC-0602 | 网络概要 TxID profile | 15 |
| DEC-0603 | Blockqs 验证数据 profile | 15 |

## 全局待决问题指针

全局待决问题（设计文档第 6 节 C 类：二者均未覆盖，仅列待决，不臆造）在**第 00 章「全局待决问题汇总」**集中索引，各章「待决问题」节承载：

| 编号 | 待决问题 | 承载章节 |
|------|---------|---------|
| C-6 | 脚本成本数值（DEC-0504 开放） | 10 |
| C-7 | 禁用指令解除方式（DEC-0505 开放） | 10 |
| C-8 | 单位体系混用（chx/Bi/毫币） | 01（承载），03、14（引用） |
| C-9 | 创世具体参数（创世时间戳、mainnet Genesis-ID 占位） | 05、11 |
| C-10 | P2P 线格式 / 版本分叉治理 / 通用子链派生协议 | 00（边界声明），13、15（呼应） |

详见 [`00.Project-Scope.md`](00.Project-Scope.md)。

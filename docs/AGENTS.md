# docs 文件结构 & 关联指南

> **注**：该文档由 AI Agent 更新，原则如下：
> - 当由设计构想创建（或修订）开发提案时（*Conception => Proposal*），创建双向对应表。
> - 当由开发提案生成（或修订）实施方案时（*Proposal => Plan*），创建 Proposal 到 Plan 的单向对应表。
> - 对应必须尽量完整，使得上级文件如果被修订，可以充分检索相关内容，然后更新下级文件。

本文档描述了 `docs` 目录下*设计构想（Conception）*、*开发提案（Proposal）*和*实施方案（Plan）*的文件结构，以及它们之间的关联。

- `Conception`: 设计构想，包含了作者对项目整体以及各个功能模块的设计构思。
- `Proposal`: 开发提案，基于设计构想生成的具体开发方案和技术细节，为生成 `plan` 提供依据。
- `Plan`: 实施方案，包含了具体的开发计划、任务分配、时间表和测试与验证等。


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


## 开发提案（Proposal）

由设计构想（conception/*）生成提案时，输出存放于 `proposal/` 之下。


### 逆向关系：Proposal → Conception（以开发提案为基准）

| 开发提案文件（proposal/） | 相关设计构想文件（conception/） | 说明 |
|--------------------------|-------------------------------|------|
| `1.Consensus-PoH.md` | `1.共识-历史证明（PoH）.md`<br>`2.共识-端点约定.md`<br>`附.交易.md` | 铸凭哈希算法、防护、出块规则、铸凭交易 |
| `2.Services(Third-party).md` | `3.公共服务.md`<br>`4.激励机制.md` | 服务架构、节点发现、数据驿站、经济激励 |
| `3.Evidence-Design.md` | `5.信用结构.md`<br>`附.交易.md` | 三种信元（币金、凭信、存证）的完整设计 |
| `4.Script-of-Stack.md` | `6.脚本系统.md` | 栈机制、指令集、执行环境 |
| `5.Transaction.md` | `附.交易.md`<br>`5.信用结构.md`<br>`6.脚本系统.md`<br>`附.组队校验.md` | 交易结构、信元编码、脚本验证机制、UTXO/UTCO 概念 |
| `6.Checks-by-Team.md` | `附.组队校验.md`<br>`2.共识-端点约定.md` | 校验团队架构、首领校验、冗余机制 |
| `blockchain-core.md` | `blockchain.md`<br>`1.共识-历史证明（PoH）.md`<br>`2.共识-端点约定.md` | 区块头结构、年块机制、分叉处理、链连续性 |
| `Instruction/*.md` | `Instruction/*.md` | 脚本指令集（见下文） |


### 正向关系总结

关键的对应规律：

> **注：**
> `7` => `blockchain-core.md`

| Conception 设计构想 | 相关的 Proposal | 说明 |
|---|---|---|
| `1.共识-历史证明（PoH）.md` | 1, 7 | PoH核心算法涉及共识和区块 |
| `2.共识-端点约定.md` | 1, 6, 7 | 出块规则、分叉处理、链维持 |
| `3.公共服务.md` | 2 | 直接支撑服务提案 |
| `4.激励机制.md` | 2 | 经济模型补充服务设计 |
| `5.信用结构.md` | 3, 5 | 信元定义和交易具体化 |
| `6.脚本系统.md` | 4, 5 | 脚本指令和交易验证 |
| `附.交易.md` | 1, 3, 5 | 交易结构和铸凭交易 |
| `附.组队校验.md` | 5, 6 | 主要支撑校验提案，但UTXO/UTCO概念也体现在交易提案 |
| `blockchain.md` | 7 | 区块链基础概念 |

> **说明：**
> - 逆向表：显示每个Proposal文件依赖的所有Conception文件（一个提案可对应多个构想）
> - 正向表：显示每个Conception文件被哪些Proposal引用（一个构想可支撑多个提案）
> - 关系分析基于 2026年3月份完整内容比对


### 脚本指令集

脚本指令集的设计构想位于 `conception/Instruction/` 目录下，开发提案位于 `proposal/Instruction/` 目录下：

对应的关系如下（严格对应，路径目录省略）：

| 设计构想文件 | 输出提案文件 |
|--------------|----------------|
| `0.基本约束.md` | `0.Base-Constraints.md` |
| `1.值指令.md` | `1.Value-Instructions.md` |
| `2.截取指令.md` | `2.Capture-Instructions.md` |
| `3.栈操作指令.md` | `3.Stack-Operations.md` |
| `4.集合指令.md` | `4.Collection-Operations.md` |
| `5.交互指令.md` | `5.Interaction-Instructions.md` |
| `6.结果指令.md` | `6.Result-Instructions.md` |
| `7.流程控制.md` | `7.Flow-Control.md` |
| `8.转换指令.md` | `8.Conversion-Instructions.md` |
| `9.运算指令.md` | `9.Arithmetic-Instructions.md` |
| `10.比较指令.md` | `10.Comparison-Instructions.md` |
| `11.逻辑指令.md` | `11.Logic-Instructions.md` |
| `12.模式指令.md` | `12.Pattern-Instructions.md` |
| `13.环境指令.md` | `13.Environment-Instructions.md` |
| `14.工具指令.md` | `14.Tool-Instructions.md` |
| `15.系统指令.md` | `15.System-Instructions.md` |
| `16.函数指令.md` | `16.Function-Instructions.md` |
| `17.模块指令.md` | `17.Module-Instructions.md` |
| `18.扩展指令.md` | `18.Extension-Instructions.md` |


## 实施方案（Plan）

由开发提案（proposal/*）生成实施方案时，输出存放于 `plan/` 之下。

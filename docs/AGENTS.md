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


### 「构想 => 提案」关系对应表

此为代理（AI Agent）维护「构想 => 提案」的对应关系表，确保每个提案都能追溯到相关的构想文件。

| 开发提案文件（proposal/） | 相关设计构想文件（conception/） | 说明 |
|--------------------------|-------------------------------|------|
| `blockchain-core.md` | `blockchain.md`<br>`1.共识-历史证明（PoH）.md`<br>`2.共识-端点约定.md`<br>`附.组队校验.md` | 区块头链、年块衔接、CheckRoot、初始主链验证、手动切链与链标识 |
| `1.Consensus-PoH.md` | `1.共识-历史证明（PoH）.md`<br>`2.共识-端点约定.md`<br>`blockchain.md`<br>`附.组队校验.md` | PoH 铸凭哈希、择优池、出块约定、区块竞争、分叉进入与裁决规则 |
| `2.Services(Third-party).md` | `README.md`<br>`3.公共服务.md`<br>`4.激励机制.md`<br>`附.交易.md` | 公共服务网络、Depots/Blockqs/stun2p、铸币时间表、交易费销毁、公共服务奖励与兑奖 |
| `3.Evidence-Design.md` | `5.信用结构.md`<br>`附.交易.md`<br>`blockchain.md` | 币金、凭信、存证、输出配置、附件 ID、公钥哈希、地址和签名模型 |
| `5.Transaction.md` | `附.交易.md`<br>`5.信用结构.md`<br>`blockchain.md`<br>`附.组队校验.md`<br>`4.激励机制.md` | 交易头/体、输入输出、Coinbase、UTXO/UTCO 指纹、交易存储与费用约束 |
| `6.Checks-by-Team.md` | `附.组队校验.md`<br>`附.交易.md`<br>`blockchain.md`<br>`1.共识-历史证明（PoH）.md`<br>`2.共识-端点约定.md` | 校验组角色、首领校验、冗余复核、铸造协作、区块发布、UTXO/UTCO 指纹 |
| `4.Script-of-Stack.md` | `6.脚本系统.md`<br>`Instruction/*.md`<br>`examples/*.md` | 栈式脚本系统设计相对独立，本轮按用户要求暂未修订 |

> **说明：**
> 如果有多个文件，每个文件引用用换行分隔。


### 脚本指令集

脚本指令集的设计构想位于 `conception/Instruction/` 目录下，开发提案位于 `proposal/Instruction/` 目录下：

此为严格对应关系（路径目录省略）：

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


### Proposal → Plan 关联表

此为代理（AI Agent）维护「提案 => 方案」的对应关系表，确保每个实施方案都能追溯到相关的提案文件。

| 开发提案文件（proposal/） | 相关实施方案文件（plan/） | 说明 |
|--------------------------|--------------------------|------|
| `blockchain-core.md` | `phase1-types-crypto.md`<br>`phase2-blockchain-core.md` | 基础类型/密码学 + 区块链核心结构 |
| `3.Evidence-Design.md` | `phase1-types-crypto.md`<br>`phase3-transaction-model.md` | 三种信元类型定义 + 交易输出结构 |
| `5.Transaction.md` | `phase3-transaction-model.md`<br>`phase4-utxo-utco-state.md` | 交易完整模型 + UTXO/UTCO 状态 |
| `6.Checks-by-Team.md` | `phase4-utxo-utco-state.md`<br>`phase7-checkteam-verification.md` | UTXO/UTCO 指纹结构 + 组队校验逻辑 |
| `1.Consensus-PoH.md` | `phase6-poh-consensus.md` | PoH 铸凭哈希、择优池、分叉解决 |
| `2.Services(Third-party).md` | `phase8-services-interfaces.md` | 第三方服务接口 + 铸造时间表 |
| `4.Script-of-Stack.md`<br>`Instruction/*.md` | `phase5-script-engine.md` | 脚本引擎完整实现（254 条操作码、执行环境、锁定/解锁管道） |

> **说明：**
> - `phase1-types-crypto.md` 被多个提案依赖（作为基础层）。
> - 关系分析基于 2026年4月份内容比对。

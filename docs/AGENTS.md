# docs 文件结构 & 关联指南

> 该文档由 AI Agent 更新，原则如下：
> - 当由设计构想创建（或修订）开发提案时（*Conception => Proposal*）。
> - 当由开发提案生成（或修订）实施方案时（*Proposal => Plan*）。
> - 对应务必完整，便于后期关联性查询修改，提升效率。

本文档在 `docs` 目录之下，因此文档文件相对于此目录，不再从项目根目录计算路径。

## 目录结构概览

- `Conception`: 设计构想，包含了作者对项目整体以及各个功能模块的设计构思。
- `Proposal`: 开发提案/技术规格（`Proposal / Technical Spec`），基于设计构想生成，为生成实施方案提供依据。
- `Plan`: 实施方案（`Implementation Plan`），包含了具体的开发计划、任务分配、时间表和测试与验证等。


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
| `proposal/00.Project-Scope.md` | `conception/README.md`<br>`conception/blockchain.md` | 第一阶段边界、文档层级、包分层。 |
| `proposal/01.Types-And-Encoding.md` | `conception/blockchain.md`<br>`conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/6.脚本系统.md` | 基础类型、整数、字节串、列表和结构体编码。 |
| `proposal/02.Cryptography-And-Hashing.md` | `conception/blockchain.md`<br>`conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/1.共识-历史证明（PoH）.md` | 密码学算法分配、Hash 输入、地址哈希、签名抽象。 |
| `proposal/03.Identifiers-And-Constants.md` | `conception/blockchain.md`<br>`conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/6.脚本系统.md`<br>`conception/1.共识-历史证明（PoH）.md` | 标识符长度、系统常量、时间高度换算。 |
| `proposal/04.Hash-Trees.md` | `conception/blockchain.md`<br>`conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/附.组队校验.md` | 哈希树、含序叶子、证明路径、状态指纹树。 |
| `proposal/05.Blockchain-Core.md` | `conception/blockchain.md`<br>`conception/附.交易.md` | 区块头链、入块验证、CheckRoot、年块、初始主链验证。 |
| `proposal/06.Transaction-Model.md` | `conception/附.交易.md`<br>`conception/blockchain.md`<br>`conception/5.信用结构.md` | 交易头、输入输出、签名消息、单签/多签、Coinbase 边界。 |
| `proposal/07.Coin-Credit-Proof-Units.md` | `conception/README.md`<br>`conception/5.信用结构.md`<br>`conception/附.交易.md` | Coin、Credit、Proof、附件 ID、Mediator、Custom 输出。 |
| `proposal/08.UTXO-UTCO-State.md` | `conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/blockchain.md` | UTXO/UTCO 状态、状态转移、状态指纹、状态同步与回滚。 |
| `proposal/09.Script-System.md` | `conception/6.脚本系统.md`<br>`conception/Instruction/*.md`<br>`conception/examples/*.md` | 脚本 VM、运行时空间、公共验证、私有监听、资源与成本边界。 |
| `proposal/10.PoH-Consensus.md` | `conception/1.共识-历史证明（PoH）.md`<br>`conception/2.共识-端点约定.md`<br>`conception/附.组队校验.md` | PoH 铸凭交易、铸凭哈希、择优池、同步池。 |
| `proposal/11.Endpoint-Conventions-And-Fork-Choice.md` | `conception/2.共识-端点约定.md`<br>`conception/1.共识-历史证明（PoH）.md`<br>`conception/附.组队校验.md` | 出块共约、快速转播、区块竞争、分叉选择、交易池共约。 |
| `proposal/12.Team-Validation.md` | `conception/附.组队校验.md`<br>`conception/2.共识-端点约定.md`<br>`conception/1.共识-历史证明（PoH）.md` | 校验组角色、首领校验、冗余复核、铸造协作、区块发布。 |
| `proposal/13.Public-Service-Interfaces.md` | `conception/README.md`<br>`conception/3.公共服务.md`<br>`conception/4.激励机制.md`<br>`conception/附.组队校验.md` | 节点发现、Depots、Blockqs、stun2p 的接口边界。 |
| `proposal/14.Incentives-And-Coinbase-Rewards.md` | `conception/4.激励机制.md`<br>`conception/3.公共服务.md`<br>`conception/附.组队校验.md`<br>`conception/附.交易.md` | 发行、交易费销毁、奖励分配、公共服务兑奖、成熟期。 |

> **说明：**
> 如果有多个文件，每个文件引用用换行分隔。


### 脚本指令集

脚本指令集的设计构想位于 `conception/Instruction/` 目录下，开发提案应存放于 `proposal/Instruction/` 目录下：

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

指令集 Proposal 已输出至 `proposal/Instruction/`，另含索引文件 `proposal/Instruction/README.md`。


### 架构决策记录（ADR）

本项目较大，考虑追加一层架构决策记录 ADR（Architecture Decision Records），存放在 `docs/adr` 目录下。


## 实施方案（Plan）

由开发提案（proposal/*）生成实施方案时，输出存放于 `plan/` 之下。


### 「Proposal => Plan」关系对应表

此为代理（AI Agent）维护「提案 => 方案」的对应关系表，确保每个实施方案都能追溯到相关的提案文件。

|  实施方案文件（plan/） | 相关的提案文件（proposal/） | 说明 |
|--------------------------|--------------------------|------|
| `plan/00-Implementation-Roadmap.md` | `proposal/00.Project-Scope.md`<br>`proposal/01.Types-And-Encoding.md`<br>`proposal/02.Cryptography-And-Hashing.md`<br>`proposal/03.Identifiers-And-Constants.md`<br>`proposal/04.Hash-Trees.md`<br>`proposal/05.Blockchain-Core.md`<br>`proposal/06.Transaction-Model.md`<br>`proposal/07.Coin-Credit-Proof-Units.md`<br>`proposal/08.UTXO-UTCO-State.md`<br>`proposal/09.Script-System.md`<br>`proposal/10.PoH-Consensus.md`<br>`proposal/11.Endpoint-Conventions-And-Fork-Choice.md`<br>`proposal/12.Team-Validation.md`<br>`proposal/13.Public-Service-Interfaces.md`<br>`proposal/14.Incentives-And-Coinbase-Rewards.md`<br>`proposal/Instruction/*.md` | 总体实施路线、分层架构、阶段门禁和全局验证命令。 |
| `plan/01-Foundation-Types-Crypto.md` | `proposal/00.Project-Scope.md`<br>`proposal/01.Types-And-Encoding.md`<br>`proposal/02.Cryptography-And-Hashing.md`<br>`proposal/03.Identifiers-And-Constants.md`<br>`proposal/04.Hash-Trees.md` | 基础类型、规范化编码、Hash/ID、签名抽象和通用哈希树。 |
| `plan/02-Blockchain-Core.md` | `proposal/05.Blockchain-Core.md` | 区块头、BlockID、头链、链身份、年块和 CheckRoot。 |
| `plan/03-Transaction-And-Units.md` | `proposal/06.Transaction-Model.md`<br>`proposal/07.Coin-Credit-Proof-Units.md`<br>`proposal/14.Incentives-And-Coinbase-Rewards.md` | 交易头、输入输出、Coin/Credit/Proof、签名消息、Coinbase 边界。 |
| `plan/04-UTXO-UTCO-State.md` | `proposal/08.UTXO-UTCO-State.md`<br>`proposal/04.Hash-Trees.md`<br>`proposal/06.Transaction-Model.md`<br>`proposal/07.Coin-Credit-Proof-Units.md` | UTXO/UTCO entry、引用解析、状态转移、状态指纹和快照。 |
| `plan/05-Script-System.md` | `proposal/09.Script-System.md`<br>`proposal/Instruction/*.md` | 脚本 VM、bytecode、运行时、公共/私有模式、指令注册表和分批指令。 |
| `plan/06-PoH-Consensus-And-Fork-Choice.md` | `proposal/10.PoH-Consensus.md`<br>`proposal/11.Endpoint-Conventions-And-Fork-Choice.md` | PoH MintHash、择优池、同步池、快速转播、区块竞争和分叉选择。 |
| `plan/07-Team-Validation-Services-Incentives.md` | `proposal/12.Team-Validation.md`<br>`proposal/13.Public-Service-Interfaces.md`<br>`proposal/14.Incentives-And-Coinbase-Rewards.md` | 校验组接口、公共服务接口、发行、奖励、兑奖和成熟期。 |
| `plan/08-Open-Questions-And-Acceptance.md` | `proposal/00.Project-Scope.md`<br>`proposal/01.Types-And-Encoding.md`<br>`proposal/02.Cryptography-And-Hashing.md`<br>`proposal/03.Identifiers-And-Constants.md`<br>`proposal/04.Hash-Trees.md`<br>`proposal/05.Blockchain-Core.md`<br>`proposal/06.Transaction-Model.md`<br>`proposal/07.Coin-Credit-Proof-Units.md`<br>`proposal/08.UTXO-UTCO-State.md`<br>`proposal/09.Script-System.md`<br>`proposal/10.PoH-Consensus.md`<br>`proposal/11.Endpoint-Conventions-And-Fork-Choice.md`<br>`proposal/12.Team-Validation.md`<br>`proposal/13.Public-Service-Interfaces.md`<br>`proposal/14.Incentives-And-Coinbase-Rewards.md`<br>`proposal/Instruction/*.md` | 全局未决项、测试覆盖、集成测试、依赖检查和最终验收标准。 |

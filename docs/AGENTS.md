# docs 文件结构 & 关联指南

> 该文档由 AI Agent 更新：
> - 当由构想创建（或修订）提案时：Conception => Proposal
> - 当由提案生成（或修订）实施方案时：Proposal => Plan

本文档描述了 `docs` 目录下*设计构想（Conception）*、*开发提案（Proposal）*和*实施方案（Plan）*的文件结构，以及它们之间的关联。

- `Conception`: 设计构想，包含了作者对项目整体以及各个功能模块的设计构思。
- `Proposal`: 开发提案，基于设计构想生成的具体开发方案和技术细节，为生成 `plan` 提供依据。
- `Plan`: 实施方案，包含了具体的开发计划、任务分配、时间表和测试与验证等。


## 设计构想（Conception）

位于 `conception/` 目录下，包含以下内容：

| 功能 | 设计构想文件 |
|------|-----------------|
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

对应文件包含：

| 功能 | 输出提案文件（proposal/） | 相关设计构想文件（conception/） |
|------|----------------------|-------------------------------|
| 共识 | `1.Consensus-PoH.md` | `1.共识-历史证明（PoH）.md`,<br> `2.共识-端点约定.md` |
| 服务 | `2.Services(Third-party).md` | `3.公共服务.md`,<br> `4.激励机制.md` |
| 信用 | `3.Evidence-Design.md` | `5.信用结构.md` |
| 脚本 | `4.Script-of-Stack.md` | `6.脚本系统.md` |
| 交易 | `5.Transaction.md` | `附.交易.md` |
| 校验 | `6.Checks-by-Team.md` | `附.组队校验.md` |
| 核心 | `blockchain-core.md` | `blockchain.md` |
| 脚本指令集 | `Instruction/*.md` | `Instruction/*.md` |

> **注意：**
> 上面的对应关系并非严格的一一对应，只是主体内容上对应。
> 一个设计构想文件可能会生成多个提案文件，反之一个提案文件可能由多个构想形成。


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

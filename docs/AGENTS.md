# docs 文件结构 & 关联指南

> 该文档由 AI Agent 更新，原则如下：
> - 当由设计构想创建（或修订）开发提案时（*Conception => Proposal*）。
> - 当由开发提案生成（或修订）实施方案时（*Proposal => Plan*）。
> - 对应务必完整，便于后期关联性查询修改，提升效率。

本文档在 `docs` 目录之下，因此文档文件相对于此目录，不再从项目根目录计算路径。

## 目录结构概览

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
| 待定 | 待定 | 待定 |

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


## 实施方案（Plan）

由开发提案（proposal/*）生成实施方案时，输出存放于 `plan/` 之下。


### 「Proposal => Plan」关系对应表

此为代理（AI Agent）维护「提案 => 方案」的对应关系表，确保每个实施方案都能追溯到相关的提案文件。

|  实施方案文件（plan/） | 相关的提案文件（proposal/） | 说明 |
|--------------------------|--------------------------|------|
| 待定 | 待定 | 待定 |

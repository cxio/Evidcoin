# docs 文件结构 & 关联指南

本文档在 `docs` 目录之下，因此文档文件相对于此目录，不再从项目根目录计算路径。

## 目录结构概览

Evidcoin 文档的正式结构收敛为两层：

| 层级 | 目录 | 说明 |
|------|------|------|
| Conception | `conception/` | 设计构想，包含作者对协议、系统和应用边界的原始设计。 |
| Decision | `decision/` | 架构决策，仅记录 conception 尚未明确的补充决策。 |

`proposal/` 与 `plan/` 当前为待重构材料，不作为新的正式决策依据。后续重构时应从 `conception/` 与 `decision/` 重新生成。


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

位于 `decision/` 目录。Decision 的作用是补充 conception 尚未直接固定的规范化细节，例如字节编码、哈希域隔离、极端边界、字段宽度和实现路径。

维护规则：

- 新增 Decision 前必须先检查 `conception/` 是否已经明确该规则。
- 若 conception 已明确，直接引用 conception，不新增 Decision。
- 若后续 conception 修订吸收了某个 Decision，应删除或标记该 Decision 已被吸收。
- Decision 文件命名为 `DECISION-NNNN-<short-description>.md`。

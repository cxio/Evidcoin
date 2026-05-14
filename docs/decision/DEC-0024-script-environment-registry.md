# DEC-0024: Script Environment Registry（脚本环境注册表）

Status: Proposed

## Context

conception 的 `ENV`、`IN`、`OUT`、`INOUT`、`CREDIT` 等环境指令使用名称说明目标条目，但链上附参需要稳定的数值标识。

## Decision

建议建立脚本环境注册表：

- 每个环境指令拥有独立的 `uint8` 条目标识空间。
- 文档名称只用于文本脚本和调试展示，链上编码使用数值标识。
- 已在 conception 中列出的名称进入保留集合；新增名称必须追加，不得复用。
- 未知标识在共识验证中失败，不得被忽略。
- 私有扩展不得占用基础环境标识空间。

## Rationale

数值注册表避免不同语言实现对字符串大小写、编码和排序产生差异。

## Consequences

需要补充一份正式注册表表格。本 DEC 在表格冻结前保持 Proposed。

## Conception references

- `docs/conception/Instruction/13.环境指令.md`
- `docs/conception/6.脚本系统.md`

## Open questions

- 各名称的具体数值分配。
- `Timestamp`、`BlockTime` 等时间名称是否统一单位展示。
- 环境查询失败时是返回 NIL 还是脚本失败，需逐项定义。

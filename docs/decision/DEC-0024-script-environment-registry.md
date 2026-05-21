# DEC-0024: Script Registry Spaces（脚本注册表空间）

Status: Proposed

## Context

conception 的 `ENV`、`IN`、`OUT`、`INOUT`、`CREDIT` 等环境指令使用名称说明目标条目，但链上附参需要稳定的数值标识。类似问题也存在于 `XFROM`、`SIGNED`、`SOURCE`、`VALUE`、哈希算法名、字符串格式名、模式标识、函数和模块编号。

## Decision

建议建立脚本注册表空间：

- 每类带名称或别名的附参拥有独立的数值空间，默认宽度按对应指令附参定义。
- 文档名称只用于文本脚本和调试展示，链上编码使用数值标识。
- 已在 conception 中列出的名称进入保留集合；新增名称必须追加，不得复用。
- 未知标识在共识验证中失败，不得被忽略。
- 私有扩展不得占用基础环境、函数或模块标识空间。
- 第一批需冻结的空间至少包括：`ENV`、`IN`、`OUT`、`INOUT`、`CREDIT`、`XFROM`、`SIGNED`、`SYS_TIME`、`SOURCE`、`VALUE`、`FN_X`、`MO_XX`、`EXT_MO`、哈希算法名、字符串格式名和模式标识。
- `ENV{Timestamp}` 与 `SYS_TIME{Stamp}` 都依赖节点当前实际时间，不得用于公共验证共识路径，除非 conception 后续定义为区块推导时间。

## Rationale

数值注册表避免不同语言实现对字符串大小写、编码和排序产生差异。扩大注册表范围可避免只有 `ENV` 被规范而其它别名型附参继续分叉。

## Consequences

需要补充一份正式注册表表格。本 DEC 在表格冻结前保持 Proposed。脚本公共验证实现需要同时拦截 `SYS_TIME` 和 `ENV{Timestamp}` 这类非确定性时间源。

## Conception references

- `docs/conception/Instruction/13.环境指令.md`
- `docs/conception/Instruction/14.工具指令.md`
- `docs/conception/Instruction/15.系统指令.md`
- `docs/conception/Instruction/16.函数指令.md`
- `docs/conception/Instruction/17.模块指令.md`
- `docs/conception/Instruction/18.扩展指令.md`
- `docs/conception/6.脚本系统.md`

## Open questions

- 各名称的具体数值分配。
- `Timestamp`、`BlockTime` 等时间名称是否统一单位展示。
- 环境查询失败时是返回 NIL 还是脚本失败，需逐项定义。
- `ENV{Timestamp}` 是否应改名或改义，以避免与公共验证禁用的 `SYS_TIME` 重叠。

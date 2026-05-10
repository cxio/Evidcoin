# DEC-0017: 脚本 VM Float 确定性

## Status（状态）

Accepted

## Context（背景）

`conception/6.脚本系统.md` 将 `Float` 列为脚本数据类型，但未固定精度、舍入和 NaN 行为。浮点结果若参与公共验证，跨平台差异可能造成不同节点得出不同验证结果。

## Decision（决策）

脚本 VM 的 `Float` 类型采用 IEEE 754 binary64，即 Go `float64`。

运行规则：

- 舍入模式为 round-to-nearest-even。
- 所有 NaN 规范化为 quiet NaN，位型为 `0x7ff8000000000000`。
- 禁止 signaling NaN 进入数据栈、实参区或变量域。
- `+Inf` 和 `-Inf` 保留 IEEE 754 语义。

公共验证约束：

- Float 可作为中间值存在。
- Float 不得作为 `PASS` 或 `CHECK` 的直接实参。
- 若公共验证路径中 Float 直接进入 `PASS` 或 `CHECK`，验证失败。

## Rationale（理由）

- binary64 在主流平台上行为稳定，足以支持脚本中的数值表达。
- NaN 规范化消除最常见的跨平台差异。
- 禁止 Float 直接决定 pass 状态，可降低共识分歧风险。

## Consequences（影响）

- VM 实现需在 Float 入栈或运算后规范化 NaN。
- 测试应覆盖 NaN、Inf、普通浮点运算和 Float 进入 `CHECK` 的拒绝路径。

## Conception Relationship（与构想关系）

- 补充 `Float` 的确定性运行语义。
- 不改变脚本系统支持 `Float` 的 conception 设定。

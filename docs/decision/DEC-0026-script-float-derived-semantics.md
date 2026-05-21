# DEC-0026: Script Float Derived Semantics（脚本 Float 派生语义）

Status: Proposed

## Context

conception 的转换指令允许多种类型转为 Float，运算指令也会将数值转换到 Float 统一计算。旧限制若禁止部分输入，会与 conception 冲突。

构想层当前 `BOOL` 转换没有列出 Float；旧 Decision 中关于 `BOOL` 极小 Float 的问题不应继续作为本 DEC 的开放项，除非作者先修订 conception。

## Decision

建议只补充边界，不改变 conception 的转换主体：

- `FLOAT` 接受 conception 已列类型：Nil、Bool、Byte、Rune、Int 和合法字符串。
- `BigInt` 转 Float 是否允许尚待裁决；若允许，超出精确表达范围时必须定义舍入规则。
- Float 运算后的 NaN/Inf 传播、失败或规范化处理按 DEC-0022 最终裁决；在裁决前不得固定为拒绝或允许。
- Float 到 Int、Byte、Rune 的转换按 conception 的截断或取整语义，但越界必须失败。
- Float 到整数的负数取整方向需冻结，建议采用 toward zero 以匹配“截断小数部分”的表述。
- 字符串转 Float 使用固定语法子集，不依赖本地化、小数逗号或平台解析差异。
- `POW`、表达式运算和 `MO_MATH` 的返回类型与异常处理需要同一 Float profile 约束。

## Rationale

该 DEC 修正旧方案与 conversion 指令的冲突，将输入限制降级为确定性边界。移除 `BOOL Float` 开放项可避免 Decision 擅自扩大构想层类型转换范围。

## Consequences

脚本 VM 需要自带或固定浮点解析规则，不能直接依赖可能随语言版本变化的宽松解析。

## Conception references

- `docs/conception/Instruction/8.转换指令.md`
- `docs/conception/Instruction/9.运算指令.md`
- `docs/conception/Instruction/17.模块指令.md`

## Open questions

- `BigInt` 是否可转 Float。
- NaN/Inf 作为运算中间值或结果时的公共验证语义。
- 字符串 Float 语法是否完全采用 Go `strconv.ParseFloat` 的合法子集。
- Float 到 Int、Byte、Rune 的负数取整方向是否采用 toward zero。

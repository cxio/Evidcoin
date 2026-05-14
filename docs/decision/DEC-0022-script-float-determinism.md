# DEC-0022: Script Float Determinism（脚本浮点确定性）

Status: Accepted

## Context

conception 已明确脚本 `Float` 为 `float64`，值指令固定 8 字节，运算内部转换到 Float 统一计算。Decision 只补充跨平台确定性边界。

## Decision

- `Float` 使用 IEEE 754 binary64。
- 链上字节编码使用 big-endian 的 64 位位型。
- 共识执行不得启用平台相关扩展精度或非确定性舍入模式。
- NaN 输入和运算结果在共识脚本中无效。
- `+Inf`、`-Inf` 作为运算结果无效；是否允许字面量编码为无效值由字节编码 DEC 统一处理。
- `-0.0` 在比较中等于 `+0.0`，规范编码应归一为 `+0.0`。

## Rationale

float64 主体已被 conception 吸收；这里只记录共识实现必须一致的边界。

## Consequences

脚本 VM 需要在每次 Float 运算和转换后检查 NaN/Inf。

## Conception references

- `docs/conception/Instruction/1.值指令.md`
- `docs/conception/Instruction/8.转换指令.md`
- `docs/conception/Instruction/9.运算指令.md`

## Open questions

无。

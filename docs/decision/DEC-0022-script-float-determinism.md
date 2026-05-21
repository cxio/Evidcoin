# DEC-0022: Script Float Determinism（脚本浮点确定性）

Status: Proposed

## Context

conception 已明确脚本 `Float` 为 `float64`，值指令固定 8 字节，运算内部转换到 Float 统一计算。Decision 只补充跨平台确定性边界。

构想层当前实际异常浮点检测指令为 `ISEFV`，不是旧草案中的 `ISNAN`。

## Decision

- `Float` 使用 IEEE 754 binary64。
- 链上字节编码使用 big-endian 的 64 位位型。
- 共识执行不得启用平台相关扩展精度或非确定性舍入模式。
- NaN 与 `+Inf`、`-Inf` 的字面量编码、运算传播和公共验证处理仍需冻结；在作者裁决前，实现不能把禁止或允许任一方案作为最终共识规则。
- `-0.0` 在比较中等于 `+0.0`，规范编码应归一为 `+0.0`。
- 异常浮点检测指令名称按 conception 当前指令表为 `ISEFV`。

## Rationale

float64 主体已被 conception 吸收；这里只记录共识实现必须一致的边界。由于 conception 已定义 `ISEFV` 并说明浮点计算可能出现 NaN 或 Inf，本 DEC 不能直接禁止异常值，而应先保留为待裁决边界。

## Consequences

脚本 VM 需要在每次 Float 运算和转换后按最终裁决处理 NaN/Inf。所有使用 `ISNAN` 名称的旧草案和测试说明应改为 `ISEFV` 或废弃。

## Conception references

- `docs/conception/Instruction/1.值指令.md`
- `docs/conception/Instruction/8.转换指令.md`
- `docs/conception/Instruction/9.运算指令.md`
- `docs/conception/Instruction/10.比较指令.md`

## Open questions

- NaN 是否允许作为运行时值，或仅允许由 `ISEFV` 检测后被脚本显式处理。
- `+Inf`、`-Inf` 是否允许作为中间值、最终结果或字面量编码。
- NaN payload 是否必须规范化，若规范化应采用哪个位型。

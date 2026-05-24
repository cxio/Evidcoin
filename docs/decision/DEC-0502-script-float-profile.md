# DEC-0502: Script Float Profile（脚本浮点配置）

Status: Proposed

## Context（背景）

Conception 允许脚本使用 `Float`，并说明字面量不支持 NaN/Inf，但运算中可能产生异常浮点，可用 `ISEFV` 检测。浮点跨实现确定性、字节编码、取整、比较和异常传播仍需冻结。

## Decision（决策）

建议 profile：

- `Float` 使用 IEEE 754 binary64。
- 字节编码使用 8 字节大端 bit pattern。
- 输入字面量不得表达 NaN、+Inf、-Inf。
- 运算产生 NaN 或 Inf 时不立即崩溃，由 `ISEFV` 检测；但进入公共验证最终 PASS 前若仍存在异常浮点，验证失败。
- `-0.0` 在数值比较中等于 `+0.0`，但字节编码保持原 bit pattern。
- `Float -> Int` 默认向零截断。
- `Float -> String` 使用固定规范格式，不依赖本地 locale。

建议比较规则：

- 任一操作数为 NaN 时，除 `ISEFV` 外的比较返回 `false`。
- `EQUAL(+0.0, -0.0)` 返回 `true`。
- 排序类比较对 NaN 返回失败状态，而不是任意排序。

## Rationale（理由）

完全禁止浮点会削弱脚本表达力；保留 binary64 并冻结异常语义，可在可用性和确定性之间折中。异常浮点不应静默通过公共验证。

## Consequences（影响）

- 需要跨平台测试向量覆盖 NaN、Inf、-0.0、舍入和字符串格式。
- 共识实现不得使用会受 CPU 或编译器 fast-math 影响的非标准行为。
- 高精度整数逻辑应使用 `Int` 或 `BigInt`，不要用 `Float` 表达金额。

## Conception References（构想层依据）

- `docs/conception/6.脚本系统.md`
- `docs/conception/Instruction/8.转换指令.md`
- `docs/conception/Instruction/9.运算指令.md`
- `docs/conception/Instruction/10.比较指令.md`

## Open Questions（开放问题）

- `Float -> String` 的精确格式，是最短 round-trip 还是固定小数/科学计数法。
- `POW`、除零、溢出产生异常浮点时是保留值还是立即失败。
- `PACK` 和 `BYTES` 对 `Float` 是否直接输出 IEEE 754 bit pattern。

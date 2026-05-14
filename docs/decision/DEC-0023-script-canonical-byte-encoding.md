# DEC-0023: Script Canonical Byte Encoding（脚本规范字节编码）

Status: Proposed

## Context

conception 已定义脚本是编译后的指令码值序列，并为值指令、附参和关联数据给出描述。仍需固定唯一字节编码。

## Decision

建议如下：

- 每条指令以 1 字节 opcode 开始。
- 固定长度附参紧随 opcode。
- 变长附参使用 DEC-0001 的 varint。
- 关联数据使用长度前缀加原始字节，长度不包含 opcode 与附参。
- `Int` 使用 DEC-0001 的有符号 varint。
- `BigInt` 使用最短大端补码或无符号幅值表示尚待裁决。
- `Float` 使用 DEC-0022 的 big-endian binary64，并拒绝非规范零值、NaN 和 Inf。
- 人类可读脚本文本不是共识编码，不参与 TxID。

## Rationale

统一字节编码是脚本哈希、脚本模式匹配和交易体编码的前提。

## Consequences

脚本编译器必须拒绝同义文本产生的非规范字节序列。该 DEC 冻结前，脚本测试向量只能作为草案。

## Conception references

- `docs/conception/6.脚本系统.md`
- `docs/conception/Instruction/1.值指令.md`

## Open questions

- `BigInt` 的负数编码形式。
- 字符串转义只影响文本编译，是否需要单独规范。
- `Script` 类型的字节克隆是否允许携带未规范化脚本。

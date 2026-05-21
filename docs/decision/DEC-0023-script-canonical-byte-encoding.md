# DEC-0023: Script Canonical Byte Encoding（脚本规范字节编码）

Status: Proposed

## Context

conception 已定义脚本是编译后的指令码值序列，并为值指令、附参和关联数据给出描述。仍需固定唯一字节编码。

构想层中的指令结构为 `opcode + 附参 + 关联数据`。对 `DATA{}(~)`、`CODE{}(~)` 等变长指令，关联数据长度已经由附参表达；Decision 不应额外引入第二个长度前缀。

## Decision

建议如下：

- 每条指令以 1 字节 opcode 开始。
- 固定长度附参紧随 opcode。
- 变长附参使用 DEC-0001 的 varint，并作为后续关联数据的长度或结构参数。
- 关联数据不再额外添加通用长度前缀；具体长度由该指令的附参定义。
- `Int` 使用 DEC-0001 的有符号 varint。
- `Rune` 固定为 4 字节 Unicode code point，建议 big-endian；该点待作者确认。
- `BigInt` 使用最短大端补码或无符号幅值表示尚待裁决。
- `Float` 使用 DEC-0022 的 big-endian binary64；`-0.0` 应规范为 `+0.0`。NaN/Inf 的编码合法性按 DEC-0022 最终裁决，在裁决前不得固定为拒绝或允许。
- `MODEL{}(2)` 等复用附参高位的指令必须在指令级规范中固定有效位宽和标志位含义。
- 人类可读脚本文本不是共识编码，不参与 TxID；文本别名到数值附参的映射需由注册表冻结。

## Rationale

统一字节编码是脚本哈希、脚本模式匹配和交易体编码的前提。避免额外长度前缀可保持与 conception 的指令结构一致。

## Consequences

脚本编译器必须拒绝同义文本产生的非规范字节序列。该 DEC 冻结前，脚本测试向量只能作为草案。

## Conception references

- `docs/conception/6.脚本系统.md`
- `docs/conception/Instruction/0.基本约束.md`
- `docs/conception/Instruction/1.值指令.md`
- `docs/conception/Instruction/12.模式指令.md`

## Open questions

- `BigInt` 的负数编码形式。
- `Rune` 的字节序。
- NaN/Inf 字面量编码是否允许，以及若允许是否需要规范化 payload。
- 字符串转义、UTF-8 非法序列和正则文本只影响文本编译，是否需要单独规范。
- `Script` 类型的字节克隆是否允许携带未规范化脚本。

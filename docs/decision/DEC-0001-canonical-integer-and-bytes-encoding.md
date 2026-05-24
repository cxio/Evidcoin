# DEC-0001: Canonical Integer and Bytes Encoding（规范整数与字节编码）

Status: Proposed

## Context（背景）

Conception 多处使用变长整数、固定宽度整数和长度受限字节序列，但尚未冻结统一字节序、变长整数格式、负数编码和非规范编码的拒绝规则。

## Decision（决策）

建议采用以下规范：

- 固定宽度整数统一使用大端序。该规则适用于 `uint16`、`uint32`、`uint64`、`int64` 和固定宽度脚本附参。
- 无符号变长整数使用 ULEB128：每字节低 7 位承载数据，高位为续位，低位组先出现。
- 变长整数必须使用最短编码。`0` 编码为 `0x00`，任何带冗余高位零组的编码非法。
- 协议字段若需要有符号变长整数，必须单独在字段 DEC 中声明 ZigZag 或固定宽度编码；默认不得使用有符号 varint。
- 字节序列默认编码为 `varint(length) || bytes`；固定长度字段不带长度前缀。
- 受限长度字段先按编码长度检查，再按语义长度检查。超过 conception 限制的字段非法。

## Rationale（理由）

大端固定宽度编码便于人工审查和跨语言实现。ULEB128 简单、成熟，且能高效表达输出序位、年度、附件大小等小整数。最短编码规则可避免同一数据拥有多个 TxID 或 BlockID。

## Consequences（影响）

- 所有参与哈希或签名的结构必须使用规范编码。
- 解码器必须拒绝非最短 varint，而不是容忍后再重编码。
- 若后续某字段确需小端、定长或 ZigZag，需要在对应 DEC 中显式覆盖本规则。

## Conception References（构想层依据）

- `docs/conception/附.交易.md`
- `docs/conception/5.信用结构.md`
- `docs/conception/6.脚本系统.md`
- `docs/conception/Instruction/0.基本约束.md`

## Open Questions（开放问题）

- 脚本 `BigInt` 的负数编码是否采用二进制补码、符号位加绝对值，或独立类型标签。
- 交易年度字段是否使用真实年份数值，还是从创世年起算的无符号偏移量。

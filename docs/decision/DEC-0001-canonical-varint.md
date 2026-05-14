# DEC-0001: Canonical Varint（规范变长整数）

Status: Accepted

## Context

conception 多处使用变长整数表达高度、年度、输出序位、金额和脚本长度，但未固定唯一字节编码。不同实现若接受多种等价编码，会导致 TxID、BlockID、脚本字节序列和状态指纹不一致。

## Decision

- 无符号整数使用 ULEB128，每字节低 7 位承载数据，最高位表示后续字节，最低有效组在前。
- 有符号整数使用 ZigZag 后再按 ULEB128 编码。
- 编码必须最短；任何非最短编码无效。
- 解码结果必须落入字段声明的取值范围，溢出无效。
- 字节序列长度字段计数的是紧随其后的原始字节数，不包含长度字段自身。

## Rationale

ULEB128 易实现且与 conception 中“多数下标只需 1 字节”的空间目标一致。最短编码消除同一数值的多重表示。

## Consequences

所有共识哈希、签名消息和脚本字节编码必须拒绝非规范 varint。实现需要为每个字段分别检查范围。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/5.信用结构.md`
- `docs/conception/Instruction/1.值指令.md`

## Open questions

无。

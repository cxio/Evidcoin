# DECISION-0003: 规范化无符号 Varint 编码

## Status（状态）

Accepted

## Context（背景）

`docs/conception` 多处使用变长整数表达年度、输出下标、长度等字段，但没有固定具体 varint 算法。若不同实现采用不同编码，将影响规范化字节、哈希输入和跨实现互操作。

## Decision（决策）

无符号 varint 采用 LEB128，即 Protocol Buffers 风格的 base-128 varint。

每个字节低 7 位存储数据，最高位为延续标志：

- `1` 表示后续还有字节。
- `0` 表示当前字节为最后一字节。

示例：

```text
0      -> 00
127    -> 7f
128    -> 80 01
16383  -> ff 7f
16384  -> 80 80 01
```

规范化要求：

- 必须使用最短编码。
- 解码方必须拒绝非最短编码。
- `uint64` 最大值最多使用 10 字节。

## Rationale（理由）

- LEB128 有成熟实现和测试向量。
- Go 标准库 `encoding/binary` 已提供对应基础能力。
- 最短编码约束可避免同一数值存在多个合法字节表示。

## Consequences（影响）

- 所有参与哈希或签名的规范化编码都必须拒绝非最短 varint。
- 字节级测试应覆盖边界值和非最短编码拒绝。

## Conception Relationship（与构想关系）

- 补充 conception 中“变长整数”的具体字节编码。
- 不新增 conception 未定义的字段。

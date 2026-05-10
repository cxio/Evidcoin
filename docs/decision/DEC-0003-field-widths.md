# DEC-0003: 区块头与交易头字段宽度

## Status（状态）

Accepted

## Context（背景）

`conception/blockchain.md` 和 `conception/附.交易.md` 已给出区块头与交易头字段，但多数字段以 `int` 等示意类型表达。为了生成稳定的 BlockID 和 TxID，需要固定字段宽度、顺序和字节序。

## Decision（决策）

区块头规范化编码：

| 字段 | 类型 | 宽度 | 字节序 |
|------|------|------|--------|
| `Version` | `uint32` | 4 字节 | big-endian |
| `Height` | `uint32` | 4 字节 | big-endian |
| `PrevBlock` | `[48]byte` | 48 字节 | 原样 |
| `CheckRoot` | `[48]byte` | 48 字节 | 原样 |
| `Stakes` | `uint64` | 8 字节 | big-endian |
| `YearBlock` | `[48]byte` | 48 字节 | 原样 |

区块头字段顺序：

```text
Version || Height || PrevBlock || CheckRoot || Stakes || YearBlock
```

区块头总长度为 112 或 160 字节，非年块边界的 `YearBlock` 省略，节省 48 字节。

交易头规范化编码：

| 字段 | 类型 | 宽度 | 字节序 |
|------|------|------|--------|
| `Version` | `uint16` | 2 字节 | big-endian |
| `HashInputs` | `[32]byte` | 32 字节 | 原样 |
| `HashOutputs` | `[32]byte` | 32 字节 | 原样 |
| `Timestamp` | `int64` | 8 字节 | big-endian |

交易头字段顺序：

```text
Version || HashInputs || HashOutputs || Timestamp
```

交易头总长度为 74 字节。

其它未专项固定的整数字段默认使用 `DEC-0001` 的无符号 varint，除非其语义要求有符号整数。

## Rationale（理由）

- 固定宽度头部便于直接计算 BlockID 和 TxID。
- 区块高度使用 `uint32` 已可覆盖远超百年的区块数量。
- 交易版本使用 `uint16` 足以表达格式演进。
- 时间戳使用 `int64` 毫秒与 conception 交易时间戳一致。

## Consequences（影响）

- BlockID 和 TxID 测试向量必须使用上述规范化编码。
- 修改任一字段宽度、顺序或字节序都属于破坏性协议变更。

## Conception Relationship（与构想关系）

- 固定 conception 示意结构的字节级编码。
- 不改变区块头和交易头包含的字段集合。

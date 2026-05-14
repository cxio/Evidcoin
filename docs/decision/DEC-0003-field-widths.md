# DEC-0003: Field Widths（字段宽度）

Status: Proposed

## Context

conception 的区块头已使用固定宽度字段；交易头仍以示意类型表达，签名消息和交易体编码会受字段宽度影响。

## Decision

建议采用以下字段宽度，待作者裁决后冻结：

| 字段 | 类型 | 编码 |
|------|------|------|
| `BlockHeader.Version` | `uint32` | big-endian |
| `BlockHeader.Height` | `uint32` | big-endian |
| `BlockHeader.PrevBlock` | `[48]byte` | 原始字节 |
| `BlockHeader.CheckRoot` | `[48]byte` | 原始字节 |
| `BlockHeader.Stakes` | `uint64` | big-endian |
| `BlockHeader.YearBlock` | `[48]byte` | 年块高度存在，否则全零 |
| `TxHeader.Version` | `uint16` | big-endian |
| `TxHeader.HashInputs` | `[32]byte` | 原始字节 |
| `TxHeader.HashOutputs` | `[32]byte` | 原始字节 |
| `TxHeader.Timestamp` | `int64` | big-endian Unix 毫秒 |

## Rationale

区块头宽度跟随 conception。交易版本保留 `uint16` 是因为交易头版本空间足够且可减少 TxHeader 体积，但该点需要与签名消息一起裁决。

## Consequences

在 Proposed 冻结前，不得发布最终 TxID、签名消息或交易测试向量。旧 Decision 中交易版本宽度冲突已不沿用。

## Conception references

- `docs/conception/blockchain.md`
- `docs/conception/附.交易.md`

## Open questions

- `TxHeader.Version` 是否最终采用 `uint16`。
- `YearBlock` 在非年块高度是否编码全零，或从区块头规范编码中省略。

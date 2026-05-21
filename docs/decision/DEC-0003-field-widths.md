# DEC-0003: Field Widths（字段宽度）

Status: Proposed

## Context

conception 的区块头已使用固定宽度字段；交易头仍有一处内部冲突：`docs/conception/附.交易.md` 使用 `Timestamp int64`，而 `docs/conception/blockchain.md` 的创世 Coinbase 示例写作 `Timestamp uint64`。在作者修正前，Decision 只能给出建议编码，并标记为 Proposed。

最近 conception 已新增 `MintPKHash`，并明确 Coinbase 省略 `HashInputs`。

## Decision

建议采用以下字段宽度，待作者裁决后冻结：

| 字段 | 类型 | 编码 |
|------|------|------|
| `BlockHeader.Version` | `uint32` | big-endian |
| `BlockHeader.Height` | `uint32` | big-endian |
| `BlockHeader.PrevBlock` | `[48]byte` | 原始字节 |
| `BlockHeader.CheckRoot` | `[48]byte` | 原始字节 |
| `BlockHeader.Stakes` | `uint64` | big-endian，单位为 `chx * hour` |
| `BlockHeader.YearBlock` | `[48]byte` | 仅 `height % 87661 == 0` 时编码；创世块编码全零 |
| `TxHeader.Version` | `uint16` | big-endian |
| `TxHeader.HashInputs` | `[32]byte` | 普通交易编码；Coinbase 省略 |
| `TxHeader.HashOutputs` | `[32]byte` | 原始字节 |
| `TxHeader.Timestamp` | `int64` | big-endian Unix 毫秒，暂按 `附.交易.md` |
| `TxHeader.MintPKHash` | `[32]byte` | 可选；空值时省略 |
| `TxHeader.BlockHeight` | `uint32` | 仅 Coinbase 编码 |
| `TxHeader.Minter` | `MintProof` | 仅 Coinbase 编码；创世块可省略 |
| `TxHeader.FreeData` | `varbytes` | 仅 Coinbase 编码，长度 `<256` |

## Rationale

区块头宽度跟随 conception。`YearBlock` 采用“非年块省略”可保持普通区块头为固定 112 字节。交易头暂按 `附.交易.md` 的 `int64` 时间戳，以兼容普通交易的有符号时间表达；创世 Coinbase 示例的 `uint64` 已列入构想层待修正清单。

## Consequences

在 Proposed 冻结前，不得发布最终 TxID、签名消息或创世交易测试向量。Coinbase 的 TxID 必须按“省略 `HashInputs`”重新生成，不得使用旧占位哈希。

## Conception references

- `docs/conception/blockchain.md`
- `docs/conception/附.交易.md`

## Open questions

- `Timestamp int64` 与 `Timestamp uint64` 的构想层冲突需作者修正。
- `MintProof` 的字段级字节编码仍需在 Coinbase 规范中冻结。

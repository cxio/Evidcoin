# DEC-0003: Block and Transaction Field Encoding（区块与交易字段编码）

Status: Proposed

## Context（背景）

Conception 已给出 `BlockHeader` 与 `TxHeader` 的伪结构，但字段存在可选项、Coinbase 特例和若干宽度待冻结问题。字段编码会直接影响 BlockID、TxID、签名消息和验证路径。

## Decision（决策）

建议按以下顺序编码区块头：

1. `Version uint32`
2. `Height uint32`
3. `PrevBlock [48]byte`
4. `CheckRoot [48]byte`
5. `Stakes uint64`
6. `YearBlock [48]byte`，仅年块高度编码

建议按以下顺序编码普通交易头：

1. `Version uint16`
2. `HashInputs [32]byte`
3. `HashOutputs [32]byte`
4. `Timestamp int64`
5. `MintPKHash optional [32]byte`

建议按以下顺序编码 Coinbase 交易头：

1. `Version uint16`
2. `HashOutputs [32]byte`
3. `Timestamp int64`
4. `MintPKHash [32]byte`
5. `BlockHeight uint32`
6. `Minter optional MintProof`
7. `FreeData bytes<256>`

可选字段编码建议：

- `MintPKHash` 在普通交易中为空时省略，并由一个字段 presence 位图标识。
- `YearBlock` 在非年块中完全省略，不编码全零占位。
- `Minter` 在创世 Coinbase 中省略，并由 Coinbase profile 标识。

## Rationale（理由）

字段顺序采用 conception 伪代码顺序，降低实现与文档偏差。非年块省略 `YearBlock` 可保持区块头常规尺寸较小。Coinbase 省略 `HashInputs` 已由 conception 明确，应在编码层直接体现。

## Consequences（影响）

- 冻结前生成的 TxID 或 BlockID 不能作为长期测试向量。
- 可选字段必须有唯一 presence 规则，不能依赖全零值隐式判断。
- Coinbase 与普通交易头必须使用不同解析 profile，避免字段错位。

## Conception References（构想层依据）

- `docs/conception/blockchain.md#区块头`
- `docs/conception/blockchain.md#创世交易coinbase`
- `docs/conception/附.交易.md#交易头`

## Open Questions（开放问题）

- `TxHeader.Timestamp` 是否最终固定为 `int64`，以及是否允许负值。
- `YearBlock` 的年块判定是否严格为 `height % 87661 == 0`。
- presence 位图放在交易头起始处、版本后，还是由 profile 外部指定。

# DEC-0012: Coinbase HashInputs 计算

## Status（状态）

Accepted

## Context（背景）

`conception/附.交易.md` 已明确 Coinbase 是没有输入源的特殊交易，并要求 Coinbase 位于区块交易序列首位。交易头仍包含 `HashInputs`，因此 Coinbase 需要确定性的输入哈希占位规则。

## Decision（决策）

Coinbase 的 `HashInputs` 使用区块高度计算：

```text
HashInputs = BLAKE3-256(DomainTag("CoinbaseInputs") || uint64_be(blockHeight))
```

其中：

- `DomainTag("CoinbaseInputs")` 按 `DEC-0002` 生成。
- `blockHeight` 使用 8 字节 big-endian `uint64`。
- 输出为 32 字节。

## Rationale（理由）

- Coinbase 没有真实输入，不能复用普通输入哈希树。
- 绑定区块高度可避免同构 Coinbase 在不同高度产生相同输入根。
- 使用 domain tag 可避免与普通交易输入哈希混淆。

## Consequences（影响）

- Coinbase TxID 生成必须使用此特殊 `HashInputs`。
- 测试应覆盖高度 `0`、高度 `1` 和大高度值。

## Conception Relationship（与构想关系）

- 补充 Coinbase 无输入源时的交易头字段计算。
- 不改变 Coinbase 特殊交易和首位收录约定。

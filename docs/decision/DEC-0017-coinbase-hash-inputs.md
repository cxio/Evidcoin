# DEC-0017: Coinbase Hash Inputs（Coinbase 输入哈希）

Status: Proposed

## Context

conception 明确 Coinbase 没有输入源，但交易头仍包含 `HashInputs`。需要为 Coinbase 的输入根提供确定性占位。

## Decision

建议 Coinbase 的 `HashInputs` 为：

```text
BLAKE3-256(DomainTag("coinbase.inputs") || uint32_be(blockHeight))
```

- `blockHeight` 为 Coinbase 所在区块高度。
- Coinbase 不编码普通输入列表，也不使用交易输入的 `LeadHash`、`LeadPKHash`、`RestHash` 结构。

## Rationale

绑定高度可避免不同高度同构 Coinbase 拥有相同输入占位。独立域标签避免与普通输入根混淆。

## Consequences

Coinbase TxID 依赖该占位规则；在 Coinbase 序列化冻结前保持 Proposed。

## Conception references

- `docs/conception/附.交易.md`

## Open questions

- 高度是否采用 `uint32`，或与长期链高度兼容改为 varint。

# DEC-0017: Coinbase HashInputs Removed（Coinbase 输入哈希已移除）

Status: Deprecated

## Context

旧草案曾为 Coinbase 的 `HashInputs` 定义确定性占位哈希。最新 conception 已明确 Coinbase 没有输入项，且 `HashInputs` 字段省略。

## Decision

- Coinbase 不编码 `HashInputs`。
- Coinbase 不使用 `coinbase.inputs` 域标签。
- Coinbase 不编码普通输入列表，也不使用普通交易输入的 `LeadHash`、`LeadPKHash`、`RestHash` 结构。
- Coinbase 的特殊字段和 TxID 规范转入 DEC-0018。

## Rationale

保留占位哈希会与 conception 的“字段省略”直接冲突，并改变 Coinbase TxID、创世块哈希和签名消息。

## Consequences

所有使用 `BLAKE3-256(DomainTag("coinbase.inputs") || height)` 的测试向量和实现全部废弃。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/blockchain.md`

## Open questions

无。

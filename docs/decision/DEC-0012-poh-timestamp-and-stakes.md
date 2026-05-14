# DEC-0012: PoH Timestamp and Stakes（PoH 时间戳与 Stakes）

Status: Accepted

## Context

conception 规定区块时间戳由创世时间和高度推导，Stakes 表达区块币权销毁并参与铸凭哈希。Decision 只补充实现边界。

## Decision

- 区块 `H` 的时间戳为 `genesisTimestamp + H * 6min`，单位为 Unix 毫秒。
- PoH 中 `timeStamp` 使用待创建区块高度的推导时间戳。
- `Stakes` 使用评参区块头中的 `Stakes` 字段。
- `Stakes` 单位为 `chx * hour`，币龄不足 1 小时按 0 计。
- Coinbase 不计入 Stakes；只统计非 Coinbase 交易输入销毁的币权。
- 计算 Stakes 时使用交易实际入块高度对应的时间，不使用交易自身时间戳推导币龄。

## Rationale

这些规则来自 conception，但集中记录可避免共识实现混用本机时间或交易时间戳。

## Consequences

区块验证必须拒绝任何试图携带独立区块时间戳的头部格式。

## Conception references

- `docs/conception/1.共识-历史证明（PoH）.md`
- `docs/conception/2.共识-端点约定.md`
- `docs/conception/blockchain.md`

## Open questions

无。

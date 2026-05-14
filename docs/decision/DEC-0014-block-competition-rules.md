# DEC-0014: Block Competition Rules（区块竞争规则）

Status: Accepted

## Context

conception 同时定义同一铸造者多区块、候选铸造者冗余发布和交易量约束。Decision 只统一比较顺序和边界表达。

## Decision

- 同一高度同一铸造者发布多个合法区块时，交易费收益更低者胜出。
- 不同候选者发布合法区块时，先按择优池序位确认主候选者，再应用交易量约束。
- 若候选区块 Stakes 严格大于主区块 Stakes 的 3 倍，候选区块胜出。
- 主区块 Stakes 为 0 时，任何 Stakes 大于 0 的合法候选区块均满足交易量约束。
- 候选区块之间均满足交易量约束时，按择优池序位靠前者胜出。

## Rationale

“超过 3 倍”按严格大于处理，避免等于 3 倍时实现分歧。

## Consequences

区块竞争实现必须同时取得择优池序位、交易费收益和 Stakes。交易费收益的精确计算依赖 Coinbase 编码。

## Conception references

- `docs/conception/2.共识-端点约定.md`
- `docs/conception/附.组队校验.md`

## Open questions

无。

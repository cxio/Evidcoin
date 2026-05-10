# DEC-0022: Credit 31 年过期边界

## Status（状态）

Accepted

## Context（背景）

`conception/5.信用结构.md` 已明确超过 31 年未消费的 Credit 会过期并从 UTCO 集中移除，且年度按 `87661` 块计算。实现层需要固定边界比较符和活跃集移除时机。

## Decision（决策）

Credit 的 31 年期限按区块数计算：

```text
CreditLifetimeBlocks = 31 * 87661
expiryHeight = createdHeight + CreditLifetimeBlocks
```

边界语义：

- 若验证区块高度 `H <= expiryHeight`，该 Credit 未因 31 年期限过期。
- 若验证区块高度 `H > expiryHeight`，该 Credit 不得作为有效输入。
- 当链状态推进到 `H > expiryHeight` 时，该 Credit 可从 UTCO 活跃集中移除。
- `createdHeight` 为创建该 Credit 输出的交易所在区块高度，不使用交易头时间戳。

## Rationale（理由）

- conception 已明确起始时间为交易所在区块，使用创建高度可避免交易时间戳扰动 UTCO 生命周期。
- `H > createdHeight + 31*87661` 表达“超过 31 年未消费后不再有效”的首个失效高度，保留 31 年整点边界本身仍有效。
- 允许从活跃集移除可控制 UTCO 集规模，符合构想层限制目标。

## Consequences（影响）

- 区块验证必须拒绝消费已超过失效边界的 Credit。
- 状态维护可在区块应用前或应用后执行过期清理，但对高度 `H` 的输入有效性判断必须一致。
- 测试应覆盖 `H == expiryHeight` 仍有效，`H == expiryHeight + 1` 失效。

## Conception Relationship（与构想关系）

- 补充 `conception/5.信用结构.md` 中 Credit 31 年过期规则的边界语义。
- 不改变 Credit 单次寿命、消费即终止或 UTCO 集约束。

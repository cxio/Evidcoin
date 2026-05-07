# DECISION-0001: PoH 铸凭哈希内层算法

## Status（状态）

Accepted

## Context（背景）

`docs/conception/1.共识-历史证明（PoH）.md` 已定义铸凭哈希流程：先计算 `Hash(铸凭交易ID || 评参区块铸凭哈希 || X)`，再由铸造者签名，最后计算铸凭哈希。

`docs/conception/blockchain.md` 已明确最终铸凭哈希使用 `BLAKE3-256`，但没有固定内层 `Hash(...)` 的算法。

## Decision（决策）

PoH 铸凭哈希的内层哈希使用 `BLAKE3-256`，输出 32 字节。

```text
hashData = BLAKE3-256(mintTxID || referenceMintHash || X)
mintHash = BLAKE3-256(Sign(hashData))
```

## Rationale（理由）

- 内层输入均来自链上确定性数据，主要需求是高频计算效率。
- `BLAKE3-256` 已用于树结构内部节点，复用同一算法可降低实现复杂度。
- 最终比较值由 32 字节铸凭哈希给出，空间足够大。

## Consequences（影响）

- 实现 PoH 铸凭哈希时不得把内层 `Hash(...)` 替换为 SHA3 或其它算法。
- 测试向量应分别覆盖 `hashData` 与最终 `mintHash`。

## Conception Relationship（与构想关系）

- 补充 `docs/conception/1.共识-历史证明（PoH）.md` 中未命名的内层 `Hash(...)`。
- 不改变 conception 已明确的最终铸凭哈希 `BLAKE3-256`。

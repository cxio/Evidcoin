# ADR-0001: PoH 铸凭哈希内层算法

## Status（状态）

Accepted

## Context（背景）

在 PoH（Proof of Historical）共识机制中，铸凭哈希（MintHash）的计算分为两层：

1. **内层哈希**：`hashData = Hash(mintTxID || referenceBlockMintHash || X)`
2. **外层哈希**：`MintHash = BLAKE3-512-XOF(Sign(hashData), 64 bytes)`

提案层（`docs/proposal/10.PoH-Consensus.md`）和哈希算法分配表（`docs/proposal/02.Cryptography-And-Hashing.md`）均未指定内层 `Hash(...)` 的具体算法，被标记为未决项（OQ-016）。

网络中可能同时有大量节点参与铸凭资格竞争，因此内层哈希在短时间内会被频繁计算。

## Decision（决策）

**内层哈希算法采用 BLAKE3-256**，输出 32 字节。

具体计算公式：

```
hashData = BLAKE3-256(mintTxID || referenceBlockMintHash || X)
MintHash = BLAKE3-512-XOF(Sign(hashData), 64)
```

## Rationale（理由）

1. **内容为确定性数据**：内层哈希的输入（铸凭交易 ID、评参区块铸凭哈希、X 参数）均为链上确定性数据，不存在敌对方可控的不可信输入，无额外的量子安全需求。
2. **效率优先**：可能有大量参与者同时提交铸凭资格，内层哈希需被高频计算。BLAKE3 是目前主流算法中性能最优的之一，适合此高并发场景。
3. **与树结构一致**：BLAKE3-256 已被用于哈希树的内部节点计算，在同一系统内复用同一算法可减少依赖，降低实现复杂度。
4. **外层安全兜底**：最终 MintHash 由 BLAKE3-512-XOF（64 字节）包裹，提供 512 位空间的安全强度，内层的相对较低安全余量不影响整体共识安全性。

## Consequences（影响）

- 需在 `docs/proposal/02.Cryptography-And-Hashing.md` 的哈希算法分配表中补充本条目。
- 需在 `docs/proposal/10.PoH-Consensus.md` 中将 `Hash(...)` 替换为 `BLAKE3-256(...)`。
- 实现时 `pkg/crypto` 中的 `HashMintInner` 函数使用 BLAKE3-256。
- OQ-016 关闭。

## References（参考）

- `docs/proposal/10.PoH-Consensus.md` — 铸凭哈希算法
- `docs/proposal/02.Cryptography-And-Hashing.md` — 哈希算法分配表
- `docs/plan/06-PoH-Consensus-And-Fork-Choice.md` — Task 1
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-016

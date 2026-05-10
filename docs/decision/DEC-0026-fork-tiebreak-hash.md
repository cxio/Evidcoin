# DEC-0026: 分叉平局 Hash 算法收敛

## Status（状态）

Proposed

## Context（背景）

`conception/2.共识-端点约定.md` 已定义分叉竞争平局时使用 tie-breaker，但文本说明“选用抗 ASIC 的 HashX”，伪代码却写为 `Hash256(分叉点区块ID || 分叉首块ID)`。`conception/1.共识-历史证明（PoH）.md` 未进一步固定该平局哈希算法。因此当前构想层存在算法口径冲突，不能 Accepted。

## Decision（决策）

本节为候选规则，状态转为 Accepted 前不得作为最终共识依据。

建议收敛为下列候选规则：

```text
TieBreakHash = BLAKE3-256(
    DomainTag("ForkDecision") ||
    forkPointBlockID[48] ||
    firstForkBlockID[48]
)
```

比较规则建议如下：

- 对每条平局分叉，取同一分叉点区块 ID 和该分叉的首块区块 ID 计算 `TieBreakHash`。
- `TieBreakHash` 按原始字节字典序升序比较，较小者胜出。
- 若 `TieBreakHash` 完全相等，则比较 `firstForkBlockID` 原始 48 字节，较小者胜出。
- 输入编码固定为 domain tag 后串接两个 48 字节区块 ID，不加入长度前缀。

待裁决事项：

- 是否按构想层文字采用 `HashX`。
- 是否按既有哈希策略采用 BLAKE3-256 作为 `Hash256` 的具体算法。
- 是否需要使用 SHA3-384 以匹配区块 ID 的长期安全强度。
- domain tag 名称是否固定为 DEC-0002 已列的 `ForkDecision`。

## Rationale（理由）

- 现有 conception 对 `HashX` 与 `Hash256` 同时出现，不能由实现自行解释。
- BLAKE3-256 与项目中 `Hash256` 场景一致，且计算简单，适合低频 tie-breaker。
- domain tag 可避免 tie-breaker 哈希与其它 Hash256 用途混淆。

## Consequences（影响）

- 在构想层口径修订或后续 Accepted 决策前，分叉平局哈希不能作为最终共识规则实现。
- 若最终选择 `HashX`，需要新增依赖、参数和测试向量。
- 若最终选择 BLAKE3-256 或 SHA3-384，需要同步更新分叉竞争测试向量。

## Conception Relationship（与构想关系）

- 标记 `conception/2.共识-端点约定.md` 中分叉平局哈希算法的口径冲突，并提出收敛候选。
- 不改变分叉链段 35 区块评比、20 区块接纳窗口或平局时按哈希字典序比较的构想层主体。

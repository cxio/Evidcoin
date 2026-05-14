# DEC-0004: Hash Tree Edge Cases（哈希树边界）

Status: Proposed

## Context

conception 明确区块交易集、输出集、附件片组和 UTXO/UTCO 指纹使用哈希树，但空树、单叶、奇数叶和叶子序号编码仍需统一。交易输入的 `RestHash` 是专用两层结构，不能误套通用 Merkle 规则。

## Decision

建议如下：

- 通用树叶子哈希：`SHA3-384(DomainTag("hash-tree.leaf") || leafIndex || payload)`。
- 分支哈希：`BLAKE3-256(DomainTag("hash-tree.branch") || left || right)`。
- 单叶树根为该叶哈希。
- 奇数叶提升时，最后一个节点直接提升到下一层，不复制自身。
- 空树根按具体结构定义；交易输出集和区块交易集不得为空，附件片组空值由附件规范另行定义。
- 交易输入根继续按 conception 的 `Hash256(LeadHash || LeadPKHash || RestHash)`，不使用本通用树规则。

## Rationale

不复制奇数叶可避免引入人工重复数据；显式排除交易输入根可防止把 RestHash 当作通用树分支处理。

## Consequences

所有哈希树证明格式必须携带叶序号和兄弟方向。该 DEC 冻结前，证明包格式保持 Proposed。

## Conception references

- `docs/conception/blockchain.md`
- `docs/conception/附.交易.md`

## Open questions

- 空附件片组是否需要统一空根。
- 区块交易叶子的 3 字节序号应作为 `payload` 前缀还是 `leafIndex` 的专用编码。

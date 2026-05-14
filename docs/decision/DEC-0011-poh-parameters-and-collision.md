# DEC-0011: PoH Parameters and Collision（PoH 参数与碰撞）

Status: Accepted

## Context

conception 已明确 PoH 的关键参数：铸凭交易区间、评参区块、择优池大小、授权同步成员和铸凭哈希主体。Decision 只补充碰撞和排序边界。

## Decision

- 铸凭交易有效区间为 `[-80000, -240]`，初段例外见 DEC-0013。
- 评参区块为链末端的 `-8` 号区块。
- 择优池容量为 20，按铸凭哈希字节序从小到大排序。
- 排名前 5 之后的 15 位候选者可发起同步。
- 若不同铸凭交易计算出相同铸凭哈希，按完整铸凭交易 ID 字节序从小到大排序；仍相同时按铸造者公钥字节序排序。

## Rationale

参数主体来自 conception。碰撞排序只处理极端等值情况，避免实现自行选择导致择优池不一致。

## Consequences

择优池实现必须保留完整铸凭交易 ID 和铸造者公钥用于稳定排序。

## Conception references

- `docs/conception/1.共识-历史证明（PoH）.md`

## Open questions

无。

# DEC-0011: PoH Mint Profile（PoH 铸凭配置）

Status: Proposed

## Context

conception 已明确 PoH 的关键参数：铸凭交易区间、评参区块、`Stakes` 取值区块、择优池大小、授权同步成员和铸凭哈希主体。最近修订把铸造身份改为 `MintPKHash` 可选，并将币权销毁因子与评参区块剥离。

Decision 只补充跨实现需要一致的身份来源、碰撞排序和仍未冻结的字节级输入。

## Decision

- 铸凭交易有效区间为 `[-80000, -240]`，初段例外见 DEC-0013。
- 评参区块为链末端的 `-8` 号区块。
- `Stakes` 因子来自链末端的 `-32` 号区块，详见 DEC-0012。
- 择优池容量为 20，按铸凭哈希字节序从小到大排序。
- 排名前 5 之后的 15 位候选者可发起同步。
- 铸造者身份为铸凭交易的 `MintPKHash`；若铸凭交易未设置 `MintPKHash`，则回退到首领输入源公钥哈希 `LeadPKHash`。
- 如果回退到 `LeadPKHash`，择优凭证必须提供可验证 `HashInputs = Hash256(ListHash || LeadPKHash)` 的 `ListHash` 或等价证明材料。
- 若不同铸凭交易计算出相同铸凭哈希，按完整铸凭交易 ID 字节序从小到大排序；仍相同时按铸造者公钥字节序排序。

## Rationale

参数主体来自 conception。碰撞排序只处理极端等值情况，避免实现自行选择导致择优池不一致。身份来源规则使 `MintPKHash` 的可选性和旧首领输入身份逻辑可以在同一验证路径中收敛。

## Consequences

择优池实现必须保留完整铸凭交易 ID、铸造者公钥和身份来源证明。区块证明包需要携带足以验证 `MintPKHash` 或 `LeadPKHash` 的材料。

## Conception references

- `docs/conception/1.共识-历史证明（PoH）.md`
- `docs/conception/附.交易.md`

## Open questions

- 铸凭哈希前像中的 `pubKey` 使用完整公钥还是由 `MintPKHash` 对应的公钥证明派生，构想层文字仍需冻结。
- `X = Bytes(timeStamp * Stakes * Mix)` 的整数宽度、端序和任意精度编码尚未冻结。
- `评参区块:铸凭哈希` 在前像中的字段格式尚未冻结。

# DEC-0027: Block Proof Package（区块证明包）

Status: Proposed

## Context

conception 允许区块在完整验证前通过适当区块证明先行转播，并列出最少证明性数据。需要规范证明包字段，以支持快速转播和轻节点验证。

## Decision

建议最小区块证明包包含：

- 区块头。
- Coinbase 交易体。
- Coinbase 到区块交易树根的验证路径。
- UTXO/UTCO 指纹值或其纳入 `CheckRoot` 的证明片段。
- 铸造者对 `CheckRoot` 的签名。
- 择优凭证及其验证所需的铸凭交易定位信息。

证明包只证明“可先行转播”，不替代完整交易集验证。

## Rationale

该集合对应 conception 的快速转播描述和组队校验的最少证明数据。

## Consequences

节点收到证明包后仍需同步完整交易 ID 序列和缺失交易，完成最终合法性验证。

## Conception references

- `docs/conception/2.共识-端点约定.md`
- `docs/conception/附.组队校验.md`
- `docs/conception/blockchain.md`

## Open questions

- Coinbase 验证路径的精确方向编码。
- UTXO/UTCO 指纹是直接携带根值，还是携带可验证路径。
- 择优凭证中铸凭交易定位信息的最小字段集。

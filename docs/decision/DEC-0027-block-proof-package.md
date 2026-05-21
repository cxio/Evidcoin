# DEC-0027: Block Proof Package（区块证明包）

Status: Proposed

## Context

conception 允许区块在完整验证前通过适当区块证明先行转播，并列出最少证明性数据。需要规范证明包字段，以支持快速转播和轻节点验证。

最近 conception 修订影响了证明包：UTXO/UTCO 指纹不再包含 `DataID`，Coinbase 省略 `HashInputs`，PoH 身份可由 `MintPKHash` 或 `LeadPKHash` 得出。

## Decision

建议最小区块证明包包含：

- 区块头。
- Coinbase 交易体与 Coinbase 特殊头字段。
- Coinbase 到区块交易树根的验证路径。
- UTXO/UTCO 指纹值或其纳入 `CheckRoot` 的证明片段，按 `TxID || FlagOutputs` 状态位模型验证。
- 铸造者对 `CheckRoot` 的签名。
- 择优凭证及其验证所需的铸凭交易定位信息。
- 若铸凭交易设置 `MintPKHash`，提供对应公钥与签名证明。
- 若铸凭交易未设置 `MintPKHash`，提供可验证 `LeadPKHash` 与输入根关系的 `ListHash` 或等价证明材料。

证明包只证明“可先行转播”，不替代完整交易集验证。

## Rationale

该集合对应 conception 的快速转播描述和组队校验的最少证明数据。新增身份证明分支可覆盖 `MintPKHash` 可选设计。

## Consequences

节点收到证明包后仍需同步完整交易 ID 序列和缺失交易，完成最终合法性验证。旧包含 `DataID` 或 Coinbase `HashInputs` 的证明包草案废弃。

## Conception references

- `docs/conception/2.共识-端点约定.md`
- `docs/conception/附.组队校验.md`
- `docs/conception/blockchain.md`
- `docs/conception/1.共识-历史证明（PoH）.md`

## Open questions

- Coinbase 验证路径的精确方向编码。
- UTXO/UTCO 指纹是直接携带根值，还是携带可验证路径。
- 择优凭证中铸凭交易定位信息的最小字段集。
- `MintPKHash` 对应完整公钥的证明格式。

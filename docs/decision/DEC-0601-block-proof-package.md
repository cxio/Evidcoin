# DEC-0601: Block Proof Package（区块证明包）

Status: Proposed

## Context（背景）

Conception 允许新区块先广播最小证明，再同步区块概要和交易数据。初始节点验证也需要末端区块的 Coinbase、验证路径、UTXO/UTCO 指纹和铸造者签名。证明包字段尚未冻结。

## Decision（决策）

建议区块证明包包含：

1. `BlockHeader`
2. `CoinbaseTx`
3. `CoinbaseTxIndex`，必须为 0
4. `CoinbaseMerklePath`
5. `UTXORoot`
6. `UTCORoot`
7. `CheckRoot`
8. `MinterCheckRootSignature`
9. `MintProof`
10. `ChainScope`

验证流程建议：

- 验证 Coinbase TxID。
- 用 Coinbase 路径验证其属于区块交易树。
- 用交易树根、UTXO/UTCO 指纹重算 `CheckRoot`。
- 验证区块头 `CheckRoot`。
- 验证铸造者对 `CheckRoot` 的签名。
- 验证 `MintProof` 中的 `MintHash` 和签名。

## Rationale（理由）

证明包应足够小，使节点可先转播区块证明；同时必须包含独立验证铸造资格和区块头合法性的最小材料。

## Consequences（影响）

- 证明包不能替代完整区块验证，只能支持快速预验证和转播。
- 若 UTXO/UTCO 指纹不携带完整证明，验证者仍需信任自己已有状态或后续查询。
- 初始同步至少需要最近 31 块证明包以覆盖分叉安全窗口。

## Conception References（构想层依据）

- `docs/conception/blockchain.md#初始主链验证`
- `docs/conception/2.共识-端点约定.md#区块发布`
- `docs/conception/附.组队校验.md#区块发布`

## Open Questions（开放问题）

- 证明包是否包含完整交易树根，还是只通过 `CheckRoot` 间接表达。
- UTXO/UTCO 指纹是否需要附带状态证明路径。
- 铸造者公钥在证明包中引用 `MintProof` 即可，还是需要独立字段。

# DEC-0603: Blockqs Verification Data Profile（Blockqs 验证数据配置）

Status: Proposed

## Context（背景）

Conception 将完整区块、交易数据和附件存储交给公共服务，并要求 Blockqs 提供交易查询和验证路径。Blockqs 与 Depots 的边界、响应格式、验证数据最小集合尚未冻结。

## Decision（决策）

建议 Blockqs 提供以下响应类型：

- `TxLookup`：按年度和 TxID 返回完整交易、区块高度、区块内序位。
- `TxProof`：返回交易到区块交易树根的验证路径。
- `BlockTxList`：返回区块完整 TxID 序列或网络概要。
- `StateProof`：返回 UTXO/UTCO 状态位证明和输出详情。
- `RecentBlockProofs`：返回最近至少 31 个区块证明包。
- `AttachmentIndex`：返回小附件或大附件分片索引。

建议服务边界：

- Blockqs 负责小数据、即时查询、验证路径和索引。
- Depots 负责完整区块文件、大附件和分片数据。
- 恰好 `10MB` 的附件边界需冻结后才能进入服务规范。

建议响应验证：

- 所有响应必须可由区块头链、`CheckRoot`、TxID 或附件指纹验证。
- 服务节点签名只证明服务来源，不证明数据真实。
- 客户端应向多个 Blockqs 节点交叉查询关键数据。

## Rationale（理由）

Blockqs 是查询加速层，不应成为信任根。响应格式围绕可验证数据组织，可让轻节点以较小数据量核实交易和状态。

## Consequences（影响）

- Blockqs API 可以演进，但验证数据结构必须保持与共识编码一致。
- Depots 和 Blockqs 对同一数据的提供可能重叠，但验证口径必须相同。
- 初始同步流程依赖 `RecentBlockProofs` 的完整性。

## Conception References（构想层依据）

- `docs/conception/3.公共服务.md#区块查询blockqs`
- `docs/conception/3.公共服务.md#数据驿站depots`
- `docs/conception/附.交易.md#交易的存储与验证`
- `docs/conception/blockchain.md#初始主链验证`

## Open Questions（开放问题）

- 恰好 `10MB` 的附件由 Blockqs 还是 Depots 负责，见 `CONCEPTION-CONFLICTS.md` 的 `C-010`。
- Blockqs 响应是否需要统一二进制 canonical encoding，还是服务层可用 JSON/CBOR。
- 服务节点身份签名是否使用链上地址对应密钥，还是服务网络独立密钥。

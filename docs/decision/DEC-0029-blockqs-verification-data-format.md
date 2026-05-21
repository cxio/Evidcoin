# DEC-0029: Blockqs Verification Data Format（Blockqs 验证数据格式）

Status: Proposed

## Context

conception 说明 Blockqs 提供区块头、交易定位、交易 ID 清单、哈希验证路径、脚本检索、UTXO/UTCO 集和小附件等服务。Decision 仅定义轻节点和初始上线节点所需的验证数据格式；更宽的查询 API 留到服务接口阶段。

最近 conception 修订要求验证数据跟随创世细节、Coinbase 省略 `HashInputs` 和 UTXO/UTCO 指纹简化。

## Decision

建议 Blockqs 验证响应分为三类：

- `HeaderRangeProof`：连续区块头、可选年块衔接信息、起止高度和起止 BlockID。
- `TxInclusionProof`：交易年度、TxID、区块高度、区块内序位、交易树验证路径和目标区块头摘要。
- `TipBootstrapProof`：创世区块信息、区块头链摘要、末端 31 个区块的 Coinbase 证明包、当前 UTXO/UTCO 指纹摘要。

所有响应必须声明 `Protocol-ID`、`Chain-ID` 和数据生成时的服务节点身份。Blockqs 响应只是验证材料，最终信任来自本地重算哈希和签名。

验证数据必须遵守以下边界：

- UTXO/UTCO 指纹按 DEC-0010 的 `TxID || FlagOutputs` 模型解释。
- Coinbase 验证按 DEC-0018，不得要求 Coinbase `HashInputs`；DEC-0017 仅记录旧占位输入哈希已废弃。
- 创世块字段按 DEC-0013。
- Blockqs 可提供脚本、小附件和交易集合查询，但这些非验证查询格式不由本 DEC 冻结。

## Rationale

将服务响应拆成明确类型，有利于轻节点按需获取最小数据，并避免把服务节点声明当作共识事实。

## Consequences

Blockqs API 可以独立演进，但共识验证所需字段必须保持兼容。Depots 与 Blockqs 对完整区块、大附件和附件索引的边界仍需在服务接口阶段细化。

## Conception references

- `docs/conception/blockchain.md`
- `docs/conception/附.交易.md`
- `docs/conception/3.公共服务.md`

## Open questions

- 服务节点身份格式和签名算法。
- `TipBootstrapProof` 是否必须包含完整末端 31 个区块头，或可用区块头链摘要替代。
- Blockqs 与 Depots 对完整区块数据、大附件和附件索引的职责边界。

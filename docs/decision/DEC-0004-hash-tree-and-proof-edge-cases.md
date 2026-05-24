# DEC-0004: Hash Tree and Proof Edge Cases（哈希树与证明边界）

Status: Proposed

## Context（背景）

Conception 使用多类哈希树：区块交易树、交易输出树、附件片组树、UTXO/UTCO 状态树。其算法用途已明确，但空树、单叶、奇数节点、叶子序号和验证路径编码尚未统一。

## Decision（决策）

建议采用以下通用规则：

- 叶子前像为 `DomainTag("tree.leaf") || leafIndex || payload`。
- 分支前像为 `DomainTag("tree.branch") || leftHash || rightHash`。
- `leafIndex` 使用该树规定的固定宽度或 varint 编码，必须唯一且从 0 开始。
- 奇数层最后一个节点直接提升到下一层，不复制自身。
- 单叶树根等于该叶哈希，不额外套一层分支哈希。
- 空树不得使用通用规则，必须由具体结构单独定义空根。

专用规则：

- 区块交易树叶子序号为 3 字节，大端，从 0 开始。
- 附件片组树叶子序号为 2 字节，大端，从 0 开始。
- 交易输入根继续使用 `Hash256(ListHash || LeadPKHash)`，不套通用哈希树。
- UTXO/UTCO 指纹是宽成员状态树，不直接套二叉树证明格式。

证明路径建议编码为：

- `leafIndex`
- `leafHash`
- `siblings[]`，每项包含 `direction` 和 `hash`
- `rootHash`

## Rationale（理由）

不复制奇数叶可避免人为重复数据。单叶即根是常见且可验证的最小规则。不同结构保留自己的序号宽度，可贴合 conception 中的 3 字节交易序号与 2 字节附件分片序号。

## Consequences（影响）

- 验证路径必须携带方向信息。
- 区块交易树与附件片组树不能混用序号宽度。
- 交易输入根、UTXO/UTCO 指纹需在各自 DEC 中继续细化。

## Conception References（构想层依据）

- `docs/conception/blockchain.md#交易约束`
- `docs/conception/blockchain.md#哈希树结构`
- `docs/conception/附.交易.md#哈希校验树`
- `docs/conception/5.信用结构.md#附件id的结构`
- `docs/conception/附.组队校验.md#utxoutco-指纹`

## Open Questions（开放问题）

- 各结构空根是否统一由域标签哈希产生，还是保留全零特殊值。
- 区块交易树叶子前像中的 3 字节序号是否作为 `leafIndex`，还是作为 `payload` 内部前缀。

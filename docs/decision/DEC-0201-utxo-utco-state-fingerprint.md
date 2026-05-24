# DEC-0201: UTXO/UTCO State Fingerprint（UTXO/UTCO 状态指纹）

Status: Proposed

## Context（背景）

Conception 已明确 UTXO/UTCO 指纹按年度、TxID 字节 `[8,13,18]` 分层，末端叶为 `Hash384(TxID || FlagOutputs)`。仍需冻结状态位顺序、空分组哈希、年度排序和缓存详情边界。

## Decision（决策）

建议状态位规则：

- `FlagOutputs` 的第 `i` 位对应输出序位 `i`。
- 位序为每字节低位优先，即 bit0 对应较小输出序位。
- `Count` 表示有效输出数量，而非字节数量。
- 尾部未使用 bit 必须为 0。
- `1` 表示未花费或未转出，`0` 表示已无效。

建议分层规则：

- 顶层年度按数值升序排列。
- 年度内按 TxID 字节 `[8,13,18]` 分入三级宽成员节点。
- 同一末端分组内按完整 TxID 字典序排列。
- 空年度和空分组不编码；整棵空状态树使用专用空根。

建议叶子规则：

```text
StateLeaf = SHA3-384(DomainTag("utxo.leaf" or "utco.leaf") || TxID || Count || FlagBytes)
```

缓存边界：

- 状态位集合参与指纹。
- 输出详情缓存不参与指纹，只是同形分层的检索优化。

## Rationale（理由）

按输出序位映射状态位可以避免把输出详情纳入状态根，保持指纹轻量。年度和完整 TxID 排序可保证跨实现构造相同根。

## Consequences（影响）

- 查询服务必须能提供状态位证明和输出详情证明的边界说明。
- 逆向推导验证只依赖状态位集合，不依赖缓存详情。
- 短 TxID 引用碰撞时，末端分组排序规则与交易输入解析一致。

## Conception References（构想层依据）

- `docs/conception/附.组队校验.md#utxoutco-指纹`
- `docs/conception/附.组队校验.md#从-utxoutco-集检索输出项数据`
- `docs/conception/附.交易.md#附utxoutco-集合`

## Open Questions（开放问题）

- 空状态树根使用全零、域标签哈希，还是 `SHA3-384(empty)`。
- `Count` 是否保留在叶子前像中，或仅由 `FlagBytes` 长度推导。
- Credit 过期移除时状态位是置 0 后保留，还是从 UTCO 叶中删除。

# DEC-0010: UTXO/UTCO Fingerprint Payload（状态指纹载荷）

Status: Proposed

## Context

conception 定义 UTXO/UTCO 四层宽成员指纹树，并说明末端叶子节点为 `TxID || DataID || FlagOutputs`。仍需固定 payload 序列化、状态位顺序和分组空节点处理。

## Decision

建议末端 payload 为：

```text
LeafPayload = txid || data_id || flag_count || flag_bytes
LeafHash = SHA3-384(DomainTag("state.utxo"|"state.utco") || LeafPayload)
```

- `txid` 为 48 字节完整交易 ID。
- `data_id` 为该 TxID 下仍有效输出项 payload 按 `OutIndex` 升序编码后的 SHA3-384。
- `flag_count` 表示有效标记覆盖的输出项数量，不是字节数。
- `flag_bytes` 从低位到高位映射同一字节内递增的 `OutIndex`；位值 `1` 表示未花费或未转出，`0` 表示无效或不存在。
- 年度层、三级字节分层 `[8,13,18]` 和末端按 TxID 排序跟随 conception。

## Rationale

显式 `flag_count` 可区分尾部填充零与真实输出数量。低位优先便于位运算，但需要作者确认。

## Consequences

状态指纹测试向量必须覆盖空分组、单输出、多输出、尾部不足 8 位和 UTCO 过期删除。

## Conception references

- `docs/conception/附.组队校验.md`
- `docs/conception/blockchain.md`
- `docs/conception/5.信用结构.md`

## Open questions

- 位顺序是否最终采用低位优先。
- `data_id` 是否应包含已无效输出的占位，或只包含仍有效输出。
- 空年度或空分组的哈希表示尚未冻结。

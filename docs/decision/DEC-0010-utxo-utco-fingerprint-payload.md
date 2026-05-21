# DEC-0010: UTXO/UTCO Fingerprint Payload（状态指纹载荷）

Status: Proposed

## Context

最新 conception 已简化 UTXO/UTCO 指纹：末端叶子节点只表达输出项未花费/未转出状态位序列，即 `Hash384(TxID || FlagOutputs)`。输出项详细数据和旧 `DataID` 不参与状态指纹。

仍需固定 `FlagOutputs` 的字节级序列化、位顺序和空分组处理。

## Decision

建议末端 payload 为：

```text
FlagOutputs = flag_byte_count || flag_bytes
LeafPayload = txid || FlagOutputs
LeafHash = SHA3-384(DomainTag("state.utxo"|"state.utco") || LeafPayload)
```

- `txid` 为 48 字节完整交易 ID。
- `flag_byte_count` 表示 `flag_bytes` 的字节数，跟随 conception 伪代码中的 `Count int // 标记位字节数`。
- `flag_bytes` 每一位对应一个输出项；位值 `1` 表示未花费或未转出，`0` 表示无效或不存在。
- 同一字节内建议低位到高位映射递增的 `OutIndex`，但该位序在作者确认前保持 Proposed。
- 年度层、三级字节分层 `[8,13,18]` 和末端按 TxID 排序跟随 conception。
- 输出项详细 payload、缓存器数据和任何 `DataID` 均不进入指纹。

## Rationale

只承诺状态位可显著减小 UTXO/UTCO 指纹维护成本，并与 conception 的“轻量级状态位集”一致。显式字节数可区分尾部填充零与实际状态位范围。

## Consequences

状态指纹测试向量必须覆盖空分组、单输出、多输出、尾部不足 8 位和 UTCO 过期删除。所有旧 `DataID` 测试向量废弃。

## Conception references

- `docs/conception/附.组队校验.md`
- `docs/conception/blockchain.md`
- `docs/conception/5.信用结构.md`

## Open questions

- 位顺序是否最终采用低位优先。
- 空年度或空分组的哈希表示尚未冻结。

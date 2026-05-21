# DEC-0007: Transaction Body Canonical Encoding（交易体规范编码）

Status: Proposed

## Context

conception 定义输入项、输出项、输入根和输出根，但缺少交易体的字节级规范。最近构想层用可选 `MintPKHash` 解耦铸造身份与首领输入，因此旧“首领输入必须使用完整 48 字节 TxID”的限制不再有明确依据。

## Decision

建议交易体编码按下列结构冻结：

```text
TxBody = inputs_count || Input* || outputs_count || Output*
Input = year || txid_part_len || txid_part || out_index || unlock_script_len || unlock_script
Output = serial || config || payload_len || payload || lock_script_len || lock_script
```

- 所有计数和长度使用 DEC-0001。
- 普通输入的 `txid_part_len` 不得小于 16；是否使用完整 48 字节由交易构造者决定。
- 首领输入不再强制 `txid_part_len = 48`；铸造身份优先由 `TxHeader.MintPKHash` 指定，未设置时回退到首领输入源公钥哈希 `LeadPKHash`。
- `serial` 必须等于输出在输出集合中的序位，从 0 连续递增。
- `payload` 由 Coin、Credit、Proof 类型决定，具体字段顺序需在信用结构编码中冻结。
- 见证信息不进入 `TxBody`，也不进入 `HashInputs` 或 `HashOutputs`。
- Coinbase 没有普通输入列表，并省略 `HashInputs`，详见 DEC-0018。

## Rationale

显式长度前缀使脚本和 payload 可安全解析。取消首领输入完整 TxID 要求可回到 conception 的短引用设计，并避免与可选 `MintPKHash` 重复约束。

## Consequences

该 DEC 仍为 Proposed，因为 Coin、Credit、Proof 的 payload 字段级编码尚未全部冻结。交易索引与 UTXO/UTCO 查询必须处理首领输入短引用碰撞。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/5.信用结构.md`

## Open questions

- Coin、Credit、Proof payload 的精确字段顺序。
- 备注、附件 ID、创建者等可选字段为空时使用零长度还是显式 NIL 标记。

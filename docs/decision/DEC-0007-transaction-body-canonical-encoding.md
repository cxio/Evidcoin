# DEC-0007: Transaction Body Canonical Encoding（交易体规范编码）

Status: Proposed

## Context

conception 定义输入项、输出项、输入根和输出根，但缺少交易体的字节级规范。跨实现若字段顺序、长度前缀或可选字段处理不同，会直接导致 TxID 不一致。

## Decision

建议交易体编码按下列结构冻结：

```text
TxBody = inputs_count || Input* || outputs_count || Output*
Input = year || txid_part_len || txid_part || out_index || unlock_script_len || unlock_script
Output = serial || config || payload_len || payload || lock_script_len || lock_script
```

- 所有计数和长度使用 DEC-0001。
- 首领输入的 `txid_part_len` 必须为 48，其余输入不得小于 16。
- `serial` 必须等于输出在输出集合中的序位，从 0 连续递增。
- `payload` 由信元类型决定，具体字段顺序需在信用结构编码中冻结。
- 见证信息不进入 `TxBody`，也不进入 `HashInputs` 或 `HashOutputs`。

## Rationale

显式长度前缀使脚本和 payload 可安全解析；首领输入规则与 conception 的快速验证和铸凭身份检索一致。

## Consequences

该 DEC 仍为 Proposed，因为 Coin、Credit、Proof 的 payload 字段级编码尚未全部冻结。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/5.信用结构.md`

## Open questions

- Coin、Credit、Proof payload 的精确字段顺序。
- 备注、附件 ID、创建者等可选字段为空时使用零长度还是显式 NIL 标记。

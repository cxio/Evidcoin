# DEC-0008: Witness Encoding and Pruning（见证编码与剪枝边界）

Status: Proposed

## Context

conception 明确见证信息与解锁脚本分离，且 TxID 不包含见证信息。签名验证、长期存储和剪枝需要统一见证容器边界。

## Decision

建议见证编码如下：

```text
WitnessSet = witness_count || Witness*
Witness = input_index || witness_type || item_count || WitnessItem*
WitnessItem = item_type || item_len || item_bytes
```

- `input_index` 必须引用交易输入序位。
- `witness_type` 至少区分单签、多签和自定义验证。
- 见证不进入 TxID、交易输入根、交易输出根、区块交易树根或 UTXO/UTCO 指纹。
- 区块完整验证完成并超过 31 个区块安全边界后，节点可剪枝标准见证；若应用需要长期审计，可由外部服务保存。
- 定制验证若把签名数据放入解锁脚本，则该数据属于交易体，不属于可剪枝见证。

## Rationale

见证剪枝能降低长期存储压力，同时不破坏 TxID 和链上状态。31 个区块边界与 conception 的分叉安全口径一致。

## Consequences

轻节点验证历史签名需要从归档节点或外部服务取回见证。交易数据服务需要区分交易体与见证包。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/6.脚本系统.md`
- `docs/conception/2.共识-端点约定.md`

## Open questions

- 标准见证是否必须保存至少 240 个区块以匹配校验组运行习惯。
- 多签见证中补全集排序与 `FN_MPUBHASH` 的精确编码。

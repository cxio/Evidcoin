# DEC-0009: Signature Message Encoding（签名消息编码）

Status: Proposed

## Context

conception 支持按授权标记选择输入、输出、接收者、内容、脚本或完整输出进行签名。旧方案把 TxID 放入签名消息，但 TxID 覆盖完整输入输出，可能破坏选择性授权语义。

## Decision

签名消息必须按授权标记选择性构造，而不是无条件签署 TxID。建议结构如下：

```text
SigMsg = chain_context || tx_header_selected || input_scope || output_scope || auth_flag || current_input_index
```

- `chain_context` 使用 conception 的 `Protocol-ID || Chain-ID || Genesis-ID || Bound-ID`。
- `tx_header_selected` 只包含版本、时间戳和被授权范围需要的哈希摘要；不得因便捷而强制包含完整 TxID。
- `input_scope` 根据 `SIGIN_ALL` 或 `SIGIN_SELF` 编码输入范围。
- `output_scope` 根据 `SIGOUT_ALL`、`SIGOUT_SELF` 和辅项标记编码输出范围。
- 字段宽度暂按 DEC-0003；该 DEC 冻结前本文件保持 Proposed。

## Rationale

选择性授权的意义是只承诺指定范围。直接签 TxID 会把未选择部分也纳入承诺，导致 `SIGOUT_SELF|SIGRECEIVER` 等模式失去语义。

## Consequences

签名验证实现需要按授权标记重建消息，不能只比较 TxID。需要为每种授权组合生成测试向量。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/blockchain.md`
- `docs/conception/Instruction/16.函数指令.md`

## Open questions

- `tx_header_selected` 是否应包含交易时间戳。
- 未被授权的字段是否完全省略，或以固定空摘要占位。
- `SIGIN_ALL` 是否包含当前输入的解锁脚本字节。

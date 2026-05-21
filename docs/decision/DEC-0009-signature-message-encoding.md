# DEC-0009: Signature Message Encoding（签名消息编码）

Status: Proposed

## Context

conception 支持按授权标记选择输入、输出、接收者、内容、脚本或完整输出进行签名。旧方案把 TxID 放入签名消息，但 TxID 覆盖完整输入输出，可能破坏选择性授权语义。

最新构想层已明确：所有签名消息默认包含交易头内的版本、时间戳和 `MintPKHash`（如果有）；未指定部分不包含；`SIGIN_ALL` 与 `SIGIN_SELF` 包含对应输入的解锁脚本。Coinbase 不使用选择性授权，而是对整个交易签名。

## Decision

普通交易签名消息必须按授权标记选择性构造，而不是无条件签署 TxID。建议结构如下：

```text
SigMsg = chain_context || tx_header_common || input_scope || output_scope || auth_flag || current_input_index
```

- `chain_context` 使用 conception 的 `Protocol-ID || Chain-ID || Genesis-ID || Bound-ID`。
- `tx_header_common` 固定包含 `Version`、`Timestamp` 和非空 `MintPKHash`。
- `input_scope` 根据 `SIGIN_ALL` 或 `SIGIN_SELF` 编码输入范围，且包含被纳入输入的解锁脚本字节。
- `output_scope` 根据 `SIGOUT_ALL`、`SIGOUT_SELF` 和辅项标记编码输出范围。
- 未被授权的字段完全省略，不使用空摘要占位，除非后续授权标记另行定义。
- Coinbase 签名消息为完整 Coinbase TxID 加链上下文，不使用上述选择性授权范围。
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

- 授权标记组合的精确字节顺序和重复字段去重规则。
- Coinbase “完整 TxID 加链上下文”的签名消息是否还需专用域标签或固定前缀。

# DEC-0602: Network Summary TxID Profile（网络概要交易 ID 配置）

Status: Proposed

## Context（背景）

Conception 建议区块概要中每个交易 ID 仅截取前 16 字节，以降低同步数据量；碰撞时发布者补充更长信息。短摘要格式、碰撞回退和与完整交易树的关系尚未冻结。

## Decision（决策）

建议区块概要格式：

```text
Summary = BlockID || TxCount || TxIDPrefixLen || TxIDPrefix* || OptionalResolution*
```

规则：

- 默认 `TxIDPrefixLen = 16`。
- `TxIDPrefix` 按区块交易序列顺序排列，包含 Coinbase。
- 接收方发现本地候选交易中有多个匹配时，请求指定序位的完整 TxID。
- 发布方可对碰撞位置提供更长前缀或完整 TxID。
- 最终验证必须使用完整 TxID 序列计算交易树根。

## Rationale（理由）

16 字节前缀足以覆盖常规同步场景，并显著降低区块概要大小。碰撞回退不应影响共识，因为最终哈希树必须使用完整 TxID。

## Consequences（影响）

- 区块概要只是网络优化，不是共识数据。
- 节点不得因短前缀无法解析就接受不完整区块。
- 恶意发布方提供错误摘要会在完整交易树验证阶段失败。

## Conception References（构想层依据）

- `docs/conception/附.组队校验.md#同步优化`
- `docs/conception/附.交易.md#输入项`

## Open Questions（开放问题）

- `TxIDPrefixLen` 是否固定为 16，还是允许协商。
- 碰撞回退消息是否按序位请求，还是按前缀请求。
- 区块概要是否需要发布方签名或仅依赖区块证明包。

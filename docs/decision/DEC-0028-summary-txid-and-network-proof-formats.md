# DEC-0028: Summary TxID and Network Proof Formats（概要 TxID 与网络证明格式）

Status: Proposed

## Context

网络同步可使用短 TxID 降低传输量，但 conception 已明确交易输入短引用主体规则，二者不能混同。短 TxID 只应是网络优化，不得进入共识哈希。

## Decision

建议如下：

- 区块同步摘要可使用 TxID 前 16 字节作为概要 TxID。
- 概要 TxID 只用于节点间匹配本地已知交易，不进入区块头、TxID、CheckRoot、签名消息或状态指纹。
- 若同一摘要或本地候选集合中出现前缀碰撞，接收方必须请求完整 TxID 或完整交易。
- 网络证明消息必须携带 `Protocol-ID`、`Chain-ID`、当前高度、相关 BlockID 和消息签名。
- 网络证明格式只证明对端声明，不作为链上共识事实。

## Rationale

将概要 TxID 限定在网络层，可避免前缀碰撞影响共识安全。

## Consequences

同步实现需要保留碰撞回退路径。轻节点不能用概要 TxID 完成最终交易验证。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/附.组队校验.md`
- `docs/conception/blockchain.md`
- `docs/conception/2.共识-端点约定.md`

## Open questions

- 概要 TxID 长度是否固定 16 字节，或允许协商更长长度。
- 网络证明签名是否使用节点身份密钥，还是链上账户密钥。

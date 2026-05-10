# DEC-0018: 交易过期边界语义

## Status（状态）

Accepted

## Context（背景）

`conception/2.共识-端点约定.md` 已将未确认交易超过 `240` 个区块视为过期失效定义为协议，并明确区块不得收录未来交易。实现层仍需固定按高度判定时的边界语义，避免不同节点在刚好跨越 240 个区块时产生分歧。

## Decision（决策）

交易过期按交易头 `Timestamp` 映射到标准区块时间高度后判定：

```text
txHeight = floor((TxHeader.Timestamp - GenesisBlockTime) / BlockInterval)
expired  = H > txHeight + 240
```

其中：

- `H` 为待收录交易的区块高度。
- `TxHeader.Timestamp` 与 `GenesisBlockTime` 的单位均为 Unix 毫秒。
- `BlockInterval` 为固定 6 分钟，即 `360000` 毫秒。
- `HeightAtBlockTime` 使用创世块时间戳与固定区块间隔推导，不使用本地时钟。
- 若 `TxHeader.Timestamp` 不在标准区块时间整点，`txHeight` 向下取整。
- 若 `TxHeader.Timestamp < GenesisBlockTime`，该交易非法。
- 计算 `TxHeader.Timestamp - GenesisBlockTime` 时必须检查 `int64` 溢出，或使用可证明安全的整数计算。
- 当 `H > txHeight + 240` 时，该交易已过期，不得收录。
- 当 `H <= txHeight + 240` 时，交易未因 240 区块期限过期，仍需通过其它合法性验证。
- 若 `TxHeader.Timestamp > BlockTime(H)`，该交易是未来交易，不得被高度 `H` 的区块收录。

## Rationale（理由）

- conception 已固定 240 个区块期限，本决策只补充严格边界。
- 使用标准区块时间高度避免节点本地时钟差异进入共识。
- `H > txHeight + 240` 保留第 240 个区块边界本身，使“超过 240 个区块”与构想层文字一致。

## Consequences（影响）

- 交易池可在当前高度满足 `H > txHeight + 240` 后删除未确认交易。
- 区块验证必须独立检查未来交易和过期交易。
- 测试应覆盖 `H == txHeight + 240`、`H == txHeight + 241`、非整点时间戳向下取整、早于创世时间和未来时间戳。

## Conception Relationship（与构想关系）

- 补充 `conception/2.共识-端点约定.md` 中交易过期协议的高度边界。
- 不改变 240 个区块期限，也不改变未来交易不得收录的构想层规则。

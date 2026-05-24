# DEC-0102: Signature Message Profile（签名消息配置）

Status: Proposed

## Context（背景）

Conception 定义了普通交易的授权标记和链标识前缀，但签名消息的精确字段顺序、缺省字段、输入/输出子集编码和 Coinbase 签名消息尚未冻结。

## Decision（决策）

建议签名消息统一为：

```text
DomainTag("signature.message") || ChainScope || SigScope || TxHeaderCore || CoveredInputs || CoveredOutputs
```

其中：

- `ChainScope = Protocol-ID || Chain-ID || Genesis-ID || Bound-ID`。
- `SigScope = chk_type || auth_flag || input_index`。
- `TxHeaderCore` 固定包含 `Version`、`Timestamp` 和可选 `MintPKHash`。
- `CoveredInputs` 按 `SIGIN_ALL` 或 `SIGIN_SELF` 选择；含解锁脚本，但不含见证信息。
- `CoveredOutputs` 按 `SIGOUT_ALL` 或 `SIGOUT_SELF` 选择，并按 `SCRIPT`、`CONTENT`、`RECEIVER`、`OUTPUT` 细分。
- 未被授权标记覆盖的字段不进入消息。

Coinbase 签名建议：

- Coinbase 不使用普通授权标记。
- Coinbase 交易签名消息为完整 Coinbase TxID，外加链标识域。
- 铸造者对区块 `CheckRoot` 的签名独立于 Coinbase 交易签名。

## Rationale（理由）

签名消息需要同时支持普通支付授权、局部授权和链重放隔离。把链标识作为签名前缀与 conception 一致。Coinbase 无输入，使用完整交易签名可避免授权子集带来的歧义。

## Consequences（影响）

- 签名验证必须知道当前输入序位。
- `SIGOUT_SELF` 在输入序位没有对应输出时应失败。
- 剪枝见证后，长期安全依赖 TxID 和区块哈希链，不依赖签名重验。

## Conception References（构想层依据）

- `docs/conception/blockchain.md#主链和分叉标识`
- `docs/conception/附.交易.md#签名消息`
- `docs/conception/6.脚本系统.md#例币金支付验证`

## Open Questions（开放问题）

- `Bound-ID` 为空时是否编码零长度字段，还是从 `ChainScope` 中省略。
- `SCRIPT|CONTENT|RECEIVER|OUTPUT` 组合冲突时如何规范化。
- 多签签名集合是否必须按公钥哈希排序，还是保留见证提供顺序。

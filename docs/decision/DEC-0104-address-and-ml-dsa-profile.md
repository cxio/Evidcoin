# DEC-0104: Address and ML-DSA Profile（地址与 ML-DSA 配置）

Status: Proposed

## Context（背景）

Conception 已明确单签和多签公钥哈希算法，以及地址校验码流程。但地址前缀清单、Base58 字母表、ML-DSA 参数、签名序列化和公钥编码尚未冻结。

## Decision（决策）

地址建议：

- 地址文本为 `prefix || Base58(pubKeyHash || checksum)`。
- `checksum = last4(SHA2-256(SHA2-256(prefix || pubKeyHash)))`。
- Base58 使用 Bitcoin 字母表。
- `prefix` 参与校验码，但不进入 Base58 负载。
- 主网、测试网和开发网必须使用不同 prefix。

公钥哈希建议：

- 单签：`SHA3-256(BLAKE2b-512(pubKeyBytes))`。
- 多签：各公钥先计算 `BLAKE3-256(pubKeyBytes)`，字典序排序后串联，前置 `m || n` 两字节，再计算 `SHA3-256(BLAKE2b-512(...))`。
- 多签中 `m` 和 `n` 均不得为 0，且 `m <= n`。

ML-DSA 建议：

- 使用 ML-DSA-65。
- 公钥、私钥和签名序列化采用所选库的 canonical byte encoding。
- 签名验证输入为 DEC-0102 定义的签名消息字节序列。

## Rationale（理由）

地址校验流程已由 conception 明确，Decision 只补充前缀、Base58 字母表和密码学序列化边界。多签先哈希再排序可保护公钥隐私并稳定地址生成。

## Consequences（影响）

- 地址 prefix 变更会导致校验码变化。
- 不同网络地址不能混用。
- 若 Go 标准库与第三方库的 ML-DSA 编码不一致，必须在实现前冻结一种 profile。

## Conception References（构想层依据）

- `docs/conception/附.交易.md#单签名`
- `docs/conception/附.交易.md#多重签名的地址`
- `docs/conception/附.交易.md#公钥哈希的地址编码`

## Open Questions（开放问题）

- 主网、测试网、开发网 prefix 的具体文本。
- 重复公钥在多签地址中是非法、去重，还是按原样参与排序。
- ML-DSA-65 使用标准库还是 `circl`，以及两者测试向量是否一致。

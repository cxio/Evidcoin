# DEC-0005: Address Encoding（地址编码）

Status: Proposed

## Context

conception 规定公钥哈希为 `SHA3-256(BLAKE2b-512(pubKey))`，并定义文本地址为前缀、公钥哈希、校验码和 Base58 的组合。`FN_ADDRESS` 指令说明使用两次哈希取校验码，而 `附.交易.md` 的地址附录只说执行哈希运算，二者需要统一。

## Decision

建议地址文本编码如下：

- 二进制账户标识为 32 字节公钥哈希或复合公钥哈希。
- 校验输入为 `prefix || pubKeyHash`。
- 校验码为 `SHA3-256(SHA3-256(checkInput))` 的末尾 4 字节。
- Base58 负载为 `pubKeyHash || checksum`，最终文本为 `prefix || base58(payload)`。
- 解码时根据已知 `prefix` 分离文本，校验通过后返回 32 字节公钥哈希。

## Rationale

该方案与 `FN_ADDRESS` 的两次哈希描述一致，同时保留 conception 的前缀参与校验和前缀不进入 Base58 负载的结构。

## Consequences

在作者裁决前，地址测试向量和 UI 输入校验应标记为实验性。`FN_ADDRESS` 与地址附录的一次/两次哈希差异记录为开放问题。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/Instruction/16.函数指令.md`
- `docs/conception/blockchain.md`

## Open questions

- 校验码是否最终采用两次 SHA3-256。
- 主网、测试网和私有链的 `prefix` 字符串清单尚未定义。

# DEC-0005: Address Encoding（地址编码）

Status: Accepted

## Context

conception 规定公钥哈希为 `SHA3-256(BLAKE2b-512(pubKey))`，并定义文本地址为前缀、公钥哈希、校验码和 Base58 的组合。最新构想层已明确校验码算法为两次 `SHA2-256`。

## Decision

地址文本编码如下：

- 二进制账户标识为 32 字节单公钥哈希或复合公钥哈希。
- 校验输入为 `prefix || pubKeyHash`。
- 校验码为 `SHA2-256(SHA2-256(checkInput))` 的末尾 4 字节。
- Base58 负载为 `pubKeyHash || checksum`，最终文本为 `prefix || base58(payload)`。
- 解码时根据已知 `prefix` 分离文本，校验通过后返回 32 字节公钥哈希。
- 多签地址的复合公钥哈希构造跟随 conception：先对每个公钥取 `BLAKE3-256(pubKey)`，按字典序排序并串联，再前置 `m/N` 配比计算 `SHA3-256(BLAKE2b-512(...))`。

## Rationale

该方案直接跟随 `附.交易.md` 与 `FN_ADDRESS` 的两次哈希流程。校验码使用 SHA2-256 只用于人类可读地址的错误检测，不改变链上公钥哈希算法。

## Consequences

旧草案中使用双 `SHA3-256` 的地址测试向量全部废弃。钱包和 UI 必须把地址前缀作为校验输入的一部分，但前缀不进入 Base58 负载。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/Instruction/16.函数指令.md`
- `docs/conception/blockchain.md`

## Open questions

- 主网、测试网和私有链的 `prefix` 字符串清单尚未定义。

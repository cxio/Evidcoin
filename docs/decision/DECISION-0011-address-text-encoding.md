# DECISION-0011: 地址文本编码

## Status（状态）

Accepted

## Context（背景）

`docs/conception/附.交易.md` 已规定地址编码流程：公钥哈希添加识别前缀后计算校验码，公钥哈希附加校验码后编码为文本，最后再前置识别前缀。它没有固定识别前缀、文本编码算法和校验哈希算法。

## Decision（决策）

地址文本格式为：

```text
address = "Cx" || Base58(AddressHash || checksum)
```

字段规则：

- 识别前缀为 ASCII 字符串 `Cx`。
- `AddressHash` 为 32 字节公钥哈希。
- `checksum = SHA3-256("Cx" || AddressHash)` 的末尾 4 字节。
- Base58 使用 Bitcoin 字母表 `123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz`。

校验时先拆分 `Cx` 前缀，再 Base58 解码，提取 32 字节公钥哈希和 4 字节校验码，重新计算并比较校验码。

## Rationale（理由）

- `Cx` 简短，便于区分 Evidcoin 地址。
- Base58 避免常见易混字符，适合人类输入。
- 校验码计算包含前缀，符合 conception 中“公钥哈希前置识别前缀再哈希”的流程。

## Consequences（影响）

- 地址编码实现必须把前缀纳入校验码计算。
- 变更前缀或 checksum 规则会改变所有地址文本，应视为破坏性变更。

## Conception Relationship（与构想关系）

- 补充 conception 已给出的地址编码流程中的具体前缀和编码算法。
- 本决策按 conception 使用“末尾 4 字节校验码”。

# ADR-0020: 地址文本编码前缀

## Status（状态）

Accepted

## Context（背景）

Evidcoin 地址的内部表示为 32 字节的 `AddressHash`（SHA3-256(BLAKE2b-512(pubkey))）。为了在用户界面、钱包、API 等场景中展示地址，需要定义一种人类可读的文本编码格式。

提案层将地址文本编码（前缀、Base58、checksum 等）标记为完全未固定（OQ-015）。

## Decision（决策）

**地址文本编码前缀为 `Cx`**，后接 Base58Check 编码的地址哈希。

完整格式：

```
address_text = "Cx" + Base58Check(AddressHash)
```

### Base58Check 规范

采用 Bitcoin 风格的 Base58Check 编码：
1. 对 `AddressHash`（32 字节）计算校验码：`checksum = SHA3-256(SHA3-256(AddressHash))[0:4]`（前 4 字节）
2. 拼接：`payload = AddressHash || checksum`（36 字节）
3. 对 payload 进行 Base58 编码（使用 Bitcoin 字母表：`123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz`）
4. 在编码结果前加前缀 `Cx`

### 示例格式

```
Cx7nK3mPqRsW2xYvBcDfGhJkLmNpQtUvWxYzAaBb...
```

## Rationale（理由）

1. **`Cx` 前缀**：`C` 取自 "Credit/Coin"（凭信/币金），`x` 代表通用地址（cross-type）。两个字符的前缀足以区分 Evidcoin 地址与其他区块链地址，同时保持简短。
2. **Base58Check**：业界广泛使用（Bitcoin、许多山寨币），有充分的库支持（`github.com/mr-tron/base58` 已在 AGENTS.md 中列为计划依赖），可读性好（无 0/O/I/l 等易混淆字符），内置 4 字节校验码可防止地址输错。
3. **SHA3-256 双哈希校验码**：与项目整体的 SHA3 优先策略一致，避免引入额外的哈希算法。

## Consequences（影响）

- 需在 `docs/proposal/02.Cryptography-And-Hashing.md` 或专门的地址规范文档中记录完整格式。
- `pkg/crypto` 中需实现 `EncodeAddress(AddressHash) string` 和 `DecodeAddress(string) (AddressHash, error)` 两个函数。
- OQ-015 关闭。

## References（参考）

- `docs/proposal/02.Cryptography-And-Hashing.md` — 地址哈希定义
- `docs/proposal/03.Identifiers-And-Constants.md` — AddressHash 类型
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-015
- AGENTS.md — `github.com/mr-tron/base58` 计划依赖

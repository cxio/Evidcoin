# DEC-0002: 哈希域分隔标签格式

## Status（状态）

Accepted

## Context（背景）

`conception/blockchain.md` 已为不同用途分配哈希算法，但未规定域分隔标签。缺少统一域分隔会让不同语义的哈希输入依赖调用者自行区分，增加 hash confusion 风险。

## Decision（决策）

Domain Tag 采用如下字节格式：

```text
"Evidcoin:" || Purpose || ":v" || Version || 00
```

字段含义：

| 字段 | 规则 |
|------|------|
| `"Evidcoin:"` | ASCII 固定前缀。 |
| `Purpose` | ASCII 用途名。 |
| `":v"` | ASCII 版本分隔符。 |
| `Version` | 按 `DEC-0001` 编码的无符号 varint，当前为 `01`。 |
| `00` | NUL 终止符。 |

当前用途名：

| 用途 | Purpose |
|------|---------|
| 区块头 | `BlockHeader` |
| 交易头 | `TxHeader` |
| CheckRoot | `CheckRoot` |
| 哈希树分支 | `TreeBranch` |
| 哈希树叶子 | `TreeLeaf` |
| 状态叶子 | `StateLeaf` |
| 状态数据 | `StateData` |
| 附件指纹 | `Attachment` |
| 地址哈希 | `AddressHash` |
| PoH 铸凭哈希 | `PoHCredential` |
| Coinbase 输入哈希 | `CoinbaseInputs` |
| 签名消息 | `SignMessage` |
| 同步池签名 | `SyncPool` |
| 分叉裁决签名 | `ForkDecision` |
| 区块铸造签名 | `BlockMintSign` |

使用方式：

```text
Hash(domainTag || payload)
```

## Rationale（理由）

- 可读 ASCII 前缀便于调试。
- 版本字段允许未来按用途演进。
- NUL 终止符避免用途名前缀混淆。
- PoH 铸凭哈希采用唯一 purpose，与 conception 固定的单层计算结构一致。

## Consequences（影响）

- 哈希函数封装应内部固定 domain tag，不应让调用方任意传入。
- 修改 Purpose 或版本会改变历史哈希值，必须视为协议升级。

## Conception Relationship（与构想关系）

- 补充 conception 中哈希算法分配之外的域隔离格式。
- 不改变任何 conception 已固定的哈希算法。

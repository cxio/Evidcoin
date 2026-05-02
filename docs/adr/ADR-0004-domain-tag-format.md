# ADR-0004: 哈希域分隔标签（Domain Tag）格式

## Status（状态）

Accepted

## Context（背景）

为防止不同用途的哈希输入在语义上发生混淆（hash confusion），提案层（`docs/proposal/02.Cryptography-And-Hashing.md`）要求为每种哈希用途设置独立的域分隔标签（domain tag）。

但具体的字节格式跨 5 个提案文档反复被提及为"未固定"状态，涉及场景包括：
- CheckRoot 计算
- 哈希树节点（分支/叶子）
- 签名消息混入
- 铸凭哈希

该问题属于阻塞性未决项（OQ-003）：domain tag 一旦确定后，修改成本极高（所有历史哈希值全部变更）。

## Decision（决策）

**Domain Tag 格式**：

```
"Evidcoin:" + Purpose + ":v" + version_number + "\x00"
```

其中：

| 部分 | 说明 |
|------|------|
| `"Evidcoin:"` | 固定项目前缀，ASCII 编码 |
| `Purpose` | 表示哈希用途的短字符串，见下表，ASCII 编码 |
| `":v"` | 版本分隔符，ASCII 编码 |
| `version_number` | 无符号变长整数（LEB128，见 ADR-0003），当前均为 `1`，即字节 `0x01` |
| `"\x00"` | NUL 终止符，确保字节流的明确边界 |

### Purpose 字符串对照表

| 哈希用途 | Purpose 字符串 |
|----------|---------------|
| 区块头（BlockID） | `BlockHeader` |
| 交易头（TxID） | `TxHeader` |
| CheckRoot | `CheckRoot` |
| 哈希树分支节点 | `TreeBranch` |
| 哈希树叶子节点 | `TreeLeaf` |
| UTXO/UTCO 状态指纹叶子 | `StateLeaf` |
| UTXO/UTCO DataID | `StateData` |
| 附件指纹 | `Attachment` |
| 地址哈希（AddressHash） | `AddressHash` |
| 铸凭哈希内层（见 ADR-0001） | `MintInner` |
| 铸凭哈希外层（MintHash） | `MintHash` |
| Coinbase HashInputs（见 ADR-0010） | `CoinbaseInputs` |
| 签名消息（链识别混入） | `SignMessage` |
| 同步池签名消息 | `SyncPool` |
| 分叉裁决签名消息 | `ForkDecision` |

### 示例

```
BlockHeader 的 domain tag 字节序列：
"Evidcoin:BlockHeader:v" + 0x01 + 0x00
= 45 76 69 64 63 6F 69 6E 3A 42 6C 6F 63 6B 48 65 61 64 65 72 3A 76 01 00
```

### 使用方式

Domain tag 作为哈希函数的**前缀输入**，与有效载荷拼接后一同送入哈希函数：

```
Hash(domain_tag || payload)
```

若某哈希函数本身支持 personalization（如 BLAKE2b），则使用 personalization 参数；否则直接前置拼接。

## Rationale（理由）

1. **可读性**：前缀采用人类可读的 ASCII 字符串，便于调试和日志输出识别。
2. **可扩展性**：`":v" + version_number` 允许未来在不破坏现有用途的情况下对特定哈希算法或语义进行版本化演进。
3. **明确边界**：NUL 终止符（`\x00`）防止 Purpose 字符串与紧随其后的有效载荷发生前缀碰撞（如 `"Sign"` 和 `"Signature"` 不可能混淆）。
4. **统一管理**：集中维护 Purpose 字符串对照表，避免各模块自行定义格式。

## Consequences（影响）

- 需在 `docs/proposal/02.Cryptography-And-Hashing.md` 中补充 Domain Tag 格式规范和对照表。
- `pkg/crypto` 中每个哈希用途函数（如 `HashBlockHeader`、`HashTreeBranch`）需在内部硬编码对应的 domain tag。
- 禁止调用方自行传入 domain tag；domain tag 应被封装在各哈希函数实现内部。
- OQ-003 关闭。

## References（参考）

- `docs/proposal/02.Cryptography-And-Hashing.md` — 密码学与哈希
- `docs/proposal/04.Hash-Trees.md` — 哈希树
- `docs/proposal/05.Blockchain-Core.md` — CheckRoot
- `docs/plan/01-Foundation-Types-Crypto.md` — Task 4
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-003
- ADR-0003（LEB128 varint，version_number 编码依赖）

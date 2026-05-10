# DEC-0001: PoH 铸凭哈希算法

## Status（状态）

Accepted（2026-05 修订：移除签名参与）

## Context（背景）

`docs/conception/1.共识-历史证明（PoH）.md` 已修改铸凭哈希流程，取消之前签名参与计算的设计。

`docs/conception/blockchain.md` 已明确铸凭哈希使用 `BLAKE3-256`，但没有明确详细的计算结构。

## Decision（决策）

PoH 铸凭哈希使用 `BLAKE3-256`，输出 32 字节。

```text
mintHash = BLAKE3-256(domainTag || pubKey || mintTxID || referenceMintHash || X)
```

字段规则：

- `mintTxID`：铸凭交易的 TxID（48 字节，按 conception/附.交易.md 定义）。
- `referenceMintHash`：评参区块（`-9` 号）的铸凭哈希（32 字节）。
- `X`：按 `DEC-0002` 编码的字节序列。
- `pubKey`：铸造者公钥的完整字节（铸凭交易首笔输入对应的公钥），编码与签名验证所用一致。
- `domainTag`：按 `DEC-0004` 格式构造，purpose 为 `"PoH-MintHash"`，version 为 `1`。
- `mintHash` 不依赖任何签名值；铸造者签名仅用于资格证明（见 conception/1.共识-历史证明（PoH）.md「择优凭证」），不参与本计算。

## Rationale（理由）

- 公开确定性形式彻底消除签名 grinding 路径，避免 PoH 退化为算力竞争。
- 引入 `pubKey` 作为输入因子既增强信息熵，也将铸凭哈希与铸造者身份强绑定，便于后续资格验证。
- 域分隔标签防止 `mintHash` 计算与系统中其它哈希用途产生跨用途碰撞或重放。

## Consequences（影响）

- 实现 `mintHash` 时不得引入签名值或任何运行时随机量。
- 铸造者签名机制独立保留，仅用于资格证明，与 `mintHash` 计算解耦。
- 测试向量应包含至少一组明确公钥与域标签的端到端用例。

## Conception Relationship（与构想关系）

- 不改变 conception 已明确的最终铸凭哈希 `BLAKE3-256` 选择。
- 不改变 conception 中铸造者签名作为资格证明的保留地位。

# DEC-0002: Domain Tags and Hash Profiles（域标签与哈希配置）

Status: Proposed

## Context（背景）

Conception 已指定区块头、交易头、树枝、附件、公钥哈希和铸凭哈希的算法分配，但尚未冻结域隔离标签、标签编码方式、版本号和各类哈希前像的精确分隔。

## Decision（决策）

建议采用以下域标签 profile：

- 域标签编码为 ASCII 字符串：`"Evidcoin/v1/" || name || 0x00`。
- 域标签必须作为哈希前像首段，不参与用户可控字段的长度解释。
- 同一算法不同用途必须使用不同标签，即使输入结构当前不可混淆。
- 建议初始标签：
  - `block.header`
  - `tx.header`
  - `tree.leaf`
  - `tree.branch`
  - `checkroot`
  - `utxo.leaf`
  - `utco.leaf`
  - `mint.hash`
  - `signature.message`
  - `attachment.fingerprint`
  - `attachment.piece-tree`
  - `address.single`
  - `address.multi`

算法 profile 沿用 conception：

- 区块头、交易头、CheckRoot、UTXO/UTCO 叶：`SHA3-384`。
- 通用哈希树分支、交易输出树根、附件片组树：`BLAKE3-256`。
- 附件完整指纹：`SHA3-512`。
- 公钥哈希：`SHA3-256(BLAKE2b-512(...))`。
- 铸凭哈希：`BLAKE3-256`。

## Rationale（理由）

域隔离可防止不同结构的相同字节前像产生跨用途混淆。字符串标签比纯整数表更易审计；尾部 `0x00` 分隔符可避免标签名与后续字段拼接歧义。

## Consequences（影响）

- 任何冻结后的哈希前像都必须列明域标签。
- 历史版本升级时，应通过 `Evidcoin/vN` 或具体结构版本实现隔离。
- 未冻结的标签不得进入不可变主网数据。

## Conception References（构想层依据）

- `docs/conception/blockchain.md#哈希策略`
- `docs/conception/附.交易.md`
- `docs/conception/5.信用结构.md#关于附件`
- `docs/conception/1.共识-历史证明（PoH）.md#铸凭哈希`

## Open Questions（开放问题）

- 标签是否作为可见常量进入实现，或压缩为注册表编号后进入前像。
- `BLAKE3` 是否使用 keyed mode；当前建议不使用 keyed mode，仅使用普通 hash 加域标签。

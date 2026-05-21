# DEC-0002: Domain Tags（域分隔标签）

Status: Accepted

## Context

conception 已为不同用途分配哈希算法，但没有统一定义域分隔标签。PoH、签名消息、状态指纹、Coinbase 特殊数据和服务证明若共用裸哈希输入，容易产生跨上下文复用风险。

最近 conception 已明确 Coinbase 省略 `HashInputs`，因此旧的 `coinbase.inputs` 域标签不再分配。

## Decision

域标签编码为：

```text
"Evidcoin" || 0x00 || ascii(purpose) || 0x00 || varuint(version)
```

- `purpose` 只允许 ASCII 字母、数字、点号、下划线和连字符。
- `version` 当前为 `1`，按 DEC-0001 的无符号 varint 编码。
- 域标签必须作为对应哈希输入的第一个片段。
- 已分配用途包括：`hash-tree.branch`、`hash-tree.leaf`、`tx.body`、`tx.witness`、`tx.sigmsg`、`coinbase.body`、`coinbase.mint-proof`、`poh.mint`、`state.utxo`、`state.utco`、`block.proof`、`network.summary`、`blockqs.proof`。
- `coinbase.inputs` 为历史废弃用途，不得在新测试向量或实现中使用。

## Rationale

显式零分隔可避免前缀拼接歧义，ASCII purpose 便于审计和测试向量展示。删除 `coinbase.inputs` 可避免 Decision 层继续保留与 Coinbase 省略输入字段相冲突的编码路径。

## Consequences

新增共识哈希用途必须先分配 purpose。历史测试向量需要注明域标签版本。若旧草案使用 `coinbase.inputs`，必须废弃并重新生成。

## Conception references

- `docs/conception/blockchain.md`
- `docs/conception/1.共识-历史证明（PoH）.md`
- `docs/conception/附.交易.md`

## Open questions

无。

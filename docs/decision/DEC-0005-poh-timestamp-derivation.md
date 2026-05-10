# DEC-0005: PoH 时间戳的推导与隔离

## Status（状态）

Accepted

## Context（背景）

`conception/blockchain.md` 明确区块头不含独立时间戳字段，区块时间通过高度推算。`conception/1.共识-历史证明（PoH）.md` 的铸凭哈希参数 `X` 中含 `timeStamp`。`conception/附.交易.md` 与 `DEC-0003` 又定义了交易头中的 `Timestamp int64`（毫秒）。两类时间戳同名但语义不同，实现者易混淆，可能导致 PoH 计算分歧。

## Decision（决策）

PoH 计算中使用的“区块时间戳”由高度按下式确定性推导：

```text
timeStamp_for_PoH(height) = genesisTime + height * BlockInterval
```

字段规则：

- 单位：毫秒。
- `BlockInterval = 6 * 60 * 1000` 毫秒。
- `genesisTime` 是协议常量，与创世块绑定，在协议参数表中固定（数值由实现层在创世前确定并冻结）。
- 该值仅用于 PoH 与其它需要“标准区块时间”的协议计算，不写入区块头，不参与编码与传输。
- 该值与交易头 `Timestamp` 字段无关：交易头 `Timestamp` 由交易作者填写、可在合理范围内偏移，是交易级时间戳。

实现层命名建议：

- 协议内部统一使用 `protocolBlockTime(height)` 或同义命名表示该确定性时间戳，避免与交易 `Timestamp` 同名。
- PoH 实现接收高度作为输入直接计算，不应从任何区块字段中读取该值。

## Rationale（理由）

- 显式命名隔离消除“哪个 timeStamp”歧义，便于跨实现产生一致的测试向量。
- 由高度推导保证全网 PoH 输入的确定性，无需任何节点本地时钟参与。
- 不写入区块头维持 `conception/blockchain.md` 既有结论，避免无意义的字节冗余。

## Consequences（影响）

- PoH 测试向量必须以“高度 → 推导时间戳 → X 计算”的链路生成。
- 实现中若误把交易头 `Timestamp` 用于 PoH，将产生不可互通的链，应在审查清单中单列。
- `genesisTime` 的具体数值变更属于破坏性协议变更。

## Conception Relationship（与构想关系）

- 显式化 `conception/1.共识-历史证明（PoH）.md` 已采用的“由高度推算”语义。
- 不改变 `conception/blockchain.md` 关于“区块头不含时间戳”的设定。
- 与 `DEC-0003` 中的交易头 `Timestamp` 字段在命名与用途上明确分离。

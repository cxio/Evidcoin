# DEC-0004: PoH 参数编码与碰撞处理

## Status（状态）

Accepted

## Context（背景）

`conception/1.共识-历史证明（PoH）.md` 已固定 PoH 铸凭哈希为单层 `mintHash = BLAKE3-256(...)`，并定义 `X := Bytes(timeStamp * Stakes * Mix)`。本决策不重述铸凭哈希算法主体，只补充 `X` 的字节编码，以及完全相同 `mintHash` 的择优池碰撞处理。

PoH 使用评参区块自身的 `Stakes` 字段参与 `X` 计算。`timeStamp` 的来源由 `DEC-0005` 明确。

PoH 铸凭哈希的 domain tag 使用 `DEC-0002` 中的 `DomainTag("PoHCredential")`。

## Decision（决策）

`X` 的计算使用任意精度非负整数，不允许固定宽度溢出截断。

```text
XBig = timeStamp * Stakes * Mix
X    = minimal_big_endian_bytes(XBig)
```

编码规则：

- 若 `XBig == 0`，编码为单字节 `0x00`。
- 若 `XBig > 0`，编码为无前导零的大端序最短字节序列。
- `timeStamp` 使用按区块高度确定性推导的 PoH 时间戳，单位为毫秒。
- `Stakes` 使用评参区块自身的 `Stakes` 字段。

若两个不同铸凭交易产生完全相同的 `mintHash`，节点只保留先到达择优池的候选者，拒绝后到达的同哈希候选者。

此规则只处理不同 `mintTxID` 的哈希碰撞，不处理同一铸造者在分叉上的双签场景。

## Rationale（理由）

- 三项乘积可能超过 `uint64`，截断会损失高位信息并导致实现差异。
- 最短大端编码具有唯一性，适合进入哈希输入。
- PoH 时间戳使用毫秒单位，与交易时间戳、区块标准时间和脚本环境保持一致。
- 铸凭哈希为 32 字节，真实碰撞概率极低。
- 先到先得与择优池的实时传播模型一致。
- 引入额外排序字段会增加协议复杂度，但几乎不会带来实际收益。

## Consequences（影响）

- 实现铸凭哈希时需要使用大整数或等价的无溢出乘法。
- 测试向量应覆盖 `Stakes = 0`、普通值和超过 64 位的乘积。
- 择优池实现需要以铸凭哈希为唯一键之一检测重复。
- 极端碰撞导致不同节点先到顺序不一致时，后续分叉选择机制处理结果。

## Conception Relationship（与构想关系）

- 补充 `Bytes(...)` 的规范化语义。
- 补充择优池同值碰撞的边界语义。
- 不改变 conception 已固定的单层 PoH 铸凭哈希算法主体。
- 不改变 conception 已明确的“值小者优”规则。

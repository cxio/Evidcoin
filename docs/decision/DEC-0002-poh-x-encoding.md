# DEC-0002: PoH 铸凭哈希 X 参数编码

## Status（状态）

Accepted

## Context（背景）

`docs/conception/1.共识-历史证明（PoH）.md` 定义了：

```text
X := Bytes(timeStamp * Stakes * Mix)
```

其中 `timeStamp` 已明确为 Unix 毫秒，`Stakes` 来自 `-27` 号区块，`Mix = 0x517cc1b727220a95`。但 `Bytes(...)` 的溢出行为和字节编码方式未在 conception 中固定。

## Decision（决策）

`X` 的计算使用任意精度非负整数，不允许固定宽度溢出截断。

```text
XBig = timeStamp * Stakes * Mix
X    = minimal_big_endian_bytes(XBig)
```

编码规则：

- 若 `XBig == 0`，编码为单字节 `0x00`。
- 若 `XBig > 0`，编码为无前导零的大端序最短字节序列。
- `timeStamp` 使用当前区块确定性时间戳，单位为毫秒。

## Rationale（理由）

- 三项乘积可能超过 `uint64`，截断会损失高位信息并导致实现差异。
- 最短大端编码具有唯一性，适合进入哈希输入。
- 毫秒单位与 conception 中交易时间戳、区块标准时间和脚本环境保持一致。

## Consequences（影响）

- 实现铸凭哈希时需要使用大整数或等价的无溢出乘法。
- 测试向量应覆盖 `Stakes = 0`、普通值和超过 64 位的乘积。

## Conception Relationship（与构想关系）

- 补充 `Bytes(...)` 的规范化语义。
- 不改变 PoH 对 `timeStamp`、`Stakes`、`Mix` 的 conception 定义。

# DEC-0015: Fork Tiebreaker（分叉平局裁决）

Status: Proposed

## Context

conception 已将分叉竞争窗口修订为 31 个区块，并给出平局时使用抗 ASIC `RandomX` 的示意算法。Decision 只补充字段拼接和实现边界；RandomX 具体参数仍需冻结。

## Decision

建议分叉平局裁决如下：

- 分叉链段竞争长度为 31 个区块；一方先胜出 16 个区块时提前结束。
- 新发现分叉长度必须小于等于 20 才进入评比。
- 对长度 20 且临近 #21 铸造时间才发现的分叉，按 conception 由本链当前区块择优池前 5 名签名确认是否接纳。
- 平局时计算 `tieValue = RandomX(seed, input)`。
- `seed = forkPointBlockID`，即分叉点区块 ID 的 48 字节原始值。
- `input = firstForkBlockID`，即目标分叉首块 ID 的 48 字节原始值。
- `tieValue` 按字典序比较，值小者胜出。
- 若 `tieValue` 仍相等，比较分叉首块 BlockID，值小者胜出。
- RandomX 的具体 profile、输出长度和库参数待裁决；冻结前不得写入最终共识实现。

## Rationale

31/16/20 参数和 RandomX 方向直接来自 conception。保留 Proposed 是因为 RandomX profile 未在 conception 中给出到可生成测试向量的程度。

## Consequences

分叉处理可以先实现非平局路径；平局路径需等待 RandomX 参数裁决或以实验标志隔离。旧 `HashX256` 名称废弃。

## Conception references

- `docs/conception/2.共识-端点约定.md`

## Open questions

- RandomX 的具体实现、参数集、输出截取长度和版本标识。
- 分叉首块 ID 比较是否需要带链方向或发现时间信息，当前建议不带。

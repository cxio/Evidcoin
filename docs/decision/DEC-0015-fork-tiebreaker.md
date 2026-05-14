# DEC-0015: Fork Tiebreaker（分叉平局裁决）

Status: Proposed

## Context

conception 已将分叉竞争窗口修订为 31 个区块，并给出平局时使用抗 ASIC `HashX` 的示意算法。但 `HashX` 未具体指定。

## Decision

建议分叉平局裁决如下：

- 分叉链段竞争长度为 31 个区块；一方先胜出 16 个区块时提前结束。
- 新发现分叉长度必须小于等于 20 才进入评比。
- 平局时计算 `tieValue = HashX256(forkPointBlockID || firstForkBlockID)`，值小者胜出。
- 若 `tieValue` 仍相等，比较分叉首块 BlockID，值小者胜出。
- `HashX256` 的具体算法待裁决；冻结前不得写入共识实现。

## Rationale

31/16/20 参数直接来自 conception。HashX 未定义，因此只能 Proposed。

## Consequences

分叉处理可以先实现非平局路径；平局路径需等待 HashX 算法裁决或以实验标志隔离。

## Conception references

- `docs/conception/2.共识-端点约定.md`

## Open questions

- `HashX256` 是否采用现有算法，还是项目自定义参数化算法。
- 分叉首块 ID 比较是否需要带链方向或发现时间信息，当前建议不带。

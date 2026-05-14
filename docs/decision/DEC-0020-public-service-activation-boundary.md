# DEC-0020: Public Service Activation Boundary（公共服务激活边界）

Status: Accepted

## Context

conception 的 PoH 初段规则说明百日扩张期间 Coinbase 只有一笔输出，无公共服务奖励；第 101 日开始接受公共服务并启动对外奖励。

## Decision

- 区块 0 为创世块。
- 区块 1 至 24000 属于百日扩张范围，Coinbase 不包含公共服务奖励输出。
- 从第 24001 个区块开始，Coinbase 启用公共服务奖励输出和兑奖槽。
- 百日扩张期间 Coinbase 的单输出目标和收益归属由创世/启动规范指定，Decision 不另行伪造。

## Rationale

以 conception 给出的 `7201-24000` 百日阶段作为边界，下一块进入第 101 日后的正常阶段。

## Consequences

Coinbase 验证需要按高度切换输出集合。公共服务节点在激活前不能凭空获得链上奖励。

## Conception references

- `docs/conception/1.共识-历史证明（PoH）.md`
- `docs/conception/4.激励机制.md`

## Open questions

无。

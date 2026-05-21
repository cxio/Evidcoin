# DEC-0020: Public Service Activation Boundary（公共服务激活边界）

Status: Accepted

## Context

conception 说明百日扩张期间 Coinbase 中没有对公共服务的奖励；第 101 日开始接受公共服务并启动对外奖励。旧 Decision 曾把百日扩张误写为 Coinbase 单输出，已与创世 Coinbase 两输出示例冲突。

## Decision

- 区块 0 为创世块。
- 区块 0 至 24000 属于百日扩张范围，Coinbase 不包含公共服务奖励输出和兑奖槽。
- 从高度 `24001` 的区块开始，Coinbase 启用公共服务奖励输出和兑奖槽。
- `24001` 起 Coinbase 必须包含 5 类奖励目标：铸凭者、校验组、Blockqs、Depots、STUN。
- 百日扩张期间 Coinbase 输出集合不由本 DEC 固定为单输出；创世块已明确为校验组与铸凭者两项币金输出。

## Rationale

以 conception 给出的百日扩张结束高度 `24000` 作为边界；其中 `7201-24000` 是百日扩张中的抽奖扩张阶段，下一块 `24001` 进入第 101 日后的正常阶段。

## Consequences

Coinbase 验证需要按高度切换输出集合。公共服务节点在激活前不能凭空获得链上奖励。公共服务奖励的兑奖窗口、31 区块安全边界和 48 区块确认窗口见 DEC-0018。

## Conception references

- `docs/conception/blockchain.md`
- `docs/conception/4.激励机制.md`
- `docs/conception/附.交易.md`

## Open questions

无。

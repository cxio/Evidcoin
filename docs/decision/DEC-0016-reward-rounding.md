# DEC-0016: Reward Rounding（奖励取整）

Status: Accepted

## Context

conception 已定义奖励比例和“不能整除的余数留给最终分配者”。需要统一分配顺序和余数归属。

构想层在不同文件中列举 Coinbase 输出时顺序不完全一致；Decision 以 `docs/conception/附.交易.md` 的输出项配置值作为编码顺序。

## Decision

- 所有金额以 `chx` 为最小单位计算，使用整数除法向下取整。
- Coinbase 奖励输出编号按输出项配置值顺序：铸凭者、校验组、Blockqs、Depots、STUN。
- 先计算前四项的向下取整金额，最后一项 STUN 接收剩余金额。
- 公共服务奖励未激活时，Coinbase 只省略公共服务输出；不得把未激活服务的份额发给空地址。
- 百日前 Coinbase 不应被实现为固定“单输出”；创世块明确有校验组和铸凭者两项币金输出，启动期输出集合由创世/启动规范和公共服务激活边界共同约束。

## Rationale

按 Coinbase 输出配置值处理，可统一编码顺序和余数归属，并避免叙述性列举顺序影响共识编码。

## Consequences

公共服务未激活阶段的输出集合需要由 DEC-0020 和创世/启动规范共同确定。

## Conception references

- `docs/conception/4.激励机制.md`
- `docs/conception/附.交易.md`
- `docs/conception/blockchain.md`

## Open questions

无。

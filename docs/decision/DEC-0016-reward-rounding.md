# DEC-0016: Reward Rounding（奖励取整）

Status: Accepted

## Context

conception 已定义奖励比例和“不能整除的余数留给最终分配者”。需要统一分配顺序和余数归属。

## Decision

- 所有金额以 `chx` 为最小单位计算，使用整数除法向下取整。
- Coinbase 奖励输出编号按 conception 的配置值顺序：铸凭者、校验组、Blockqs、Depots、STUN。
- 先计算前四项的向下取整金额，最后一项 STUN 接收剩余金额。
- 若公共服务奖励未激活，则按激活边界规则重新定义输出集合，不能把未激活服务的份额错误发给空地址。

## Rationale

按 Coinbase 输出编号顺序处理，可同时统一编码顺序和余数归属。

## Consequences

公共服务未激活阶段的单输出 Coinbase 需要由 DEC-0020 明确边界。

## Conception references

- `docs/conception/4.激励机制.md`
- `docs/conception/附.交易.md`

## Open questions

无。

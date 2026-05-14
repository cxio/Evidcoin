# DEC-0018: Coinbase Serialization and Award Slots（Coinbase 序列化与兑奖槽）

Status: Proposed

## Context

conception 已定义 Coinbase 主体信息、输出配置、公共服务奖励比例和兑奖窗口。Decision 只补充字段顺序、服务命名、计算顺序与输出编码顺序。

## Decision

建议 Coinbase body 编码为：

```text
CoinbaseBody = block_height || mint_proof || total_reward || free_data_len || free_data || reward_outputs || award_slots
```

- `reward_outputs` 按 Coinbase 输出编号编码：铸凭者、校验组、Blockqs、Depots、STUN。
- 公共服务计算或评估可按服务实现需求执行，但输出编码顺序必须以 Coinbase 输出编号为准。
- 服务命名在正式文档中统一显示为 `Blockqs`、`Depots`、`STUN`；若引用 conception 中小写写法，仅视为同一服务的名称变体。
- `award_slots` 建议为 `BlockqsSlots[6] || DepotsSlots[6] || STUNSlots[6]`，每项覆盖前 48 个区块。
- 当前高度 `H` 的 Coinbase 只能标记 `[H-48, H-1]` 的公共服务奖励；满足 31 个区块安全边界且达到确认数后可兑奖。

## Rationale

输出编号来自 conception，是最稳定的顺序依据。计算顺序与输出编码顺序分离，可避免把实现优化误写成共识格式。

## Consequences

验证实现需要从最近 48 个区块的 Coinbase 中统计公共服务确认。该 DEC 依赖 DEC-0017 与交易输出 payload 编码。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/4.激励机制.md`
- `docs/conception/Instruction/15.系统指令.md`

## Open questions

- `mint_proof` 的精确字段顺序。
- 公共服务槽位内第 0 位对应 `H-1` 还是 `H-48`。
- 未确认截留进入第 49 个后续区块 Coinbase 的编码位置。

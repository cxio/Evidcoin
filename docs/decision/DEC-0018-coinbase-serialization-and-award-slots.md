# DEC-0018: Coinbase Serialization and Award Slots（Coinbase 序列化与兑奖槽）

Status: Proposed

## Context

conception 已定义 Coinbase 主体信息、输出配置、公共服务奖励比例和兑奖窗口。最近修订明确 Coinbase 特有字段位于交易头示意中，且 `HashInputs` 省略。

## Decision

建议 Coinbase 规范编码按下列逻辑冻结：

```text
CoinbaseHeader = version || hash_outputs || timestamp || mint_pk_hash || block_height || mint_proof? || free_data_len || free_data
CoinbaseBody = reward_outputs || award_slots?
```

- Coinbase 不编码 `HashInputs`。
- `mint_pk_hash` 为 Coinbase 铸造者身份，创世块和普通 Coinbase 均应设置。
- `mint_proof` 为择优凭证；创世块省略。
- `free_data` 长度 `<256`。
- `reward_outputs` 按 Coinbase 输出配置值顺序编码：铸凭者、校验组、Blockqs、Depots、STUN。
- 百日前 Coinbase 不包含 Blockqs、Depots、STUN 公共服务奖励输出；从高度 `24001` 开始必须包含 5 类输出。
- 服务命名在正式文档中统一显示为 `Blockqs`、`Depots`、`STUN`；若引用 conception 中小写写法，仅视为同一服务的名称变体。
- `award_slots` 建议为 `BlockqsSlots[6] || DepotsSlots[6] || STUNSlots[6]`，每项覆盖前 48 个区块。
- 当前高度 `H` 的 Coinbase 只能标记 `[H-48, H-1]` 的公共服务奖励；满足 31 个区块安全边界且达到确认数后可兑奖。
- 收益总额只包含铸币、未销毁交易费和兑奖截留；交易费销毁的 50% 不进入 Coinbase 输出。

## Rationale

输出配置值来自 conception，是最稳定的顺序依据。将 Coinbase 特殊字段从普通交易输入根中剥离，可避免旧占位 `HashInputs` 继续污染 TxID。

## Consequences

验证实现需要从最近 48 个区块的 Coinbase 中统计公共服务确认。该 DEC 依赖 DEC-0003 的字段宽度和交易输出 payload 编码；`HashInputs` 移除直接跟随 conception，DEC-0017 仅保留历史废止说明。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/4.激励机制.md`
- `docs/conception/blockchain.md`
- `docs/conception/Instruction/15.系统指令.md`

## Open questions

- `mint_proof` 的精确字段顺序。
- 公共服务槽位内第 0 位对应 `H-1` 还是 `H-48`。
- 未确认截留进入第 49 个后续区块 Coinbase 的编码位置。

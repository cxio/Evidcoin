# DEC-0401: Coinbase Serialization, Rewards and Award Slots（Coinbase 序列化、奖励与兑奖槽）

Status: Proposed

## Context（背景）

Conception 已明确 Coinbase 无输入、收益来源、奖励比例、公共服务 48 块兑奖窗口、百日前不启用公共服务奖励，以及长期发行规则。但输出顺序、取整、兑奖槽位、截留回收和百日前 profile 仍需冻结。

## Decision（决策）

本 DEC 中涉及 Coinbase 输出顺序和百日前奖励 profile 的内容受 `CONCEPTION-CONFLICTS.md` 的 `C-001`、`C-016` 阻塞。冻结前不得把下列选项作为可实现的最终规则。

可先保留 Coinbase 输出配置值候选，具体顺序待作者裁决：

- `1`：铸凭者，10%。
- `2`：校验组，40%。
- `3`：Blockqs，20%。
- `4`：Depots，20%。
- `5`：STUN，10%。

待裁决输出顺序：

- 选项 A：按配置值升序排列，即铸凭者、校验组、Blockqs、Depots、STUN。
- 选项 B：按 `blockchain.md` 的叙述顺序排列，即校验组、铸凭者、Blockqs、Depots、STUN。
- 选项 C：按 `4.激励机制.md` 的服务叙述顺序排列，即校验组、铸凭者、Depots、Blockqs、STUN。
- 百日前 Coinbase 输出集合和比例重标定需先解决 `C-016`。
- 百日后是否必须包含五类输出已由 conception 倾向明确，但其序列仍依赖 `C-001`。

可独立推进的金额规则候选：

- `RewardBase = issuance + unburned_tx_fee + reclaimed_award`。
- 交易费 50% 销毁；奇数 `chx` 时建议销毁部分向下取整，未销毁部分获得余数。
- 奖励比例逐项整数除法；最终余数归属需随输出顺序一起裁决。

建议兑奖槽：

- Blockqs、Depots、STUN 各自 6 字节，共 18 字节。
- 每个服务槽覆盖前 48 个区块，每块 1 bit。
- bit 顺序建议 bit0 对应 `H-1`，bit47 对应 `H-48`。
- 后续 48 块内达到 2 次确认即可在 31 块安全边界后兑奖。
- 未完整确认的部分由第 49 块 Coinbase 回收进入 `reclaimed_award`。

## Rationale（理由）

Coinbase 输出顺序会直接影响 TxID，因此在构想层冲突解决前只保留候选项。把交易费销毁和奖励分配放在同一 DEC，可避免 Coinbase 金额校验分散。兑奖槽分服务独立，符合 conception 对后期调整的预留。

## Consequences（影响）

- 本 DEC 依赖构想层先解决 Coinbase 输出顺序和百日前 profile 冲突。
- `SYS_AWARD` 必须能识别公共服务未激活期。
- 第 49 块回收编码会影响 Coinbase TxID，需冻结后才能生成测试向量。

## Conception References（构想层依据）

- `docs/conception/blockchain.md#百日扩张`
- `docs/conception/4.激励机制.md`
- `docs/conception/附.交易.md#铸币交易coinbase`

## Open Questions（开放问题）

- 构想层中 Coinbase 输出顺序存在冲突，见 `CONCEPTION-CONFLICTS.md` 的 `C-001`。
- 百日前奖励比例是保留 10/40，还是重标定为 20/80。
- `reclaimed_award` 在 Coinbase 中作为头字段、输出项，还是隐含在收益总额中。

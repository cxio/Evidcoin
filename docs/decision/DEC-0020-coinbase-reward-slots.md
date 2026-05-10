# DEC-0020: Coinbase 完整编码与公共服务兑奖槽

## Status（状态）

Proposed

## Context（背景）

`conception/附.交易.md` 已定义 Coinbase 的核心信息、输出配置和公共服务奖励目标，`conception/4.激励机制.md` 已定义公共服务 48 区块兑奖窗口与三个服务各自分配比例。`DEC-0012` 已固定 Coinbase 的 `HashInputs`。剩余缺口是 Coinbase 字段顺序、收益输出顺序和兑奖槽位的字节级编码。

## Decision（决策）

本节为候选规则，状态转为 Accepted 前不得作为最终共识依据。

Coinbase 交易体建议按下列顺序编码：

```text
CoinbaseBody =
    uint64_be(blockHeight) ||
    MintCredential ||
    uint64_be(totalRewardChx) ||
    RewardOutputs ||
    AwardSlots ||
    VarBytes(freeData)
```

字段规则建议如下：

- `MintCredential` 包含择优凭证，并能验证铸造者身份、铸凭哈希和铸造者签名。
- `totalRewardChx` 为原始铸币、未销毁交易费和兑奖截留回收的合计金额，单位为 chx。
- `RewardOutputs` 固定按下列目标顺序编码：铸凭者、校验组、Blockqs、Depots、STUN。
- `freeData` 长度必须小于 256 字节。
- Coinbase 必须位于区块交易序列首位。
- Coinbase 的 `HashInputs` 继续使用 `DEC-0012`。

公共服务兑奖槽建议固定为 18 字节：

```text
AwardSlots = BlockqsSlots[6] || DepotsSlots[6] || StunSlots[6]
```

槽位位序建议如下：

- 每类服务 6 字节，合计 48 bit，对应前段 48 个区块。
- bit 0 对应 `H-1` 的同类服务奖励目标，bit 47 对应 `H-48` 的同类服务奖励目标。
- 每个字节内按低位到高位表达更近到更远的区块。
- 当前高度 `H` 的 Coinbase 只能确认 `[H-48, H-1]` 范围内的公共服务奖励。

兑奖与回收建议如下：

- 公共服务奖励在后续 48 个区块内获得 1 次确认可兑奖 50%，获得 2 次确认可兑奖 100%。
- 满足 35 个区块确认且已获得足额确认后，可执行兑奖。
- 若至第 49 个后续区块仍未获得完整确认，未确认部分进入该区块 Coinbase 的 `totalRewardChx` 回收。
- 回收金额必须可由链上历史 Coinbase 与兑奖槽确定性推导。

待冻结参数：

- `MintCredential` 的精确内部字段与序列化格式。
- `RewardOutputs` 中每个输出项复用普通输出结构还是使用 Coinbase 专用输出结构。
- `totalRewardChx` 与各分成输出、销毁输出之间的校验公式。
- 兑奖交易或 `SYS_AWARD` 对兑奖槽的引用编码。

## Rationale（理由）

- 固定输出顺序可消除收益分成解释差异。
- Coinbase 输出顺序以 `conception/附.交易.md` 中输出项配置编号为准，使编码顺序与既有配置值一致。
- 将三类服务槽分开符合 conception 中“各自定义、便于后期修改调整”的实现参考。
- 近端 bit 使用低位便于按高度差直接定位。

## Consequences（影响）

- Coinbase 编码在待冻结参数确定前不能作为最终共识格式。
- 验证实现需要能从最近 48 个区块的 Coinbase 中统计公共服务奖励确认数。
- 测试应覆盖 0、1、2 次确认、48 区块末端确认和第 49 区块回收。

## Conception Relationship（与构想关系）

- 补充 `conception/附.交易.md` 与 `conception/4.激励机制.md` 中 Coinbase、奖励分成和兑奖槽位的编码细节。
- 不改变公共服务奖励比例、48 区块兑奖窗口或 `DEC-0012` 的 Coinbase `HashInputs` 规则。

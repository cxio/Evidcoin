# DEC-0303: Fork Choice and RandomX Tiebreaker（分叉选择与 RandomX 平局裁决）

Status: Proposed

## Context（背景）

Conception 已明确 31 块分叉竞争、16 块过半胜出、长度 20 临界裁决、同铸造者低收益原则和 RandomX 平局方案。但完整比较算法、低收益定义、RandomX profile 和 3 倍币权销毁流程仍需冻结。

## Decision（决策）

建议分叉链段比较：

- 只比较分叉点之后最多 31 个区块。
- 逐高度比较两条链对应区块的 `MintHash`。
- 单高度 `MintHash` 较小者得 1 分。
- 任一链先达到 16 分即胜出。
- 31 个高度比较完成仍平局时进入 RandomX 裁决。

建议同高度同铸造者多签归一化：

- 先按铸造者公钥哈希分组。
- 每组仅保留“铸造者个人可得收益最低”的区块。
- 若个人可得收益相同，保留交易费总额更低者。
- 仍相同则保留 BlockID 更小者。

建议 RandomX 平局：

```text
seed = ForkPointBlockID
input = FirstForkBlockID
score = RandomX(seed, input)
```

- `score` 按字典序升序，较小者胜。
- `score` 相同则比较分叉首块 ID，较小者胜。

## Rationale（理由）

先对同铸造者多签做归一化，可减少人为制造平局的空间。RandomX 仅作为低概率平局裁决，不应进入常规出块路径。

## Consequences（影响）

- RandomX 版本和参数未冻结前，平局裁决不可用于主网。
- 低收益定义需与 Coinbase 输出和交易费销毁规则一致。
- 3 倍币权销毁的冗余出块规则需在区块接收阶段独立实现。

## Conception References（构想层依据）

- `docs/conception/2.共识-端点约定.md#分叉竞争`
- `docs/conception/2.共识-端点约定.md#平局的可能性及解决`
- `docs/conception/附.组队校验.md#低收益原则`
- `docs/conception/附.组队校验.md#交易量约束`

## Open Questions（开放问题）

- RandomX 的具体库、版本、输出长度、内存参数和轻/重模式。
- “低收益”最终是个人收益、交易费收益，还是 Coinbase 总输出。
- 3 倍币权销毁在连续后位超越时的完整确定性算法，见 `CONCEPTION-CONFLICTS.md` 的 `C-007`。

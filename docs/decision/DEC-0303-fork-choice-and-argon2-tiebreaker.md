# DEC-0303: Fork Choice and Argon2 Tiebreaker（分叉选择与 Argon2 平局裁决）

Status: Accepted

## Context（背景）

Conception 已明确 31 块分叉竞争、16 块过半胜出、长度 20 临界裁决、同铸造者低收益原则和 Argon2 平局方案。本决策冻结完整比较算法、低收益定义、Argon2 profile 和交易量约束流程（Stakes 严格超3倍则替换）。

## Decision（决策）

分叉链段比较：

- 只比较分叉点之后最多 31 个区块。
- 每个高度先按本决策的同铸造者多签归一化和交易量约束选出有效候选块。
- 逐高度比较两条链对应有效候选块的 `MintHash`。
- 单高度 `MintHash` 较小者得 1 分。
- 单高度 `MintHash` 完全相等时双方都不得分。
- 任一链先达到 16 分即胜出。
- 31 个高度比较完成仍平局时进入 Argon2 裁决。

同高度同铸造者多签归一化：

- 先按铸造者公钥哈希分组。
- 每组仅保留“铸造者个人可得收益最低”的区块。
- 若个人可得收益相同，保留交易费总额更低者。
- 仍相同则保留 BlockID 更小者。

其中“铸造者个人可得收益”指 Coinbase 中直接分配给该铸造者身份的金额，不包含校验组工作报酬、公共服务奖励或其它第三方收益。

Argon2 平局：

```text
seed  = ForkPointBlockID[:16]   // 分叉点区块 ID 的前 16 字节
input = FirstForkBlockID        // 分叉首块 ID，48 字节
score = Argon2id(password=input, salt=seed, time=3, memory=64MiB, threads=4, keyLen=32)
```

即 `argon2.IDKey(input, seed, 3, 64*1024, 4, 32)`。

- `score` 按字典序升序，较小者胜。
- `score` 相同则比较分叉首块 ID，较小者胜。

Argon2 profile：

- 实现固定为 `golang.org/x/crypto/argon2` 的 `IDKey` 函数，即 Argon2id（RFC 9106）。
- 参数冻结：`time=3` 轮次、`memory=64*1024` KiB（64 MiB）、`threads=4`、输出 `keyLen=32` 字节。
- `password` 为 48 字节 `FirstForkBlockID`；`salt` 为 `ForkPointBlockID` 的前 16 字节（满足 `IDKey` 对 salt 至少 8 字节的要求）。
- Argon2id 输出由 RFC 9106 与冻结参数唯一决定，符合规范的实现输出一致；参考实现随 `go.mod` 的 `golang.org/x/crypto v0.52.0`。
- 不得使用会改变哈希结果的参数变体或非规范实现。

交易量约束确定性算法（Stakes `>3x`）：

- 只比较同一高度、同一前一区块上的冗余出块。
- 候选块按铸造者在择优池中的排名升序排列。
- 缺位候选者跳过，不生成空候选。
- 从当前最优候选 `winner` 开始，依次考察后位候选 `challenger`。
- `challenger.Stakes > winner.Stakes * 3` 时，替换 `winner` 并继续考察后位候选；否则停止。
- `TxCount` 仅作上层统计与展示，不参与候选归一化比较。
- 若 `winner.Stakes == 0`，仍按上述公式处理；因此后位候选只要 `Stakes > 0` 即可满足 `>3x`。
- 相等不算超越，必须严格 `> 3x`。

## Rationale（理由）

先对同铸造者多签做归一化，低收益定义采用铸造者个人可得收益，直接对应抑制多签收益动机。

Argon2 仅作为低概率平局裁决，不进入常规出块路径；但一旦触发，哈希正确性直接影响主链选择，因此冻结参数与参考实现。Argon2 是内存硬函数，可抑制 GPU/ASIC 的哈希优化动机；`golang.org/x/crypto/argon2` 为纯 Go 实现，避免了 RandomX 方案所需的 CGO 封装与跨平台负担。

交易量不参与归一化比较，否则可能导致铸造者竟相构造大量微交易来追求数量优势，反而创造了攻击面。币权更能反应真实交易价值，也更难被操纵。

## Consequences（影响）

- Argon2 裁决复用既有依赖 `golang.org/x/crypto/argon2`，无需 CGO 封装或外部 C/C++ 库；既有 RandomX 封装实现需替换为 Argon2 实现。
- 低收益比较依赖 Coinbase 中铸造者个人收益金额的可验证计算。
- 交易量约束（Stakes 严格 `>3x`）的冗余出块规则需在区块接收阶段独立实现，并在分叉链段比较前完成候选块选择。
- 测试需要覆盖同铸造者多签、`MintHash` 相等、31 块平局、Argon2 裁决、`score` 相等回退首块 ID、连续后位超越、`Stakes=0`、仅 Stakes 超越和相等边界。

## Conception References（构想层依据）

- `docs/conception/2.共识-端点约定.md#分叉竞争`
- `docs/conception/2.共识-端点约定.md#平局的可能性及解决`
- `docs/conception/附.组队校验.md#低收益原则`
- `docs/conception/附.组队校验.md#交易量约束`

## Open Questions（开放问题）

- 无。

# ADR-0023: Coinbase 交易作为特殊交易独立处理

## Status（状态）

Accepted

## Context（背景）

普通交易的输出项使用一个配置字节（Config/Envelope byte）来区分输出类型（Coin/Credit/Proof/Mediator），其低 4 位（`0x0F`）表示输出类型编码。

在评审过程中（C-2）提出了一个疑问：Coinbase 交易的输出配置低 4 位与普通交易输出类型字段是否存在冲突。

## Decision（决策）

**Coinbase 交易是一笔特殊交易，在所有处理逻辑中与普通交易完全独立，两者之间不存在字段冲突。**

具体规则：
- Coinbase 交易的输出**只能是 Coin 类型**，没有 Credit/Proof/Mediator 输出的概念。
- Coinbase 输出不存在"输出类型字段（低 4 位）"的逻辑——Coinbase 输出有其独立的序列化格式，不与普通输出共享相同的配置字段布局。
- 验证节点在处理交易时，首先判断是否为 Coinbase 交易（通过版本号或其他识别符），若是则走 Coinbase 独立验证路径；若否则走普通交易路径。两条路径在输出解析阶段不相互干扰。

## Rationale（理由）

1. **职责明确**：Coinbase 交易的语义和约束与普通交易根本不同——它没有输入（或仅有伪输入），其输出直接代表区块铸造奖励，这种特殊性要求独立处理，而非强行套用普通输出的字段布局。
2. **冲突不成立**：低 4 位是否冲突的前提是两种交易走同一解析路径。由于两者已经在最外层分叉处理，Coinbase 的字段布局独立定义，不存在"同一字段被两种语义复用"的冲突。
3. **实现简洁**：将 Coinbase 与普通交易分开处理，使得各自的验证逻辑可以专注于自身约束，避免在通用逻辑中引入过多的特例判断。

## Consequences（影响）

- `internal/tx` 中的输出解析函数需在入口处区分 Coinbase 与普通交易，路由到各自的独立解析函数。
- Coinbase 输出的序列化格式应在 `docs/proposal/06.Transaction-Model.md` 中独立描述，不与普通输出格式文档混在一起。
- 测试套件需要覆盖：尝试在 Coinbase 交易中使用非 Coin 输出类型时应被拒绝。

## References（参考）

- `docs/proposal/06.Transaction-Model.md` — 交易模型
- `docs/proposal/07.Coin-Credit-Proof-Units.md` — 输出类型定义
- `docs/plan/03-Transaction-And-Units.md` — Task 9

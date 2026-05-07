# DECISION-0010: ML-DSA-65 集成路径

## Status（状态）

Accepted

## Context（背景）

Evidcoin 使用 ML-DSA-65 作为主要签名算法。`docs/conception` 固定了算法方向，但没有固定 Go 实现库。Go 标准库和第三方库状态会随实现时点变化，因此需要明确选择策略而不是过早绑定某个库。

## Decision（决策）

ML-DSA-65 实现按以下优先级选择：

| 优先级 | 实现 | 条件 |
|--------|------|------|
| 1 | Go 标准库 | 当前 Go 版本已提供稳定 ML-DSA-65 API。 |
| 2 | `github.com/cloudflare/circl` | 标准库不可用或 API 不稳定。 |
| 3 | 其它同行评审实现 | 前两者均不可用或存在已知缺陷。 |

无论选择哪个底层库，都必须封装在项目密码学接口之后。上层内部包不得直接依赖具体 ML-DSA 库。

## Rationale（理由）

- 标准库优先可减少供应链和维护成本。
- CIRCL 是成熟备选。
- 接口隔离可降低未来替换底层实现的成本。

## Consequences（影响）

- 实现签名模块前需要检查当前 Go 版本和库状态。
- 若引入第三方库，应固定版本并验证 `go.sum`。

## Conception Relationship（与构想关系）

- 补充 ML-DSA-65 的工程集成路径。
- 不改变 conception 对后量子签名算法的选择。

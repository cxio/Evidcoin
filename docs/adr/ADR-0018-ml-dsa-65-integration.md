# ADR-0018: ML-DSA-65 签名库集成路径

## Status（状态）

Accepted

## Context（背景）

Evidcoin 采用后量子密码学签名算法 ML-DSA-65（CRYSTALS-Dilithium Level 3）作为主要签名方案。实施计划（`docs/plan/01-Foundation-Types-Crypto.md`，Task 5）使用测试签名器（Test Signer）回避了正式密码学库的集成，但后续没有明确的 Task 来完成从测试签名器到正式实现的切换。

Go 1.26 可能已将 ML-DSA-65 纳入标准库（`crypto/mlkem` 已在 1.23 引入，ML-DSA 在 1.25+ 的讨论中），具体情况需在实现时确认。

## Decision（决策）

**按行业最佳实践执行，由实施 Agent 在 Task 5 时自行判断，并在选定方案前告知用户确认。**

### 实施 Agent 的参考决策框架

按以下优先级顺序选择 ML-DSA-65 实现：

1. **优先：Go 标准库**（若 Go 1.26+ 已包含 `crypto/mldsa`）
   - 无额外依赖，长期维护由 Go 官方负责。
   - 选择条件：标准库 API 稳定且支持 ML-DSA-65（Level 3 参数集）。

2. **其次：`github.com/cloudflare/circl`**
   - 由 Cloudflare 维护，API 成熟，已被业界广泛验证。
   - 选择条件：标准库不可用，或标准库 API 不稳定。

3. **备选：`filippo.io/mldsa`** 或其他经过同行评审的实现
   - 选择条件：上述两者均不可用或存在已知缺陷。

### 接口隔离要求（已在 Plan 01 中定义，此处重申）

无论使用哪个底层库，ML-DSA-65 的实现必须封装在 `pkg/crypto` 的 `Signer`/`Verifier` 接口后面，上层代码（`internal/*`）不得直接 import 密码学库。

## Rationale（理由）

由于 Go 版本和标准库在实施时的具体状态无法事先完全预测，将最终决策留给实施 Agent 在可以检查实际环境时作出更为合理。行业最佳实践（标准库优先）是明确的，接口隔离确保了未来切换的成本最小化。

## Consequences（影响）

- 实施 Agent 在实现 `pkg/crypto` Task 5 前，需检查 Go 标准库是否包含 ML-DSA-65，并向用户报告选择方案和理由后再继续。
- `go.mod` 中的密码学库依赖在此 ADR 接受后仍为空（待实施时填入）。
- 若使用第三方库，需在 `go.mod` 中固定版本，并在 `go.sum` 中验证完整性。

## References（参考）

- `docs/proposal/02.Cryptography-And-Hashing.md` — 密码学提案（ML-DSA-65）
- `docs/plan/01-Foundation-Types-Crypto.md` — Task 5
- AGENTS.md — 计划中的外部依赖（`github.com/cloudflare/circl`）

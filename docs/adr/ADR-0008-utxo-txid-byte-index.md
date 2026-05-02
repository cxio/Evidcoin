# ADR-0008: UTXO/UTCO 指纹树中间层 TxID 字节索引

## Status（状态）

Accepted

## Context（背景）

UTXO 和 UTCO 状态指纹采用四层宽成员哈希树结构（`docs/proposal/08.UTXO-UTCO-State.md`）：

- **顶层**：按年度分组
- **第 2-4 层**：按 TxID 的特定字节位置进行分级路由，索引为 `[8, 13, 18]`

问题在于这里的字节索引是 **0-based**（`TxID[0]` 为第一个字节）还是 **1-based**（`TxID[1]` 为第一个字节）？两者产生完全不同的树结构，如果不同实现对此理解不同，将生成不同的状态根，导致共识分歧（OQ-012 部分相关）。

## Decision（决策）

**TxID 字节索引采用 0-based 索引**，遵循编程语言的通常惯例。

即：
- 第 2 层路由键 = `TxID[8]`（第 9 个字节）
- 第 3 层路由键 = `TxID[13]`（第 14 个字节）
- 第 4 层路由键 = `TxID[18]`（第 19 个字节）

其中 TxID 为 48 字节的 SHA3-384 哈希值，有效索引范围为 `[0, 47]`。

## Rationale（理由）

0-based 索引是绝大多数编程语言（包括 Go）的数组/切片默认约定，也是密码学协议文档的惯例（如比特币 BIP 文档中的字节描述均为 0-based）。选择 1-based 反而需要在每次引用时进行心理转换，容易引发实现错误。

## Consequences（影响）

- 需在 `docs/proposal/08.UTXO-UTCO-State.md` 中明确标注"字节索引为 0-based"。
- `internal/utxo` 和 `internal/utco` 的状态指纹树实现中，路由函数使用 `txID[8]`、`txID[13]`、`txID[18]`（Go 切片语法）。
- 需在测试套件中提供具体的 TxID 示例，验证树路由结果与预期一致。
- OQ-012 部分关闭（此 ADR 只处理字节索引语义；状态指纹树的根组合规则见各包实现）。

## References（参考）

- `docs/proposal/08.UTXO-UTCO-State.md` — UTXO/UTCO 状态
- `docs/proposal/04.Hash-Trees.md` — 哈希树
- `docs/plan/04-UTXO-UTCO-State.md` — Task 7
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-012

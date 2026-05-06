# ADR-0034: 区块头与交易头字段宽度

## Status（状态）

Accepted

## Context（背景）

Proposal 层对区块头（`BlockHeader`）和交易头（`TxHeader`）中多个整数字段的固定宽度从未正式确定，导致：

1. `BlockHeader.CanonicalBytes()` 无法实现最终版本
2. `TxHeader` 规范化编码同理
3. 所有字节级协议测试向量均无法生成
4. 跨实现互操作缺乏基础保证

构想层（`docs/conception/blockchain.md`）曾以"如果整数按 4 字节计算，区块头约 112 字节"作为示例描述，但未正式固定。

## Decision（决策）

### 1. 区块头字段宽度

| 字段 | 类型 | 宽度 | 字节序 | 说明 |
|------|------|------|--------|------|
| `Version` | uint32 | 4 字节 | big-endian | 区块格式版本号 |
| `Height` | uint32 | 4 字节 | big-endian | 区块高度，uint32 可支持约 42 亿区块，远超百年用量（每年 87661 块 × 100 年 ≈ 876 万块） |
| `PrevBlock` | [48]byte | 48 字节 | - | 前一区块 BlockID（SHA3-384） |
| `CheckRoot` | [48]byte | 48 字节 | - | 校验根（SHA3-384） |
| `Stakes` | uint64 | 8 字节 | big-endian | 币权销毁总值（按 ADR-0033） |
| `YearBlock` | [48]byte | 48 字节 | - | 前一年块 BlockID（非年块边界时为全零） |

区块头规范化编码字段顺序固定为：`Version || Height || PrevBlock || CheckRoot || Stakes || YearBlock`，总长度为 `4 + 4 + 48 + 48 + 8 + 48 = 160` 字节。

> **注（YearBlock）：** 非年块边界高度（即 `height % 87661 ≠ 0`）的区块，`YearBlock` 字段仍**保留 48 字节空间**，填充全零（`[48]byte{}`）。年块边界高度（`height % 87661 == 0`）的区块，`YearBlock` 填充前一个年块的 BlockID。创世块（height = 0）的 `YearBlock` 填充全零。

### 2. 交易头字段宽度

| 字段 | 类型 | 宽度 | 字节序 | 说明 |
|------|------|------|--------|------|
| `Version` | uint16 | 2 字节 | big-endian | 交易格式版本号 |
| `HashInputs` | [32]byte | 32 字节 | - | 输入集哈希（BLAKE3-256） |
| `HashOutputs` | [32]byte | 32 字节 | - | 输出集哈希（BLAKE3-256） |
| `Timestamp` | int64 | 8 字节 | big-endian | 交易时间戳，单位毫秒（Unix 时间，UTC） |

交易头规范化编码字段顺序固定为：`Version || HashInputs || HashOutputs || Timestamp`，总长度为 `2 + 32 + 32 + 8 = 74` 字节。

`TxID = SHA3-384(TxHeaderCanonicalBytes)`，输出 48 字节。

### 3. 其他整数字段的默认规则

对于本 ADR 未明确列出的其他整数字段（如输入项中的 `Year`、`OutIndex`、`TransferIndex` 等），**默认使用变长整数编码（Uvarint，按 ADR-0003）**，除非未来有专项 ADR 固定为固定宽度。

具体字段列表的规范化编码，由对应的 Proposal 或后续 ADR 明确。

## Rationale（理由）

1. **区块头 Version/Height 用 uint32**：与构想层示意（"按 4 字节计算"）一致，且 uint32 对高度已充分够用，不会溢出。
2. **交易头 Version 用 uint16**：交易版本需求远低于区块头，2 字节（65536 种版本）已足够，节省空间。
3. **Timestamp 用 int64 毫秒**：毫秒精度可支持 `timestamp * Stakes * Mix` 的 X 参数计算（ADR-0002），同时 int64 毫秒覆盖约 ±2.9 亿年，不存在实际溢出风险。使用 int64（有符号）以兼容 Unix 时间戳惯例（负数表示 1970 年之前）。
4. **其他字段默认 Uvarint**：节省编码空间，与项目已确立的 varint 优先原则（ADR-0003）一致。

## Consequences（影响）

- **关闭 H-3 未决项**：本 ADR 完整关闭区块头/交易头字段宽度的未决问题。
- `pkg/types` 中 `BlockHeader` 和 `TxHeader` 的规范化编码实现可给出最终版本，总长度分别为 160 字节和 74 字节。
- 字节级测试向量（区块头 Hash、TxID）现在可以生成。
- 需在 `docs/proposal/05.Blockchain-Core.md` 和 `docs/proposal/06.Transaction-Model.md` 中补充 ADR-0034 追溯，并移除"字段宽度待定"相关描述。
- 需在 `docs/plan/01-Foundation-Types-Crypto.md`（区块头编码）和 `docs/plan/03-Transaction-And-Units.md`（交易头编码）中补充 ADR-0034 追溯。

## References（参考）

- `docs/conception/blockchain.md` — 区块头结构原始定义
- `docs/conception/附.交易.md` — 交易头结构原始定义
- `docs/adr/ADR-0003-varint-encoding.md` — Varint 编码规则
- `docs/adr/ADR-0002-poh-x-encoding.md` — X 参数（含 Timestamp）
- `docs/adr/ADR-0033-stakes-definition.md` — Stakes 字段类型
- `docs/proposal/05.Blockchain-Core.md` — 区块链核心
- `docs/proposal/06.Transaction-Model.md` — 交易模型

# ADR-0003: 规范化可变长度整数编码（varint）

## Status（状态）

Accepted

## Context（背景）

Evidcoin 的规范化编码（`docs/proposal/01.Types-And-Encoding.md`）中大量使用可变长度整数（varint）来编码：

- `Bytes` 类型的长度前缀
- 列表（List）的元素数量
- 可选字段（Optional）的内容

提案层声明使用 "canonical unsigned varint"，但具体算法从未确认，被标记为阻塞性未决项（OQ-001）。此未决项阻碍了整个编码层的最终测试向量生成，影响所有依赖编码的哈希计算。

## Decision（决策）

**采用 LEB128（Little-Endian Base-128）无符号变长整数编码**，即 Protocol Buffers 使用的 varint 格式。

### 编码规则

每个字节低 7 位存储数据，最高位（bit 7）为延续标志：
- `1`：后续还有字节
- `0`：这是最后一个字节

```
值 0         → 0x00                    (1 字节)
值 127       → 0x7F                    (1 字节)
值 128       → 0x80 0x01              (2 字节)
值 16383     → 0xFF 0x7F              (2 字节)
值 16384     → 0x80 0x80 0x01        (3 字节)
```

### 规范化约束（canonical 要求）

- **禁止非最短编码**：接收方必须拒绝含多余前导零的 varint（如 `0x80 0x00` 表示 0 是非法的，应为 `0x00`）。
- 最大编码长度：uint64 最大值需 10 字节，实现时需防止超长读取。

## Rationale（理由）

1. **广泛验证**：LEB128/Protocol Buffers varint 是行业中最常用的变长整数格式之一，有大量参考实现和测试向量可供验证。
2. **Go 标准库支持**：`encoding/binary` 包提供 `PutUvarint`/`Uvarint` 函数，可直接使用，降低实现错误概率。
3. **规范化约束明确**：非最短编码拒绝规则已在提案层要求，LEB128 的规范化约束与此完全对应。

## Consequences（影响）

- 需在 `docs/proposal/01.Types-And-Encoding.md` 中将 "canonical unsigned varint" 替换为明确的 LEB128 定义并附编码示例。
- `pkg/types` 中需实现 `EncodeVarint(n uint64) []byte` 和 `DecodeVarint(b []byte) (uint64, int, error)` 两个函数，并验证规范化约束。
- 提供标准测试向量（至少覆盖：0、127、128、16383、16384、uint64 最大值）。
- OQ-001 关闭。

## References（参考）

- `docs/proposal/01.Types-And-Encoding.md` — 规范化编码
- `docs/plan/01-Foundation-Types-Crypto.md` — Task 3
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-001
- Protocol Buffers Encoding: https://protobuf.dev/programming-guides/encoding/#varints

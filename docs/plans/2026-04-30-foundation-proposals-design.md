# Foundation Proposals Design（基础提案设计）

## 背景

Evidcoin 当前处于预实施阶段，`docs/conception/` 已包含较完整的设计构想，但 `docs/proposal/` 尚未形成可供 Implementation Plan 直接引用的技术规格。

直接从构想层编写实施方案会把过多协议细节留到编码阶段决定，容易导致序列化、哈希输入、标识符、状态指纹等基础规则返工。因此第一轮应先生成基础层 Proposal/Specs。

## 目标

第一轮 Proposal 的目标是稳定后续所有模块共享的基础协议规则，包括类型、编码、密码学、标识符、常量和哈希树。

这些文档应当达到以下标准：

- 能被后续 Implementation Plan 直接引用。
- 能为 Go 包 `pkg/types/`、`pkg/crypto/` 以及基础哈希树实现提供规格依据。
- 每个关键规则都能追溯到 `docs/conception/` 的来源文件。
- 明确哪些内容已确定，哪些内容仍为未决问题。

## 推荐路线

采用“基础层优先，规格先闭合”的路线。

与完整骨架优先相比，该路线避免生成大量空壳文档；与最小链路闭环优先相比，该路线先固定所有下游模块共享的编码和哈希规则，降低后续返工风险。

## 第一轮输出

第一轮生成以下 Proposal 文件：

- `docs/proposal/00.Project-Scope.md`
- `docs/proposal/01.Types-And-Encoding.md`
- `docs/proposal/02.Cryptography-And-Hashing.md`
- `docs/proposal/03.Identifiers-And-Constants.md`
- `docs/proposal/04.Hash-Trees.md`

同时更新：

- `docs/AGENTS.md`

## 各文档职责

### `00.Project-Scope.md`

定义第一阶段实现边界、非目标、术语、层级依赖、包划分和文档追溯原则。

该文档不定义具体字节格式，只说明基础层 Proposal 的适用范围和与后续交易、区块、脚本、共识 Proposal 的关系。

### `01.Types-And-Encoding.md`

定义基础类型和二进制编码规则，包括：

- 固定长度字节数组。
- 无符号整数和有符号整数的范围。
- varint 编码规则。
- 字节序。
- 字节串长度前缀。
- 可选字段编码。
- 列表编码。
- 结构体序列化规则。

### `02.Cryptography-And-Hashing.md`

定义密码学和 Hash 规则，包括：

- SHA3-384、SHA3-512、SHA3-256 的用途。
- BLAKE3-256、BLAKE3-512/XOF 的用途。
- BLAKE2b-512 在地址哈希中的用途。
- ML-DSA-65 签名抽象。
- Hash 输入拼接原则。
- 域分隔策略。
- 测试向量要求。

### `03.Identifiers-And-Constants.md`

定义系统级标识符和常量，包括：

- `BlockID`。
- `TxID`。
- `AddressHash`。
- `AttachmentHash`。
- `TreeHash`。
- `MintHash`。
- 区块时间、高度、年度换算。
- 脚本和交易基础限制常量。

### `04.Hash-Trees.md`

定义所有哈希树的公共规则，包括：

- 二元哈希树。
- 含序叶子。
- 空树和单叶树规则。
- 证明路径格式。
- 交易树、输入树、输出树、附件片组树、UTXO/UTCO 指纹树的共同约束。

## 本轮非目标

第一轮不处理以下内容：

- 不写 Go 代码。
- 不写 Implementation Plan。
- 不完整定义交易状态机。
- 不完整定义 UTXO/UTCO 存储实现。
- 不完整定义脚本 VM 指令语义。
- 不完整定义 PoH 共识流程。
- 不定义 P2P 或公共服务实现。

这些内容应在基础 Proposal 稳定后，由后续 Proposal 分批生成。

## 验收标准

第一轮完成后应满足：

- `docs/proposal/` 下存在 5 个基础 Proposal 文件。
- 每个 Proposal 都包含来源构想、目标、规则、未决问题和后续 Plan 影响。
- `docs/AGENTS.md` 的 `Conception => Proposal` 对应表已更新。
- 文档之间术语一致，不互相冲突。

## 后续步骤

基础 Proposal 完成后，再生成 Implementation Plan。Implementation Plan 应以这些 Proposal 为直接输入，按 Go 包拆成可执行任务。

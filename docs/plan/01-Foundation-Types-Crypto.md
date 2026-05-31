# Foundation Types Crypto Implementation Plan

**Goal:** 实现 Evidcoin 的基础类型、规范化编码、Hash/ID 类型、密码学抽象和基础哈希树工具。

**Architecture:** `pkg/types/` 只提供无内部依赖的类型、常量和编码能力，`pkg/crypto/` 在其上提供 Hash、域标签、地址/多签复合公钥哈希和签名抽象，`pkg/hashtree/` 提供通用二叉树与专用树规则。DEC-0004 已固定哈希树边界策略（空根由各结构自定义、单叶根按 tree.branch profile 归一化为 32B、奇数层提升不复制、验证路径不含 leafIndex）为协议默认。

**Tech Stack:** Go 1.26.2、`golang.org/x/crypto/blake2b`、`lukechampine.com/blake3`、`github.com/cloudflare/circl`（ML-DSA-65，DEC-0104）、`github.com/mr-tron/base58`、表驱动测试。

---

## 来源提案

- `docs/proposal/00.Project-Scope.md`
- `docs/proposal/01.Types-And-Encoding.md`
- `docs/proposal/02.Cryptography-And-Hashing.md`
- `docs/proposal/03.Identifiers-And-Constants.md`
- `docs/proposal/04.Hash-Trees.md`

## 包边界

| 包 | 允许依赖 | 禁止依赖 | 职责 |
|----|----------|----------|------|
| `pkg/types` | 标准库 | `internal/*`、`pkg/crypto` | 固定长度类型、ID、常量、编码 |
| `pkg/crypto` | 标准库、第三方 crypto、`pkg/types` | `internal/*` | Hash、地址哈希、签名接口 |
| `pkg/hashtree` | 标准库、`pkg/types`、`pkg/crypto` | `internal/*` | 通用哈希树和证明路径 |

如果希望减少包数量，可以把 `pkg/hashtree` 延后到 `internal/tx` 和 `internal/utxo` 需要时再创建；但交易树、附件树和状态树都会复用，优先建议单独建包。

## 建议文件

| 文件 | 内容 |
|------|------|
| `pkg/types/fixed_bytes.go` | 固定长度字节辅助、复制、防别名 API |
| `pkg/types/hash.go` | `Hash32`、`Hash48`、`Hash64` |
| `pkg/types/id.go` | `BlockID`、`TxID`、`CheckRoot`、`AddressHash`、`TreeHash`、`AttachmentHash`、`MintHash` |
| `pkg/types/constants.go` | `BlockInterval`、`BlocksPerYear`、脚本和交易限制常量 |
| `pkg/types/encoding.go` | `Encoder`、`Decoder`、规范化编码入口 |
| `pkg/types/varint.go` | DEC-0001 ULEB128 canonical unsigned varint（最短编码强制） |
| `pkg/types/int.go` | 固定宽度 big-endian 整数编码 |
| `pkg/types/bytes.go` | 可变长度 Bytes、列表、optional 编码工具 |
| `pkg/types/errors.go` | 类型和编码错误 |
| `pkg/crypto/hash.go` | SHA3、BLAKE3、domain tag Hash API |
| `pkg/crypto/address.go` | 单签 `SHA3-256(BLAKE2b-512(pubkey))`、多签复合公钥哈希、DEC-0104 `EncodeAddress` / `DecodeAddress`（Cx/Tx/Dx + Base58Check） |
| `pkg/crypto/signature.go` | `Signer`、`Verifier`、`PublicKey`、`Signature` 抽象 |
| `pkg/crypto/testsigner_test.go` | 测试签名器，避免早期绑定 ML-DSA 库 |
| `pkg/hashtree/tree.go` | 二元树构建和策略 |
| `pkg/hashtree/proof.go` | 证明路径结构和验证 |
| `pkg/hashtree/ordered_leaf.go` | 2-byte、3-byte 含序叶子 Hash |

## Task 1: 固定长度 Hash 与 ID 类型

**Files:**
- Create: `pkg/types/hash.go`
- Create: `pkg/types/id.go`
- Create: `pkg/types/hash_test.go`
- Create: `pkg/types/id_test.go`

**Step 1: 写失败测试**

覆盖以下表驱动用例：

- 32、48、64 字节输入可构造成功。
- 短 1 字节拒绝。
- 长 1 字节拒绝。
- 返回字节切片时不得暴露内部数组别名。
- `BlockID` 与 `TxID` 同为 48B 但类型不可直接赋值。

**Step 2: 运行测试确认失败**

```bash
go test ./pkg/types -run 'Test(Hash|ID)' -v
```

Expected: 因类型和构造函数不存在而失败。

**Step 3: 最小实现**

实现建议 API：

```go
type Hash32 [32]byte
type Hash48 [48]byte
type Hash64 [64]byte

type BlockID Hash48
type TxID Hash48
type CheckRoot Hash48
type AddressHash Hash32
type TreeHash Hash32       // 通用树枝 / BLAKE3-256
type AttachmentHash Hash64 // 附件完整指纹 / SHA3-512（DEC-0002）
type MintHash Hash32       // 铸凭哈希 / BLAKE3-256 32B（DEC-0301，非 64B）
```

构造函数命名建议：`NewHash32`、`NewBlockID`、`MustBlockID`。`Must*` 只在测试和静态向量中使用。

> **注（与旧 plan 的修正）：** `MintHash` 为 **32 字节**（DEC-0301 `MintHash [32]byte`，BLAKE3-256），不是 64 字节。

**Step 4: 验证**

```bash
go test ./pkg/types -run 'Test(Hash|ID)' -v
```

Expected: PASS。

**Step 5: 提交（仅在用户明确要求提交时执行）**

```bash
git add pkg/types/hash.go pkg/types/id.go pkg/types/hash_test.go pkg/types/id_test.go
git commit -m "feat: add fixed hash identifier types"
```

## Task 2: 常量与高度时间工具

**Files:**
- Create: `pkg/types/constants.go`
- Create: `pkg/types/time.go`
- Create: `pkg/types/time_test.go`

**Step 1: 写失败测试**

测试：

- `BlockInterval == 6 * time.Minute`。
- `BlocksPerYear == 87661`。
- `IsYearBoundary(0) == true`。
- `HeightYear(0) == 0`。
- `HeightYear(87660) == 0`。
- `HeightYear(87661) == 1`。
- `BlockTime(genesis, 0) == genesis`。
- `BlockTime(genesis, 1) == genesis + 6 minutes`。
- `IsYearBoundary(BlocksPerYear) == true`。

**Step 2: 运行测试确认失败**

```bash
go test ./pkg/types -run 'Test(BlockTime|Constants|YearBoundary)' -v
```

**Step 3: 最小实现**

实现 `BlockHeight` 命名类型、`BlockTime`、`HeightYear`、`IsYearBoundary`。创世高度 0 是年度边界，年度公式为 `height / BlocksPerYear`（`BlocksPerYear = 87661`）。

`BlockHeight` 底层类型固定为 `uint32`（与第 01 章 §1.4 定宽白名单中区块头 `Height` 为 uint32 一致）；凡参与区块头哈希/签名输入的高度编码必须走 `AppendUint32BE`，不得使用 `int`/`uint64` 底层或 varint，否则将导致 `BlockID` 不唯一。

> **注（两套年度口径，命名须显式区分，勿混用）：**
> - **高度年度（`HeightYear`）**：按 87661 区块计的年度边界，`HeightYear(height) = height / BlocksPerYear`，用于年块/年度边界判定（本 Task）。
> - **自然年度（`CalendarYear`）**：「UTC 自然年份数值」口径（DEC-0001，如 `2025`），用于交易输入项短引用、UTXO/UTCO 状态指纹分层（详见第 03、09 章 Stakes/年度承载）。
>
> 两者是不同维度，实现与测试中必须用不同命名（`HeightYear` vs `CalendarYear`）区分，禁止互相替代。

**Step 4: 验证并提交**

```bash
go test ./pkg/types -run 'Test(BlockTime|Constants|YearBoundary)' -v
git add pkg/types/constants.go pkg/types/time.go pkg/types/time_test.go
git commit -m "feat: add block time constants"
```

## Task 3: 规范化整数、LEB128 Varint、Bytes、列表和 optional 编码

**Files:**
- Create: `pkg/types/encoding.go`
- Create: `pkg/types/int.go`
- Create: `pkg/types/varint.go`
- Create: `pkg/types/bytes.go`
- Create: `pkg/types/encoding_test.go`
- Create: `pkg/types/errors.go`

**Step 1: 写失败测试**

测试：

- `uint16`、`uint32`、`uint64` big-endian 编码。
- 固定宽度整数短输入、长输入拒绝。
- DEC-0001 ULEB128 unsigned varint 测试向量：`0 -> 0x00`、`127 -> 0x7F`、`128 -> 0x80 0x01`、`16383 -> 0xFF 0x7F`、`16384 -> 0x80 0x80 0x01`。
- `uint64` 最大值编码长度为 10 字节。
- 非最短 varint 编码必须拒绝，例如 `0x80 0x00`。
- varint 溢出和缺少终止字节的输入必须拒绝。
- Bytes 编码为 `varint length + raw bytes`。
- 空 Bytes 只编码长度 0。
- optional absent 为 `0x00`。
- optional present 为 `0x01 + value`。
- optional 非法 marker 拒绝。
- list 先编码 count，再编码元素。

**Step 2: 运行测试确认失败**

```bash
go test ./pkg/types -run 'TestCanonicalEncoding' -v
```

**Step 3: 最小实现**

实现小而明确的函数，不先做泛型序列化框架：

- `AppendUint16BE(dst []byte, v uint16) []byte`
- `AppendUint32BE(dst []byte, v uint32) []byte`
- `AppendUint64BE(dst []byte, v uint64) []byte`
- `AppendVarUint(dst []byte, v uint64) []byte`
- `ReadVarUint(src []byte) (value uint64, n int, err error)`
- `AppendBytes(dst []byte, b []byte) []byte`
- `AppendOptional(dst []byte, present bool, appendValue func([]byte) []byte) []byte`

**Step 4: 验证并提交**

```bash
go test ./pkg/types -run 'TestCanonicalEncoding' -v
git add pkg/types/encoding.go pkg/types/int.go pkg/types/varint.go pkg/types/bytes.go pkg/types/encoding_test.go pkg/types/errors.go
git commit -m "feat: add canonical encoding helpers"
```

## Task 4: Hash API 和用途隔离

**Files:**
- Create: `pkg/crypto/hash.go`
- Create: `pkg/crypto/hash_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: 写失败测试**

测试：

- `SHA3-384` 输出 48B；`SHA3-512` 输出 64B；`BLAKE3-256` 输出 32B。
- DEC-0002 域标签格式为 `"Evidcoin/v1/" || name || 0x00`（ASCII 常量，作为前像首段）。
- 14 项域标签全集齐备：`block.header`、`tx.header`、`tree.leaf`、`tree.branch`、`checkroot`、`utxo.leaf`、`utco.leaf`、`mint.hash`、`signature.message`、`attachment.fingerprint`、`address.single`、`address.multi`、`utxo.empty`、`utco.empty`。
- 同 payload 不同域标签输出不同。
- 调用方不能传入协议域标签；协议标签必须由 `pkg/crypto` 内部按用途绑定。
- 附件片组树为唯一免域标签例外（DEC-0002）：`BLAKE3-256(2-byte seq || BLAKE3-256(piece))`，前像不前置域标签。

**Step 2: 添加依赖**

```bash
go get golang.org/x/crypto/blake2b lukechampine.com/blake3
```

**Step 3: 运行测试确认失败**

```bash
go test ./pkg/crypto -run TestHash -v
```

**Step 4: 最小实现**

实现用途明确的函数（算法 profile 见 DEC-0002 / proposal 02 §3）：

- `HashBlockHeader(data []byte) types.BlockID`（SHA3-384 + `block.header`）
- `HashTxHeader(data []byte) types.TxID`（SHA3-384 + `tx.header`）
- `HashCheckRoot(data []byte) types.CheckRoot`（SHA3-384 + `checkroot`）
- `HashTreeLeaf(data []byte) types.Hash48`（SHA3-384 + `tree.leaf`）
- `HashTreeBranch(data []byte) types.TreeHash`（BLAKE3-256 + `tree.branch`）
- `HashUTXOLeaf(data []byte) types.Hash48`（SHA3-384 + `utxo.leaf`）
- `HashUTCOLeaf(data []byte) types.Hash48`（SHA3-384 + `utco.leaf`）
- `HashAttachment(data []byte) types.AttachmentHash`（SHA3-512 + `attachment.fingerprint`）
- `HashMint(data []byte) types.MintHash`（BLAKE3-256 + `mint.hash`，32B）
- `EmptyUTXORoot() types.TreeHash` / `EmptyUTCORoot() types.TreeHash`（`BLAKE3-256(DomainTag("utxo.empty"))` / `utco.empty`，32B）
- `HashAttachmentPieceTree(...)`：免域标签路径，与协议树明确隔离。

域标签已由 DEC-0002 固定，API 内部必须绑定对应用途，不允许调用方传入协议标签；附件片组树是唯一免域标签例外。非协议测试辅助函数必须与协议 Hash API 明确隔离。

**Step 5: 验证并提交**

```bash
go test ./pkg/crypto -run TestHash -v
git add go.mod go.sum pkg/crypto/hash.go pkg/crypto/hash_test.go
git commit -m "feat: add protocol hash functions"
```

## Task 5: 地址哈希、地址文本编码与签名抽象

**Files:**
- Create: `pkg/crypto/address.go`
- Create: `pkg/crypto/signature.go`
- Create: `pkg/crypto/address_test.go`
- Create: `pkg/crypto/signature_test.go`

**Step 1: 写失败测试**

测试：

- 单签：`SHA3-256(BLAKE2b-512(pubkey))` 生成 32B `AddressHash`（`address.single` 域）。
- 多签（复合公钥哈希，`address.multi` 域）：各公钥 `BLAKE3-256` 初级哈希→字典序排序串联→前置 `m||n`→`SHA3-256(BLAKE2b-512(...))`；`m,n` 均非 0 且 `m<=n`，重复公钥非法。
- 不同公钥材料生成不同 `AddressHash`；单签与多签地址外观无区别。
- `EncodeAddress` 输出 `prefix || Base58(pubKeyHash || checksum)`；`checksum = last4(SHA2-256(SHA2-256(prefix || pubKeyHash)))`，`prefix` 参与校验但不进 Base58 负载（DEC-0104）。
- 网络前缀：主网 `Cx`、测试网 `Tx`、开发网 `Dx`；Base58 用 Bitcoin 字母表。
- `DecodeAddress` 可恢复原始 32B `AddressHash`；对错误前缀、非法 Base58、长度错误和 checksum 失败返回错误。
- 签名器签名后验证通过；消息/公钥变更、算法 ID 不匹配验证失败。

**Step 2: 实现测试签名器**

早期不要强绑 ML-DSA-65 具体库。测试签名器可以放在 `_test.go` 中，用 HMAC 或确定性 fake 结构验证接口行为。

**Step 3: ML-DSA-65 profile（DEC-0104 已固定 circl）**

DEC-0104 已冻结 ML-DSA-65 profile 为 `github.com/cloudflare/circl`：

- 公钥/私钥/签名序列化采用 circl 的 canonical byte encoding。
- 签名验证输入为第 04 章（DEC-0102）定义的签名消息字节序列。

> **A-2 未来兼容观察项：** 本 Plan 以 **DEC-0104（circl）为准**；是否随 Go 标准库成熟迁移不属于当前全局待决项。编码时不得混用标准库实现，上层只能依赖 `pkg/crypto` 的 `Signer` / `Verifier` 接口，以便未来单独评估迁移。

**Step 4: 验证并提交**

```bash
go test ./pkg/crypto -run 'Test(Address|Signature)' -v
git add pkg/crypto/address.go pkg/crypto/signature.go pkg/crypto/address_test.go pkg/crypto/signature_test.go
git commit -m "feat: add address hash and signature interfaces"
```

## Task 5A: Amount 与 Bi/chx 显示层转换

**Files:**
- Modify: `pkg/types/constants.go`
- Create: `pkg/types/amount.go`
- Create: `pkg/types/amount_test.go`

**Step 1: 写失败测试**

测试（单位口径 C-8 已裁决：`1 Bi = 10^8 chx`，`chx` 为最小承载单位，proposal 01）：

- `type Amount uint64` 可表达协议层金额（以 `chx` 整数承载）。
- `ChxPerBi == 100_000_000`。
- `Amount(1 * ChxPerBi)` 显示为 `1.00000000 Bi` 或项目选定的等价格式。
- 显示层转换能处理 0、1 chx、1 Bi 和 `uint64` 边界值。
- 用户输入解析不得用浮点数参与协议层运算；如提供 decimal parser，必须测试 8 位小数、超 8 位拒绝和溢出拒绝。

**Step 2: 最小实现**

实现 `Amount`、`ChxPerBi` 和显示层转换辅助函数。协议层计算只接受 `Amount` / `uint64 chx`，不得用 `float64` 表示金额。`Bi`（= 币）仅为展示/换算单位。

**Step 3: 验证并提交**

```bash
go test ./pkg/types -run TestAmount -v
git add pkg/types/constants.go pkg/types/amount.go pkg/types/amount_test.go
git commit -m "feat: add amount unit helpers"
```

## Task 6: 通用哈希树骨架

**Files:**
- Create: `pkg/hashtree/tree.go`
- Create: `pkg/hashtree/proof.go`
- Create: `pkg/hashtree/ordered_leaf.go`
- Create: `pkg/hashtree/tree_test.go`
- Create: `pkg/hashtree/proof_test.go`

**Step 1: 写失败测试**

测试：

- 左右子节点调换后根不同。
- 内部分支输出 32B。
- 同一叶子主体配不同 3-byte sequence prefix 得到不同 leaf hash。
- 同一叶子主体配不同 2-byte sequence prefix 得到不同 leaf hash。
- 证明路径方向错误时验证失败；验证路径不携带 `leafIndex`（DEC-0004）。
- 单叶树根不等于 48B 叶哈希，而是 `BLAKE3-256(DomainTag("tree.branch") || leafHash)`，长度为 32B。
- 单叶证明兄弟路径为空，但验证时必须先对 48B 叶哈希执行一元归一化。
- 奇数层最后一个节点**直接提升**到下一层，**不复制自身**。
- 同一叶子主体配不同 3-byte 序号前缀得到不同 leaf hash（区块交易树）。
- 空根由各结构自定义（通用树不预设全零根；UTXO/UTCO 空根 `utxo.empty`/`utco.empty` 由第 05 章承载）。

**Step 2: 最小实现**

实现 DEC-0004 固定的默认协议行为：**奇数层直接提升（不复制）、单叶树根一元归一化为 32B、验证路径携带方向位且不含 `leafIndex`**。分支 BLAKE3-256 + `tree.branch`，叶 SHA3-384 + `tree.leaf`（附件片组树走免域标签路径）。空根策略由各专用树自定义，不得在通用树中硬编为全零。

**Step 3: 验证并提交**

```bash
go test ./pkg/hashtree -v
git add pkg/hashtree/tree.go pkg/hashtree/proof.go pkg/hashtree/ordered_leaf.go pkg/hashtree/tree_test.go pkg/hashtree/proof_test.go
git commit -m "feat: add hash tree primitives"
```

## 阶段验收

运行：

```bash
go fmt ./...
go test ./pkg/types ./pkg/crypto ./pkg/hashtree
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- `pkg/types` 不依赖 `pkg/crypto` 或 `internal/*`。
- `pkg/crypto` 不依赖 `internal/*`。
- 所有 Hash 输出长度与 Proposal 一致。
- 固定长度 ID 不能语义混用。
- DEC-0004 固定的哈希树边界策略（奇数层提升不复制、单叶根按 tree.branch profile 一元归一化为 32B、路径不含 leafIndex）已实现；空根交由各专用树承载，未被默认为全零协议事实。

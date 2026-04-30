# Foundation Types Crypto Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 Evidcoin 的基础类型、规范化编码、Hash/ID 类型、密码学抽象和基础哈希树工具。

**Architecture:** `pkg/types/` 只提供无内部依赖的类型、常量和编码能力，`pkg/crypto/` 在其上提供 Hash、地址哈希和签名抽象。哈希树工具应优先放在基础层可复用包中，但未决规则必须通过显式策略参数或拒绝路径处理。

**Tech Stack:** Go 1.26.2、`golang.org/x/crypto/sha3`、`golang.org/x/crypto/blake2b`、`lukechampine.com/blake3`、表驱动测试。

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
| `pkg/types/varint.go` | canonical unsigned varint，实现前必须确认算法 |
| `pkg/types/int.go` | 固定宽度 big-endian 整数编码 |
| `pkg/types/bytes.go` | 可变长度 Bytes、列表、optional 编码工具 |
| `pkg/types/errors.go` | 类型和编码错误 |
| `pkg/crypto/hash.go` | SHA3、BLAKE3、domain tag Hash API |
| `pkg/crypto/address.go` | `SHA3-256(BLAKE2b-512(pubkey material))` |
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
type TreeHash Hash32
type AttachmentHash Hash64
type MintHash Hash64
```

构造函数命名建议：`NewHash32`、`NewBlockID`、`MustBlockID`。`Must*` 只在测试和静态向量中使用。

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
- `BlockTime(genesis, 0) == genesis`。
- `BlockTime(genesis, 1) == genesis + 6 minutes`。
- `IsYearBoundary(0)` 和 `IsYearBoundary(BlocksPerYear)` 的预期需在测试名中说明。

**Step 2: 运行测试确认失败**

```bash
go test ./pkg/types -run 'Test(BlockTime|Constants|YearBoundary)' -v
```

**Step 3: 最小实现**

实现 `BlockHeight` 命名类型、`BlockTime`、`IsYearBoundary`。创世高度 0 是否年度边界必须与测试和 Proposal 注释一致。

**Step 4: 验证并提交**

```bash
go test ./pkg/types -run 'Test(BlockTime|Constants|YearBoundary)' -v
git add pkg/types/constants.go pkg/types/time.go pkg/types/time_test.go
git commit -m "feat: add block time constants"
```

## Task 3: 规范化整数、Bytes、列表和 optional 编码

**Files:**
- Create: `pkg/types/encoding.go`
- Create: `pkg/types/int.go`
- Create: `pkg/types/varint.go`
- Create: `pkg/types/bytes.go`
- Create: `pkg/types/encoding_test.go`
- Create: `pkg/types/errors.go`

**Step 1: 先确认 varint 算法**

如果 `docs/proposal/01.Types-And-Encoding.md` 仍未固定具体 varint 算法，先向用户确认。确认前只实现固定宽度整数和固定 bytes，不实现依赖 varint 字节的最终测试向量。

**Step 2: 写失败测试**

测试：

- `uint16`、`uint32`、`uint64` big-endian 编码。
- 固定宽度整数短输入、长输入拒绝。
- Bytes 编码为 `varint length + raw bytes`。
- 空 Bytes 只编码长度 0。
- optional absent 为 `0x00`。
- optional present 为 `0x01 + value`。
- optional 非法 marker 拒绝。
- list 先编码 count，再编码元素。

**Step 3: 运行测试确认失败**

```bash
go test ./pkg/types -run 'TestCanonicalEncoding' -v
```

**Step 4: 最小实现**

实现小而明确的函数，不先做泛型序列化框架：

- `AppendUint16BE(dst []byte, v uint16) []byte`
- `AppendUint32BE(dst []byte, v uint32) []byte`
- `AppendUint64BE(dst []byte, v uint64) []byte`
- `AppendVarUint(dst []byte, v uint64) []byte`
- `ReadVarUint(src []byte) (value uint64, n int, err error)`
- `AppendBytes(dst []byte, b []byte) []byte`
- `AppendOptional(dst []byte, present bool, appendValue func([]byte) []byte) []byte`

**Step 5: 验证并提交**

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

- `SHA3-384` 输出 48B。
- `SHA3-512` 输出 64B。
- `BLAKE3-256` 输出 32B。
- `BLAKE3-512-XOF` 输出 64B，不能等于默认 32B digest 补零。
- 同 payload 不同 domain tag 输出不同。

**Step 2: 添加依赖**

```bash
go get golang.org/x/crypto/sha3 golang.org/x/crypto/blake2b lukechampine.com/blake3
```

**Step 3: 运行测试确认失败**

```bash
go test ./pkg/crypto -run TestHash -v
```

**Step 4: 最小实现**

实现用途明确的函数：

- `HashBlockHeader(data []byte) types.BlockID`
- `HashTransaction(data []byte) types.TxID`
- `HashCheckRoot(data []byte) types.CheckRoot`
- `HashTreeBranch(data []byte) types.TreeHash`
- `HashAttachment(data []byte) types.AttachmentHash`
- `HashMint(data []byte) types.MintHash`

Domain tag 是否进入协议输入未最终固定时，API 内部不要擅自改变协议数据；可以提供 `HashWithDomain(domain string, payload []byte)` 供非协议测试和签名消息使用。

**Step 5: 验证并提交**

```bash
go test ./pkg/crypto -run TestHash -v
git add go.mod go.sum pkg/crypto/hash.go pkg/crypto/hash_test.go
git commit -m "feat: add protocol hash functions"
```

## Task 5: 地址哈希与签名抽象

**Files:**
- Create: `pkg/crypto/address.go`
- Create: `pkg/crypto/signature.go`
- Create: `pkg/crypto/address_test.go`
- Create: `pkg/crypto/signature_test.go`

**Step 1: 写失败测试**

测试：

- 相同公钥材料生成相同 32B `AddressHash`。
- 不同公钥材料生成不同 `AddressHash`。
- 签名器签名后验证通过。
- 消息变更验证失败。
- 公钥变更验证失败。
- 算法 ID 不匹配验证失败。

**Step 2: 实现测试签名器**

早期不要强绑 ML-DSA-65 具体库。测试签名器可以放在 `_test.go` 中，用 HMAC 或确定性 fake 结构验证接口行为。

**Step 3: 验证并提交**

```bash
go test ./pkg/crypto -run 'Test(Address|Signature)' -v
git add pkg/crypto/address.go pkg/crypto/signature.go pkg/crypto/address_test.go pkg/crypto/signature_test.go
git commit -m "feat: add address hash and signature interfaces"
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
- 证明路径方向错误时验证失败。
- 空树策略未指定时返回明确错误。
- 奇数叶策略未指定时返回明确错误。

**Step 2: 最小实现**

实现 `OddLeafPolicy`、`EmptyTreePolicy`，初始只支持测试明确选择的策略。不要把空根或复制末叶作为默认协议事实。

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
- 未决 Hash 树策略没有被默认为协议事实。

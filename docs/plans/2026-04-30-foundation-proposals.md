# Foundation Proposals Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 生成 Evidcoin 基础层 Proposal/Specs，为后续 Go 实现计划提供稳定的协议依据。

**Architecture:** 以 `docs/conception/` 为唯一设计来源，先生成基础层规格，再更新 `docs/AGENTS.md` 的追溯关系。第一轮只覆盖项目边界、类型编码、密码学、标识符常量和哈希树，不进入交易状态机、脚本 VM、PoH 完整流程或代码实现。

**Tech Stack:** Markdown 文档、Evidcoin 三层文档体系、Go 1.26+ 目标实现、SHA3、BLAKE3、BLAKE2b、ML-DSA-65。

---

## 前置说明

执行本计划时必须遵守：

- 只修改 `docs/proposal/`、`docs/AGENTS.md`，以及必要时补充 `docs/plans/`。
- 不写 Go 源码。
- 不创建 `go.mod`。
- 不生成实施代码目录。
- 每份 Proposal 都必须包含“来源构想”“目标”“非目标”“规则”“未决问题”“对后续 Plan 的影响”。
- 所有文档说明使用中文，技术名词和文件名保持英文。

## 参考文档

- `docs/plans/2026-04-30-foundation-proposals-design.md`
- `docs/AGENTS.md`
- `docs/conception/README.md`
- `docs/conception/blockchain.md`
- `docs/conception/附.交易.md`
- `docs/conception/5.信用结构.md`
- `docs/conception/6.脚本系统.md`
- `docs/conception/1.共识-历史证明（PoH）.md`
- `docs/conception/附.组队校验.md`

---

### Task 1: Project Scope Proposal

**Files:**
- Create: `docs/proposal/00.Project-Scope.md`
- Read: `docs/AGENTS.md`
- Read: `docs/conception/README.md`
- Read: `docs/conception/blockchain.md`

**Step 1: Draft the document structure**

Create `docs/proposal/00.Project-Scope.md` with these sections:

```markdown
# Project Scope（项目范围）

## 来源构想

## 目标

## 非目标

## 文档层级

## 第一阶段实现边界

## 包分层建议

## 术语约定

## 未决问题

## 对后续 Plan 的影响
```

**Step 2: Fill scope rules**

Define the first implementation boundary as:

- `pkg/types/`: 基础类型、固定长度 Hash、ID、常量。
- `pkg/crypto/`: Hash 函数、地址哈希、签名接口抽象。
- Shared specs only: 不实现区块、交易、UTXO/UTCO、脚本或共识。

**Step 3: Verify the document is self-contained**

Read `docs/proposal/00.Project-Scope.md` and check that a new engineer can understand why this Proposal exists without reading this plan.

**Step 4: Commit**

```bash
git add docs/proposal/00.Project-Scope.md
git commit -m "docs: add project scope proposal"
```

---

### Task 2: Types And Encoding Proposal

**Files:**
- Create: `docs/proposal/01.Types-And-Encoding.md`
- Read: `docs/conception/blockchain.md`
- Read: `docs/conception/附.交易.md`
- Read: `docs/conception/5.信用结构.md`
- Read: `docs/conception/6.脚本系统.md`

**Step 1: Draft the document structure**

Create `docs/proposal/01.Types-And-Encoding.md` with these sections:

```markdown
# Types And Encoding（类型与编码）

## 来源构想

## 目标

## 非目标

## 基础类型

## 整数编码

## 字节序

## 字节串编码

## 列表编码

## 可选字段编码

## 结构体序列化

## 规范化编码要求

## 测试向量要求

## 未决问题

## 对后续 Plan 的影响
```

**Step 2: Define initial encoding decisions**

Specify:

- Fixed bytes use exact length and reject mismatches.
- Numeric fields use unsigned integers unless the domain explicitly needs signed values.
- Fixed-width integers use big-endian byte order for canonical serialization.
- Variable-length byte strings use a canonical unsigned varint length prefix.
- Lists encode item count first, then canonical encoding of each item.
- Optional fields encode presence as `0x00` or `0x01`, then value if present.

**Step 3: Mark unresolved choices**

Add unresolved questions for:

- Exact varint algorithm if not yet author-approved.
- Whether human-readable source script encoding follows the same canonical byte rules.
- Whether signed integers are needed outside script runtime values.

**Step 4: Commit**

```bash
git add docs/proposal/01.Types-And-Encoding.md
git commit -m "docs: add types and encoding proposal"
```

---

### Task 3: Cryptography And Hashing Proposal

**Files:**
- Create: `docs/proposal/02.Cryptography-And-Hashing.md`
- Read: `docs/conception/blockchain.md`
- Read: `docs/conception/附.交易.md`
- Read: `docs/conception/5.信用结构.md`
- Read: `docs/conception/1.共识-历史证明（PoH）.md`

**Step 1: Draft the document structure**

Create `docs/proposal/02.Cryptography-And-Hashing.md` with these sections:

```markdown
# Cryptography And Hashing（密码学与哈希）

## 来源构想

## 目标

## 非目标

## Hash 算法分配

## Hash 输入规范

## 域分隔

## 地址哈希

## 签名算法

## 铸凭哈希

## 测试向量要求

## 安全注意事项

## 未决问题

## 对后续 Plan 的影响
```

**Step 2: Capture algorithm assignments**

Document:

- Block header: SHA3-384.
- CheckRoot: SHA3-384.
- TxID: SHA3-384.
- Tree internal nodes: BLAKE3-256.
- Tree leaves: SHA3-384 where specified by conception docs.
- Attachment fingerprint: SHA3-512.
- Address hash: SHA3-256(BLAKE2b-512(pubkey material)).
- Mint hash: BLAKE3-512 through explicit XOF output length.

**Step 3: Define implementation constraints**

Specify:

- Hash inputs must use canonical encoding from `01.Types-And-Encoding.md`.
- BLAKE3-512 must not use a default 32-byte digest API.
- Signature abstraction should allow ML-DSA-65 without exposing concrete dependency choices to upper layers.

**Step 4: Commit**

```bash
git add docs/proposal/02.Cryptography-And-Hashing.md
git commit -m "docs: add cryptography and hashing proposal"
```

---

### Task 4: Identifiers And Constants Proposal

**Files:**
- Create: `docs/proposal/03.Identifiers-And-Constants.md`
- Read: `docs/conception/blockchain.md`
- Read: `docs/conception/附.交易.md`
- Read: `docs/conception/5.信用结构.md`
- Read: `docs/conception/6.脚本系统.md`
- Read: `docs/conception/1.共识-历史证明（PoH）.md`

**Step 1: Draft the document structure**

Create `docs/proposal/03.Identifiers-And-Constants.md` with these sections:

```markdown
# Identifiers And Constants（标识符与常量）

## 来源构想

## 目标

## 非目标

## 标识符类型

## 时间与高度

## 脚本限制常量

## 交易限制常量

## 共识基础常量

## 地址文本编码

## 未决问题

## 对后续 Plan 的影响
```

**Step 2: Define identifier lengths**

Document:

- `BlockID`: 48 bytes.
- `TxID`: 48 bytes.
- `CheckRoot`: 48 bytes.
- `AddressHash`: 32 bytes.
- `TreeHash`: 32 bytes.
- `AttachmentHash`: 64 bytes.
- `MintHash`: 64 bytes.

**Step 3: Capture constants**

Document:

- `BlockInterval = 6 * time.Minute`.
- `BlocksPerYear = 87661`.
- `MaxStackHeight = 256`.
- `MaxStackItem = 1024`.
- `MaxLockScript = 1024`.
- `MaxUnlockScript = 4096`.
- `MaxTxSize = 65535`, noting conception docs also describe `<64KB` and the exact byte limit must be fixed before implementation.

**Step 4: Commit**

```bash
git add docs/proposal/03.Identifiers-And-Constants.md
git commit -m "docs: add identifiers and constants proposal"
```

---

### Task 5: Hash Trees Proposal

**Files:**
- Create: `docs/proposal/04.Hash-Trees.md`
- Read: `docs/conception/blockchain.md`
- Read: `docs/conception/附.交易.md`
- Read: `docs/conception/5.信用结构.md`
- Read: `docs/conception/附.组队校验.md`

**Step 1: Draft the document structure**

Create `docs/proposal/04.Hash-Trees.md` with these sections:

```markdown
# Hash Trees（哈希树）

## 来源构想

## 目标

## 非目标

## 通用节点类型

## 二元哈希树

## 含序叶子

## 空树与单叶树

## 证明路径

## 交易相关哈希树

## UTXO/UTCO 指纹树

## 附件片组树

## 测试向量要求

## 未决问题

## 对后续 Plan 的影响
```

**Step 2: Define shared tree rules**

Specify:

- Internal branch hash uses BLAKE3-256 unless a later Proposal narrows a specific tree.
- Ordered leaves include the documented sequence prefix before hashing.
- Proof paths must include sibling hash, sibling side, and enough position data to reconstruct the root.
- Empty tree root is unresolved unless conception already fixes it.

**Step 3: Capture tree-specific constraints**

Document:

- Transaction set tree uses transaction IDs with 3-byte sequence prefix.
- Attachment shard tree uses shard hash with 2-byte sequence prefix.
- UTXO/UTCO fingerprint tree is a separate state tree and should not be conflated with transaction Merkle-like trees.

**Step 4: Commit**

```bash
git add docs/proposal/04.Hash-Trees.md
git commit -m "docs: add hash trees proposal"
```

---

### Task 6: Update Documentation Traceability

**Files:**
- Modify: `docs/AGENTS.md`
- Read: all newly created `docs/proposal/*.md`

**Step 1: Replace placeholder mapping**

Update the `Conception => Proposal` table in `docs/AGENTS.md` so it includes the five new Proposal files and their source conception files.

Use this mapping:

```markdown
| `proposal/00.Project-Scope.md` | `conception/README.md`<br>`conception/blockchain.md` | 第一阶段边界、文档层级、包分层。 |
| `proposal/01.Types-And-Encoding.md` | `conception/blockchain.md`<br>`conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/6.脚本系统.md` | 基础类型、整数、字节串、列表和结构体编码。 |
| `proposal/02.Cryptography-And-Hashing.md` | `conception/blockchain.md`<br>`conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/1.共识-历史证明（PoH）.md` | 密码学算法分配、Hash 输入、地址哈希、签名抽象。 |
| `proposal/03.Identifiers-And-Constants.md` | `conception/blockchain.md`<br>`conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/6.脚本系统.md`<br>`conception/1.共识-历史证明（PoH）.md` | 标识符长度、系统常量、时间高度换算。 |
| `proposal/04.Hash-Trees.md` | `conception/blockchain.md`<br>`conception/附.交易.md`<br>`conception/5.信用结构.md`<br>`conception/附.组队校验.md` | 哈希树、含序叶子、证明路径、状态指纹树。 |
```

**Step 2: Verify table formatting**

Read `docs/AGENTS.md` and confirm the Markdown table remains readable.

**Step 3: Commit**

```bash
git add docs/AGENTS.md
git commit -m "docs: map foundation proposals to conceptions"
```

---

### Task 7: Final Review

**Files:**
- Read: `docs/proposal/00.Project-Scope.md`
- Read: `docs/proposal/01.Types-And-Encoding.md`
- Read: `docs/proposal/02.Cryptography-And-Hashing.md`
- Read: `docs/proposal/03.Identifiers-And-Constants.md`
- Read: `docs/proposal/04.Hash-Trees.md`
- Read: `docs/AGENTS.md`

**Step 1: Check consistency**

Verify:

- Identifier lengths match across files.
- Encoding rules are referenced by cryptography and hashing rules.
- Hash tree rules do not define transaction state machine behavior.
- Every Proposal has unresolved questions instead of silently guessing ambiguous protocol details.

**Step 2: Check git status**

Run:

```bash
git status --short
```

Expected: clean except for files intentionally left uncommitted by the user.

**Step 3: Report completion**

Summarize:

- Created Proposal files.
- Updated traceability.
- Noted unresolved decisions for author review.

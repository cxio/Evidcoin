# Hash Tree Single Leaf Root Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 DEC-0004 新的单叶树根规则同步到 proposal、plan 与 `pkg/hashtree` 实现，使通用树根始终为 32 字节。

**Architecture:** 通用树叶仍使用 `SHA3-384 + tree.leaf` 生成 48 字节叶哈希；任意对外树根使用 `tree.branch` profile 生成 32 字节根。多叶树沿用二叉分支计算，奇数层末尾节点仍直接提升；只有整棵树最终只剩一个原始叶哈希时，额外执行一元根归一化 `BLAKE3-256(DomainTag("tree.branch") || leafHash)`。

**Tech Stack:** Go 1.26+；`pkg/crypto` 的 `HashTreeLeaf` / `HashTreeBranch`；`pkg/hashtree`；Markdown 文档。

**Commit Policy:** 本次按用户要求不提交，所有变更只保留在工作区；计划中的验证命令不包含 `git commit`。

---

## Context（上下文）

DEC-0004 已更新为：单叶树根使用 `tree.branch` profile 归一化为 32 字节，即 `BLAKE3-256(DomainTag("tree.branch") || leafHash)`，不复制叶子，也不构造 `leafHash || leafHash`。当前 proposal、plan 与代码仍保留旧语义：单叶根直接等于 48 字节叶哈希。

本计划只同步现有正式文档与实现，不引入新依赖，不改附件片组树例外规则，不改变交易输入根专用规则。

---

### Task 1: 同步 Proposal 哈希树规格

**Files:**
- Modify: `docs/proposal/04.Hash-Trees.md:25-29`
- Modify: `docs/proposal/04.Hash-Trees.md:48-52`
- Modify: `docs/proposal/04.Hash-Trees.md:70-73`
- Modify: `docs/proposal/04.Hash-Trees.md:83-87`
- Modify: `docs/proposal/04.Hash-Trees.md:93-98`

**Step 1: 更新通用树规则**

将旧的“单叶根直接使用叶哈希”表述改为：

```markdown
- **单叶树根：** 单叶树根使用 `tree.branch` profile 一元归一化为 32 字节：`BLAKE3-256( DomainTag("tree.branch") || leafHash )`；不复制叶子，也不构造 `leafHash || leafHash` 双子分支。
```

**Step 2: 更新区块交易树单交易说明**

在区块交易树规则中保留“单交易区块叶子仍前置 3 字节序号”，并补充：

```markdown
- 若区块仅含一笔交易，先计算含 `seq=000` 的 48 字节叶哈希，再按单叶树根规则归一化为 32 字节交易树根。
```

**Step 3: 更新交易输出树说明**

将旧的“树根外再做一次 BLAKE3-256”表述改为：

```markdown
- 输出树根 = `Hash256( Tree<Outputs> )`，其中 `Tree<Outputs>` 本身按 §1 产出 32 字节树根；`Hash256(...)` 表示采用 256 位树根 profile，不表示在树根外再套一层额外哈希。
```

**Step 4: 更新边界与 Plan 约束**

将单叶相关旧表述改为“单叶根一元归一化，奇数叶不复制”。确保本文件不再保留会暗示旧语义的措辞。

**Step 5: 验证文档检索**

Run:

```bash
rg '单叶.*叶哈希|额外套哈希' docs/proposal/04.Hash-Trees.md
```

Expected: 无输出。

---

### Task 2: 同步 Plan 中的实施要求

**Files:**
- Modify: `docs/plan/01-Foundation-Types-Crypto.md:1-5`
- Modify: `docs/plan/01-Foundation-Types-Crypto.md:349-365`
- Modify: `docs/plan/01-Foundation-Types-Crypto.md:389-395`
- Modify: `docs/plan/03-Transaction-And-Units.md:223-235`
- Modify: `docs/plan/00-Implementation-Roadmap.md:120-123`

**Step 1: 更新阶段 01 的架构摘要**

把单叶根旧语义改为“单叶根按 `tree.branch` profile 归一化为 32B”。

**Step 2: 更新 Task 6 测试要求**

将旧测试项：

```markdown
- 单叶树根不等于 48B 叶哈希，而是 `BLAKE3-256(DomainTag("tree.branch") || leafHash)`，长度为 32B。
```

改为：

```markdown
- 单叶树根不等于 48B 叶哈希，而是 `BLAKE3-256(DomainTag("tree.branch") || leafHash)`，长度为 32B。
- 单叶证明的兄弟路径为空，但验证时会先按单叶根规则归一化后比对根。
```

**Step 3: 更新 Task 6 实现说明**

把单叶根旧语义改为“单叶树根一元归一化为 32B”。保留“奇数层直接提升、不复制”。

**Step 4: 更新阶段验收与交易输出计划**

在 `03-Transaction-And-Units.md` 中改为“单叶根归一化”。在 `00-Implementation-Roadmap.md` 中确认“单叶根策略由 DEC-0004 关闭”不再暗示旧语义。

**Step 5: 验证文档检索**

Run:

```bash
rg '单叶.*叶哈希|额外套哈希' docs/plan
```

Expected: 无旧语义输出；若 `单叶根` 出现，应是“归一化”语义。

---

### Task 3: 修改 hashtree 单叶根实现

**Files:**
- Modify: `pkg/hashtree/tree.go:1-5`
- Modify: `pkg/hashtree/tree.go:34-46`
- Modify: `pkg/hashtree/tree.go:56-70`

**Step 1: 写失败测试前先理解现状**

当前 `BuildTree` 在 `len(leafHashes)==1` 时不会进入循环，`Root()` 返回原始 48B 叶哈希。需要改为返回 32B 归一化根。

**Step 2: 添加一元根辅助函数**

在 `pkg/hashtree/tree.go` 中新增私有函数：

```go
// singleRootHash 计算单叶树根：BLAKE3-256(tree.branch || leafHash)。
// 该规则只用于整棵树只有一个叶子时，避免根宽度随叶子数量变化。
func singleRootHash(leaf []byte) []byte {
    h := crypto.HashTreeBranch(leaf)
    return h.Bytes()
}
```

注意：`crypto.HashTreeBranch(data)` 已经在内部添加 `tree.branch` 域标签，因此这里不要手工追加域标签。

**Step 3: 修改 BuildTree 的单叶路径**

在复制叶哈希后，增加单叶特判：

```go
if len(level) == 1 {
    t := &Tree{levels: [][][]byte{level}}
    t.levels = append(t.levels, [][]byte{singleRootHash(level[0])})
    return t, nil
}
```

也可以用等价结构实现，但必须保证：

- `t.levels[0][0]` 仍是原始 48B 叶哈希，供 Proof 使用；
- `t.Root()` 返回 32B 根；
- 单叶 proof 的 `Siblings` 为空。

**Step 4: 保持多叶与奇数提升逻辑**

不要改变现有多叶循环。特别是 3 叶树仍应计算：

```text
root = branchHash(branchHash(leaf0, leaf1), leaf2)
```

这里提升的 `leaf2` 可作为右侧输入参与上层分支哈希，不要提前归一化。

**Step 5: 更新 package 注释**

把 `tree.go` 顶部“单叶根即该 48 字节叶哈希本身”改为“单叶根按 tree.branch profile 归一化为 32 字节”。

---

### Task 4: 更新 hashtree 测试

**Files:**
- Modify: `pkg/hashtree/tree_test.go:16-28`
- Modify: `pkg/hashtree/proof_test.go:7-23`

**Step 1: 修改单叶根测试**

将 `TestSingleLeafRootEqualsLeaf` 改名为：

```go
func TestSingleLeafRootNormalizedToBranchHash(t *testing.T) {
    l := leaves(1)
    tree, err := BuildTree(l)
    if err != nil {
        t.Fatal(err)
    }
    want := branchHash(l[0], nil)
    if !bytes.Equal(tree.Root(), want) {
        t.Fatal("single leaf root must be normalized with tree.branch profile")
    }
    if bytes.Equal(tree.Root(), l[0]) {
        t.Fatal("single leaf root must not equal the 48-byte leaf hash")
    }
    if len(tree.Root()) != 32 {
        t.Fatalf("single leaf root len = %d, want 32", len(tree.Root()))
    }
}
```

如果实现采用 `singleRootHash` 而非 `branchHash(l[0], nil)`，测试可以直接使用 `crypto.HashTreeBranch(l[0]).Bytes()` 作为期望值。避免测试依赖错误的 `left||right` 语义。

**Step 2: 添加单叶证明路径测试**

在 `proof_test.go` 增加：

```go
func TestSingleLeafProofHasNoSiblingsAndVerifies(t *testing.T) {
    tree, err := BuildTree(leaves(1))
    if err != nil {
        t.Fatal(err)
    }
    p, err := tree.Proof(0)
    if err != nil {
        t.Fatal(err)
    }
    if len(p.Siblings) != 0 {
        t.Fatalf("single leaf proof siblings = %d, want 0", len(p.Siblings))
    }
    if !Verify(p) {
        t.Fatal("single leaf proof must verify after root normalization")
    }
}
```

**Step 3: 先运行测试确认失败**

Run:

```bash
go test ./pkg/hashtree -run 'TestSingleLeaf' -v
```

Expected before implementation: FAIL，因为当前单叶根仍是 48B，且 `Verify` 对空 siblings 不归一化。

**Step 4: 实现后再次运行**

Run:

```bash
go test ./pkg/hashtree -v
```

Expected after implementation: PASS。

---

### Task 5: 修改 proof 验证逻辑

**Files:**
- Modify: `pkg/hashtree/proof.go:65-79`

**Step 1: 修改 Verify 的空 siblings 路径**

当前 `Verify` 在 `len(Siblings)==0` 时直接比较 `LeafHash` 与 `Root`。改为：

```go
func Verify(p Proof) bool {
    cur := cloneBytes(p.LeafHash)
    if len(p.Siblings) == 0 {
        cur = singleRootHash(cur)
        return bytes.Equal(cur, p.Root)
    }
    for _, s := range p.Siblings {
        switch s.Direction {
        case SiblingLeft:
            cur = branchHash(s.Hash, cur)
        case SiblingRight:
            cur = branchHash(cur, s.Hash)
        default:
            return false
        }
    }
    return bytes.Equal(cur, p.Root)
}
```

**Step 2: 运行证明测试**

Run:

```bash
go test ./pkg/hashtree -run 'TestProof|TestSingleLeaf' -v
```

Expected: PASS。

**Step 3: 注意安全边界**

不要用 `len(Siblings)==0` 接受任意 32B `Root` 与 48B `LeafHash` 的直接相等；必须经过 `singleRootHash`。

---

### Task 6: 全局检索与验证

**Files:**
- Review only: `docs/decision/DEC-0004-hash-tree-and-proof-edge-cases.md`
- Review only: `docs/proposal/04.Hash-Trees.md`
- Review only: `docs/plan/01-Foundation-Types-Crypto.md`
- Review only: `docs/plan/03-Transaction-And-Units.md`
- Review only: `pkg/hashtree/*.go`

**Step 1: 检索旧语义**

Run:

```bash
rg '单叶.*直接|额外套哈希|48B leafHash' docs pkg/hashtree
```

Expected: 无旧语义输出。允许出现“单叶树根”但必须是归一化语义。

**Step 2: 格式化 Go 代码**

Run:

```bash
gofmt -w pkg/hashtree/tree.go pkg/hashtree/proof.go pkg/hashtree/tree_test.go pkg/hashtree/proof_test.go
```

Expected: 命令无输出。

**Step 3: 运行相关测试**

Run:

```bash
go test ./pkg/hashtree -v
```

Expected: PASS。

**Step 4: 运行基础层相关测试**

Run:

```bash
go test ./pkg/types ./pkg/crypto ./pkg/hashtree
```

Expected: PASS。

**Step 5: 查看最终 diff**

Run:

```bash
git diff -- docs/proposal/04.Hash-Trees.md docs/plan/00-Implementation-Roadmap.md docs/plan/01-Foundation-Types-Crypto.md docs/plan/03-Transaction-And-Units.md pkg/hashtree/tree.go pkg/hashtree/proof.go pkg/hashtree/tree_test.go pkg/hashtree/proof_test.go
```

Expected: diff 仅包含单叶根语义同步、测试更新和必要注释更新。

---

## Risks（风险）

- 当前 `pkg/hashtree` API 用 `[]byte` 表示根。同步后根应稳定为 32B，但本计划不额外改 API 类型，避免扩大范围。
- 附件片组树是 DEC-0002 例外。本计划不修改附件片组树实现；如果后续发现附件片组单叶根也有独立歧义，应另开决策或在 DEC-0002 中明确。
- UTXO/UTCO 宽成员树不直接套通用二叉证明格式，但其分支仍使用 `tree.branch`。本计划只同步通用树与相关文字，不重写状态树。

## Final Verification（最终验证）

至少运行：

```bash
rg '单叶.*直接|额外套哈希|48B leafHash' docs pkg/hashtree
gofmt -w pkg/hashtree/tree.go pkg/hashtree/proof.go pkg/hashtree/tree_test.go pkg/hashtree/proof_test.go
go test ./pkg/hashtree -v
go test ./pkg/types ./pkg/crypto ./pkg/hashtree
```

若修改影响更广，再运行：

```bash
go test ./...
go build ./...
```

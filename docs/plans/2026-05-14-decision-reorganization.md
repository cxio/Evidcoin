# Decision Reorganization 2026-05-14 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `docs/decision/` 全面重整为只补充 `docs/conception/` 未明确内容的规范化决策集。

**Architecture:** 以 `docs/conception/` 为唯一正式上游，按基础编码、交易见证、状态指纹、PoH、激励 Coinbase、通告脚本、证明服务格式分组重建 DEC。README 作为索引和迁移账本，记录旧 DEC 去向、已吸收内容和仍需作者裁决的问题。

**Tech Stack:** Markdown 文档、Git、shell 校验命令（rg/find/grep）。

---

### Task 1: 基线与依据确认

**Files:**
- Read: `docs/conception/**/*.md`
- Read: `docs/decision/*.md`

**Step 1:** 确认在 `.worktrees/decision-reorganization-2026-05-14` 工作，分支为 `docs/decision-reorganization-2026-05-14`。

**Step 2:** 搜索 conception 中与 PoH、Coinbase、分叉、脚本、地址、见证和证明包相关的最新规则。

**Step 3:** 搜索旧 DEC 中的过时口径和冲突项。

**Step 4:** 提交设计文档和本实施计划。

### Task 2: 重建 DEC 文件集

**Files:**
- Delete/Rewrite: `docs/decision/DEC-*.md`
- Modify: `docs/decision/README.md`

**Step 1:** 删除旧 DEC 文件，保留 README 作为待重写文件。

**Step 2:** 创建新 `DEC-0001` 至 `DEC-0029`，每个文件使用统一结构。

**Step 3:** 将无法由 conception 直接裁定的项目标为 `Proposed`，并列明待裁决参数。

**Step 4:** 将 conception 已明确内容从 Decision 主体移出，仅作为引用或 README 吸收清单。

**Step 5:** 重写 README，包含当前索引、迁移索引、已吸收/不再单列清单、开放问题。

### Task 3: 自审与修正

**Files:**
- Modify: `docs/decision/*.md`

**Step 1:** 检查是否错误保留了 conception 已明确内容作为 Decision 主体。

**Step 2:** 检查是否伪造 conception 不存在的裁决，必要时改为 `Proposed`。

**Step 3:** 检查 README 链接、迁移表和状态统计是否一致。

**Step 4:** 修正 35 区块、Version 宽度、Merkle 边界、公告初始根、FLOAT 输入限制、FN_ADDRESS 哈希冲突、服务命名顺序等重点问题。

### Task 4: 校验与提交

**Files:**
- All changed docs.

**Step 1:** 运行 `find docs/decision -maxdepth 1 -type f -name 'DEC-*.md' | sort`。

**Step 2:** 检查 README 链接对应文件存在。

**Step 3:** 运行 `rg --line-number -- '35 个区块|35区块|uint32_be\(TxHeader.Version\)|COMFLO|MintInner|MintHash|内层|外层|-27' docs/decision || true`。

**Step 4:** 运行 `rg --line-number 'Status: (Accepted|Proposed|Deprecated|Superseded|Absorbed)' docs/decision/DEC-*.md`。

**Step 5:** 运行 `go test ./...` 作为 worktree baseline/最终校验；若仍无 Go package，记录输出。

**Step 6:** 分别提交 `design/plan`、`reorganize decisions`、`verification fixes`；若某阶段无变更则跳过该提交。

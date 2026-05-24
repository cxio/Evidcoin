# Decision Rewrite Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 重写 `docs/decision/`，使其与最新构想层一致，并只保留构想层未冻结的补充决策。

**Architecture:** 采用主题段编号重建 Decision 文件集。`README.md` 作为索引和维护规则入口，`CONCEPTION-CONFLICTS.md` 作为构想层矛盾清单，其余 DEC 文件按基础、交易、状态、共识、激励、脚本、证明服务分组。

**Tech Stack:** Markdown 文档；验证以文件存在性、内部链接、旧文件清理和 `git diff --stat` 为主。

---

### Task 1: 写入重写设计文档

**Files:**
- Create: `docs/plans/2026-05-21-decision-rewrite-design.md`

**Step 1: 写入设计**

保存本次结构设计、编号策略、文件清单、删除/吸收策略和冲突处理策略。

**Step 2: 核对文件存在**

Run: `test -f docs/plans/2026-05-21-decision-rewrite-design.md`
Expected: exit code 0

### Task 2: 写入实施计划

**Files:**
- Create: `docs/plans/2026-05-21-decision-rewrite.md`

**Step 1: 写入计划**

保存本文档，列出每一步的目标文件和验证方式。

**Step 2: 核对文件存在**

Run: `test -f docs/plans/2026-05-21-decision-rewrite.md`
Expected: exit code 0

### Task 3: 删除旧决策文件并创建新文件集

**Files:**
- Delete: `docs/decision/DEC-0001-canonical-varint.md` through old `docs/decision/DEC-0029-*.md`
- Modify: `docs/decision/README.md`
- Create: `docs/decision/CONCEPTION-CONFLICTS.md`
- Create: new `docs/decision/DEC-*.md` files from the design list

**Step 1: 删除旧 DEC**

用 `apply_patch` 删除旧编号文件，保留目录。

**Step 2: 新增新 DEC**

按设计文件清单新增新编号文件。

**Step 3: 更新 README**

重写索引、状态定义、主题分组、已吸收/移除清单和维护规则。

### Task 4: 审查链接和一致性

**Files:**
- Review: `docs/decision/README.md`
- Review: `docs/decision/*.md`

**Step 1: 检查旧编号残留**

Run: `rg "DEC-00(0[5-9]|1[0-9]|2[0-9])-" docs/decision`
Expected: no references to removed old filenames except absorbed history text if explicitly marked.

**Step 2: 检查新索引链接**

Run: `for f in docs/decision/DEC-*.md docs/decision/CONCEPTION-CONFLICTS.md; do test -f "$f" || exit 1; done`
Expected: exit code 0

### Task 5: 最终验证

**Files:**
- Verify: `docs/decision/`
- Verify: `docs/plans/2026-05-21-decision-rewrite*.md`

**Step 1: 查看状态**

Run: `git status --short`
Expected: only intended docs changes.

**Step 2: 查看差异统计**

Run: `git diff --stat`
Expected: decision rewrite and plan docs are visible.

**Step 3: 复核关键内容**

确认 `CONCEPTION-CONFLICTS.md` 包含构想层自我矛盾，`README.md` 明确 Decision 不替代 conception。

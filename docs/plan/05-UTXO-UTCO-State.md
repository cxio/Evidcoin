# UTXO UTCO State Implementation Plan

**Goal:** 实现 Coin 的 UTXO 状态、Credit 的 UTCO 状态、局部引用解析、状态转移、状态指纹和基础回滚接口。

**Architecture:** `internal/utxo/` 和 `internal/utco/` 分别维护 Coin 与 Credit 状态，语义隔离但共享相同状态指纹算法。状态层通过接口调用脚本验证，不直接依赖脚本 VM 具体执行器，以避免循环依赖。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、`pkg/hashtree`、`internal/tx`、表驱动测试。

---

## 来源提案

- `docs/proposal/09.UTXO-UTCO-State.md`
- 依赖 `docs/proposal/06.Transaction-Model.md`、`07.Coin-Credit-Proof-Units.md`、`04.Hash-Trees.md`、`02.Cryptography-And-Hashing.md`
- DEC-0201：五层宽成员树（年度层 + TxID `[8]`/`[13]`/`[18]` 三层 + 末端完整 TxID 叶子层）、叶子前像 `TxID || Count || FlagBytes`、FlagBytes 位序低位优先、空根 `utxo.empty`/`utco.empty`、UTCO 过期、缓存边界。
- DEC-0002（引用）：`utxo.leaf`/`utco.leaf`/`utxo.empty`/`utco.empty`/`tree.branch` 域标签。
- 状态根口径（proposal 09 §7）：UTXORoot/UTCORoot 取**前一区块完成后**状态集，供 plan 02 的 CheckRoot 合并。
- 短引用碰撞拒绝按 proposal 06 §5 / DEC-0101（TxID 排序首个匹配）。
- 同块链式消费禁止：输入只能引用已确认历史区块输出（proposal 09 §1，交易独立性）。

## 包边界

| 包 | 职责 | 禁止事项 |
|----|------|----------|
| `internal/utxo` | Coin entry、消费、插入、UTXO root | 不处理 Credit 语义 |
| `internal/utco` | Credit entry、转移、到期、UTCO root | 不处理 Coin 金额守恒 |
| `internal/state` 可选 | 共享引用解析、快照接口、批处理上下文 | 不承载领域规则 |

如果 Go import 关系更简单，可以先不创建 `internal/state`，在 `utxo` 和 `utco` 内部重复少量结构，等重复超过 3 处再抽取。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/utxo/entry.go` | UTXO entry |
| `internal/utxo/store.go` | UTXO store 接口 |
| `internal/utxo/resolver.go` | 完整和局部引用解析 |
| `internal/utxo/apply.go` | Coin 状态转移 |
| `internal/utxo/fingerprint.go` | UTXO 指纹叶子和 root |
| `internal/utxo/snapshot.go` | 快照接口 |
| `internal/utxo/errors.go` | 错误定义 |
| `internal/utco/entry.go` | UTCO entry |
| `internal/utco/store.go` | UTCO store 接口 |
| `internal/utco/resolver.go` | Credit 引用解析 |
| `internal/utco/apply.go` | Credit 状态转移 |
| `internal/utco/expiry.go` | 到期和激活清理 |
| `internal/utco/fingerprint.go` | UTCO 指纹叶子和 root |
| `internal/utco/snapshot.go` | 快照接口 |
| `internal/utco/errors.go` | 错误定义 |

## Task 1: UTXO entry 与 store 接口

**Files:**
- Create: `internal/utxo/entry.go`
- Create: `internal/utxo/store.go`
- Create: `internal/utxo/entry_test.go`
- Create: `internal/utxo/store_test.go`

**Step 1: 写失败测试**

测试：

- Entry 包含年度、完整 `TxID`、`OutIndex`、Coin payload 摘要或完整 payload、金额、接收者、锁定脚本、创建高度、有效位。
- 已消费 entry 不能再次消费。
- store 可按完整 outpoint 查询。
- 缺失 outpoint 返回明确错误。

**Step 2: 实现并提交**

```bash
go test ./internal/utxo -run 'Test(Entry|Store)' -v
git add internal/utxo/entry.go internal/utxo/store.go internal/utxo/entry_test.go internal/utxo/store_test.go
git commit -m "feat: add utxo entries"
```

## Task 2: UTCO entry 与 store 接口

**Files:**
- Create: `internal/utco/entry.go`
- Create: `internal/utco/store.go`
- Create: `internal/utco/entry_test.go`
- Create: `internal/utco/store_test.go`

**Step 1: 写失败测试**

测试：

- Entry 包含年度、完整 `TxID`、`OutIndex`、Credit payload（接收者/创建者/标题/描述/附件 ID）、锁定脚本、创建高度、有效位。
- Credit 为一次性转移/消费：转移即消费旧 UTCO 并新建（proposal 07 §5），不维护多次转移计数。
- 过期 Credit（`age > 31 × 87661`）不可解析为有效输入；`age == 31 × 87661` 仍可用。

**Step 2: 实现并提交**

```bash
go test ./internal/utco -run 'Test(Entry|Store)' -v
git add internal/utco/entry.go internal/utco/store.go internal/utco/entry_test.go internal/utco/store_test.go
git commit -m "feat: add utco entries"
```

## Task 3: 局部 TxIDPart 解析

**Files:**
- Create: `internal/utxo/resolver.go`
- Create: `internal/utco/resolver.go`
- Create: `internal/utxo/resolver_test.go`
- Create: `internal/utco/resolver_test.go`

**Step 1: 写失败测试**

测试：

- 完整 outpoint 可解析。
- `Year + TxIDPart + OutIndex` 唯一匹配可解析。
- 多个有效项匹配同一局部引用时拒绝。
- 歧义拒绝是固定协议策略（DEC-0101 / proposal 06 §5）：同年度内末端叶按 TxID 排序、首个匹配即引用；引用错误交易无法验证，用户应预查询或延长引用字节数，不增加动态扩展机制。
- 无匹配返回缺失错误。
- 无效项不参与有效解析。

**Step 2: 实现并提交**

```bash
go test ./internal/utxo ./internal/utco -run 'TestResolver' -v
git add internal/utxo/resolver.go internal/utco/resolver.go internal/utxo/resolver_test.go internal/utco/resolver_test.go
git commit -m "feat: resolve state references"
```

## Task 4: Coin 状态转移

**Files:**
- Create: `internal/utxo/apply.go`
- Create: `internal/utxo/apply_test.go`
- Create: `internal/utxo/errors.go`

**Step 1: 写失败测试**

测试：

- 正常 Coin 输入消费后无效。
- 同一批次重复消费拒绝。
- 输出 Coin 插入 UTXO。
- 自定义类输出（Config bit7=1）不进入 UTXO；普通交易无销毁位（销毁仅由 Coinbase `BurnCoin` 表达，不产出可花费项）。
- Proof/Credit 输入传入 UTXO apply 时拒绝。
- 同一区块 A 输出被 B 输入引用时拒绝，无论 A 是否在 B 之前；输入只能引用已确认历史区块中的 UTXO。

**Step 2: 实现**

定义 `ScriptVerifier` 接口，由调用方注入：

```go
type ScriptVerifier interface {
    VerifyCoinSpend(ctx context.Context, entry Entry, input tx.Input) error
}
```

不要 import `internal/script`。

**Step 3: 验证并提交**

```bash
go test ./internal/utxo -run 'TestApply' -v
git add internal/utxo/apply.go internal/utxo/apply_test.go internal/utxo/errors.go
git commit -m "feat: apply utxo state transitions"
```

## Task 5: Credit 状态转移

**Files:**
- Create: `internal/utco/apply.go`
- Create: `internal/utco/expiry.go`
- Create: `internal/utco/apply_test.go`
- Create: `internal/utco/expiry_test.go`
- Create: `internal/utco/errors.go`

**Step 1: 写失败测试**

测试：

- 新建 Credit 插入 UTCO。
- 转移 Credit 消费旧 UTCO 并插入新 UTCO（一次性转移，proposal 07 §5）。
- payload 不可变字段（创建者/标题/描述/附件 ID）变更拒绝。
- 过期 Credit（`age > 31 × 87661`）在区块结束清理：状态位失效，同 TxID 无任何有效 Credit 时删除该 UTCO 叶（proposal 09 §6）。
- 同 TxID 仍有其它未转出且未过期 Credit 时保留叶并 `Count` 递减。
- 同一区块 A 输出被 B 输入引用时拒绝，无论 A 是否在 B 之前；输入只能引用已确认历史区块中的 UTCO。

**Step 2: 实现**

定义 `ScriptVerifier` 接口，由调用方注入。Credit 转移验证必须同时检查签名/脚本结果和 payload 不可变字段。

**Step 3: 验证并提交**

```bash
go test ./internal/utco -run 'Test(Apply|Expiry)' -v
git add internal/utco/apply.go internal/utco/expiry.go internal/utco/apply_test.go internal/utco/expiry_test.go internal/utco/errors.go
git commit -m "feat: apply utco state transitions"
```

## Task 6: 状态指纹叶子

**Files:**
- Create: `internal/utxo/fingerprint.go`
- Create: `internal/utco/fingerprint.go`
- Create: `internal/utxo/fingerprint_test.go`
- Create: `internal/utco/fingerprint_test.go`

**Step 1: 写失败测试**

测试（DEC-0201 §3-§4）：

- 末端叶前像 = `DomainTag("utxo.leaf"/"utco.leaf") || TxID || Count || FlagBytes`，叶哈希 = `SHA3-384(前像)`。
- `Count` 为该 TxID 的**有效输出数量**（非 FlagBytes 字节数），减至零时该叶可移除。
- `FlagBytes` 第 i 位对应输出序位 i，**每字节低位优先**（bit0 对应较小序位），`1`=未花费/未转出，`0`=已花费/已转出或无效，尾部未用 bit 必须为 `0`。
- 输出详情（缓存集）**不参与**叶前像，仅作检索优化。
- UTXO 与 UTCO 同结构但类型/API 不可混用（域标签不同）。
- 任一 flag 位或 `Count` 变化会改变叶哈希。

**Step 2: 实现并提交**

```bash
go test ./internal/utxo ./internal/utco -run 'TestFingerprintLeaf' -v
git add internal/utxo/fingerprint.go internal/utco/fingerprint.go internal/utxo/fingerprint_test.go internal/utco/fingerprint_test.go
git commit -m "feat: add state fingerprint leaves"
```

## Task 7: 五层状态指纹 root

**Files:**
- Modify: `internal/utxo/fingerprint.go`
- Modify: `internal/utco/fingerprint.go`
- Create: `internal/utxo/root_test.go`
- Create: `internal/utco/root_test.go`

**Step 1: 写失败测试**

测试（DEC-0201 §2、§5）：

- 顶层按年度**数值升序**分级。
- 后三级使用 TxID 字节 `[8]`、`[13]`、`[18]` 分层（0-based）。
- 使用具体 TxID 测试向量验证第 9、14、19 个字节分别进入三级路由，避免 1-based 误实现。
- 同一末端分组内按**完整 TxID 字典序**排列。
- 空年度、空分组不编码。
- 分支节点按 `tree.branch` 域标签编码；该树不套用第 04 章通用二叉证明格式。
- 空状态 root 使用专用空根：UTXO = `SHA3-384(DomainTag("utxo.empty"))`，UTCO = `SHA3-384(DomainTag("utco.empty"))`（**非全零**）。
- 同一数据进入 UTXO root 与 UTCO root 时语义隔离（空根/叶域标签不同）。
- 单项、多项 root 稳定。

**Step 2: 实现**

DEC-0201 已冻结五层宽成员树完整结构（年度层升序 + TxID `[8]`/`[13]`/`[18]` 三层分组 + 末端完整 TxID 叶子层字典序 + `tree.branch` 分支域 + 末端 `utxo.leaf`/`utco.leaf` 域 SHA3-384 + 专用空根）。完整实现该结构，不再返回 `ErrSpecIncomplete`。

**Step 3: 验证并提交**

```bash
go test ./internal/utxo ./internal/utco -run 'TestFingerprintRoot' -v
git add internal/utxo/fingerprint.go internal/utco/fingerprint.go internal/utxo/root_test.go internal/utco/root_test.go
git commit -m "feat: add state fingerprint grouping"
```

## Task 8: 快照与回滚接口

**Files:**
- Create: `internal/utxo/snapshot.go`
- Create: `internal/utco/snapshot.go`
- Create: `internal/utxo/snapshot_test.go`
- Create: `internal/utco/snapshot_test.go`

**Step 1: 写失败测试**

测试：

- 快照绑定高度、`BlockID`、状态 root、链身份。
- 应用失败可回滚批次内变更。
- 回滚不影响批次前状态。

**Step 2: 实现并提交**

```bash
go test ./internal/utxo ./internal/utco -run 'TestSnapshot' -v
git add internal/utxo/snapshot.go internal/utco/snapshot.go internal/utxo/snapshot_test.go internal/utco/snapshot_test.go
git commit -m "feat: add state snapshots"
```

## 阶段验收

运行：

```bash
go fmt ./...
go test ./internal/utxo ./internal/utco ./internal/tx
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- UTXO 和 UTCO 状态语义隔离（空根/叶域标签不同）。
- 局部引用歧义按 DEC-0101 拒绝（TxID 排序首个匹配）。
- 同批次重复消费、同块链式引用拒绝。
- Proof 不进入任一状态集；自定义类输出不进入状态集。
- 五层宽成员树按 DEC-0201 完整实现：年度层升序 + TxID `[8]`/`[13]`/`[18]` 三层分组 + 末端完整 TxID 叶子层字典序；叶前像 `TxID || Count || FlagBytes`；空根用 `utxo.empty`/`utco.empty` 域哈希（非全零）。
- 叶前像 `Count` 为有效输出数，FlagBytes 位序低位优先、尾部 0。
- UTCO 过期（`age > 31×87661`）删叶逻辑与第 07 章联动。
- 状态根取前一区块完成态，供 plan 02 CheckRoot 合并；缓存集（输出详情）不参与状态根。

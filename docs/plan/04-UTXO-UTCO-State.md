# UTXO UTCO State Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 Coin 的 UTXO 状态、Credit 的 UTCO 状态、局部引用解析、状态转移、状态指纹和基础回滚接口。

**Architecture:** `internal/utxo/` 和 `internal/utco/` 分别维护 Coin 与 Credit 状态，语义隔离但共享相同状态指纹算法。状态层通过接口调用脚本验证，不直接依赖脚本 VM 具体执行器，以避免循环依赖。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、`pkg/hashtree`、`internal/tx`、表驱动测试。

---

## 来源提案

- `docs/proposal/08.UTXO-UTCO-State.md`
- 依赖 `docs/proposal/06.Transaction-Model.md`
- 依赖 `docs/proposal/07.Coin-Credit-Proof-Units.md`
- 依赖 `docs/proposal/04.Hash-Trees.md`

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

- Entry 包含年度、完整 `TxID`、`OutIndex`、Credit payload、接收者、创建者、配置、标题、描述、附件 ID、锁定脚本、创建/激活高度、截止高度、剩余转移次数、有效位。
- 转移次数为 0 的 Credit 不可再转移。
- 到期 Credit 不可解析为有效输入。

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
- 销毁 flag 的 Coin 不进入 UTXO。
- Proof/Credit 输入传入 UTXO apply 时拒绝。
- 同一区块内前序新输出不能被后序交易消费，除非 Proposal 未来显式允许。

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
- 转移 Credit 消费旧 UTCO 并插入新 UTCO。
- 不可变字段变更拒绝。
- 可修改字段按配置允许变更。
- 可修改性只能降级。
- 到期 Credit 在区块结束清理。
- 转移次数归零后不进入 UTCO。

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

测试：

- `LeafHash = SHA3-384(TxID || DataID || FlagOutputs)`。
- `DataID = SHA3-384(OutputsPayloadsSortedByOutIndex)`。
- `OutIndex` 升序影响稳定性。
- UTXO 和 UTCO 同结构但类型/API 不可混用。
- flag `1` 与 `0` 变化会改变叶子 Hash。

**Step 2: 实现并提交**

```bash
go test ./internal/utxo ./internal/utco -run 'TestFingerprintLeaf' -v
git add internal/utxo/fingerprint.go internal/utco/fingerprint.go internal/utxo/fingerprint_test.go internal/utco/fingerprint_test.go
git commit -m "feat: add state fingerprint leaves"
```

## Task 7: 四层状态指纹 root

**Files:**
- Modify: `internal/utxo/fingerprint.go`
- Modify: `internal/utco/fingerprint.go`
- Create: `internal/utxo/root_test.go`
- Create: `internal/utco/root_test.go`

**Step 1: 写失败测试**

测试：

- 顶层按年度分级。
- 后三级使用 `TxID` 字节 `[8,13,18]` 分层。
- 同一数据进入 UTXO root 与 UTCO root 时有语义隔离。
- 空状态 root 如 Proposal 未固定则返回明确未决错误。
- 单项、多项 root 稳定。

**Step 2: 实现**

如果四层宽成员树节点组合细节未固定，只实现 bucket 分组和叶子列表，root 函数返回 `ErrSpecIncomplete`。不要发明最终 root。

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

- UTXO 和 UTCO 状态语义隔离。
- 局部引用歧义拒绝。
- 同批次重复消费拒绝。
- Proof 不进入任一状态集。
- 状态指纹未决部分不会伪装成最终协议 root。

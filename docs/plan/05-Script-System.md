# Script System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 Evidcoin 栈式脚本 VM 的 bytecode、运行时空间、资源限制、公共/私有模式、安全状态机和分批指令注册表。

**Architecture:** `internal/script/` 是 Layer 3，依赖基础类型和上层注入的交易/状态环境接口，但交易、状态、共识包不得依赖脚本包的具体实现。先实现 VM 核心和元数据注册，再按指令组逐批实现；危险或未决指令默认解析但拒绝执行。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、显式 instruction registry、表驱动测试、fuzz 测试候选。

---

## 来源提案

- `docs/proposal/09.Script-System.md`
- `docs/proposal/Instruction/README.md`
- `docs/proposal/Instruction/0.Base-Constraints.md`
- `docs/proposal/Instruction/1.Value-Instructions.md` 到 `18.Extension-Instructions.md`

## 非目标

- 不实现源码级脚本语言解析器。
- 不执行公共验证中禁止的本机时间、Shell 或私有扩展。
- 不直接访问 UTXO/UTCO store。
- 不直接决定交易是否合法，只返回脚本验证结果。
- 不一次性实现全部 254 个 opcode 的复杂语义。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/script/opcode.go` | opcode 常量和范围 |
| `internal/script/bytecode.go` | bytecode decode、instruction frame |
| `internal/script/instruction.go` | 指令接口 |
| `internal/script/metadata.go` | opcode 元数据 schema |
| `internal/script/registry.go` | 指令注册表 |
| `internal/script/value.go` | VM 值类型 |
| `internal/script/stack.go` | 数据栈 |
| `internal/script/args.go` | 实参区 FIFO |
| `internal/script/scope.go` | 局部域、全局域 |
| `internal/script/env.go` | 环境接口 |
| `internal/script/executor.go` | VM 执行循环 |
| `internal/script/state.go` | 执行状态 |
| `internal/script/cost.go` | 资源和成本预算 |
| `internal/script/public_private.go` | 公共/私有模式限制 |
| `internal/script/errors.go` | 错误定义 |
| `internal/script/instr_value.go` | 0-18 值指令 |
| `internal/script/instr_capture.go` | 19-23 截取指令 |
| `internal/script/instr_stack.go` | 24-34 栈操作 |
| `internal/script/instr_collection.go` | 35-45 集合操作 |
| `internal/script/instr_interaction.go` | 46-50 交互指令 |
| `internal/script/instr_result.go` | 51-57 结果指令 |
| `internal/script/instr_flow.go` | 58-66 流程控制 |
| `internal/script/instr_conversion.go` | 67-79 转换指令 |
| `internal/script/instr_arithmetic.go` | 80-103 运算指令 |
| `internal/script/instr_comparison.go` | 104-111 比较指令 |
| `internal/script/instr_logic.go` | 112-115 逻辑指令 |
| `internal/script/instr_pattern.go` | 116-127 模式指令 |
| `internal/script/instr_environment.go` | 128-137 环境指令 |
| `internal/script/instr_tool.go` | 138-163 工具指令 |
| `internal/script/instr_system.go` | 164-169 系统指令 |
| `internal/script/instr_function.go` | 170-224 函数指令 |
| `internal/script/instr_module.go` | 225-250 模块指令 |
| `internal/script/instr_extension.go` | 251-253 扩展指令 |

## Task 1: opcode 与元数据注册表

**Files:**
- Create: `internal/script/opcode.go`
- Create: `internal/script/metadata.go`
- Create: `internal/script/registry.go`
- Create: `internal/script/opcode_test.go`
- Create: `internal/script/metadata_test.go`

**Step 1: 写失败测试**

测试：

- opcode `0-169` 为基础指令范围。
- opcode `170-224` 为函数指令范围。
- opcode `225-250` 为模块指令范围。
- opcode `251-253` 为扩展指令范围。
- opcode `254-255` 为系统保留，默认拒绝。
- 每条注册指令必须包含 mnemonic、附参 schema、关联数据 schema、实参数量模型、返回数量、解锁脚本可用性、确定性、公私可用性、成本等级、错误场景描述。

**Step 2: 实现并提交**

```bash
go test ./internal/script -run 'Test(Opcode|Metadata|Registry)' -v
git add internal/script/opcode.go internal/script/metadata.go internal/script/registry.go internal/script/opcode_test.go internal/script/metadata_test.go
git commit -m "feat: add script opcode registry"
```

## Task 2: bytecode 解码

**Files:**
- Create: `internal/script/bytecode.go`
- Create: `internal/script/bytecode_test.go`
- Create: `internal/script/errors.go`

**Step 1: 写失败测试**

测试：

- `opcode + attached parameters + associated data` 可解码为 instruction frame。
- 附参长度不足拒绝。
- 关联数据长度不足拒绝。
- 多余尾随数据拒绝，除非明确是下一条指令。
- `MaxLockScript`、`MaxUnlockScript` 超限拒绝。
- unlock script 只允许 opcode `0-50` 和 `SYS_NULL`。

**Step 2: 实现并提交**

```bash
go test ./internal/script -run 'TestBytecode' -v
git add internal/script/bytecode.go internal/script/bytecode_test.go internal/script/errors.go
git commit -m "feat: decode script bytecode"
```

## Task 3: 值系统、栈和实参区

**Files:**
- Create: `internal/script/value.go`
- Create: `internal/script/stack.go`
- Create: `internal/script/args.go`
- Create: `internal/script/value_test.go`
- Create: `internal/script/stack_test.go`
- Create: `internal/script/args_test.go`

**Step 1: 写失败测试**

测试：

- 支持 `Nil`、`Bool`、`Byte`、`Int`、`String`、`Bytes` 等先导类型。
- 栈 LIFO。
- 空栈 pop 拒绝。
- `MaxStackHeight = 256` 超限拒绝。
- 单个 stack item 超过 `MaxStackItem = 1024` 拒绝。
- 实参区 FIFO。

**Step 2: 实现**

先实现最小值类型集合；`Rune`、`BigInt`、`Float`、`RegExp`、`Time`、`Dict`、`Module/Object` 可用占位类型加拒绝转换，后续指令任务再扩展。

**Step 3: 验证并提交**

```bash
go test ./internal/script -run 'Test(Value|Stack|Args)' -v
git add internal/script/value.go internal/script/stack.go internal/script/args.go internal/script/value_test.go internal/script/stack_test.go internal/script/args_test.go
git commit -m "feat: add script runtime values"
```

## Task 4: 执行状态机和公共/私有模式

**Files:**
- Create: `internal/script/state.go`
- Create: `internal/script/public_private.go`
- Create: `internal/script/executor.go`
- Create: `internal/script/cost.go`
- Create: `internal/script/state_test.go`
- Create: `internal/script/public_private_test.go`
- Create: `internal/script/executor_test.go`

**Step 1: 写失败测试**

测试：

- 状态包括 `Running`、`Passed`、`Failed`、`Exited`、`Returned`、`EndedForPublicValidation`。
- `END` 在公共验证中结束公共路径。
- 公共验证遇到无数据 `INPUT` 视为隐式 `END`。
- `SYS_TIME`、`SHELL`、`EXT_PRIV` 在公共路径拒绝。
- 成本预算耗尽拒绝。

**Step 2: 实现并提交**

```bash
go test ./internal/script -run 'Test(State|Public|Executor)' -v
git add internal/script/state.go internal/script/public_private.go internal/script/executor.go internal/script/cost.go internal/script/state_test.go internal/script/public_private_test.go internal/script/executor_test.go
git commit -m "feat: add script execution state"
```

## Task 5: 值、栈、交互和结果指令最小集

**Files:**
- Create: `internal/script/instr_value.go`
- Create: `internal/script/instr_stack.go`
- Create: `internal/script/instr_interaction.go`
- Create: `internal/script/instr_result.go`
- Create: `internal/script/instr_value_test.go`
- Create: `internal/script/instr_stack_test.go`
- Create: `internal/script/instr_interaction_test.go`
- Create: `internal/script/instr_result_test.go`

**Step 1: 写失败测试**

测试：

- `NIL`、`TRUE`、`FALSE`、`DATA`、`STRING` 压栈。
- `NOP`、`PUSH`、`POP`、`TOP`、`PEEK` 基本行为。
- `INPUT`、`OUTPUT` 缓冲行为。
- `PASS`、`CHECK`、`EXIT`、`RETURN`、`END` 状态转换。
- 所有失败路径包括栈下溢、参数不足、公共模式限制。

**Step 2: 实现并提交**

```bash
go test ./internal/script -run 'TestInstr(Value|Stack|Interaction|Result)' -v
git add internal/script/instr_value.go internal/script/instr_stack.go internal/script/instr_interaction.go internal/script/instr_result.go internal/script/instr_value_test.go internal/script/instr_stack_test.go internal/script/instr_interaction_test.go internal/script/instr_result_test.go
git commit -m "feat: add core script instructions"
```

## Task 6: 环境、系统和函数指令接口

**Files:**
- Create: `internal/script/env.go`
- Create: `internal/script/instr_environment.go`
- Create: `internal/script/instr_system.go`
- Create: `internal/script/instr_function.go`
- Create: `internal/script/env_test.go`
- Create: `internal/script/instr_environment_test.go`
- Create: `internal/script/instr_system_test.go`
- Create: `internal/script/instr_function_test.go`

**Step 1: 写失败测试**

测试：

- 环境字段注册表包含字段名、类型、确定性、可用域、成本、错误规则。
- `SIGNED` 通过注入 verifier 验证，不直接依赖交易包实现。
- `SYS_CHKPASS` 通过注入接口查询，不直接依赖状态包。
- Hash 函数指令复用 `pkg/crypto`。
- `SYS_NULL` 可用于 unlock script。

**Step 2: 实现**

保持接口化：

```go
type Environment interface {
    Lookup(name string) (Value, error)
}

type SignatureChecker interface {
    Check(message, signature, publicKey []byte) error
}
```

**Step 3: 验证并提交**

```bash
go test ./internal/script -run 'Test(Env|InstrEnvironment|InstrSystem|InstrFunction)' -v
git add internal/script/env.go internal/script/instr_environment.go internal/script/instr_system.go internal/script/instr_function.go internal/script/env_test.go internal/script/instr_environment_test.go internal/script/instr_system_test.go internal/script/instr_function_test.go
git commit -m "feat: add script environment interfaces"
```

## Task 7: 集合、转换、比较、逻辑指令

**Files:**
- Create: `internal/script/instr_collection.go`
- Create: `internal/script/instr_conversion.go`
- Create: `internal/script/instr_comparison.go`
- Create: `internal/script/instr_logic.go`
- Create: `internal/script/instr_collection_test.go`
- Create: `internal/script/instr_conversion_test.go`
- Create: `internal/script/instr_comparison_test.go`
- Create: `internal/script/instr_logic_test.go`

**Step 1: 写失败测试**

测试：

- Dict 保留插入顺序，不能用 Go map 遍历顺序输出。
- 隐式跨类比较默认拒绝。
- `EVERY`、`SOME` 对空集合语义明确测试。
- 转换必须按源类型到目标类型规则表执行。

**Step 2: 实现并提交**

```bash
go test ./internal/script -run 'TestInstr(Collection|Conversion|Comparison|Logic)' -v
git add internal/script/instr_collection.go internal/script/instr_conversion.go internal/script/instr_comparison.go internal/script/instr_logic.go internal/script/instr_collection_test.go internal/script/instr_conversion_test.go internal/script/instr_comparison_test.go internal/script/instr_logic_test.go
git commit -m "feat: add deterministic script instructions"
```

## Task 8: 危险、复杂和扩展指令默认拒绝

**Files:**
- Create: `internal/script/instr_capture.go`
- Create: `internal/script/instr_flow.go`
- Create: `internal/script/instr_arithmetic.go`
- Create: `internal/script/instr_pattern.go`
- Create: `internal/script/instr_tool.go`
- Create: `internal/script/instr_module.go`
- Create: `internal/script/instr_extension.go`
- Create: `internal/script/instr_capture_test.go`
- Create: `internal/script/instr_flow_test.go`
- Create: `internal/script/instr_arithmetic_test.go`
- Create: `internal/script/instr_pattern_test.go`
- Create: `internal/script/instr_tool_test.go`
- Create: `internal/script/instr_module_test.go`
- Create: `internal/script/instr_extension_test.go`

**Step 1: 写失败测试**

测试：

- `SHELL` 公共路径拒绝。
- `RANDOM/SLRAND` 未实现确定性 PRNG 前拒绝公共执行。
- `EXT_MO`、`EXT_PRIV` 默认禁用。
- 模块指令只能通过白名单方法表调用。
- 模式匹配必须有最大步数或预算。
- `GOTO` 顶层跳转和深度限制生效。
- `EMBED <= 4` 生效。

**Step 2: 实现拒绝和安全骨架**

实现解析、元数据和拒绝路径。只有规则完全明确的简单指令可执行。

**Step 3: 验证并提交**

```bash
go test ./internal/script -run 'TestInstr(Capture|Flow|Arithmetic|Pattern|Tool|Module|Extension)' -v
git add internal/script/instr_capture.go internal/script/instr_flow.go internal/script/instr_arithmetic.go internal/script/instr_pattern.go internal/script/instr_tool.go internal/script/instr_module.go internal/script/instr_extension.go internal/script/instr_capture_test.go internal/script/instr_flow_test.go internal/script/instr_arithmetic_test.go internal/script/instr_pattern_test.go internal/script/instr_tool_test.go internal/script/instr_module_test.go internal/script/instr_extension_test.go
git commit -m "feat: guard advanced script instructions"
```

## 阶段验收

运行：

```bash
go fmt ./...
go test ./internal/script
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- unlock script opcode 限制生效。
- 公共/私有模式差异被测试覆盖。
- VM 资源限制被测试覆盖。
- 未决或危险指令默认拒绝，而不是静默执行。
- `internal/script` 不 import `internal/consensus`。

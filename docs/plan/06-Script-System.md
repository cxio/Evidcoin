# Script System Implementation Plan

**Goal:** 实现 Evidcoin 栈式脚本 VM 的 bytecode、运行时空间、资源限制、公共/私有模式、安全状态机和分批指令注册表。

**Architecture:** `internal/script/` 是 Layer 3，依赖基础类型和上层注入的交易/状态环境接口，但交易、状态、共识包不得依赖脚本包的具体实现。先实现 VM 核心和元数据注册，再按指令组逐批实现；危险或未决指令默认解析但拒绝执行。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、显式 instruction registry、表驱动测试、fuzz 测试候选。

---

## 来源提案

- `docs/proposal/10.Script-System.md`
- `docs/proposal/Instruction/`（逐条指令规格基线，冻结）
- DEC-0501：字节码编码（opcode/附参/关联数据、参数消费顺序、源码字面量歧义、`CALL` 公共验证分析）。
- DEC-0502：浮点 profile（binary64、异常浮点保留、比较与转换语义）。
- DEC-0503：注册表与环境边界（子空间独立编号、公共验证边界、环境命名 `BlockTime`/`TxTime`）。
- DEC-0504：成本模型（三层预算 `MaxInputScriptCost`/`MaxTxScriptCost`/`MaxBlockScriptCost`、外部引用计数归属调用方；具体数值待 C-6）。
- DEC-0505：执行状态机、通关状态语义、禁用指令清单（`SCRIPT`/`VALUE`/`EVAL`/`INOUT`）、执行边界（具体失败枚举待 C-7 细化）。
- 资源上限（proposal §13 / 第 03 章）：`MaxStackHeight=255`（<256）、`MaxStackItem=4095`（<4KB）、`MaxLockScript=8191`（<8KB）、`MaxUnlockScript=8191`（<8KB）。
- 签名读取边界（proposal §7 + DEC-0102/0103）：标准验证签名从见证环境读取；`FN_CHECKSIG`/`FN_MCHECKSIG` 定制验证可从脚本数据读取；解锁脚本由交易输入根承诺。

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
- opcode `254-255` 未分配（255 为基础指令集理论上限），默认拒绝。
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
- `MaxLockScript` 边界：8191 bytes 合法，8192 bytes 拒绝（识别/锁定脚本，<8KB）。
- `MaxUnlockScript` 边界：8191 bytes 合法，8192 bytes 拒绝（<8KB；不含标准内置见证，定制验证签名若入解锁脚本则计入）。
- unlock script 只允许 opcode `0-50` 和 `SYS_NULL`（opcode 169）；`SYS_NULL` 合法，其他系统指令如 `SYS_TIME` 非法。

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
- `MaxStackHeight` 边界：255 个栈元素合法，256 个拒绝（<256）。
- `MaxStackItem` 边界：4095 bytes 合法，4096 bytes 拒绝（<4KB）。
- 实参区 FIFO。
- Float 字面量不得表达 NaN/+Inf/-Inf（输入即拒绝）；运算产生的 NaN/Inf 保留为异常值继续执行，由 `ISEFV` 检测。
- `-0.0` 数值比较等于 `+0.0`，但字节编码保持原 bit pattern。

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

- 执行状态（DEC-0505）：`Running`、`PassStop`、`VerifyFail`、`ScriptError`、`CostFail`、`PrivateStop`。
- 初始通关状态为 `true`；空脚本以 `true` 产生 `PassStop`（通过）。
- `END` 与公共验证路径中的无数据 `INPUT` 以当前通关状态产生 `PassStop`。
- `PASS false` 立即 `VerifyFail`；`PASS true` 继续；`CHECK true/false` 写入后继续，后写覆盖前值。
- `CHECK(true)` 后 `CHECK(false)` 最终 `PassStop(false)`（不合法）；`CHECK(false)` 后 `CHECK(true)` 最终通过。
- `SYS_TIME`、`EXT_PRIV` 在公共验证路径触达即 `ScriptError`；`GOTO`/`EMBED` 目标缺失/不可验证即 `ScriptError`。
- `SHELL` 在公共路径不执行本地程序，但正常消费实参、做栈/类型检查、计入公共成本（不拒绝）。
- 公共验证触达禁用指令 `SCRIPT`/`INOUT` 立即 `ScriptError`。
- 成本预算耗尽产生 `CostFail`。
- 公共 `END`/`INPUT` 后的私有路径不执行、不计成本、不因其中禁用指令拒绝。

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
- `CHECK` 按 `passState = bool(x)` 覆盖，不做防覆盖保护。
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
- `SYS_CHKPASS` 签名只能来自环境中的 Witness 数据；普通数据栈、实参区或 UnlockScript 字节流中的签名字节必须不被 `SYS_CHKPASS` 读取为签名来源。
- 缺失对应 Witness 时，执行 `SYS_CHKPASS` 必须失败；未执行 `SYS_CHKPASS` 的脚本允许 Witness 为空。
- `FN_CHECKSIG` / `FN_MCHECKSIG` 可从普通数据栈或实参区读取签名参数，覆盖定制验证路径。
- 脚本 VM 不重新计算 `TxID`；UnlockScript 是否参与输入 Hash 由 `internal/tx` 的规范编码保证。
- Hash 函数指令复用 `pkg/crypto`。
- `SYS_NULL` 可用于 unlock script，且不执行计算、不访问状态、不影响栈。
- 其他系统指令如 `SYS_TIME` 用于 unlock script 必须拒绝。

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
- Float 比较：除 `ISEFV` 外任一操作数为 NaN 的比较返回 `false`；`EQUAL(+0.0,-0.0)` 为 true；排序类比较遇 NaN 导致脚本执行失败、验证不通过。
- 异常浮点由 `ISEFV` 检测（非 `ISNAN`）。
- Float 转换：`Float->Int` 默认向零截断；`BYTES`/`PACK` 对异常浮点输出 8 字节大端 bit pattern（转换后为 `Bytes`，不再触发最终异常残留检查）。

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

- 禁用指令 `SCRIPT`/`VALUE`/`EVAL`/`INOUT` 公共验证路径触达即 `ScriptError`（静态出现不拒绝）。
- `CALL`、`SHELL` 不属于禁用指令；`SHELL` 公共路径不执行本地程序但正常消费实参、计入成本。
- `RANDOM/SLRAND` 未实现确定性 PRNG 前拒绝公共执行。
- `EXT_PRIV` 公共验证路径 `ScriptError`；`EXT_MO` 模块扩展须经白名单方法表。
- 模块指令只能通过白名单方法表调用。
- 模式匹配必须有最大步数或预算。
- `GOTO` 跳转次数 `<= 2`、跳转深度 `<= 3` 生效。
- `EMBED` 嵌入次数 `<= 4`、嵌入深度 `== 0`（嵌入脚本不可再 `EMBED`/`GOTO`）生效。

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

- unlock script opcode 限制（`[0-50]` + `SYS_NULL`）生效。
- 公共/私有模式差异被测试覆盖。
- 资源上限 255/4095/8191/8191 被边界值测试覆盖。
- 执行状态机六态（DEC-0505）与通关状态覆盖语义被测试覆盖。
- 禁用指令（`SCRIPT`/`VALUE`/`EVAL`/`INOUT`）仅在公共执行路径触达时 `ScriptError`；`CALL`/`SHELL` 不禁用。
- 脚本 hash/成本基于字节码而非源码。
- `internal/script` 不 import `internal/consensus`、`internal/utxo`、`internal/utco`。

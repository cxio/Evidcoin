# Phase 5: Script Engine（脚本引擎）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标（Goal）：** 实现 Evidcoin 栈式脚本引擎，包括完整的指令集（254 条操作码）、执行环境、锁定/解锁脚本执行管道，以及安全资源约束。

**架构（Architecture）：** 脚本引擎位于 `internal/script/` 包，对应 AGENTS.md 中的 Layer 3（脚本层）。引擎依赖 `pkg/types/` 和 `pkg/crypto/`，被 Layer 4（共识层）和 Layer 2（状态层）使用。采用表驱动指令注册机制，每条指令为独立的执行函数，通过操作码索引映射。

**技术栈（Tech Stack）：** Go 1.25+, `golang.org/x/crypto`（SHA3）, `lukechampine.com/blake3`（BLAKE3）, `github.com/mr-tron/base58`（地址编码）, `github.com/cloudflare/circl`（ML-DSA-65，如标准库未内置）

---

## 参考文档

- `docs/proposal/4.Script-of-Stack.md` — 脚本系统总体设计
- `docs/proposal/Instruction/0.Base-Constraints.md` — 基础约束
- `docs/proposal/Instruction/1.Value-Instructions.md` ~ `18.Extension-Instructions.md` — 指令详细规格

---

## 包结构（Package Layout）

```
internal/script/
    engine.go          # ScriptEngine 结构体与执行入口
    env.go             # 执行环境层级（Domain）定义
    types.go           # 脚本类型系统（ScriptValue, ScriptType）
    opcode.go          # 操作码常量与指令注册表
    decode.go          # 指令字节码解码（附参/关联数据解析）
    execute.go         # 指令调度（主执行循环）
    limits.go          # 运行时资源约束检查
    pipeline.go        # 锁定/解锁执行管道
    errors.go          # 脚本执行错误类型
    instr/             # 各类指令实现（按提案分类）
        value.go       # [0-18]  值指令
        capture.go     # [19-23] 捕获指令
        stack.go       # [24-34] 栈操作
        collection.go  # [35-45] 集合操作
        interact.go    # [46-50] 交互指令
        result.go      # [51-57] 结果指令
        flow.go        # [58-66] 流程控制
        convert.go     # [67-79] 转换指令
        arith.go       # [80-103] 算术指令
        compare.go     # [104-111] 比较指令
        logic.go       # [112-115] 逻辑指令
        pattern.go     # [116-127] 模式指令
        environ.go     # [128-137] 环境指令
        tool.go        # [138-163] 工具指令
        system.go      # [164-169] 系统指令
        function.go    # [170-224] 函数指令
        module.go      # [225-250] 模块指令
        extension.go   # [251-253] 扩展指令
    instr/testdata/    # 测试用脚本字节码文件（可选）
internal/script/script_test.go  # 集成测试：完整执行管道
```

---

## 核心常量（来自提案与 AGENTS.md）

```go
const (
    MaxStackHeight   = 256
    MaxStackItem     = 1024   // bytes
    MaxLockScript    = 1024   // bytes
    MaxUnlockScript  = 4096   // bytes
    MaxLocalScope    = 128
    MaxGlobalVars    = 256
    MaxGotoDepth     = 3
    MaxGotoCallsMain = 2
    MaxEmbedCalls    = 4
    MaxEmbedDepth    = 0
    MaxTxSize        = 65535

    // Opcode 段边界
    OpcodeBaseEnd      = 170
    OpcodeFunctionEnd  = 225
    OpcodeModuleEnd    = 251
    OpcodeExtEnd       = 254
    OpcodeReserved254  = 254
    OpcodeReserved255  = 255

    // 解锁脚本允许的操作码范围
    // [0-50] 和 [169]
)
```

---

## 任务分解

---

### Task 1：包骨架与错误类型

**文件：**
- 创建：`internal/script/errors.go`
- 创建：`internal/script/limits.go`

**Step 1：编写测试（errors）**

```go
// internal/script/errors_test.go
package script_test

import (
    "errors"
    "testing"
    "github.com/yourorg/evidcoin/internal/script"
)

func TestScriptErrors(t *testing.T) {
    tests := []struct {
        name    string
        err     error
        wantIs  error
    }{
        {"stack overflow", script.ErrStackOverflow, script.ErrStackOverflow},
        {"invalid opcode", script.ErrInvalidOpcode, script.ErrInvalidOpcode},
        {"type mismatch", script.ErrTypeMismatch, script.ErrTypeMismatch},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            if !errors.Is(tc.err, tc.wantIs) {
                t.Errorf("errors.Is(%v, %v) = false", tc.err, tc.wantIs)
            }
        })
    }
}
```

**Step 2：运行验证失败**

```bash
go test ./internal/script/... -run TestScriptErrors -v
# Expected: compile error（包不存在）
```

**Step 3：实现 errors.go**

```go
// internal/script/errors.go

// Package script 实现 Evidcoin 栈式脚本引擎。
package script

import "errors"

// 脚本执行错误类型
var (
    // ErrStackOverflow 数据栈超出最大高度 256
    ErrStackOverflow = errors.New("script: stack overflow")
    // ErrStackUnderflow 从空栈消费
    ErrStackUnderflow = errors.New("script: stack underflow")
    // ErrStackItemTooLarge 栈项超过 1024 字节
    ErrStackItemTooLarge = errors.New("script: stack item too large")
    // ErrInvalidOpcode 操作码不在合法范围内（0-253）
    ErrInvalidOpcode = errors.New("script: invalid opcode")
    // ErrTypeMismatch 操作数类型不符合指令要求
    ErrTypeMismatch = errors.New("script: type mismatch")
    // ErrDivisionByZero 除零错误
    ErrDivisionByZero = errors.New("script: division by zero")
    // ErrScriptTooLong 脚本字节数超过上限
    ErrScriptTooLong = errors.New("script: script too long")
    // ErrLocalScopeFull 局部域已达 128 条目上限
    ErrLocalScopeFull = errors.New("script: local scope full")
    // ErrIndexOutOfRange 索引越界
    ErrIndexOutOfRange = errors.New("script: index out of range")
    // ErrGotoDepthExceeded GOTO 深度超过 3
    ErrGotoDepthExceeded = errors.New("script: goto depth exceeded")
    // ErrEmbedDepthExceeded EMBED 深度不为 0（嵌入脚本内再次嵌入）
    ErrEmbedDepthExceeded = errors.New("script: embed depth exceeded")
    // ErrUnlockOpcodeNotAllowed 解锁脚本使用了禁止的操作码
    ErrUnlockOpcodeNotAllowed = errors.New("script: opcode not allowed in unlock script")
    // ErrInvalidArgCount 实参数量不匹配
    ErrInvalidArgCount = errors.New("script: invalid argument count")
    // ErrPassFailed PASS 检查失败（条件为 false）
    ErrPassFailed = errors.New("script: pass check failed")
    // ErrCodeInjection 试图将 Bytes 转换为 Script（安全禁止）
    ErrCodeInjection = errors.New("script: bytes-to-script conversion not allowed")
    // ErrEvalIsolation EVAL 隔离执行失败
    ErrEvalIsolation = errors.New("script: eval isolation error")
)
```

**Step 4：实现 limits.go**

```go
// internal/script/limits.go

package script

// 运行时资源约束常量，来源于 docs/proposal/4.Script-of-Stack.md §7.1
const (
    // MaxStackHeight 数据栈最大高度
    MaxStackHeight = 256
    // MaxStackItemSize 单个栈项的最大字节数
    MaxStackItemSize = 1024
    // MaxLockScriptSize 锁定脚本最大字节数
    MaxLockScriptSize = 1024
    // MaxUnlockScriptSize 解锁脚本最大字节数（不含标准签名数据）
    MaxUnlockScriptSize = 4096
    // MaxLocalScopeSize 每块局部域最大条目数
    MaxLocalScopeSize = 128
    // MaxGlobalVarSlots 全局变量槽数量
    MaxGlobalVarSlots = 256
    // MaxGotoDepth GOTO 最大深度
    MaxGotoDepth = 3
    // MaxGotoCallsMainScript 主脚本最多 GOTO 调用次数
    MaxGotoCallsMainScript = 2
    // MaxEmbedCalls 运行时 EMBED 总次数（含子脚本）
    MaxEmbedCalls = 4
    // MaxTxSize 交易最大字节数（不含解锁部分）
    MaxTxSize = 65535
)

// isUnlockAllowed 判断操作码是否允许在解锁脚本中使用
// 仅允许 [0-50] 和 [169]（SYS_NULL）
func isUnlockAllowed(opcode byte) bool {
    return opcode <= 50 || opcode == 169
}
```

**Step 5：运行验证通过**

```bash
go test ./internal/script/... -run TestScriptErrors -v
# Expected: PASS
go build ./internal/script/...
```

**Step 6：提交**

```bash
git add internal/script/errors.go internal/script/limits.go internal/script/errors_test.go
git commit -m "feat(script): add error types and runtime limits constants"
```

---

### Task 2：脚本类型系统

**文件：**
- 创建：`internal/script/types.go`
- 创建：`internal/script/types_test.go`

**Step 1：编写测试**

```go
// internal/script/types_test.go
package script_test

import (
    "math/big"
    "testing"
    "github.com/yourorg/evidcoin/internal/script"
)

func TestScriptTypeID(t *testing.T) {
    tests := []struct {
        value    any
        wantType script.TypeID
    }{
        {nil, script.TypeNil},
        {true, script.TypeBool},
        {false, script.TypeBool},
        {uint8(1), script.TypeByte},
        {int32(65), script.TypeRune},
        {int64(100), script.TypeInt},
        {new(big.Int), script.TypeBigInt},
        {float64(3.14), script.TypeFloat},
        {"hello", script.TypeString},
        {[]byte{1, 2}, script.TypeBytes},
        {[]any{1, 2}, script.TypeSlice},
        {map[string]any{}, script.TypeDict},
    }
    for _, tc := range tests {
        got := script.TypeOf(tc.value)
        if got != tc.wantType {
            t.Errorf("TypeOf(%T) = %v, want %v", tc.value, got, tc.wantType)
        }
    }
}

func TestTypePromotion(t *testing.T) {
    // Byte → Rune → Int → BigInt 提升链
    result := script.PromoteValueSet([]any{uint8(1), int32(200)})
    for _, v := range result {
        if _, ok := v.(int32); !ok {
            t.Errorf("expected Rune after promotion, got %T", v)
        }
    }
}
```

**Step 2：运行验证失败**

```bash
go test ./internal/script/... -run TestScriptType -v
# Expected: compile error
```

**Step 3：实现 types.go**

```go
// internal/script/types.go

package script

import (
    "math/big"
    "regexp"
    "time"
)

// TypeID 脚本运行时类型标识符
// 对应 docs/proposal/4.Script-of-Stack.md §8.1
type TypeID uint8

const (
    TypeNil    TypeID = 0
    TypeBool   TypeID = 1
    TypeByte   TypeID = 2  // uint8
    TypeRune   TypeID = 3  // int32 (Unicode code point)
    TypeInt    TypeID = 4  // int64
    TypeBigInt TypeID = 5  // *big.Int
    TypeFloat  TypeID = 6  // float64
    TypeString TypeID = 7
    TypeBytes  TypeID = 8  // []byte
    TypeRegExp TypeID = 9  // *regexp.Regexp
    TypeTime   TypeID = 10 // time.Time
    TypeScript TypeID = 11 // []byte（特殊标记，不可从 Bytes 转换）
    TypeSlice  TypeID = 12 // []any
    TypeDict   TypeID = 13 // map[string]any
)

// ScriptValue 运行时脚本值（带类型标记）
// Script 类型与 Bytes 使用相同底层类型，通过 TypeID 区分
type ScriptValue struct {
    T TypeID
    V any
}

// TypeOf 返回 Go 原生值对应的 TypeID
func TypeOf(v any) TypeID {
    switch v.(type) {
    case nil:
        return TypeNil
    case bool:
        return TypeBool
    case uint8:
        return TypeByte
    case int32:
        return TypeRune
    case int64:
        return TypeInt
    case *big.Int:
        return TypeBigInt
    case float64:
        return TypeFloat
    case string:
        return TypeString
    case []byte:
        return TypeBytes
    case *regexp.Regexp:
        return TypeRegExp
    case time.Time:
        return TypeTime
    case []any:
        return TypeSlice
    case map[string]any:
        return TypeDict
    default:
        return TypeNil
    }
}

// typePromotionOrder 类型提升链顺序（数值类型）
// Byte → Rune → Int → BigInt
// Byte → Rune → Int → Float
var typePromotionOrder = map[TypeID]int{
    TypeByte:  0,
    TypeRune:  1,
    TypeInt:   2,
    TypeBigInt: 3,
    TypeFloat: 3, // 与 BigInt 同级，Float 为终止类型
}

// PromoteValueSet 对值集合 {v1, v2, ...} 进行类型提升
// 将所有成员提升为集合中最高类型
// 来源：docs/proposal/Instruction/1.Value-Instructions.md §ValueSet
func PromoteValueSet(vals []any) []any {
    if len(vals) == 0 {
        return vals
    }
    // 找出最高优先级类型
    maxType := TypeByte
    for _, v := range vals {
        t := TypeOf(v)
        if order, ok := typePromotionOrder[t]; ok {
            if maxOrder, ok2 := typePromotionOrder[maxType]; ok2 {
                if order > maxOrder {
                    maxType = t
                }
            }
        }
    }
    result := make([]any, len(vals))
    for i, v := range vals {
        result[i] = promoteValue(v, maxType)
    }
    return result
}

// promoteValue 将单个值提升为目标类型
func promoteValue(v any, target TypeID) any {
    switch target {
    case TypeRune:
        switch x := v.(type) {
        case uint8:
            return int32(x)
        }
    case TypeInt:
        switch x := v.(type) {
        case uint8:
            return int64(x)
        case int32:
            return int64(x)
        }
    case TypeBigInt:
        switch x := v.(type) {
        case uint8:
            return big.NewInt(int64(x))
        case int32:
            return big.NewInt(int64(x))
        case int64:
            return big.NewInt(x)
        }
    case TypeFloat:
        switch x := v.(type) {
        case uint8:
            return float64(x)
        case int32:
            return float64(x)
        case int64:
            return float64(x)
        }
    }
    return v
}
```

**Step 4：运行验证通过**

```bash
go test ./internal/script/... -run TestScriptType -v
```

**Step 5：提交**

```bash
git add internal/script/types.go internal/script/types_test.go
git commit -m "feat(script): add script type system with TypeID and type promotion"
```

---

### Task 3：操作码常量与指令结构

**文件：**
- 创建：`internal/script/opcode.go`
- 创建：`internal/script/opcode_test.go`

**Step 1：编写测试**

```go
// internal/script/opcode_test.go
package script_test

import (
    "testing"
    "github.com/yourorg/evidcoin/internal/script"
)

func TestOpcodeSegments(t *testing.T) {
    // 验证各段边界
    if script.OpNIL != 0 { t.Errorf("OpNIL should be 0") }
    if script.OpSYS_NULL != 169 { t.Errorf("OpSYS_NULL should be 169") }
    if script.OpFN_BASE58 != 170 { t.Errorf("OpFN_BASE58 should be 170") }
    if script.OpFN_X != 224 { t.Errorf("OpFN_X should be 224") }
    if script.OpMO_MATH != 225 { t.Errorf("OpMO_MATH should be 225") }
    if script.OpMO_XX != 250 { t.Errorf("OpMO_XX should be 250") }
    if script.OpEXT_MO != 251 { t.Errorf("OpEXT_MO should be 251") }
    if script.OpEXT_PRIV != 253 { t.Errorf("OpEXT_PRIV should be 253") }
}

func TestIsUnlockAllowed(t *testing.T) {
    // [0-50] 允许
    for i := 0; i <= 50; i++ {
        if !script.IsUnlockAllowed(byte(i)) {
            t.Errorf("opcode %d should be allowed in unlock script", i)
        }
    }
    // 51 不允许（PASS）
    if script.IsUnlockAllowed(51) {
        t.Errorf("opcode 51 (PASS) should NOT be allowed in unlock script")
    }
    // 169 允许（SYS_NULL）
    if !script.IsUnlockAllowed(169) {
        t.Errorf("opcode 169 (SYS_NULL) should be allowed in unlock script")
    }
}
```

**Step 2：实现 opcode.go（操作码常量，完整列表）**

```go
// internal/script/opcode.go

package script

// 操作码常量，对应 docs/proposal/4.Script-of-Stack.md §3 及各 Instruction 提案
// 分为四段：Base [0-169], Function [170-224], Module [225-250], Extension [251-253]

// ─── 值指令 [0-18] ──────────────────────────────────────────────────────────
const (
    OpNIL     byte = 0  // NIL：压入 nil
    OpTRUE    byte = 1  // TRUE：压入 true
    OpFALSE   byte = 2  // FALSE：压入 false
    OpBYTE    byte = 3  // Byte 字面量（关联数据：1 byte）
    OpRUNE    byte = 4  // Rune 字面量（关联数据：4 bytes）
    OpINT     byte = 5  // Int 字面量（关联数据：varint）
    OpBIGINT  byte = 6  // BigInt 字面量（附参：1字节长度；关联数据：大整数字节）
    OpFLOAT   byte = 7  // Float 字面量（关联数据：8 bytes IEEE754）
    OpSTRING  byte = 8  // String 字面量（附参：varint 长度；关联数据：UTF-8）
    OpVALUESET byte = 9 // 值集合 {} （附参：varint 总长度；关联数据：值指令序列）
    OpDATA    byte = 10 // DATA{}：原始字节（附参：varint 长度）
    OpREGEXP  byte = 11 // /.../：正则表达式（附参：1字节长度；关联数据：RE2 文本）
    OpDATE    byte = 12 // DATE{}：时间字面量（关联数据：signed varint Unix ms）
    OpDICT    byte = 13 // DICT：创建空字典（运行时消耗 keys+values）
    // 14-15 保留
    OpCODE    byte = 16 // CODE{}：惰性脚本（附参：varint 长度；不执行）
    OpSCRIPT  byte = 17 // SCRIPT：从链上加载外部脚本（附参：varint+48B+varint）
    OpVALUE   byte = 18 // VALUE：按索引读取（附参：1 byte 索引）
)

// ─── 捕获指令 [19-23] ────────────────────────────────────────────────────────
const (
    OpARG_CAPTURE byte = 19 // @：捕获后续指令返回值 → argument space
    OpLOCAL_STORE byte = 20 // $：捕获后续指令返回值 → local scope
    OpLOCAL_READ  byte = 21 // $()：从 local scope 读取（附参：1 byte signed 索引）
    OpLOOP_VAR    byte = 22 // $X()：循环变量引用（附参：1 byte 变量标识）
    OpSTACK_DIRECT byte = 23 // ~：强制后续指令从 stack 取参，绕过 argument space
)

// ─── 栈操作 [24-34] ─────────────────────────────────────────────────────────
const (
    OpNOP   byte = 24 // NOP：消费 argument space 并丢弃
    OpPUSH  byte = 25 // PUSH：将 argument space 内容压入 stack
    OpPOP   byte = 26 // POP：弹出栈顶
    OpPOPS  byte = 27 // POPS(1)：弹出 N 项（附参：1 byte 数量）
    OpTOP   byte = 28 // TOP：复制栈顶（不移除）
    OpTOPS  byte = 29 // TOPS(1)：复制栈顶 N 项
    OpPEEK  byte = 30 // PEEK：按位置引用（argument 提供位置）
    OpPEEKS byte = 31 // PEEKS(1)：按起始位置引用 N 项
    OpSHIFT byte = 32 // SHIFT(1)：弹出 N 项为 slice
    OpCLONE byte = 33 // CLONE(1)：复制 N 项为 slice（不移除）
    OpVIEW  byte = 34 // VIEW(1)：按起始位置引用 N 项为 slice（不移除）
)

// ─── 集合操作 [35-45] ────────────────────────────────────────────────────────
const (
    OpSLICE   byte = 35 // SLICE(1)：子切片
    OpREVERSE byte = 36 // REVERSE：反转 slice
    OpMERGE   byte = 37 // MERGE：合并多个 slice
    OpEXTEND  byte = 38 // EXTEND：追加成员到 slice
    OpPACK    byte = 39 // PACK：slice → Bytes（紧凑编码）
    OpSPREAD  byte = 40 // SPREAD：slice → auto-expand 多个值
    OpINDEX   byte = 41 // INDEX：按整数索引取成员
    OpITEM    byte = 42 // ITEM：按 key 取 dict 成员
    OpSET     byte = 43 // SET：设置 dict 成员
    OpCALL    byte = 44 // CALL(1)：调用模块方法（附参：1 byte 实参数）
    OpSIZE    byte = 45 // SIZE：集合大小
)

// ─── 交互指令 [46-50] ────────────────────────────────────────────────────────
const (
    OpINPUT   byte = 46 // INPUT(1)：从 INPUT 缓存区导入
    OpOUTPUT  byte = 47 // OUTPUT：导出到 OUTPUT 缓存区
    OpBUFDUMP byte = 48 // BUFDUMP(1)：刷出 OUTPUT 缓存区（附参：1 byte tag）
    // 49 保留
    OpPRINT byte = 50 // PRINT：调试输出
)

// ─── 结果指令 [51-57] ────────────────────────────────────────────────────────
const (
    OpPASS   byte = 51 // PASS：通关检查（失败立即终止）
    OpCHECK  byte = 52 // CHECK：通关检查（失败不终止，可被后续 CHECK 覆盖）
    OpGOTO   byte = 53 // GOTO(~,48,~)：独立跳转到外部脚本
    OpEMBED  byte = 54 // EMBED(~,48,~)：合并嵌入外部脚本
    OpEXIT   byte = 55 // EXIT：立即终止脚本
    OpRETURN byte = 56 // RETURN：从 MAP{}/FILTER{} 返回值
    OpEND    byte = 57 // END：公共验证结束标记
)

// ─── 流程控制 [58-66] ────────────────────────────────────────────────────────
const (
    OpIF       byte = 58 // IF{}(~)：条件分支
    OpELSE     byte = 59 // ELSE{}(~)：条件否分支
    OpSWITCH   byte = 60 // SWITCH{}(~)：多路分支
    OpCASE     byte = 61 // CASE{}(~)：分支子句
    OpDEFAULT  byte = 62 // DEFAULT{}(~)：默认子句
    OpEACH     byte = 63 // EACH{}(~)：有限迭代
    OpCONTINUE byte = 64 // CONTINUE：跳过当前迭代
    OpBREAK    byte = 65 // BREAK：退出循环/SWITCH
    OpBLOCK    byte = 66 // BLOCK{}(~)：块域
)

// ─── 转换指令 [67-79] ────────────────────────────────────────────────────────
const (
    OpToBOOL   byte = 67 // BOOL：转 bool
    OpToBYTE   byte = 68 // BYTE：转 uint8
    OpToRUNE   byte = 69 // RUNE：转 int32
    OpToINT    byte = 70 // INT：转 int64
    OpToBIGINT byte = 71 // BIGINT：转 *big.Int
    OpToFLOAT  byte = 72 // FLOAT：转 float64
    OpToSTRING byte = 73 // STRING(1){}：转 string（附参：进制/格式）
    OpToBYTES  byte = 74 // BYTES：转 []byte
    OpToRUNES  byte = 75 // RUNES：转 []rune
    OpToTIME   byte = 76 // TIME：转 time.Time
    OpToREGEXP byte = 77 // REGEXP：转 *regexp.Regexp
    // 78 保留
    OpANYS byte = 79 // ANYS(1){}：slice 类型转换
)

// ─── 算术指令 [80-103] ───────────────────────────────────────────────────────
const (
    OpEXPR   byte = 80 // ()：表达式容器（附参：1 byte 长度）
    OpMULF   byte = 81 // * ：乘（仅在表达式内）
    OpDIVF   byte = 82 // / ：除（仅在表达式内）
    OpMODF   byte = 83 // % ：取模（仅在表达式内）
    OpADDF   byte = 84 // + ：加/一元正（仅在表达式内）
    OpSUBF   byte = 85 // - ：减/一元负（仅在表达式内）
    OpMUL    byte = 86 // MUL：乘（Int/Float）
    OpDIV    byte = 87 // DIV：除（Int/Float）
    OpADD    byte = 88 // ADD：加（Int/Float/String/Bytes/Dict）
    OpSUB    byte = 89 // SUB：减（Int/Float）
    OpMOD    byte = 90 // MOD：取模（Int/Float）
    OpPOW    byte = 91 // POW：幂（Int/Float）
    OpLMOV   byte = 92 // LMOV：左移（Int）
    OpRMOV   byte = 93 // RMOV：右移（Int）
    OpAND    byte = 94 // AND：位与（Int）
    OpANDX   byte = 95 // ANDX：位清除 x&^y（Int）
    OpOR     byte = 96 // OR：位或（Int）
    OpXOR    byte = 97 // XOR：位异或（Int）
    OpNEG    byte = 98 // NEG：取负（Int/Float）
    OpNOT    byte = 99 // NOT：逻辑非（Bool）
    OpDIVMOD byte = 100 // DIVMOD：同时返回商和余数
    OpREP    byte = 101 // REP(1)：复制 N 份
    OpDEL    byte = 102 // DEL：删除 dict 键
    OpCLEAR  byte = 103 // CLEAR：清空 dict
)

// ─── 比较指令 [104-111] ──────────────────────────────────────────────────────
const (
    OpEQUAL  byte = 104 // EQUAL：等于
    OpNEQUAL byte = 105 // NEQUAL：不等于
    OpLT     byte = 106 // LT：小于
    OpLTE    byte = 107 // LTE：小于等于
    OpGT     byte = 108 // GT：大于
    OpGTE    byte = 109 // GTE：大于等于
    OpISNAN  byte = 110 // ISNAN：是否 NaN（Float）
    OpWITHIN byte = 111 // WITHIN：范围检查 [min, max)
)

// ─── 逻辑指令 [112-115] ──────────────────────────────────────────────────────
const (
    OpBOTH   byte = 112 // BOTH：逻辑与（二元）
    OpEITHER byte = 113 // EITHER：逻辑或（二元）
    OpEVERY  byte = 114 // EVERY：全称与（[]Bool）
    OpSOME   byte = 115 // SOME(1)：阈值或（[]Bool，附参：最小 true 数）
)

// ─── 模式指令 [116-127] ──────────────────────────────────────────────────────
const (
    OpMODEL      byte = 116 // MODEL{}(2)：模式匹配块
    OpWILD1      byte = 117 // _：单指令通配
    OpWILDN      byte = 118 // _(~)：N 指令通配
    OpOPTIONAL   byte = 119 // ?{}(~)：可选序列
    OpWILD_IND   byte = 120 // ^?：通配指示符
    OpEXTRACT    byte = 121 // #(1)：提取指令
    OpTYPE_MATCH byte = 122 // !?Type：类型匹配
    OpINT_RANGE  byte = 123 // >{}(~,~)：整数范围匹配
    OpFLT_RANGE  byte = 124 // >{}(8,8,4)：浮点范围匹配
    OpRE_SEARCH  byte = 125 // RE{}(1,1)：正则搜索
    OpRE_EXTRACT byte = 126 // &(1)：正则提取
    OpWILD_ANY   byte = 127 // ...：任意长度通配（非贪心）
)

// ─── 环境指令 [128-137] ──────────────────────────────────────────────────────
const (
    OpENV    byte = 128 // ENV(1){}：读取环境变量
    OpIN     byte = 129 // IN(1){}：读取输入条目
    OpOUT    byte = 130 // OUT(~,1){}：读取当前交易输出条目
    OpINOUT  byte = 131 // INOUT(~,1){}：读取源交易同级输出
    OpXFROM  byte = 132 // XFROM(1){}：读取来源脚本信息
    OpVAR    byte = 133 // VAR(1)：读取全局变量
    OpSETVAR byte = 134 // SETVAR(1)：写入全局变量
    OpSOURCE byte = 135 // SOURCE(1)：提取脚本字节序列
    OpSIGNED byte = 136 // SIGNED(1)：读取签名验证结果
    // 137 保留
)

// ─── 工具指令 [138-163] ──────────────────────────────────────────────────────
const (
    OpEVAL    byte = 138 // EVAL：隔离执行 Script 值
    OpCOPY    byte = 139 // COPY：浅拷贝
    OpDCOPY   byte = 140 // DCOPY：深拷贝
    OpKEYVAL  byte = 141 // KEYVAL(1)：提取 dict 键值
    OpMATCH   byte = 142 // MATCH(1)：正则匹配
    OpSUBSTR  byte = 143 // SUBSTR(~)：子字符串
    OpREPLACE byte = 144 // REPLACE(1)：字符串替换
    OpRANDOM  byte = 145 // RANDOM：确定性随机数（ChaCha8）
    OpSLRAND  byte = 146 // SLRAND：确定性随机重排
    OpCMPFLO  byte = 147 // CMPFLO(1)：带 epsilon 的浮点比较
    // 148-151 未使用
    OpRANGE  byte = 152 // RANGE(1)：生成数值序列
    OpMAP    byte = 153 // MAP{}(~)：映射迭代
    OpFILTER byte = 154 // FILTER{}(~)：过滤迭代
    OpSHELL  byte = 155 // SHELL{}(~)：Shell 执行（仅私有节点）
    // 156-163 保留给量子安全密码
)

// ─── 系统指令 [164-169] ──────────────────────────────────────────────────────
const (
    OpSYS_TIME    byte = 164 // SYS_TIME(1){}：读取节点时间（END 后才有效）
    OpSYS_AWARD   byte = 165 // SYS_AWARD(1,~)：铸造奖励验证（仅 Coinbase）
    OpSYS_CHKPASS byte = 166 // SYS_CHKPASS：系统内置签名验证（门指令）
    // 167-168 保留
    OpSYS_NULL byte = 169 // SYS_NULL：空操作标记（SOURCE 分段边界）
)

// ─── 函数指令 [170-224] ──────────────────────────────────────────────────────
const (
    OpFN_BASE58    byte = 170 // FN_BASE58：Base58 编解码
    OpFN_BASE32    byte = 171 // FN_BASE32：Base32 编解码
    OpFN_BASE64    byte = 172 // FN_BASE64：Base64 编解码
    OpFN_ADDRESS   byte = 173 // FN_ADDRESS：地址编解码
    OpFN_PUBHASH   byte = 174 // FN_PUBHASH：公钥哈希
    OpFN_MPUBHASH  byte = 175 // FN_MPUBHASH：多签公钥哈希
    OpFN_CHECKSIG  byte = 176 // FN_CHECKSIG：单签验证
    OpFN_MCHECKSIG byte = 177 // FN_MCHECKSIG：多签验证
    OpFN_HASH224   byte = 178 // FN_HASH224(1)：224 位哈希（附参：算法）
    OpFN_HASH256   byte = 179 // FN_HASH256(1)：256 位哈希
    OpFN_HASH384   byte = 180 // FN_HASH384(1)：384 位哈希
    OpFN_HASH512   byte = 181 // FN_HASH512(1)：512 位哈希
    // 182-222 保留
    OpFN_PRINTF byte = 223 // FN_PRINTF：格式化输出（调试）
    OpFN_X      byte = 224 // FN_X(1){}：扩展函数引用
)

// ─── 模块指令 [225-250] ──────────────────────────────────────────────────────
const (
    OpMO_MATH byte = 225 // MO_MATH：数学模块
    OpMO_FMT  byte = 226 // MO_FMT：格式化模块
    // 227-249 保留
    OpMO_XX byte = 250 // MO_XX(1){}：扩展模块引用
)

// ─── 扩展指令 [251-253] ──────────────────────────────────────────────────────
const (
    OpEXT_MO   byte = 251 // EXT_MO(2){}：扩展模块（2字节索引）
    // 252 保留
    OpEXT_PRIV byte = 253 // EXT_PRIV(2){}：私有扩展（排除在共识外）
)

// IsUnlockAllowed 判断操作码是否允许在解锁脚本中使用
// 仅允许 [0-50] 和 [169] (SYS_NULL)
// 来源：docs/proposal/4.Script-of-Stack.md §5.2
func IsUnlockAllowed(op byte) bool {
    return op <= 50 || op == 169
}
```

**Step 4：运行验证通过**

```bash
go test ./internal/script/... -run TestOpcode -v
go build ./internal/script/...
```

**Step 5：提交**

```bash
git add internal/script/opcode.go internal/script/opcode_test.go
git commit -m "feat(script): add complete opcode constants (254 opcodes, 4-segment layout)"
```

---

### Task 4：执行环境（Domain）层级

**文件：**
- 创建：`internal/script/env.go`
- 创建：`internal/script/env_test.go`

**Step 1：编写测试**

```go
// internal/script/env_test.go
package script_test

import (
    "testing"
    "github.com/yourorg/evidcoin/internal/script"
)

func TestScriptDomain(t *testing.T) {
    sd := script.NewScriptDomain()
    if sd.PassState { t.Error("initial pass state should be false") }

    // GlobalVars 256 个槽
    sd.GlobalVars[0] = int64(42)
    if sd.GlobalVars[0] != int64(42) {
        t.Errorf("global var read back failed")
    }
}

func TestBlockDomain(t *testing.T) {
    bd := script.NewBlockDomain(nil)
    for i := 0; i < script.MaxLocalScopeSize; i++ {
        if err := bd.Append(i); err != nil {
            t.Fatalf("append failed at %d: %v", i, err)
        }
    }
    // 第 129 个应失败
    if err := bd.Append("overflow"); err == nil {
        t.Error("should return error when local scope is full")
    }
}
```

**Step 2：实现 env.go**

```go
// internal/script/env.go

package script

// ValidationContext 校验上下文（Validation Domain）
// 在一次输出校验中共享，跨 GOTO/EMBED 继承
type ValidationContext struct {
    GotoCount  int // 当前已 GOTO 调用次数（主脚本层计数）
    EmbedCount int // 运行时 EMBED 总次数
    GotoDepth  int // 当前 GOTO 嵌套深度
    EmbedDepth int // 当前 EMBED 嵌套深度（嵌入脚本内为 1，禁止再嵌入）

    // INPUT/OUTPUT 缓存区（两个独立通道）
    InputBuffer  []any
    OutputBuffer []any

    // 外部 OUTPUT 处理器（私有节点可注册）
    OutputHandler func(tag byte, data []any)
}

// ScriptDomain 脚本域（Script Domain）
// 每次 GOTO 创建新的 ScriptDomain；EMBED 共享调用者的 ScriptDomain
type ScriptDomain struct {
    Stack      []any         // 数据栈（LIFO）
    ArgSpace   []any         // 实参区（FIFO）
    GlobalVars [MaxGlobalVarSlots]any // 全局变量（256 槽）
    PassState  bool          // 通关状态（true=PASS, false=FAIL/初始）
    HasPassed  bool          // 是否至少调用过一次 PASS/CHECK
    // 当前执行位置（PC）由 execute.go 管理
}

// NewScriptDomain 创建新的脚本域
func NewScriptDomain() *ScriptDomain {
    return &ScriptDomain{
        Stack:    make([]any, 0, 16),
        ArgSpace: make([]any, 0, 8),
    }
}

// BlockDomain 块域（Block Domain）
// 每个 IF/EACH/BLOCK 等块拥有独立的局部域
type BlockDomain struct {
    Local  []any       // 局部域（追加式，最大 128）
    Parent *BlockDomain // 父块域（用于嵌套块）
}

// NewBlockDomain 创建新的块域
func NewBlockDomain(parent *BlockDomain) *BlockDomain {
    return &BlockDomain{
        Local:  make([]any, 0, 8),
        Parent: parent,
    }
}

// Append 向局部域追加值
func (bd *BlockDomain) Append(v any) error {
    if len(bd.Local) >= MaxLocalScopeSize {
        return ErrLocalScopeFull
    }
    bd.Local = append(bd.Local, v)
    return nil
}

// Get 按索引读取局部域（支持负索引）
func (bd *BlockDomain) Get(idx int) (any, error) {
    n := len(bd.Local)
    if idx < 0 {
        idx = n + idx
    }
    if idx < 0 || idx >= n {
        return nil, ErrIndexOutOfRange
    }
    return bd.Local[idx], nil
}

// ExecContext 完整执行上下文（传递给每条指令）
type ExecContext struct {
    Validation *ValidationContext
    Script     *ScriptDomain
    Block      *BlockDomain
    IsUnlock   bool // 是否为解锁脚本执行（限制操作码）
    IsEmbed    bool // 是否在 EMBED 环境中（深度检查）
    IsPublic   bool // 是否为公共验证节点（忽略 EXT_PRIV/SHELL/SYS_TIME before END）
    EndReached bool // 是否已经过 END 指令
}

// LoopVarID 循环变量标识符（用于 $X() 指令）
type LoopVarID uint8

const (
    LoopVarValue LoopVarID = 0 // $Value：当前条目值
    LoopVarIndex LoopVarID = 1 // $Index/$Key：当前索引或键
    LoopVarSlice LoopVarID = 2 // $Slice/$Dict：完整集合
    LoopVarSize  LoopVarID = 3 // $Size：集合大小
)
```

**Step 3：运行验证通过**

```bash
go test ./internal/script/... -run TestScriptDomain -run TestBlockDomain -v
```

**Step 4：提交**

```bash
git add internal/script/env.go internal/script/env_test.go
git commit -m "feat(script): add execution environment domain hierarchy"
```

---

### Task 5：字节码解码器

**文件：**
- 创建：`internal/script/decode.go`
- 创建：`internal/script/decode_test.go`

**说明：** 解码器负责从字节流解析指令附参（Auxiliary Parameters）和关联数据（Associated Data）的长度，不负责语义执行。执行引擎基于解码器推进 PC（程序计数器）。

**Step 1：编写测试**

```go
// internal/script/decode_test.go
package script_test

import (
    "testing"
    "github.com/yourorg/evidcoin/internal/script"
)

func TestReadVarint(t *testing.T) {
    tests := []struct {
        data    []byte
        wantVal uint64
        wantN   int
    }{
        {[]byte{0x00}, 0, 1},
        {[]byte{0x7F}, 127, 1},
        {[]byte{0x80, 0x01}, 128, 2},
        {[]byte{0xFF, 0xFF, 0x03}, 65535, 3},
    }
    for _, tc := range tests {
        val, n, err := script.ReadVarint(tc.data)
        if err != nil { t.Fatalf("unexpected error: %v", err) }
        if val != tc.wantVal { t.Errorf("ReadVarint val = %d, want %d", val, tc.wantVal) }
        if n != tc.wantN { t.Errorf("ReadVarint n = %d, want %d", n, tc.wantN) }
    }
}

func TestDecodeInstruction(t *testing.T) {
    // NIL 指令：单字节，无附参无关联数据
    script_bytes := []byte{0} // OpNIL
    instr, n, err := script.DecodeInstruction(script_bytes, 0)
    if err != nil { t.Fatalf("DecodeInstruction NIL: %v", err) }
    if instr.Op != 0 { t.Errorf("Op = %d, want 0", instr.Op) }
    if n != 1 { t.Errorf("n = %d, want 1", n) }

    // STRING 指令：附参 varint 长度 + 关联数据
    script_bytes2 := []byte{8, 5, 'h', 'e', 'l', 'l', 'o'} // OpSTRING, len=5, "hello"
    instr2, n2, err2 := script.DecodeInstruction(script_bytes2, 0)
    if err2 != nil { t.Fatalf("DecodeInstruction STRING: %v", err2) }
    if n2 != 7 { t.Errorf("n = %d, want 7", n2) }
    if string(instr2.Data) != "hello" { t.Errorf("data = %q, want \"hello\"", instr2.Data) }
}
```

**Step 2：实现 decode.go**

```go
// internal/script/decode.go

package script

import (
    "encoding/binary"
    "fmt"
)

// Instruction 已解码的指令（一个逻辑执行单元）
type Instruction struct {
    Op      byte   // 操作码
    AuxParams []byte // 附参原始字节（按指令规格解析）
    Data    []byte // 关联数据（如块内容、字面量字节）
    Size    int    // 该指令在字节流中占用的总字节数
}

// ReadVarint 从字节切片 data[offset:] 读取 unsigned varint（LEB128）
// 返回：值, 消耗字节数, 错误
func ReadVarint(data []byte) (uint64, int, error) {
    var result uint64
    var shift uint
    for i, b := range data {
        result |= uint64(b&0x7F) << shift
        if b&0x80 == 0 {
            return result, i + 1, nil
        }
        shift += 7
        if shift >= 64 {
            return 0, 0, fmt.Errorf("varint overflow")
        }
    }
    return 0, 0, fmt.Errorf("unexpected end of varint")
}

// ReadSignedVarint 从字节切片读取 signed varint（ZigZag 编码）
func ReadSignedVarint(data []byte) (int64, int, error) {
    u, n, err := ReadVarint(data)
    if err != nil {
        return 0, 0, err
    }
    // ZigZag decode
    return int64((u >> 1) ^ -(u & 1)), n, nil
}

// instrAuxLayout 描述每条指令的附参布局
// 格式：每个元素代表一个附参段
//   0  = 无附参
//  -1  = varint（变长）
//   n  = 固定 n 字节
type instrLayout struct {
    // AuxParams：附参各段字节长度描述（0=无，-1=varint，n=固定）
    Aux []int
    // DataLen：关联数据长度来源
    //   0  = 无关联数据
    //  -1  = 由最后一个附参（varint）指定
    //  -2  = 由附参1（varint）指定
    //   n  = 固定 n 字节
    DataLen int
}

// opcodeLayouts 操作码布局表（索引 = opcode）
// 未列出的操作码默认为无附参无关联数据
var opcodeLayouts [256]instrLayout

func init() {
    // 值指令
    opcodeLayouts[OpBIGINT] = instrLayout{Aux: []int{1}, DataLen: -1} // 1字节长度 + 数据
    opcodeLayouts[OpSTRING] = instrLayout{Aux: []int{-1}, DataLen: -1} // varint长度 + UTF-8
    opcodeLayouts[OpVALUESET] = instrLayout{Aux: []int{-1}, DataLen: -1}
    opcodeLayouts[OpDATA] = instrLayout{Aux: []int{-1}, DataLen: -1}
    opcodeLayouts[OpREGEXP] = instrLayout{Aux: []int{1}, DataLen: -1}
    opcodeLayouts[OpDATE] = instrLayout{DataLen: -3} // special: signed varint
    opcodeLayouts[OpCODE] = instrLayout{Aux: []int{-1}, DataLen: -1}
    opcodeLayouts[OpSCRIPT] = instrLayout{Aux: []int{-1, 48, -1}} // varint+48B TxID+varint
    opcodeLayouts[OpVALUE] = instrLayout{Aux: []int{1}}

    // 捕获指令
    opcodeLayouts[OpLOCAL_READ] = instrLayout{Aux: []int{1}}   // signed int8
    opcodeLayouts[OpLOOP_VAR] = instrLayout{Aux: []int{1}}

    // 栈操作
    opcodeLayouts[OpPOPS] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpTOPS] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpPEEKS] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpSHIFT] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpCLONE] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpVIEW] = instrLayout{Aux: []int{1}}

    // 集合操作
    opcodeLayouts[OpSLICE] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpCALL] = instrLayout{Aux: []int{1}}

    // 交互指令
    opcodeLayouts[OpINPUT] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpBUFDUMP] = instrLayout{Aux: []int{1}}

    // 结果指令（GOTO/EMBED：varint+48+varint）
    opcodeLayouts[OpGOTO] = instrLayout{Aux: []int{-1, 48, -1}}
    opcodeLayouts[OpEMBED] = instrLayout{Aux: []int{-1, 48, -1}}

    // 流程控制（子块：varint 长度 + 子块数据）
    for _, op := range []byte{OpIF, OpELSE, OpSWITCH, OpCASE, OpDEFAULT, OpEACH, OpBLOCK} {
        opcodeLayouts[op] = instrLayout{Aux: []int{-1}, DataLen: -1}
    }

    // 转换
    opcodeLayouts[OpToSTRING] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpANYS] = instrLayout{Aux: []int{1}}

    // 算术
    opcodeLayouts[OpEXPR] = instrLayout{Aux: []int{1}, DataLen: -1}
    opcodeLayouts[OpREP] = instrLayout{Aux: []int{1}}

    // 逻辑
    opcodeLayouts[OpSOME] = instrLayout{Aux: []int{1}}

    // 模式指令
    opcodeLayouts[OpMODEL] = instrLayout{Aux: []int{2}, DataLen: -4} // 2字节，低15位为长度
    opcodeLayouts[OpWILDN] = instrLayout{Aux: []int{-1}}
    opcodeLayouts[OpOPTIONAL] = instrLayout{Aux: []int{-1}, DataLen: -1}
    opcodeLayouts[OpWILD_IND] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpEXTRACT] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpTYPE_MATCH] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpINT_RANGE] = instrLayout{Aux: []int{-1, -1}} // 两个 varint
    opcodeLayouts[OpFLT_RANGE] = instrLayout{Aux: []int{8, 8, 4}}
    opcodeLayouts[OpRE_SEARCH] = instrLayout{Aux: []int{1, 1}, DataLen: -1}
    opcodeLayouts[OpRE_EXTRACT] = instrLayout{Aux: []int{1}}

    // 环境指令
    opcodeLayouts[OpENV] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpIN] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpOUT] = instrLayout{Aux: []int{-1, 1}}
    opcodeLayouts[OpINOUT] = instrLayout{Aux: []int{-1, 1}}
    opcodeLayouts[OpXFROM] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpVAR] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpSETVAR] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpSOURCE] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpSIGNED] = instrLayout{Aux: []int{1}}

    // 工具指令
    opcodeLayouts[OpKEYVAL] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpMATCH] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpSUBSTR] = instrLayout{Aux: []int{-1}}
    opcodeLayouts[OpREPLACE] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpCMPFLO] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpRANGE] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpMAP] = instrLayout{Aux: []int{-1}, DataLen: -1}
    opcodeLayouts[OpFILTER] = instrLayout{Aux: []int{-1}, DataLen: -1}
    opcodeLayouts[OpSHELL] = instrLayout{Aux: []int{-1}, DataLen: -1}

    // 系统指令
    opcodeLayouts[OpSYS_TIME] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpSYS_AWARD] = instrLayout{Aux: []int{1, -1}}

    // 函数指令
    opcodeLayouts[OpFN_HASH224] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpFN_HASH256] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpFN_HASH384] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpFN_HASH512] = instrLayout{Aux: []int{1}}
    opcodeLayouts[OpFN_X] = instrLayout{Aux: []int{1}}

    // 模块指令
    opcodeLayouts[OpMO_XX] = instrLayout{Aux: []int{1}}

    // 扩展指令
    opcodeLayouts[OpEXT_MO] = instrLayout{Aux: []int{2}}
    opcodeLayouts[OpEXT_PRIV] = instrLayout{Aux: []int{2}}
}

// DecodeInstruction 从 code[offset:] 解码一条完整指令
// 返回：指令结构, 消耗字节数, 错误
func DecodeInstruction(code []byte, offset int) (*Instruction, int, error) {
    if offset >= len(code) {
        return nil, 0, fmt.Errorf("decode: offset out of range")
    }
    op := code[offset]
    if op == 254 || op == 255 {
        return nil, 0, ErrInvalidOpcode
    }

    layout := opcodeLayouts[op]
    pos := offset + 1
    instr := &Instruction{Op: op}

    // 解析附参
    var auxBuf []byte
    for _, auxLen := range layout.Aux {
        if auxLen == -1 {
            // varint
            if pos >= len(code) {
                return nil, 0, fmt.Errorf("decode: truncated varint aux for op %d", op)
            }
            v, n, err := ReadVarint(code[pos:])
            if err != nil {
                return nil, 0, fmt.Errorf("decode: aux varint for op %d: %w", op, err)
            }
            b := make([]byte, 8)
            binary.LittleEndian.PutUint64(b, v)
            auxBuf = append(auxBuf, b...)
            pos += n
        } else if auxLen > 0 {
            if pos+auxLen > len(code) {
                return nil, 0, fmt.Errorf("decode: truncated aux for op %d", op)
            }
            auxBuf = append(auxBuf, code[pos:pos+auxLen]...)
            pos += auxLen
        }
    }
    instr.AuxParams = auxBuf

    // 解析关联数据
    switch layout.DataLen {
    case 0:
        // 无关联数据
    case -1:
        // 由最后一个附参（varint）指定长度（已在附参中读取）
        // 需要重新从字节流中读取 varint（附参中仅保存了值，位置已推进）
        // 此处用简化方式：倒退重新解析
        // 实现简化：对有 DataLen=-1 的指令，附参中最后一段是 varint，其值即为数据长度
        // 具体解析在 execute.go 中根据 layout 完成，此处只记录 pos 推进
        dataLen, _, err := ReadVarint(code[offset+1:])
        if err != nil {
            return nil, 0, err
        }
        // 重算 pos（跳过 aux）
        auxEnd := pos
        if pos+int(dataLen) > len(code) {
            return nil, 0, fmt.Errorf("decode: truncated data for op %d", op)
        }
        instr.Data = code[auxEnd : auxEnd+int(dataLen)]
        pos = auxEnd + int(dataLen)
    case -3:
        // signed varint（DATE）
        if pos >= len(code) {
            return nil, 0, fmt.Errorf("decode: truncated signed varint for op %d", op)
        }
        _, n, err := ReadSignedVarint(code[pos:])
        if err != nil {
            return nil, 0, err
        }
        instr.Data = code[pos : pos+n]
        pos += n
    case -4:
        // MODEL：2字节附参，低15位为子块字节长度
        if len(auxBuf) >= 2 {
            subLen := int(binary.BigEndian.Uint16(auxBuf[len(auxBuf)-2:]) & 0x7FFF)
            if pos+subLen > len(code) {
                return nil, 0, fmt.Errorf("decode: truncated model block for op %d", op)
            }
            instr.Data = code[pos : pos+subLen]
            pos += subLen
        }
    default:
        // 固定长度
        if layout.DataLen > 0 {
            if pos+layout.DataLen > len(code) {
                return nil, 0, fmt.Errorf("decode: truncated fixed data for op %d", op)
            }
            instr.Data = code[pos : pos+layout.DataLen]
            pos += layout.DataLen
        }
    }

    instr.Size = pos - offset
    return instr, instr.Size, nil
}
```

> **注意：** `decode.go` 的实现是对布局的粗略实现，执行阶段需要更精确地处理各指令附参（特别是 GOTO/EMBED、MODEL 等复合附参）。实现时需按需精化。

**Step 3：运行测试**

```bash
go test ./internal/script/... -run TestReadVarint -run TestDecodeInstruction -v
```

**Step 4：提交**

```bash
git add internal/script/decode.go internal/script/decode_test.go
git commit -m "feat(script): add bytecode decoder with varint and instruction layout support"
```

---

### Task 6：执行引擎核心（ScriptEngine）

**文件：**
- 创建：`internal/script/engine.go`
- 创建：`internal/script/execute.go`

**Step 1：编写测试**

```go
// internal/script/engine_test.go
package script_test

import (
    "testing"
    "github.com/yourorg/evidcoin/internal/script"
)

func TestEngineBasicExecution(t *testing.T) {
    // 执行 TRUE PASS 脚本（最简单的成功场景）
    // 字节码：[OpTRUE, OpPASS] = [1, 51]
    code := []byte{script.OpTRUE, script.OpPASS}
    eng := script.NewEngine(script.EngineOptions{IsPublic: true})
    result, err := eng.Execute(code, nil)
    if err != nil {
        t.Fatalf("Execute failed: %v", err)
    }
    if !result.Passed {
        t.Error("expected PASS result")
    }
}

func TestEnginePassFailed(t *testing.T) {
    // FALSE PASS 应该失败
    code := []byte{script.OpFALSE, script.OpPASS}
    eng := script.NewEngine(script.EngineOptions{IsPublic: true})
    _, err := eng.Execute(code, nil)
    if err == nil {
        t.Error("expected error for FALSE PASS")
    }
}

func TestEnginePushPop(t *testing.T) {
    // TRUE POP（弹出并忽略）=> 无 PASS，pass state 未设置
    code := []byte{script.OpTRUE, script.OpPOP}
    eng := script.NewEngine(script.EngineOptions{IsPublic: true})
    result, err := eng.Execute(code, nil)
    if err != nil {
        t.Fatalf("Execute failed: %v", err)
    }
    _ = result // 未设置 pass state
}
```

**Step 2：实现 engine.go**

```go
// internal/script/engine.go

package script

// EngineOptions 引擎配置选项
type EngineOptions struct {
    IsPublic bool // 是否为公共验证节点（影响 EXT_PRIV/SHELL/SYS_TIME 行为）
    // 可扩展：TxContext, SystemContext 等（Phase 5 实现核心后补充）
}

// ExecuteResult 脚本执行结果
type ExecuteResult struct {
    Passed     bool  // 通关状态（PASS/FAIL）
    EndReached bool  // 是否遇到 END 指令
    Stack      []any // 执行结束后的栈状态（调试用）
}

// Engine 脚本引擎
type Engine struct {
    opts EngineOptions
}

// NewEngine 创建脚本引擎实例
func NewEngine(opts EngineOptions) *Engine {
    return &Engine{opts: opts}
}

// Execute 执行脚本字节码
// code：指令序列（只读）
// initialStack：初始栈内容（用于 GOTO 传参）
func (e *Engine) Execute(code []byte, initialStack []any) (*ExecuteResult, error) {
    ctx := &ExecContext{
        Validation: &ValidationContext{
            InputBuffer:  make([]any, 0),
            OutputBuffer: make([]any, 0),
        },
        Script:   NewScriptDomain(),
        Block:    NewBlockDomain(nil),
        IsPublic: e.opts.IsPublic,
    }

    if initialStack != nil {
        ctx.Script.Stack = append(ctx.Script.Stack, initialStack...)
    }

    err := executeLoop(ctx, code, 0, len(code))
    if err != nil {
        return nil, err
    }

    return &ExecuteResult{
        Passed:     ctx.Script.PassState,
        EndReached: ctx.EndReached,
        Stack:      ctx.Script.Stack,
    }, nil
}

// ExecuteUnlockThenLock 执行完整的解锁+锁定脚本管道
// 来源：docs/proposal/4.Script-of-Stack.md §5.1
func (e *Engine) ExecuteUnlockThenLock(unlockScript, lockScript []byte) (*ExecuteResult, error) {
    if len(unlockScript) > MaxUnlockScriptSize {
        return nil, ErrScriptTooLong
    }
    if len(lockScript) > MaxLockScriptSize {
        return nil, ErrScriptTooLong
    }

    ctx := &ExecContext{
        Validation: &ValidationContext{
            InputBuffer:  make([]any, 0),
            OutputBuffer: make([]any, 0),
        },
        Script:   NewScriptDomain(),
        Block:    NewBlockDomain(nil),
        IsPublic: e.opts.IsPublic,
        IsUnlock: true,
    }

    // Step 1: 执行解锁脚本（受限操作码）
    if err := executeLoop(ctx, unlockScript, 0, len(unlockScript)); err != nil {
        return nil, err
    }

    // Step 2: 切换到锁定脚本（完整指令集）
    ctx.IsUnlock = false
    if err := executeLoop(ctx, lockScript, 0, len(lockScript)); err != nil {
        return nil, err
    }

    return &ExecuteResult{
        Passed:     ctx.Script.PassState,
        EndReached: ctx.EndReached,
        Stack:      ctx.Script.Stack,
    }, nil
}
```

**Step 3：实现 execute.go（指令调度主循环）**

```go
// internal/script/execute.go

package script

// executeLoop 执行 code[start:end] 范围内的指令
func executeLoop(ctx *ExecContext, code []byte, start, end int) error {
    pc := start
    for pc < end {
        op := code[pc]

        // 系统保留操作码
        if op == 254 || op == 255 {
            return ErrInvalidOpcode
        }

        // 解锁脚本操作码限制
        if ctx.IsUnlock && !IsUnlockAllowed(op) {
            return ErrUnlockOpcodeNotAllowed
        }

        // 解码指令（获取附参和关联数据）
        instr, size, err := DecodeInstruction(code, pc)
        if err != nil {
            return err
        }

        // 调度执行
        if err := dispatchInstruction(ctx, instr, code); err != nil {
            return err
        }

        pc += size
    }
    return nil
}

// dispatchInstruction 根据操作码调度到对应的指令实现
// 指令实现位于 instr/ 子包，通过此函数统一调用
func dispatchInstruction(ctx *ExecContext, instr *Instruction, code []byte) error {
    // 指令分发：按操作码范围调用对应处理函数
    // 初始实现仅覆盖核心指令，其余指令在后续 Task 中逐步实现
    switch instr.Op {
    // ── 值指令 ──────────────────────────────────────────────
    case OpNIL:
        return execPushValue(ctx, nil)
    case OpTRUE:
        return execPushValue(ctx, true)
    case OpFALSE:
        return execPushValue(ctx, false)

    // ── 栈操作 ──────────────────────────────────────────────
    case OpNOP:
        ctx.Script.ArgSpace = ctx.Script.ArgSpace[:0]
        return nil
    case OpPUSH:
        return execPUSH(ctx)
    case OpPOP:
        return execPOP(ctx)
    case OpTOP:
        return execTOP(ctx)

    // ── 结果指令 ─────────────────────────────────────────────
    case OpPASS:
        return execPASS(ctx)
    case OpCHECK:
        return execCHECK(ctx)
    case OpEND:
        ctx.EndReached = true
        // 公共节点遇到 END 停止后续执行（私有节点继续）
        if ctx.IsPublic {
            return &haltError{} // 特殊信号，非错误
        }
        return nil
    case OpEXIT:
        return &haltError{}

    default:
        // TODO：其余操作码在后续 Task 中实现
        // 暂时忽略（NOP 语义）
        return nil
    }
}

// haltError 表示脚本正常终止（END/EXIT），不是真正的错误
type haltError struct{}
func (h *haltError) Error() string { return "script: halted" }

// execPushValue 将值压入 argument space 或直接压栈
func execPushValue(ctx *ExecContext, v any) error {
    // 检查栈高
    if len(ctx.Script.Stack) >= MaxStackHeight {
        return ErrStackOverflow
    }
    ctx.Script.Stack = append(ctx.Script.Stack, v)
    return nil
}

// execPUSH 将 argument space 全部内容按 FIFO 压入 stack
func execPUSH(ctx *ExecContext) error {
    for _, v := range ctx.Script.ArgSpace {
        if len(ctx.Script.Stack) >= MaxStackHeight {
            return ErrStackOverflow
        }
        ctx.Script.Stack = append(ctx.Script.Stack, v)
    }
    ctx.Script.ArgSpace = ctx.Script.ArgSpace[:0]
    return nil
}

// execPOP 弹出栈顶
func execPOP(ctx *ExecContext) error {
    if len(ctx.Script.Stack) == 0 {
        return ErrStackUnderflow
    }
    v := ctx.Script.Stack[len(ctx.Script.Stack)-1]
    ctx.Script.Stack = ctx.Script.Stack[:len(ctx.Script.Stack)-1]
    return execConsumeResult(ctx, v)
}

// execTOP 复制栈顶（不移除）
func execTOP(ctx *ExecContext) error {
    if len(ctx.Script.Stack) == 0 {
        return ErrStackUnderflow
    }
    v := ctx.Script.Stack[len(ctx.Script.Stack)-1]
    return execConsumeResult(ctx, v)
}

// execPASS 通关检查（失败立即终止）
func execPASS(ctx *ExecContext) error {
    v, err := popArg(ctx)
    if err != nil {
        return err
    }
    b, ok := v.(bool)
    if !ok {
        return ErrTypeMismatch
    }
    ctx.Script.HasPassed = true
    ctx.Script.PassState = b
    if !b {
        return ErrPassFailed
    }
    return nil
}

// execCHECK 通关检查（失败不终止）
func execCHECK(ctx *ExecContext) error {
    v, err := popArg(ctx)
    if err != nil {
        return err
    }
    b, ok := v.(bool)
    if !ok {
        return ErrTypeMismatch
    }
    ctx.Script.HasPassed = true
    ctx.Script.PassState = b
    return nil
}

// popArg 从 argument space 或 stack 消费一个值（遵循实参获取规则）
func popArg(ctx *ExecContext) (any, error) {
    if len(ctx.Script.ArgSpace) > 0 {
        v := ctx.Script.ArgSpace[0]
        ctx.Script.ArgSpace = ctx.Script.ArgSpace[1:]
        return v, nil
    }
    if len(ctx.Script.Stack) == 0 {
        return nil, ErrStackUnderflow
    }
    v := ctx.Script.Stack[len(ctx.Script.Stack)-1]
    ctx.Script.Stack = ctx.Script.Stack[:len(ctx.Script.Stack)-1]
    return v, nil
}

// execConsumeResult 将指令结果写入 argument space（捕获前缀时）或 stack
// 这是值流动的核心，暂时简化为直接压栈
func execConsumeResult(ctx *ExecContext, v any) error {
    if len(ctx.Script.Stack) >= MaxStackHeight {
        return ErrStackOverflow
    }
    ctx.Script.Stack = append(ctx.Script.Stack, v)
    return nil
}
```

> **注意：** `executeLoop` 中的 `haltError` 需要在调用方处理。修改 `executeLoop` 以区分 halt 和 error：

```go
// 在 engine.go 的 Execute 方法中处理 haltError
func (e *Engine) Execute(code []byte, initialStack []any) (*ExecuteResult, error) {
    // ... 省略初始化 ...
    err := executeLoop(ctx, code, 0, len(code))
    if err != nil {
        if _, ok := err.(*haltError); ok {
            // 正常终止
        } else {
            return nil, err
        }
    }
    // ...
}
```

**Step 4：运行测试**

```bash
go test ./internal/script/... -run TestEngine -v
```

**Step 5：提交**

```bash
git add internal/script/engine.go internal/script/execute.go internal/script/engine_test.go
git commit -m "feat(script): add core execution engine with basic dispatch loop"
```

---

### Task 7：捕获前缀机制（@、$、~）

**说明：** 捕获指令是实参区机制的核心，必须在大多数指令之前实现。

**文件：**
- 创建：`internal/script/instr/capture.go`
- 创建：`internal/script/instr/capture_test.go`

**设计要点：**
- `@`（OpARG_CAPTURE）：拦截紧接指令的返回值，写入 argument space 而非 stack
- `$`（OpLOCAL_STORE）：拦截紧接指令的返回值，写入 local scope
- `$()`（OpLOCAL_READ）：从 local scope 读取值，写入 argument space
- `~`（OpSTACK_DIRECT）：强制下一条指令从 stack 取参，绕过 argument space

**实现策略：** 在 execute.go 的调度循环中，维护 `captureMode` 状态标志，在调度前检查前缀，在调度后将结果重定向。

**Step 1：编写测试**

```go
// internal/script/instr/capture_test.go
package instr_test

// 测试 @ 捕获：TRUE 的返回值进 argument space，不压栈
// 字节码：[@, TRUE, NOP]（NOP 消耗 argument space）
// 期望：stack 为空（TRUE 被捕获到 arg space 后被 NOP 消耗）
func TestArgCapture(t *testing.T) {
    // 使用集成方式通过引擎执行
    // @TRUE NOP => 栈为空
    code := []byte{
        script.OpARG_CAPTURE, script.OpTRUE, // @TRUE
        script.OpNOP,                         // 消耗 arg space
    }
    eng := script.NewEngine(script.EngineOptions{})
    result, err := eng.Execute(code, nil)
    if err != nil { t.Fatalf("Execute: %v", err) }
    if len(result.Stack) != 0 {
        t.Errorf("stack should be empty, got %d items", len(result.Stack))
    }
}
```

**Step 2：修改 execute.go 以支持捕获模式**

在 `executeLoop` 中添加前缀捕获状态机：

```go
// executeLoop 主循环增加捕获前缀状态
type captureMode uint8
const (
    captureModeNone  captureMode = 0
    captureModeArg   captureMode = 1 // @ → argument space
    captureModeLocal captureMode = 2 // $ → local scope
)

// 在 executeLoop 中维护 currentCapture 状态
// 遇到 @/$/~ 时设置状态，下一条指令执行后重定向返回值
```

> **实现细节提示：** 捕获前缀的实现涉及对返回值流向的控制。推荐方式是在 `execConsumeResult` 函数中检查当前 capture 模式，并相应地将值写入 arg space 或 local scope。

**Step 3：提交**

```bash
git commit -m "feat(script): implement capture prefix mechanism (@, $, ~)"
```

---

### Task 8：值指令（完整实现）

**文件：** `internal/script/instr/value.go`

覆盖操作码 `[0–18]`，完整实现所有值类型字面量压栈：

| 操作码 | 指令 | 关键实现点 |
|--------|------|-----------|
| 0-2 | NIL/TRUE/FALSE | 直接压栈 |
| 3 | Byte literal | 读1字节关联数据 → uint8 |
| 4 | Rune literal | 读4字节 → int32 |
| 5 | Int literal | 读 varint → int64 |
| 6 | BigInt literal | 读1字节长度 + n字节 → *big.Int |
| 7 | Float literal | 读8字节 IEEE754 → float64 |
| 8 | String literal | 读 varint 长度 + UTF-8 → string |
| 9 | ValueSet {} | 读 varint 长度 + 内部值指令序列 → 执行后调用 PromoteValueSet → []any |
| 10 | DATA{} | 读 varint 长度 + 字节 → []byte |
| 11 | RegExp | 读1字节长度 + 文本 → *regexp.Regexp |
| 12 | DATE{} | 读 signed varint（Unix ms）→ time.Time |
| 13 | DICT | 从 argument space 消耗 keys([]string) + values([]any) → map[string]any |
| 16 | CODE{} | 读 varint 长度 + 字节码 → Script 类型标记值（不执行） |
| 17 | SCRIPT | 读 varint+48B+varint 附参 → 从链上加载目标脚本字节码（链上访问接口预留） |
| 18 | VALUE | 读1字节索引 → 返回当前 block domain local scope 的对应条目（待讨论） |

**测试重点：**
- ValueSet 类型提升：`{uint8(1), int32(200)}` → `[]int32{1, 200}`
- BigInt 字节序（big-endian）
- RegExp 编译失败 → 执行失败
- DATE signed varint 正负值

---

### Task 9：栈操作指令（完整实现）

**文件：** `internal/script/instr/stack.go`

覆盖操作码 `[24–34]`。

关键测试：
```go
// POPS 数量超出栈高度 → ErrStackUnderflow
// VIEW 越界 → ErrIndexOutOfRange
// SHIFT[0] → 弹出全部为 slice
```

---

### Task 10：集合操作指令

**文件：** `internal/script/instr/collection.go`

覆盖操作码 `[35–45]`。

关键点：
- `MERGE`：消耗 argument space 中可变数量的 slice
- `PACK`：将 slice 成员紧凑二进制编码（各成员按其原生类型编码，无填充）
- `SET`：原地修改 dict（dict 在 Go 中为引用类型，可安全修改）
- `CALL`：调用模块对象成员方法（模块对象为 Module 接口）

---

### Task 11：流程控制指令

**文件：** `internal/script/instr/flow.go`

覆盖操作码 `[58–66]`。

**EACH{} 实现策略：**
1. 消耗栈顶集合
2. 创建新 BlockDomain（子局部域）
3. 设置循环变量（$Value, $Index, $Slice, $Size）
4. 对每个成员执行子块指令（`executeLoop(ctx, block, 0, len(block))`）
5. 遇到 `BREAK` 抛出 breakSignal，退出
6. 遇到 `CONTINUE` 抛出 continueSignal，跳到下一成员

**IF/ELSE 实现策略：**
1. `IF`：消耗 bool 值，若 true 执行子块，在 BlockDomain 记录 if-state
2. `ELSE`：检查当前块 if-state，若 false 则执行子块

**SWITCH/CASE 实现策略：**
1. `SWITCH`：消耗 target + branch list，逐个 CASE 匹配
2. 首匹配后执行对应子块，退出 SWITCH（无 fall-through）

---

### Task 12：算术、比较、逻辑指令

**文件：**
- `internal/script/instr/arith.go`（操作码 80-103）
- `internal/script/instr/compare.go`（操作码 104-111）
- `internal/script/instr/logic.go`（操作码 112-115）

**表达式容器 `()` 的实现：**
- 操作码 80（OpEXPR）包含嵌套字节码（关联数据）
- 内部执行时所有值自动转为 float64
- 表达式操作符（81-85）仅在表达式上下文中有效
- 外部操作符（86-103）在表达式上下文中无效

**关键测试：**
```go
// ADD：int64 + int64, float64 + float64, string + string, []byte + []byte, dict + dict
// 表达式 (3 + 4 * 2) → float64(11)
// DIVMOD → 产生商和余数两个值（auto-expand）
// WITHIN：[min, max) 半开区间，包含下界
```

---

### Task 13：转换指令

**文件：** `internal/script/instr/convert.go`（操作码 67-79）

**关键安全约束（必须测试）：**
- `BYTES` 对 Script 类型调用：返回字节码副本，但结果 TypeID = TypeBytes（非 Script）
- **禁止** Bytes → Script 转换（`ErrCodeInjection`）
- `ANYS` 对 nil slice 输入 → 返回目标类型空 slice

---

### Task 14：环境指令

**文件：** `internal/script/instr/environ.go`（操作码 128-137）

**接口设计（需在 env.go 中预留）：**

环境指令需要访问交易上下文。定义接口而非具体结构，使得脚本引擎不依赖具体交易实现：

```go
// TxContext 交易执行上下文（由上层传入，脚本引擎通过此接口访问交易数据）
type TxContext interface {
    // GetInput 获取当前输入条目字段值
    GetInput(field InputField) (any, error)
    // GetOutput 获取指定输出条目字段值
    GetOutput(idx int, field OutputField) (any, error)
    // GetSourceOutput 获取源交易同级输出
    GetSourceOutput(idx int, field OutputField) (any, error)
    // GetXFrom 获取来源脚本信息
    GetXFrom(field XFromField) (any, error)
    // IsSignedBy 检查公钥哈希列表中位置 idx 的签名是否已验证
    IsSignedBy(idx int) bool
    // GetSystemInfo 获取系统域信息（链高、时间戳等）
    GetSystemInfo(field SystemField) any
}
```

**EngineOptions 中增加 TxContext 字段：**

```go
type EngineOptions struct {
    IsPublic  bool
    TxContext  TxContext // 可为 nil（简单测试时）
}
```

---

### Task 15：密码学函数指令

**文件：** `internal/script/instr/function.go`（操作码 170-224）

**哈希算法映射（附参字节）：**

| 附参值 | 算法 | Go 包 |
|--------|------|-------|
| 0 | BLAKE3 | `lukechampine.com/blake3` |
| 1 | BLAKE2 | `golang.org/x/crypto/blake2b` |
| 2 | SHA2 | `crypto/sha256`（224）/ `crypto/sha512`（384/512）|
| 3 | SHA3 | `golang.org/x/crypto/sha3` |

**FN_CHECKSIG 验证流程：**
1. 消耗栈参：`chk_type`(Byte) + `auth_flag`(Byte) + `pubKey`(Bytes) + `sig`(Bytes)
2. 使用 ML-DSA-65（或 `pkg/crypto/` 提供的签名验证函数）验证
3. 验证结果写入 ValidationContext 签名记录（供 `SIGNED` 指令查询）
4. 返回 Bool

**FN_ADDRESS 编解码：**
- 编码：`Bytes（公钥哈希）+ String（前缀）` → Base58Check 字符串
- 解码：`String（地址）+ String（前缀）` → Bytes（公钥哈希）
- Checksum：对数据进行两次 SHA3-256，取末 4 字节

---

### Task 16：模式匹配指令

**文件：** `internal/script/instr/pattern.go`（操作码 116-127）

**MODEL{} 实现策略：**

模式匹配是脚本引擎中最复杂的指令，需要单独的匹配引擎：

```go
// PatternMatcher 模式匹配引擎
type PatternMatcher struct {
    target   []byte // 目标指令序列
    pattern  []byte // 模式指令序列
    extracts []any  // 提取值列表（#指令的结果）
    targetPC int
    patternPC int
}

// Match 执行模式匹配
// 返回：是否匹配, 提取值列表
func (m *PatternMatcher) Match() (bool, []any)
```

**指令匹配规则：**
1. 普通指令：完全匹配（opcode + 所有附参 + 关联数据）
2. `_`（117）：匹配目标任意单条指令（目标 PC +1 条指令）
3. `_(N)`（118）：匹配目标连续 N 条指令
4. `?{}`（119）：尝试匹配序列，全匹配则推进，否则跳过（不回退）
5. `...`（127）：非贪心通配，需要回溯（O(n×m) 最坏情况）
6. `#`（121）：提取当前目标指令的指定部分，追加到 extracts
7. `!?Type`（122）：检查目标指令产出值的类型

---

### Task 17：系统指令

**文件：** `internal/script/instr/system.go`（操作码 164-169）

**SYS_CHKPASS 实现要点：**
- 参数来自 environment（解锁脚本提供的签名数据），不从 data stack 消费
- 通过 TxContext 接口获取签名数据
- 失败立即终止（等同 PASS false）
- 成功后在 ValidationContext 记录已验证状态

**SYS_TIME 安全约束：**
- 必须在 END 指令执行后（或 INPUT 导致的隐性 END 后）才可调用
- 公共验证节点的节点时钟不参与共识，仅用于私有逻辑

**SYS_AWARD 约束：**
- 仅可用于 Coinbase 交易的 output script
- 需通过 TxContext 检查

---

### Task 18：GOTO/EMBED 实现

**文件：** `internal/script/instr/result.go`（操作码 53-54，其余结果指令已在 Task 6 中实现）

**GOTO 实现流程：**
1. 检查 GOTO 调用次数（主脚本 ≤ 2）和深度（≤ 3）
2. 将当前 argument space 内容作为初始 stack 传给目标
3. 创建新 ScriptDomain（独立）
4. 从 TxContext 加载目标脚本字节码
5. 递归调用执行（ctx.Validation.GotoDepth++）
6. 将目标脚本的 PassState 回写到当前 ScriptDomain
7. 目标脚本的 Stack 丢弃

**EMBED 实现流程：**
1. 检查 EMBED 调用次数（≤ 4）和深度（= 0，嵌入脚本内禁止）
2. 共享当前 ScriptDomain（直接修改调用者的 stack/arg space/globals）
3. 从 TxContext 加载目标脚本字节码
4. 直接在当前上下文中执行（ctx.IsEmbed = true, EmbedDepth++）
5. 嵌入脚本不能包含 GOTO 或 EMBED（通过 EmbedDepth 检查）

---

### Task 19：模块指令与扩展指令

**文件：**
- `internal/script/instr/module.go`（操作码 225-250）
- `internal/script/instr/extension.go`（操作码 251-253）

**模块对象接口：**

```go
// Module 脚本模块接口（由 MO_ 系列指令返回）
type Module interface {
    // Call 调用模块方法
    Call(method string, args []any) ([]any, error)
}

// MathModule 数学模块（MO_MATH）
type MathModule struct{}

func (m *MathModule) Call(method string, args []any) ([]any, error) {
    switch method {
    case "Abs":   // math.Abs(float64) → float64
    case "Ceil":  // math.Ceil(float64) → float64
    case "Floor": // math.Floor(float64) → float64
    case "Pow":   // math.Pow(float64, float64) → float64
    case "Max":   // math.Max(float64, float64) → float64
    case "Min":   // math.Min(float64, float64) → float64
    case "Mod":   // math.Mod(float64, float64) → float64
    }
    return nil, fmt.Errorf("unknown method: %s", method)
}
```

**EXT_PRIV 安全约束：**
- 公共验证节点：若在 END 指令之前出现 EXT_PRIV → 执行失败（视为错误）
- 私有节点：正常执行第三方扩展

---

### Task 20：完整执行管道与集成测试

**文件：**
- `internal/script/pipeline.go`（锁定/解锁管道的完整实现）
- `internal/script/script_test.go`（集成测试）

**集成测试场景（必须覆盖）：**

```go
// Test 1：标准单签支付
// 解锁：无（签名数据在 environment 中）
// 锁定：SYS_CHKPASS
// 期望：PASS

// Test 2：哈希屏障支付
// 解锁：<hashSource>
// 锁定：FN_HASH256 DATA{<expectedHash>} EQUAL PASS END
// 期望：PASS（哈希匹配时）/ FAIL（不匹配时）

// Test 3：时间锁多路
// 锁定：SYS_CHKPASS SYS_TIME{Stamp} {expireTime} GT IF { SIGNED[0] PASS EXIT } SIGNED[1] PASS FN_HASH256 DATA{} EQUAL PASS END
// 期望：按时间条件分支

// Test 4：EACH 迭代
// 锁定：{1,2,3} EACH { $Value PUSH }
// 期望：stack = [1, 2, 3]

// Test 5：嵌套 IF/ELSE
// Test 6：SWITCH/CASE
// Test 7：类型转换链
// Test 8：模式匹配 MODEL{}
// Test 9：全局变量 VAR/SETVAR
```

**Step 1：编写集成测试（全部先失败）**

**Step 2：逐步实现各指令直到测试全部通过**

**Step 3：覆盖率检查**

```bash
go test -cover ./internal/script/...
# 目标：核心执行路径覆盖率 ≥ 80%
```

---

### Task 21：go.mod 依赖更新

**文件：** `go.mod`, `go.sum`

```bash
go get lukechampine.com/blake3
go get golang.org/x/crypto
go get github.com/mr-tron/base58
# ML-DSA-65（Go 1.25 标准库内置 crypto/mlkem 先行查阅）
go mod tidy && go mod verify
```

---

### Task 22：验收检查

```bash
# 1. 构建
go build ./...

# 2. 全部测试
go test ./internal/script/... -v -count=1

# 3. 覆盖率（核心逻辑 ≥ 80%）
go test -coverprofile=coverage.out ./internal/script/...
go tool cover -func=coverage.out | grep -E "script/|total"

# 4. 格式检查
go fmt ./... && gofmt -s -l .

# 5. 静态分析
golangci-lint run ./internal/script/...

# 6. 竞态检测
go test -race ./internal/script/...
```

---

## 实现优先级

| 优先级 | 任务 | 说明 |
|--------|------|------|
| P0（必须先实现）| Task 1-6 | 包骨架、类型、操作码、环境、解码、引擎核心 |
| P0 | Task 7 | 捕获前缀（是大多数指令的基础） |
| P1 | Task 8-12 | 值/栈/集合/流程/算术/比较/逻辑 |
| P1 | Task 13 | 转换（类型安全约束） |
| P2 | Task 14-16 | 环境/密码学/模式匹配 |
| P2 | Task 17-18 | 系统指令/GOTO/EMBED |
| P3 | Task 19 | 模块/扩展 |
| P0 | Task 21 | 依赖安装 |
| 最后 | Task 20, 22 | 集成测试 + 验收 |

---

## 待确认事项（实现前需明确）

以下事项在提案中未完全说明，实现前需要用户或后续提案明确：

1. **VALUE 指令（opcode 18）**：提案中提到 `VALUE{idx}` 按索引读取值，但未说明从何读取（是 local scope 还是 argument space 还是其他）？

2. **DICT 指令（opcode 13）**：提案说"运行时消耗 keys + values"，但如何确定 keys/values 的边界？是先压 []string keys 再压 []any values 到 argument space？

3. **PACK 指令**：各类型的二进制编码格式是什么？（用于将 slice 编码为 Bytes）

4. **SCRIPT 指令（opcode 17）** 的链上加载：需要 TxContext 提供 `GetScript(year int, txID [48]byte, outIdx int) ([]byte, error)` 接口，是否在 Phase 5 实现还是预留 stub？

5. **SYS_AWARD**：铸造验证逻辑与 PoH 共识（Phase 6）紧密耦合，Phase 5 是否仅实现接口 stub？

---

## 验收标准

| 验收项 | 标准 |
|--------|------|
| `go build ./...` | 无错误 |
| `go test ./internal/script/...` | 全部通过 |
| 核心指令覆盖率 | ≥ 80% |
| `go fmt ./...` | 无变更 |
| `golangci-lint run` | 无警告 |
| `go test -race` | 无竞态 |
| 安全约束测试 | ErrCodeInjection, ErrUnlockOpcodeNotAllowed, ErrGotoDepthExceeded 均有专项测试 |
| 集成测试 | 标准支付、哈希屏障、时间锁全部通过 |

# 脚本虚拟机实现策略

## 一、整体架构

### 1.1 核心组件

```
┌─────────────────────────────────────────────────────────┐
│                      Script Engine                       │
├─────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  Lexer   │──▶│  Parser  │──▶│ Compiler │              │
│  │ (词法器) │  │ (解析器) │  │ (编译器) │              │
│  └──────────┘  └──────────┘  └──────────┘              │
│        │              │              │                  │
│        ▼              ▼              ▼                  │
│  ┌──────────────────────────────────────────┐          │
│  │            Bytecode (指令码序列)          │          │
│  └──────────────────────────────────────────┘          │
│                         │                               │
│                         ▼                               │
│  ┌──────────────────────────────────────────┐          │
│  │              VM (虚拟机)                  │          │
│  │  ┌────────┐ ┌────────┐ ┌────────┐       │          │
│  │  │ Stack  │ │  Args  │ │ Locals │       │          │
│  │  │ 数据栈 │ │ 实参区 │ │ 局部域 │       │          │
│  │  └────────┘ └────────┘ └────────┘       │          │
│  │  ┌────────┐ ┌────────┐ ┌────────┐       │          │
│  │  │Globals │ │ Import │ │ Export │       │          │
│  │  │ 全局域 │ │导入缓存│ │导出缓存│       │          │
│  │  └────────┘ └────────┘ └────────┘       │          │
│  └──────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────┘
```

### 1.2 执行流程

```
源码文本 ──▶ Token 流 ──▶ AST ──▶ 字节码 ──▶ 执行结果
   │           │          │         │          │
   │           │          │         │          ▼
   │           │          │         │     true/false
   └───────────┴──────────┴─────────┘     (通关状态)
              编译时                  运行时
```

---

## 二、指令编码格式

### 2.1 字节码结构

每条指令的编码格式：

```
┌──────────┬──────────────┬────────────────┐
│  Opcode  │   附参区     │    关联数据    │
│  (1字节) │  (变长)      │    (变长)      │
└──────────┴──────────────┴────────────────┘
```

### 2.2 指令类型分类

根据附参和关联数据的特征，指令可分为以下几类：

| 类型 | 特征 | 示例 |
|-----|------|------|
| 纯指令 | 无附参无数据 | `NIL`, `TRUE`, `POP`, `PASS` |
| 带附参 | 固定/变长附参 | `POPS(1)`, `IF{}(~)` |
| 带数据 | 携带关联数据 | `DATA{}`, `""`, `{}` |
| 复合型 | 附参 + 数据 | `STRING(1){}`, `GOTO(~,64,~)` |

### 2.3 附参编码规则

```go
// 附参长度标记
// (~) 变长整数 varint
// (1) 固定 1 字节
// (2) 固定 2 字节
// (64) 固定 64 字节

// 变长整数编码（类似 protobuf varint）
func EncodeVarint(n int64) []byte
func DecodeVarint(data []byte) (int64, int)
```

---

## 三、虚拟机数据结构

### 3.1 核心状态

```go
// internal/script/vm.go

type VM struct {
    // 执行状态
    code     []byte    // 当前执行的字节码
    pc       int       // 程序计数器
    passed   bool      // 通关状态
    finished bool      // 是否结束

    // 数据空间
    stack    *Stack    // 数据栈（≤256 条目）
    args     *ArgQueue // 实参区（FIFO 队列）
    locals   *Locals   // 局部域栈（每层 ≤128）
    globals  *Globals  // 全局域

    // 交互缓存
    importBuf *Buffer  // 导入缓存区
    exportBuf *Buffer  // 导出缓存区

    // 执行上下文
    env       *Environment  // 环境信息（交易、区块等）
    loopVars  *LoopContext  // 循环变量上下文
    ifState   []bool        // IF 状态栈

    // 资源限制
    gasUsed   int64         // 已消耗 gas
    maxGas    int64         // 最大 gas
    depth     int           // 嵌入/跳转深度
}
```

### 3.2 数据栈实现

```go
// internal/script/stack.go

type Stack struct {
    items []any
    max   int  // 最大 256
}

func (s *Stack) Push(v any) error {
    if len(s.items) >= s.max {
        return ErrStackOverflow
    }
    s.items = append(s.items, v)
    return nil
}

func (s *Stack) Pop() (any, error) {
    if len(s.items) == 0 {
        return nil, ErrStackUnderflow
    }
    v := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return v, nil
}

func (s *Stack) Peek(offset int) (any, error)      // 引用不弹出
func (s *Stack) PeekN(n int) ([]any, error)        // 引用多项
func (s *Stack) PopN(n int) ([]any, error)         // 弹出多项
func (s *Stack) Top() (any, error)                 // 栈顶项
func (s *Stack) Size() int                         // 栈高度
```

### 3.3 实参区实现

```go
// internal/script/args.go

type ArgQueue struct {
    items []any
}

// 添加条目（持续添加）
func (a *ArgQueue) Add(v any) {
    a.items = append(a.items, v)
}

// 展开添加多项
func (a *ArgQueue) Extend(vs []any) {
    a.items = append(a.items, vs...)
}

// 一次性取出全部并清空
func (a *ArgQueue) Drain() []any {
    items := a.items
    a.items = nil
    return items
}

// 是否有值
func (a *ArgQueue) HasValues() bool {
    return len(a.items) > 0
}
```

### 3.4 局部域实现

```go
// internal/script/locals.go

// 每个语法块的局部域
type LocalScope struct {
    items []any  // 最大 128
}

// 局部域栈（嵌套语法块）
type Locals struct {
    scopes []*LocalScope
}

func (l *Locals) Push() *LocalScope     // 进入新块
func (l *Locals) Pop()                  // 离开块
func (l *Locals) Current() *LocalScope  // 当前块
func (l *Locals) Set(v any) error       // 添加到当前
func (l *Locals) Get(idx int) (any, error) // 按索引取值
```

---

## 四、指令分组实现策略

### 4.1 值指令（Opcode 0-19）

**实现重点**：解析附参和关联数据，构造具体类型值

```go
// 值指令处理器
var valueHandlers = map[Opcode]func(vm *VM) error{
    OP_NIL:    opNil,     // => nil
    OP_TRUE:   opTrue,    // => true
    OP_FALSE:  opFalse,   // => false
    OP_BYTE:   opByte,    // 读1字节 => byte
    OP_RUNE:   opRune,    // 读4字节 => rune
    OP_INT:    opInt,     // 读变长 => int64
    OP_BIGINT: opBigInt,  // 读附参长度 + 数据 => *big.Int
    OP_FLOAT:  opFloat,   // 读8字节 => float64
    OP_STRING_SHORT: opStringShort, // 附参1字节长度
    OP_STRING_LONG:  opStringLong,  // 附参2字节长度
    OP_VALUESET:     opValueSet,    // 值集 {1, 2, 3}
    OP_DATA_SHORT:   opDataShort,   // 小字节序列
    OP_DATA_LONG:    opDataLong,    // 大字节序列
    OP_REGEXP:       opRegexp,      // 正则表达式
    OP_DATE:         opDate,        // 时间对象
    OP_CODE:         opCode,        // 代码/脚本
    OP_SCRIPT:       opScript,      // 脚本引用
    OP_VALUE:        opValue,       // 通用取值
}

// 示例：整数值处理
func opInt(vm *VM) error {
    val, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    return vm.stack.Push(val)
}

// 示例：DATA{} 处理
func opDataShort(vm *VM) error {
    length := int(vm.code[vm.pc])
    vm.pc++
    data := make([]byte, length)
    copy(data, vm.code[vm.pc:vm.pc+length])
    vm.pc += length
    return vm.stack.Push(data)
}
```

**类型提升规则**（值集内）：

```go
// 类型提升优先级：Byte < Rune < Int < BigInt
//                 Byte < Rune < Int < Float
func promoteTypes(items []any) ([]any, error) {
    maxType := detectMaxType(items)
    result := make([]any, len(items))
    for i, item := range items {
        result[i] = convertTo(item, maxType)
    }
    return result, nil
}
```

---

### 4.2 截取指令（Opcode 20-24）

**实现重点**：修改下一指令的返回值目标

```go
// 截取模式
type InterceptMode int
const (
    InterceptNone   InterceptMode = iota
    InterceptToArgs               // @ => 实参区
    InterceptDirect               // ~ => 跳过实参区
    InterceptToLocal              // $ => 局部域
)

// 截取指令不直接执行，而是设置标记
func opIntercept(vm *VM) error {
    vm.interceptMode = InterceptToArgs
    return nil
}

func opLocalStore(vm *VM) error {
    vm.interceptMode = InterceptToLocal
    return nil
}

func opLocalGet(vm *VM) error {
    idx := int(int8(vm.code[vm.pc])) // 有符号，支持负数
    vm.pc++
    val, err := vm.locals.Get(idx)
    if err != nil {
        return err
    }
    vm.args.Add(val)  // 放入实参区
    return nil
}

func opLoopVar(vm *VM) error {
    varType := vm.code[vm.pc]
    vm.pc++
    var val any
    switch varType {
    case 0: val = vm.loopVars.Value
    case 1: val = vm.loopVars.Index
    case 2: val = vm.loopVars.Collection
    case 3: val = vm.loopVars.Size
    }
    vm.args.Add(val)
    return nil
}
```

**返回值分发逻辑**：

```go
// 指令执行后的返回值处理
func (vm *VM) dispatchResult(results []any) error {
    switch vm.interceptMode {
    case InterceptToArgs:
        vm.args.Extend(results)
    case InterceptToLocal:
        for _, r := range results {
            vm.locals.Current().Add(r)
        }
    default:
        // 默认入栈
        for _, r := range results {
            if err := vm.stack.Push(r); err != nil {
                return err
            }
        }
    }
    vm.interceptMode = InterceptNone
    return nil
}
```

---

### 4.3 栈操作指令（Opcode 25-35）

**实现重点**：精确处理附参定义的数量，区分引用和弹出

```go
func opPop(vm *VM) error {
    val, err := vm.stack.Pop()
    if err != nil {
        return err
    }
    return vm.dispatchResult([]any{val})
}

func opPops(vm *VM) error {
    n := int(vm.code[vm.pc])
    vm.pc++
    if n == 0 {
        n = vm.stack.Size() // 0 表示全部
    }
    vals, err := vm.stack.PopN(n)
    if err != nil {
        return err
    }
    return vm.dispatchResult(vals) // 自动展开
}

func opTop(vm *VM) error {
    val, err := vm.stack.Top()
    if err != nil {
        return err
    }
    return vm.dispatchResult([]any{val})
}

func opShift(vm *VM) error {
    n := int(vm.code[vm.pc])
    vm.pc++
    if n == 0 {
        n = vm.stack.Size()
    }
    vals, err := vm.stack.PopN(n)
    if err != nil {
        return err
    }
    // SHIFT 返回切片而非展开
    return vm.dispatchResult([]any{vals})
}

func opClone(vm *VM) error {
    n := int(vm.code[vm.pc])
    vm.pc++
    if n == 0 {
        n = vm.stack.Size()
    }
    vals, err := vm.stack.PeekN(n) // 引用不弹出
    if err != nil {
        return err
    }
    cloned := shallowCopy(vals)
    return vm.dispatchResult([]any{cloned})
}
```

---

### 4.4 集合指令（Opcode 36-45）

**实现重点**：切片/字典的基本操作

```go
func opSlice(vm *VM) error {
    size := int(vm.code[vm.pc])
    vm.pc++
    
    args := vm.getArgs(2) // 取2个实参
    slice := args[0].([]any)
    start := toInt(args[1])
    
    // 支持负数索引
    if start < 0 {
        start = len(slice) + start
    }
    
    end := start + size
    if size == 0 {
        end = len(slice)
    }
    
    if start < 0 || end > len(slice) {
        return ErrIndexOutOfRange
    }
    
    result := slice[start:end]
    return vm.dispatchResult([]any{result})
}

func opSpread(vm *VM) error {
    args := vm.getArgs(1)
    slice := args[0].([]any)
    return vm.dispatchResult(slice) // 展开为多项
}

func opMerge(vm *VM) error {
    args := vm.getArgs(-1) // 不定数量
    var result []any
    for _, arg := range args {
        if s, ok := arg.([]any); ok {
            result = append(result, s...)
        }
    }
    return vm.dispatchResult([]any{result})
}
```

---

### 4.5 结果指令（Opcode 50-56）

**实现重点**：控制脚本执行流和通关状态

```go
func opPass(vm *VM) error {
    args := vm.getArgs(1)
    val := toBool(args[0])
    if val {
        vm.passed = true
        return nil
    }
    // 失败立即结束
    vm.passed = false
    vm.finished = true
    return nil
}

func opCheck(vm *VM) error {
    args := vm.getArgs(1)
    val := toBool(args[0])
    vm.passed = val // 覆盖状态，但不退出
    return nil
}

func opExit(vm *VM) error {
    vm.finished = true
    // 可选返回值
    if vm.args.HasValues() {
        args := vm.args.Drain()
        return vm.dispatchResult(args)
    }
    return nil
}

func opGoto(vm *VM) error {
    if vm.jumpDepth >= MaxJumpDepth {
        return ErrMaxJumpDepthExceeded
    }
    
    // 读取附参
    year, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    txID := vm.code[vm.pc : vm.pc+64]
    vm.pc += 64
    outIndex, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    // 获取目标脚本
    script, err := vm.env.FetchScript(int(year), txID, int(outIndex))
    if err != nil {
        return err
    }
    
    // 传递实参作为新脚本的初始栈
    initStack := vm.args.Drain()
    
    // 创建新 VM 执行
    newVM := NewVM(script, vm.env)
    newVM.jumpDepth = vm.jumpDepth + 1
    for _, v := range initStack {
        newVM.stack.Push(v)
    }
    
    err = newVM.Execute()
    vm.passed = newVM.passed
    vm.finished = true
    return err
}

func opEmbed(vm *VM) error {
    if vm.embedCount >= MaxEmbedCount {
        return ErrMaxEmbedCountExceeded
    }
    
    // 读取附参（同 GOTO）
    year, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    txID := vm.code[vm.pc : vm.pc+64]
    vm.pc += 64
    outIndex, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    script, err := vm.env.FetchScript(int(year), txID, int(outIndex))
    if err != nil {
        return err
    }
    
    // 嵌入执行：共享当前 VM 状态
    savedPC := vm.pc
    savedCode := vm.code
    
    vm.code = script
    vm.pc = 0
    vm.embedCount++
    
    err = vm.run() // 执行嵌入脚本
    
    vm.code = savedCode
    vm.pc = savedPC
    return err
}
```

---

### 4.6 流程控制（Opcode 57-66）

**实现重点**：子语句块的跳过与执行

```go
func opIf(vm *VM) error {
    // 读取块长度
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    // 取条件
    args := vm.getArgs(1)
    cond := toBool(args[0])
    
    // 记录 IF 状态（供 ELSE 使用）
    vm.pushIfState(cond)
    
    if cond {
        // 执行块内代码
        vm.locals.Push() // 新建局部域
        err := vm.executeBlock(int(blockLen))
        vm.locals.Pop()
        return err
    }
    // 跳过块
    vm.pc += int(blockLen)
    return nil
}

func opElse(vm *VM) error {
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    ifState := vm.popIfState()
    
    if !ifState { // IF 为假时执行 ELSE
        vm.locals.Push()
        err := vm.executeBlock(int(blockLen))
        vm.locals.Pop()
        return err
    }
    vm.pc += int(blockLen)
    return nil
}

func opEach(vm *VM) error {
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    blockStart := vm.pc
    
    args := vm.getArgs(1)
    collection := args[0]
    
    switch c := collection.(type) {
    case []any:
        for i, v := range c {
            vm.loopVars = &LoopContext{
                Value:      v,
                Index:      i,
                Collection: c,
                Size:       len(c),
            }
            vm.pc = blockStart
            vm.locals.Push()
            
            err := vm.executeBlock(int(blockLen))
            vm.locals.Pop()
            
            if vm.breakFlag {
                vm.breakFlag = false
                break
            }
            if vm.continueFlag {
                vm.continueFlag = false
                continue
            }
            if err != nil {
                return err
            }
        }
    case map[string]any:
        keys := getSortedKeys(c)
        for i, k := range keys {
            vm.loopVars = &LoopContext{
                Value:      c[k],
                Index:      k, // 字典用键名
                Collection: c,
                Size:       len(c),
            }
            // ... 同上
        }
    }
    
    vm.pc = blockStart + int(blockLen)
    return nil
}

func opBreak(vm *VM) error {
    // 可选条件
    if vm.args.HasValues() {
        args := vm.args.Drain()
        if !toBool(args[0]) {
            return nil
        }
    }
    vm.breakFlag = true
    return nil
}

func opSwitch(vm *VM) error {
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    args := vm.getArgs(2)
    target := args[0]       // 标的值
    caseValues := args[1].([]any) // 分支值列表
    
    vm.switchContext = &SwitchContext{
        Target:     target,
        CaseValues: caseValues,
        CaseIndex:  0,
        Matched:    false,
    }
    
    vm.locals.Push()
    err := vm.executeBlock(int(blockLen))
    vm.locals.Pop()
    vm.switchContext = nil
    return err
}

func opCase(vm *VM) error {
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    ctx := vm.switchContext
    if ctx.Matched {
        // 已有 CASE 匹配，跳过
        vm.pc += int(blockLen)
        return nil
    }
    
    // 取对应位置的分支值比较
    caseVal := ctx.CaseValues[ctx.CaseIndex]
    ctx.CaseIndex++
    
    if equal(ctx.Target, caseVal) {
        ctx.Matched = true
        err := vm.executeBlock(int(blockLen))
        return err
    }
    vm.pc += int(blockLen)
    return nil
}
```

---

### 4.7 转换指令（Opcode 67-79）

**实现重点**：类型安全转换，边界检查

```go
func opBool(vm *VM) error {
    args := vm.getArgs(1)
    result := toBoolValue(args[0])
    return vm.dispatchResult([]any{result})
}

func toBoolValue(v any) bool {
    switch x := v.(type) {
    case nil:
        return false
    case bool:
        return x
    case byte:
        return x != 0
    case rune:
        return x != 0
    case int64:
        return x != 0
    case *big.Int:
        return x.Sign() != 0
    case float64:
        return x > math.SmallestNonzeroFloat64
    case string:
        return x != ""
    default:
        return true
    }
}

func opInt(vm *VM) error {
    args := vm.getArgs(1)
    result, err := toIntValue(args[0])
    if err != nil {
        return err
    }
    return vm.dispatchResult([]any{result})
}

func toIntValue(v any) (int64, error) {
    switch x := v.(type) {
    case nil:
        return 0, nil
    case bool:
        if x { return 1, nil }
        return 0, nil
    case byte:
        return int64(x), nil
    case rune:
        return int64(x), nil
    case int64:
        return x, nil
    case *big.Int:
        if !x.IsInt64() {
            return 0, ErrIntOverflow
        }
        return x.Int64(), nil
    case float64:
        return int64(x), nil
    case string:
        return strconv.ParseInt(x, 0, 64)
    default:
        return 0, ErrInvalidConversion
    }
}

func opString(vm *VM) error {
    // 附参：进制或格式
    format := vm.code[vm.pc]
    vm.pc++
    
    args := vm.getArgs(1)
    result := toStringValue(args[0], format)
    return vm.dispatchResult([]any{result})
}

func opDict(vm *VM) error {
    args := vm.getArgs(2)
    keys := args[0].([]string)
    vals := args[1].([]any)
    
    dict := make(map[string]any)
    for i, k := range keys {
        if i < len(vals) {
            dict[k] = vals[i]
        } else {
            dict[k] = nil
        }
    }
    return vm.dispatchResult([]any{dict})
}
```

---

### 4.8 运算指令（Opcode 80-103）

**实现重点**：表达式解析与求值

```go
// 表达式求值器
func opExpression(vm *VM) error {
    exprLen := int(vm.code[vm.pc])
    vm.pc++
    
    exprEnd := vm.pc + exprLen
    result, err := vm.evaluateExpression(exprEnd)
    if err != nil {
        return err
    }
    return vm.dispatchResult([]any{result})
}

func (vm *VM) evaluateExpression(end int) (float64, error) {
    // 使用调度场算法 (Shunting Yard) 解析中缀表达式
    var output []float64
    var operators []Opcode
    
    for vm.pc < end {
        op := Opcode(vm.code[vm.pc])
        vm.pc++
        
        switch op {
        case OP_INT, OP_FLOAT, OP_BYTE:
            // 操作数
            val := vm.readValue(op)
            output = append(output, toFloat(val))
            
        case OP_POP, OP_TOP, OP_PEEK:
            // 从栈取值
            result, _ := vm.executeOp(op)
            output = append(output, toFloat(result[0]))
            
        case OP_LPAREN: // (
            operators = append(operators, op)
            
        case OP_RPAREN: // )
            for len(operators) > 0 && operators[len(operators)-1] != OP_LPAREN {
                output = applyOp(output, operators[len(operators)-1])
                operators = operators[:len(operators)-1]
            }
            operators = operators[:len(operators)-1] // 移除 (
            
        case OP_MUL, OP_DIV, OP_MOD, OP_ADD, OP_SUB:
            for len(operators) > 0 && precedence(operators[len(operators)-1]) >= precedence(op) {
                output = applyOp(output, operators[len(operators)-1])
                operators = operators[:len(operators)-1]
            }
            operators = append(operators, op)
        }
    }
    
    for len(operators) > 0 {
        output = applyOp(output, operators[len(operators)-1])
        operators = operators[:len(operators)-1]
    }
    
    return output[0], nil
}

// 运算优先级
func precedence(op Opcode) int {
    switch op {
    case OP_MUL, OP_DIV, OP_MOD:
        return 2
    case OP_ADD, OP_SUB:
        return 1
    default:
        return 0
    }
}

// 命名运算指令
func opAdd(vm *VM) error {
    args := vm.getArgs(2)
    
    // 类型检测，支持多种类型
    switch a := args[0].(type) {
    case int64:
        b := args[1].(int64)
        return vm.dispatchResult([]any{a + b})
    case float64:
        b := toFloat(args[1])
        return vm.dispatchResult([]any{a + b})
    case string:
        b := args[1].(string)
        return vm.dispatchResult([]any{a + b})
    case []byte:
        b := args[1].([]byte)
        result := make([]byte, len(a)+len(b))
        copy(result, a)
        copy(result[len(a):], b)
        return vm.dispatchResult([]any{result})
    }
    return ErrInvalidOperands
}

func opLmov(vm *VM) error {
    args := vm.getArgs(2)
    x := args[0].(int64)
    y := args[1].(int64)
    return vm.dispatchResult([]any{x << y})
}
```

---

### 4.9 比较与逻辑指令（Opcode 104-119）

```go
func opEqual(vm *VM) error {
    args := vm.getArgs(2)
    result := deepEqual(args[0], args[1])
    return vm.dispatchResult([]any{result})
}

func opLt(vm *VM) error {
    args := vm.getArgs(2)
    result := compare(args[0], args[1]) < 0
    return vm.dispatchResult([]any{result})
}

func compare(a, b any) int {
    switch x := a.(type) {
    case int64:
        y := b.(int64)
        if x < y { return -1 }
        if x > y { return 1 }
        return 0
    case float64:
        y := toFloat(b)
        if x < y { return -1 }
        if x > y { return 1 }
        return 0
    case []byte:
        y := b.([]byte)
        return bytes.Compare(x, y)
    case string:
        y := b.(string)
        return strings.Compare(x, y)
    }
    panic("incomparable types")
}

func opWithin(vm *VM) error {
    args := vm.getArgs(2)
    val := args[0]
    rangeSlice := args[1].([]any)
    min, max := rangeSlice[0], rangeSlice[1]
    
    result := compare(val, min) >= 0 && compare(val, max) < 0
    return vm.dispatchResult([]any{result})
}

func opBoth(vm *VM) error {
    args := vm.getArgs(-1) // 不定数量
    for _, arg := range args {
        if !toBool(arg) {
            return vm.dispatchResult([]any{false})
        }
    }
    return vm.dispatchResult([]any{true})
}

func opEither(vm *VM) error {
    args := vm.getArgs(-1)
    for _, arg := range args {
        if toBool(arg) {
            return vm.dispatchResult([]any{true})
        }
    }
    return vm.dispatchResult([]any{false})
}
```

---

### 4.10 系统指令（SYS_*）

**核心：内置验证 SYS_CHKPASS**

```go
func opSysChkpass(vm *VM) error {
    // 从栈取参数
    args := vm.getArgs(3) // 类型、标记、签名、公钥
    
    sigType := toInt(args[0])
    
    switch sigType {
    case 1: // 单签名
        return vm.verifySingleSig()
    case 2: // 多重签名
        return vm.verifyMultiSig()
    }
    return ErrInvalidSignatureType
}

func (vm *VM) verifySingleSig() error {
    // 从环境取实参
    flag := vm.env.Get("flag").(byte)
    sig := vm.env.Get("sig").([]byte)
    pubKey := vm.env.Get("pubKey").([]byte)
    receiver := vm.env.Get("receiver").(Address)
    
    // 1. 验证公钥哈希 == 接收者
    pubHash := Hash512(pubKey)[:AddressLength]
    if !bytes.Equal(pubHash, receiver[:]) {
        vm.passed = false
        vm.finished = true
        return nil
    }
    
    // 2. 根据 flag 构造签名消息
    message := vm.buildSignMessage(flag)
    
    // 3. 验证签名
    valid := crypto.Verify(pubKey, message, sig)
    vm.passed = valid
    if !valid {
        vm.finished = true
    }
    return nil
}

func (vm *VM) verifyMultiSig() error {
    // 解析 m/n 配置
    sigs := vm.env.Get("sigs").([][]byte)
    pubKeys := vm.env.Get("pubKeys").([][]byte)
    supplements := vm.env.Get("supplements").([][]byte) // 补全集
    receiver := vm.env.Get("receiver").(Address)
    
    m := len(pubKeys)
    n := m + len(supplements)
    
    // 1. 计算复合公钥哈希
    var pubHashes [][]byte
    for _, pk := range pubKeys {
        pubHashes = append(pubHashes, Hash512(pk)[:AddressLength])
    }
    pubHashes = append(pubHashes, supplements...)
    sort.Slice(pubHashes, func(i, j int) bool {
        return bytes.Compare(pubHashes[i], pubHashes[j]) < 0
    })
    
    prefix := []byte{byte(m), byte(n)}
    combined := append(prefix, bytes.Join(pubHashes, nil)...)
    compositeHash := Hash512(combined)[:AddressLength]
    
    // 2. 验证复合哈希 == 接收者
    if !bytes.Equal(compositeHash, receiver[:]) {
        vm.passed = false
        vm.finished = true
        return nil
    }
    
    // 3. 验证各个签名
    message := vm.buildSignMessage(vm.env.Get("flag").(byte))
    for i, sig := range sigs {
        if !crypto.Verify(pubKeys[i], message, sig) {
            vm.passed = false
            vm.finished = true
            return nil
        }
    }
    
    vm.passed = true
    return nil
}

func opSysTime(vm *VM) error {
    timeType := vm.code[vm.pc]
    vm.pc++
    
    var result any
    switch timeType {
    case 0: // Stamp - 当前区块时间戳
        result = vm.env.BlockTimestamp()
    case 1: // Height - 当前区块高度
        result = vm.env.BlockHeight()
    // ... 其他时间相关
    }
    return vm.dispatchResult([]any{result})
}
```

---

### 4.11 函数指令（FN_*）

```go
func opFnHash256(vm *VM) error {
    args := vm.getArgs(1)
    data := toBytes(args[0])
    hash := crypto.BLAKE3(data)
    return vm.dispatchResult([]any{hash})
}

func opFnHash512(vm *VM) error {
    args := vm.getArgs(1)
    data := toBytes(args[0])
    hash := crypto.SHA512(data)
    return vm.dispatchResult([]any{hash})
}

func opFnPubhash(vm *VM) error {
    args := vm.getArgs(1)
    pubKey := args[0].([]byte)
    hash := crypto.SHA512(pubKey)[:AddressLength]
    return vm.dispatchResult([]any{hash})
}

func opFnChecksig(vm *VM) error {
    args := vm.getArgs(2)
    sig := args[0].([]byte)
    pubKey := args[1].([]byte)
    
    // 默认对整个交易签名
    message := vm.env.TxSignatureMessage()
    valid := crypto.Verify(pubKey, message, sig)
    return vm.dispatchResult([]any{valid})
}

func opFnBase32(vm *VM) error {
    mode := vm.code[vm.pc]
    vm.pc++
    
    args := vm.getArgs(1)
    
    switch mode {
    case 0: // 编码
        data := args[0].([]byte)
        result := base32.Encode(data)
        return vm.dispatchResult([]any{result})
    case 1: // 解码
        str := args[0].(string)
        result, err := base32.Decode(str)
        if err != nil {
            return err
        }
        return vm.dispatchResult([]any{result})
    }
    return ErrInvalidMode
}
```

---

## 五、实参获取策略

**核心逻辑**：先从实参区取，不足则从栈取

```go
// 获取指定数量的实参
func (vm *VM) getArgs(n int) []any {
    if n == -1 {
        // 不定数量：仅取实参区
        return vm.args.Drain()
    }
    
    if n == 0 {
        return nil
    }
    
    // 检查实参区
    if vm.directMode {
        // ~ 模式：跳过实参区，直接从栈取
        vm.directMode = false
        vals, _ := vm.stack.PopN(n)
        return vals
    }
    
    argsFromQueue := vm.args.Drain()
    
    if len(argsFromQueue) == n {
        return argsFromQueue
    }
    
    if len(argsFromQueue) > 0 && len(argsFromQueue) != n {
        // 实参区有值但数量不匹配，报错
        panic("args count mismatch")
    }
    
    // 实参区为空，从栈取
    vals, _ := vm.stack.PopN(n)
    return vals
}
```

---

## 六、编译器实现要点

### 6.1 词法分析

```go
// internal/script/lexer.go

type TokenType int
const (
    TOKEN_EOF TokenType = iota
    TOKEN_IDENT         // 指令名
    TOKEN_INT           // 整数
    TOKEN_FLOAT         // 浮点数
    TOKEN_STRING        // 字符串
    TOKEN_BYTES         // 字节序列
    TOKEN_LBRACE        // {
    TOKEN_RBRACE        // }
    TOKEN_LBRACKET      // [
    TOKEN_RBRACKET      // ]
    TOKEN_LPAREN        // (
    TOKEN_RPAREN        // )
    TOKEN_AT            // @
    TOKEN_DOLLAR        // $
    TOKEN_TILDE         // ~
    // ...
)

type Token struct {
    Type    TokenType
    Value   string
    Line    int
    Column  int
}

type Lexer struct {
    input   string
    pos     int
    line    int
    column  int
}

func (l *Lexer) NextToken() Token
```

### 6.2 语法分析与编译

```go
// internal/script/compiler.go

type Compiler struct {
    lexer   *Lexer
    output  []byte
}

func (c *Compiler) Compile() ([]byte, error) {
    for {
        tok := c.lexer.NextToken()
        if tok.Type == TOKEN_EOF {
            break
        }
        
        if err := c.compileToken(tok); err != nil {
            return nil, err
        }
    }
    return c.output, nil
}

func (c *Compiler) compileToken(tok Token) error {
    switch tok.Type {
    case TOKEN_IDENT:
        return c.compileInstruction(tok.Value)
    case TOKEN_INT:
        return c.compileInt(tok.Value)
    case TOKEN_STRING:
        return c.compileString(tok.Value)
    // ...
    }
    return nil
}

func (c *Compiler) compileInstruction(name string) error {
    opcode, ok := instructionMap[name]
    if !ok {
        return fmt.Errorf("unknown instruction: %s", name)
    }
    
    c.output = append(c.output, byte(opcode))
    
    // 处理附参和关联数据
    switch opcode {
    case OP_IF, OP_ELSE, OP_EACH:
        return c.compileBlock()
    case OP_DATA_SHORT, OP_DATA_LONG:
        return c.compileData()
    // ...
    }
    return nil
}
```

---

## 七、安全边界检查

```go
// 资源限制常量
const (
    MaxStackHeight  = 256
    MaxStackItem    = 1024
    MaxLocalScope   = 128
    MaxLockScript   = 1024
    MaxUnlockScript = 4096
    MaxEmbedCount   = 5
    MaxJumpDepth    = 3
    MaxJumpCount    = 2
)

// 栈项大小检查
func (vm *VM) checkItemSize(v any) error {
    size := estimateSize(v)
    if size > MaxStackItem {
        return ErrStackItemTooLarge
    }
    return nil
}

// 脚本长度检查
func ValidateScriptLength(script []byte, isUnlock bool) error {
    maxLen := MaxLockScript
    if isUnlock {
        maxLen = MaxUnlockScript
    }
    if len(script) > maxLen {
        return ErrScriptTooLong
    }
    return nil
}

// 解锁脚本指令限制
func ValidateUnlockOpcodes(script []byte) error {
    for i := 0; i < len(script); {
        op := Opcode(script[i])
        if op > OP_OUTPUT {  // 超出值指令~交互指令范围
            return ErrInvalidUnlockOpcode
        }
        i += instructionSize(script, i)
    }
    return nil
}
```

---

## 八、测试策略

### 8.1 指令单元测试

```go
func TestOpPop(t *testing.T) {
    vm := NewVM(nil, nil)
    vm.stack.Push(int64(100))
    vm.stack.Push(int64(200))
    
    err := opPop(vm)
    require.NoError(t, err)
    require.Equal(t, 1, vm.stack.Size())
    
    // 验证返回值在默认情况下入栈
    top, _ := vm.stack.Top()
    require.Equal(t, int64(200), top)
}

func TestOpIf(t *testing.T) {
    // 编译脚本：TRUE IF { {100} } ELSE { {200} }
    code := compile("TRUE IF { {100} } ELSE { {200} }")
    
    vm := NewVM(code, nil)
    err := vm.Execute()
    
    require.NoError(t, err)
    top, _ := vm.stack.Top()
    require.Equal(t, int64(100), top)
}
```

### 8.2 集成测试

```go
func TestStandardPayment(t *testing.T) {
    // 模拟标准支付验证
    unlockScript := compile(`
        {1}
        DATA{...签名...}
        DATA{...公钥...}
    `)
    
    lockScript := compile(`SYS_CHKPASS`)
    
    env := &MockEnv{
        Receiver: testAddress,
        // ...
    }
    
    vm := NewVM(append(unlockScript, lockScript...), env)
    err := vm.Execute()
    
    require.NoError(t, err)
    require.True(t, vm.Passed())
}
```


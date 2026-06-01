package script

import "fmt"

// Environment 是脚本 VM 运行时环境的注入接口。
// 包含交易、区块、见证等上下文数据；由外部（internal/tx 等）实现并注入。
// 此接口在 Task 6 中扩展。
// 参考：DEC-0503
type Environment interface {
	// Lookup 查询环境变量（BlockTime、TxTime 等字段名）。
	Lookup(name string) (Value, error)
}

// execFunc 是指令执行函数类型。
// vm 为当前 VM 上下文，f 为已解码的指令帧（含附参和关联数据）。
type execFunc func(vm *VM, f *InstrFrame) error

// execTable 是按 opcode 索引的执行函数表，由各 instr_*.go 文件在 init() 中注册。
var execTable [256]execFunc

// registerExec 向执行函数表注册 opcode 对应的执行函数。
// 重复注册会 panic（与 registry.go 的 Register 行为一致）。
func registerExec(op Opcode, fn execFunc) {
	if execTable[op] != nil {
		panic(fmt.Sprintf("script: execTable[%d] already registered", op))
	}
	execTable[op] = fn
}

// VM 是脚本执行虚拟机，持有完整运行时状态。
// 参考：docs/proposal/10.Script-System.md §4，DEC-0505。
type VM struct {
	// 数据栈（LIFO）
	stack *Stack
	// 实参区（FIFO）
	args *ArgsArea
	// 执行状态（六态状态机）
	state ExecState
	// 执行模式（公共/私有）
	mode ExecMode
	// 通关状态（初始 true；PASS/CHECK 写入；END 时以此为结果）
	passState bool
	// 成本预算
	budget *Budget
	// INPUT 缓冲区（外部数据注入，私有路径使用）
	inputBuf []Value
	// OUTPUT 缓冲区（OUTPUT 指令写入，外部读取）
	outputBuf []Value
	// 环境（由外部注入，可为 nil）
	env Environment
	// 全局变量（VAR/SETVAR，0-255）
	globalVars [256]Value
	// 签名验证结果标注（SIGNED 查询）
	signedFlags [256]bool
	// 签名验证接口（FN_CHECKSIG/FN_MCHECKSIG）
	sigChecker SignatureChecker
	// 见证数据接口（SYS_CHKPASS）
	witness WitnessProvider
	// passedPublicEnd 标记公共路径是否已结束（END/INPUT 触达后为 true）
	passedPublicEnd bool
}

// VMOption 是 VM 构造选项。
type VMOption func(*VM)

// WithEnvironment 注入运行时环境。
func WithEnvironment(env Environment) VMOption {
	return func(vm *VM) { vm.env = env }
}

// WithInputBuffer 注入 INPUT 缓冲区（私有路径数据）。
func WithInputBuffer(vals []Value) VMOption {
	return func(vm *VM) {
		vm.inputBuf = make([]Value, len(vals))
		copy(vm.inputBuf, vals)
	}
}

// WithBudget 设置成本预算。
func WithBudget(b *Budget) VMOption {
	return func(vm *VM) { vm.budget = b }
}

// WithMode 设置初始执行模式（默认公共模式）。
func WithMode(mode ExecMode) VMOption {
	return func(vm *VM) { vm.mode = mode }
}

// WithSignatureChecker 注入签名验证器。
func WithSignatureChecker(sc SignatureChecker) VMOption {
	return func(vm *VM) { vm.sigChecker = sc }
}

// WithWitnessProvider 注入见证数据提供者。
func WithWitnessProvider(wp WitnessProvider) VMOption {
	return func(vm *VM) { vm.witness = wp }
}

// GetGlobalVar 读取全局变量（供指令实现使用）。
func (vm *VM) GetGlobalVar(idx int) Value { return vm.globalVars[idx] }

// SetGlobalVar 写入全局变量（供指令实现使用）。
func (vm *VM) SetGlobalVar(idx int, v Value) { vm.globalVars[idx] = v }

// SetSigned 标注签名序位已通过验证。
func (vm *VM) SetSigned(idx int) { vm.signedFlags[idx] = true }

// GetSigned 查询签名序位是否已通过验证。
func (vm *VM) GetSigned(idx int) bool { return vm.signedFlags[idx] }

// PassedPublicEnd 返回公共路径是否已结束。
func (vm *VM) PassedPublicEnd() bool { return vm.passedPublicEnd }

// MarkPublicEnd 标记公共路径结束（由 END/INPUT 指令调用）。
func (vm *VM) MarkPublicEnd() { vm.passedPublicEnd = true }

func NewVM(opts ...VMOption) *VM {
	vm := &VM{
		stack:     NewStack(),
		args:      NewArgsArea(),
		state:     StateRunning,
		mode:      ModePublic,
		passState: true, // 初始通关状态为 true（DEC-0505）
		budget:    NewBudget(0),
	}
	for _, opt := range opts {
		opt(vm)
	}
	return vm
}

// State 返回当前执行状态。
func (vm *VM) State() ExecState { return vm.state }

// PassState 返回当前通关状态。
func (vm *VM) PassState() bool { return vm.passState }

// Mode 返回当前执行模式。
func (vm *VM) Mode() ExecMode { return vm.mode }

// Stack 返回数据栈的引用（测试和高级扩展使用）。
func (vm *VM) Stack() *Stack { return vm.stack }

// ArgsArea 返回实参区的引用。
func (vm *VM) ArgsArea() *ArgsArea { return vm.args }

// OutputBuffer 返回 OUTPUT 缓冲区的副本。
func (vm *VM) OutputBuffer() []Value {
	cp := make([]Value, len(vm.outputBuf))
	copy(cp, vm.outputBuf)
	return cp
}

// Run 执行已解码的指令帧序列，返回最终执行状态。
//
// 执行规则（DEC-0505）：
//   - 空脚本以 true 通关产生 PassStop（通过）。
//   - 公共路径：遇 END 或无数据 INPUT 时产生 PassStop，之后的帧不执行。
//   - PASS false：立即 VerifyFail；PASS true：继续。
//   - CHECK：后写覆盖通关状态，不终止。
//   - 禁用指令/SYS_TIME/EXT_PRIV 在公共路径触达即 ScriptError。
//   - 成本预算耗尽产生 CostFail。
func (vm *VM) Run(frames []InstrFrame) ExecState {
	// 空脚本：PassStop(true)
	if len(frames) == 0 {
		vm.state = StatePassStop
		return vm.state
	}

	for i := range frames {
		if vm.state.IsDone() {
			break
		}
		f := &frames[i]
		vm.executeFrame(f)
	}

	// 若循环结束仍在 Running，则以当前通关状态产生 PassStop
	if vm.state == StateRunning {
		vm.state = StatePassStop
	}
	return vm.state
}

// executeFrame 执行单条指令帧。
func (vm *VM) executeFrame(f *InstrFrame) {
	op := f.Op

	// 公共模式安全检查
	if err := checkPublicSafety(vm.mode, op); err != nil {
		vm.state = StateScriptError
		return
	}

	// 查找执行函数
	fn := execTable[op]
	if fn == nil {
		// 未注册执行函数，当作 ScriptError（尚未实现或系统保留）
		vm.state = StateScriptError
		return
	}

	// 执行指令
	if err := fn(vm, f); err != nil {
		vm.handleExecError(err)
		return
	}

	// 消费成本
	meta := Lookup(op)
	if meta != nil {
		cost := costForTier(meta.CostTier)
		if err := vm.budget.Consume(cost); err != nil {
			vm.state = StateCostFail
		}
	}

	// 执行后清空实参区（防止残留影响下一指令）
	vm.args.Clear()
}

// handleExecError 根据执行阶段错误设置对应状态。
func (vm *VM) handleExecError(err error) {
	switch err {
	case ErrCostExceeded:
		vm.state = StateCostFail
	case ErrDisabledInPublic, ErrSysTimeInPublic, ErrExtPrivInPublic,
		ErrGotoTargetMissing, ErrGotoDepthExceeded, ErrGotoCountExceeded,
		ErrEmbedDepthExceeded, ErrEmbedCountExceeded:
		vm.state = StateScriptError
	default:
		// 其他运行时错误（栈下溢、类型不符等）也视为 ScriptError
		vm.state = StateScriptError
	}
}

// getOneArg 从实参区取 1 个值；若实参区为空，从数据栈弹出 1 个值。
func (vm *VM) getOneArg() (Value, error) {
	if vm.args.Len() > 0 {
		return vm.args.Dequeue()
	}
	return vm.stack.Pop()
}

// getArgs 从实参区取 n 个值；若实参区为空，从数据栈弹出 n 个值。
// 固定数量模型：实参区非空时数量必须恰好为 n（DEC-0501）。
func (vm *VM) getArgs(n int) ([]Value, error) {
	if n == 0 {
		return nil, nil
	}
	if vm.args.Len() > 0 {
		if vm.args.Len() != n {
			return nil, fmt.Errorf("%w: need %d, got %d", ErrArgCountMismatch, n, vm.args.Len())
		}
		result := make([]Value, n)
		for i := range result {
			v, _ := vm.args.Dequeue()
			result[i] = v
		}
		return result, nil
	}
	// 从数据栈弹出（栈 LIFO：先弹出的是最后一个参数）
	result := make([]Value, n)
	for i := n - 1; i >= 0; i-- {
		v, err := vm.stack.Pop()
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

package script

import (
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

// instr_tool.go 实现工具指令 [138-163] 的执行函数。
// 参考：docs/proposal/Instruction/14.Tool-Instructions.md。
// 注意：EVAL(138) 为前期禁用指令，已在 disabledOpcodes 中定义，不注册 execFunc，
// executor 以"未注册"将其转为 ScriptError。

func init() {
	// EVAL(138) 不注册（禁用指令，由 checkPublicSafety 及未注册路径处理）
	registerExec(COPY, execCOPY)
	registerExec(DCOPY, execDCOPY)
	registerExec(KEYVAL, execKEYVAL)
	registerExec(MATCH, execMATCH)
	registerExec(SUBSTR, execSUBSTR)
	registerExec(REPLACE, execREPLACE)
	registerExec(RANDOM, execRANDOM)
	registerExec(SLRAND, execSLRAND)
	registerExec(CMPFLO, execCMPFLO)
	// 148-151 不注册（保留）
	registerExec(RANGE, execRANGE)
	registerExec(MAP, execMAP)
	registerExec(FILTER, execFILTER)
	registerExec(SHELL, execSHELL)
	// 156-163 不注册（量子安全保留区）
}

// execCOPY 浅复制切片（COPY，opcode 139）。
func execCOPY(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	return vm.stack.Push(SliceValue(sl))
}

// execDCOPY 深复制切片（DCOPY，opcode 140）。占位：同浅复制。
func execDCOPY(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	return vm.stack.Push(SliceValue(sl))
}

// execKEYVAL 字典键值切取（KEYVAL，opcode 141）。
// 附参=切取类型(0=键值对,1=键,2=值)。占位：返回空 Slice。
func execKEYVAL(vm *VM, _ *InstrFrame) error {
	_, err := vm.getOneArg()
	if err != nil {
		return err
	}
	return vm.stack.Push(SliceValue([]Value{}))
}

// execMATCH 正则匹配（MATCH，opcode 142）。
// 实参1=String/Bytes，实参2=RegExp。
func execMATCH(vm *VM, f *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	target, reVal := args[0], args[1]

	// 检查 RegExp 类型
	if reVal.typ != TypeRegExp {
		return ErrTypeMismatch
	}
	re, ok := reVal.data.(*regexp.Regexp)
	if !ok || re == nil {
		return ErrTypeMismatch
	}

	// 附参=匹配方式（0=MatchString,1=FindString）
	var mode uint64
	if len(f.AttrParams) > 0 {
		mode = readULEB128Param(f.AttrParams[0])
	}

	var matched bool
	switch target.typ {
	case TypeString:
		s, _ := target.AsString()
		switch mode {
		case 0:
			matched = re.MatchString(s)
		default:
			matched = re.FindString(s) != ""
		}
	case TypeBytes:
		b, _ := target.AsBytes()
		switch mode {
		case 0:
			matched = re.Match(b)
		default:
			matched = re.Find(b) != nil
		}
	default:
		return ErrTypeMismatch
	}
	return vm.stack.Push(BoolValue(matched))
}

// execSUBSTR 截取 UTF-8 子字符串（SUBSTR，opcode 143）。
// 附参=字符数(ULEB128)；实参1=String，实参2=Int(起始字符位置)。
func execSUBSTR(vm *VM, f *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	s, err := args[0].AsString()
	if err != nil {
		return err
	}
	start, err := args[1].AsInt()
	if err != nil {
		return err
	}

	var charCount uint64
	if len(f.AttrParams) > 0 {
		charCount = readULEB128Param(f.AttrParams[0])
	}

	// 将字符串转为 rune 序列进行字符级操作
	runes := []rune(s)
	n := int64(len(runes))

	// 处理负索引：从末尾计算
	if start < 0 {
		start = n + start
	}
	if start < 0 {
		start = 0
	}
	if start >= n {
		return vm.stack.Push(StringValue(""))
	}

	end := start + int64(charCount)
	if charCount == 0 || end > n {
		end = n
	}
	return vm.stack.Push(StringValue(string(runes[start:end])))
}

// execREPLACE 字符串替换（REPLACE，opcode 144）。
// 附参=替换次数(ULEB128, 0=全部)；实参1=目标串，2=模式（String/RegExp），3=替换串。
func execREPLACE(vm *VM, f *InstrFrame) error {
	args, err := vm.getArgs(3)
	if err != nil {
		return err
	}
	target, pattern, repl := args[0], args[1], args[2]

	s, err := target.AsString()
	if err != nil {
		return err
	}
	replStr, err := repl.AsString()
	if err != nil {
		return err
	}

	var n int
	if len(f.AttrParams) > 0 {
		cnt := readULEB128Param(f.AttrParams[0])
		if cnt == 0 {
			n = -1 // 全部替换
		} else {
			n = int(cnt)
		}
	} else {
		n = -1
	}

	var result string
	switch pattern.typ {
	case TypeString:
		pat, _ := pattern.AsString()
		result = strings.Replace(s, pat, replStr, n)
	case TypeRegExp:
		re, ok := pattern.data.(*regexp.Regexp)
		if !ok || re == nil {
			return ErrTypeMismatch
		}
		if n == -1 {
			result = re.ReplaceAllString(s, replStr)
		} else {
			// regexp 包无 n 次替换，手动实现
			count := 0
			result = re.ReplaceAllStringFunc(s, func(m string) string {
				if count < n {
					count++
					return re.ReplaceAllString(m, replStr)
				}
				return m
			})
		}
	default:
		return ErrTypeMismatch
	}
	return vm.stack.Push(StringValue(result))
}

// execRANDOM 确定性随机数（RANDOM，opcode 145）。
// 确定性 PRNG（ChaCha8）未实现，占位返回 ScriptError。
func execRANDOM(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return ErrTypeMismatch // 当前不支持，待 Task 实现 PRNG
}

// execSLRAND 确定性切片乱序（SLRAND，opcode 146）。
// 同 RANDOM，占位。
func execSLRAND(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return ErrTypeMismatch
}

// execCMPFLO 浮点比较（带误差，CMPFLO，opcode 147）。
// 附参=类型(0=相等,-1=<=,1=>=)；实参1/2=Float，实参3=误差 Float。
func execCMPFLO(vm *VM, f *InstrFrame) error {
	args, err := vm.getArgs(3)
	if err != nil {
		return err
	}
	a, err2 := args[0].AsFloat()
	if err2 != nil {
		return err2
	}
	b, err2 := args[1].AsFloat()
	if err2 != nil {
		return err2
	}
	eps, err2 := args[2].AsFloat()
	if err2 != nil {
		return err2
	}

	var cmpType int64
	if len(f.AttrParams) > 0 {
		raw := readULEB128Param(f.AttrParams[0])
		cmpType = int64(raw)
	}

	var result bool
	diff := a - b
	switch cmpType {
	case 0: // 相等（|a-b| <= eps）
		result = math.Abs(diff) <= eps
	case ^int64(0): // -1（a <= b + eps）
		result = diff <= eps
	case 1: // >= （a >= b - eps）
		result = diff >= -eps
	default:
		result = math.Abs(diff) <= eps
	}
	return vm.stack.Push(BoolValue(result))
}

// execRANGE 创建数值序列（RANGE，opcode 152）。
// 附参=序列长度(1字节)；实参1=起始值(Int/Float)，实参2=步进(Int/Float)。
func execRANGE(vm *VM, f *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	var seqLen uint64
	if len(f.AttrParams) > 0 {
		seqLen = readULEB128Param(f.AttrParams[0])
	}
	if seqLen == 0 {
		return vm.stack.Push(SliceValue([]Value{}))
	}

	start, step := args[0], args[1]

	// 支持 Int 和 Float 两种模式
	if start.typ == TypeInt && step.typ == TypeInt {
		s, _ := start.AsInt()
		d, _ := step.AsInt()
		vals := make([]Value, seqLen)
		for i := uint64(0); i < seqLen; i++ {
			vals[i] = IntValue(s + int64(i)*d)
		}
		return vm.stack.Push(SliceValue(vals))
	}

	// 统一转 Float
	sf, err := toFloat(start)
	if err != nil {
		return err
	}
	df, err := toFloat(step)
	if err != nil {
		return err
	}
	vals := make([]Value, seqLen)
	for i := uint64(0); i < seqLen; i++ {
		vals[i] = FloatValue(sf + float64(i)*df)
	}
	return vm.stack.Push(SliceValue(vals))
}

// execMAP 映射迭代（MAP，opcode 153）。
// 附参=子块长；实参1=Slice。占位：返回空 Slice。
func execMAP(vm *VM, _ *InstrFrame) error {
	_, err := vm.getOneArg()
	if err != nil {
		return err
	}
	return vm.stack.Push(SliceValue([]Value{}))
}

// execFILTER 成员过滤（FILTER，opcode 154）。
// 占位：返回原 Slice。
func execFILTER(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	return vm.stack.Push(SliceValue(sl))
}

// execSHELL 执行 Shell（SHELL，opcode 155）。
// 公共节点忽略（消费实参，不执行），私有节点也占位忽略。
func execSHELL(vm *VM, _ *InstrFrame) error {
	vm.args.Clear()
	return nil
}

// ─── 辅助：统计 UTF-8 字节长度（未使用，保留用于后续实现）─────────────────
var _ = utf8.RuneCountInString // 确保 utf8 import 被使用

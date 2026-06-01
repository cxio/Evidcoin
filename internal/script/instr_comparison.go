package script

import (
	"bytes"
	"fmt"
	"math"
	"strings"
)

// instr_comparison.go 实现比较指令 [104-111] 的执行函数。
// 参考：docs/proposal/Instruction/10.Comparison-Instructions.md，DEC-0502。

func init() {
	registerExec(EQUAL, execEQUAL)
	registerExec(NEQUAL, execNEQUAL)
	registerExec(LT, execLT)
	registerExec(LTE, execLTE)
	registerExec(GT, execGT)
	registerExec(GTE, execGTE)
	registerExec(ISEFV, execISEFV)
	registerExec(WITHIN, execWITHIN)
}

// compareValues 比较两个值，返回 -1/0/1。
// 支持：Byte/Rune/Int/Float/String/Bytes；浮点含 NaN 返回错误；跨类型返回错误。
func compareValues(a, b Value) (int, error) {
	if a.Typ() != b.Typ() {
		return 0, fmt.Errorf("%w: compare requires same type, got %s and %s", ErrTypeMismatch, a.Typ(), b.Typ())
	}
	switch a.Typ() {
	case TypeByte:
		av, _ := a.AsByte()
		bv, _ := b.AsByte()
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
		return 0, nil
	case TypeRune:
		av, _ := a.AsRune()
		bv, _ := b.AsRune()
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
		return 0, nil
	case TypeInt:
		av, _ := a.AsInt()
		bv, _ := b.AsInt()
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
		return 0, nil
	case TypeFloat:
		af, _ := a.AsFloat()
		bf, _ := b.AsFloat()
		if math.IsNaN(af) || math.IsNaN(bf) {
			return 0, fmt.Errorf("%w: comparison with NaN is undefined", ErrTypeMismatch)
		}
		if af < bf {
			return -1, nil
		}
		if af > bf {
			return 1, nil
		}
		return 0, nil
	case TypeString:
		as, _ := a.AsString()
		bs, _ := b.AsString()
		return strings.Compare(as, bs), nil
	case TypeBytes:
		ab, _ := a.AsBytes()
		bb, _ := b.AsBytes()
		return bytes.Compare(ab, bb), nil
	default:
		return 0, fmt.Errorf("%w: compare unsupported type %s", ErrTypeMismatch, a.Typ())
	}
}

// execEQUAL 相等比较 a==b（EQUAL，opcode 104）。
// 浮点遵循 DEC-0502：NaN != NaN；+0.0 == -0.0。
func execEQUAL(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(args[0].Equal(args[1])))
}

// execNEQUAL 不等比较 a!=b（NEQUAL，opcode 105）。
func execNEQUAL(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(!args[0].Equal(args[1])))
}

// execLT 小于比较 a<b（LT，opcode 106）。
func execLT(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	cmp, err := compareValues(args[0], args[1])
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(cmp < 0))
}

// execLTE 小于等于比较 a<=b（LTE，opcode 107）。
func execLTE(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	cmp, err := compareValues(args[0], args[1])
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(cmp <= 0))
}

// execGT 大于比较 a>b（GT，opcode 108）。
func execGT(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	cmp, err := compareValues(args[0], args[1])
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(cmp > 0))
}

// execGTE 大于等于比较 a>=b（GTE，opcode 109）。
func execGTE(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	cmp, err := compareValues(args[0], args[1])
	if err != nil {
		return err
	}
	return vm.stack.Push(BoolValue(cmp >= 0))
}

// execISEFV 判断异常浮点值（ISEFV，opcode 110）。
// 附参=标识（0=NaN, 1=+Inf, 2=-Inf）；取1 Float 实参，压入 BoolValue。
func execISEFV(vm *VM, f *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	fl, err := v.AsFloat()
	if err != nil {
		return err
	}
	var mode uint64
	if len(f.AttrParams) > 0 {
		mode = readULEB128Param(f.AttrParams[0])
	}
	var result bool
	switch mode {
	case 0:
		result = math.IsNaN(fl)
	case 1:
		result = math.IsInf(fl, 1)
	case 2:
		result = math.IsInf(fl, -1)
	default:
		return fmt.Errorf("%w: ISEFV unknown mode %d", ErrTypeMismatch, mode)
	}
	return vm.stack.Push(BoolValue(result))
}

// execWITHIN 半开区间比较 min<=x<max（WITHIN，opcode 111）。
// 实参1=值，实参2=2成员Slice(min,max)。
func execWITHIN(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	bounds, err := args[1].AsSlice()
	if err != nil {
		return err
	}
	if len(bounds) != 2 {
		return fmt.Errorf("%w: WITHIN requires bounds slice of length 2, got %d", ErrTypeMismatch, len(bounds))
	}
	// min <= x
	cmpMin, err := compareValues(bounds[0], args[0])
	if err != nil {
		return err
	}
	// x < max
	cmpMax, err := compareValues(args[0], bounds[1])
	if err != nil {
		return err
	}
	result := cmpMin <= 0 && cmpMax < 0
	return vm.stack.Push(BoolValue(result))
}

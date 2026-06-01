package script

import (
	"encoding/binary"
	"fmt"
	"math"
)

// instr_collection.go 实现集合指令 [35-45] 的执行函数。
// 参考：docs/proposal/Instruction/4.Collection-Operations.md

func init() {
	registerExec(SLICE, execSLICE)
	registerExec(REVERSE, execREVERSE)
	registerExec(MERGE, execMERGE)
	registerExec(EXTEND, execEXTEND)
	registerExec(PACK, execPACK)
	registerExec(SPREAD, execSPREAD)
	registerExec(INDEX, execINDEX)
	registerExec(ITEM, execITEM)
	registerExec(SET, execSET)
	registerExec(CALL, execCALL)
	registerExec(SIZE, execSIZE)
}

// execSLICE 截取子切片（SLICE，opcode 35）。
// 附参=子切片大小（ULEB128，0=之后全部）；实参1=目标Slice，实参2=起始下标（负数从末尾）。
func execSLICE(vm *VM, f *InstrFrame) error {
	var sz uint64
	if len(f.AttrParams) > 0 {
		sz = readULEB128Param(f.AttrParams[0])
	}
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	sl, err := args[0].AsSlice()
	if err != nil {
		return err
	}
	idxVal, err := args[1].AsInt()
	if err != nil {
		return err
	}
	n := len(sl)
	// 负数从末尾计算
	idx := int(idxVal)
	if idx < 0 {
		idx = n + idx
	}
	if idx < 0 || idx > n {
		return fmt.Errorf("%w: slice index %d out of range [0,%d]", ErrIndexOutOfRange, idxVal, n)
	}
	end := n
	if sz > 0 {
		end = idx + int(sz)
		if end > n {
			end = n
		}
	}
	sub := make([]Value, end-idx)
	copy(sub, sl[idx:end])
	return vm.stack.Push(SliceValue(sub))
}

// execREVERSE 切片成员反转（REVERSE，opcode 36）。
// 实参1=目标Slice，返回新切片（成员顺序反转）。
func execREVERSE(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	n := len(sl)
	rev := make([]Value, n)
	for i, item := range sl {
		rev[n-1-i] = item
	}
	return vm.stack.Push(SliceValue(rev))
}

// execMERGE 多切片合并（MERGE，opcode 37）。
// -1 模型：实参区有值则全取，否则空；合并所有 Slice 成员。
func execMERGE(vm *VM, _ *InstrFrame) error {
	items := vm.args.Items()
	vm.args.Clear()
	var merged []Value
	for _, item := range items {
		sl, err := item.AsSlice()
		if err != nil {
			return err
		}
		merged = append(merged, sl...)
	}
	if merged == nil {
		merged = []Value{}
	}
	return vm.stack.Push(SliceValue(merged))
}

// execEXTEND 向目标切片添加成员（EXTEND，opcode 38）。
// -1 模型：实参区全部值，第一个是目标 Slice，其余追加进去。若实参区空则返回空 Slice。
func execEXTEND(vm *VM, _ *InstrFrame) error {
	items := vm.args.Items()
	vm.args.Clear()
	if len(items) == 0 {
		return vm.stack.Push(SliceValue([]Value{}))
	}
	sl, err := items[0].AsSlice()
	if err != nil {
		return err
	}
	sl = append(sl, items[1:]...)
	return vm.stack.Push(SliceValue(sl))
}

// execPACK 切片成员打包为字节序列（PACK，opcode 39）。
// 实参1=Slice，对每个成员转为字节，串联为 Bytes。
func execPACK(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	var buf []byte
	for _, item := range sl {
		switch item.Typ() {
		case TypeBool:
			b, _ := item.AsBool()
			if b {
				buf = append(buf, 1)
			} else {
				buf = append(buf, 0)
			}
		case TypeByte:
			b, _ := item.AsByte()
			buf = append(buf, b)
		case TypeRune:
			r, _ := item.AsRune()
			var rb [4]byte
			binary.BigEndian.PutUint32(rb[:], uint32(r))
			buf = append(buf, rb[:]...)
		case TypeInt:
			n, _ := item.AsInt()
			var nb [8]byte
			binary.BigEndian.PutUint64(nb[:], uint64(n))
			buf = append(buf, nb[:]...)
		case TypeFloat:
			f, _ := item.AsFloat()
			var fb [8]byte
			binary.BigEndian.PutUint64(fb[:], math.Float64bits(f))
			buf = append(buf, fb[:]...)
		case TypeBytes:
			b, _ := item.AsBytes()
			buf = append(buf, b...)
		case TypeString:
			s, _ := item.AsString()
			buf = append(buf, []byte(s)...)
		default:
			return fmt.Errorf("%w: PACK unsupported element type %s", ErrTypeMismatch, item.Typ())
		}
	}
	return vm.stack.Push(BytesValue(buf))
}

// execSPREAD 展开切片成员到栈（SPREAD，opcode 40）。
// 实参1=Slice，把每个成员依次压栈。
func execSPREAD(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	sl, err := v.AsSlice()
	if err != nil {
		return err
	}
	for _, item := range sl {
		if err := vm.stack.Push(item); err != nil {
			return err
		}
	}
	return nil
}

// execINDEX 取切片成员（INDEX，opcode 41）。
// 实参1=Slice，实参2=Int（单下标）或 Slice（下标集）。
func execINDEX(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	sl, err := args[0].AsSlice()
	if err != nil {
		return err
	}
	switch args[1].Typ() {
	case TypeInt:
		idx, _ := args[1].AsInt()
		n := len(sl)
		i := int(idx)
		if i < 0 {
			i = n + i
		}
		if i < 0 || i >= n {
			return fmt.Errorf("%w: INDEX index %d out of range", ErrIndexOutOfRange, idx)
		}
		return vm.stack.Push(sl[i])
	case TypeSlice:
		idxSl, _ := args[1].AsSlice()
		result := make([]Value, 0, len(idxSl))
		n := len(sl)
		for _, idxVal := range idxSl {
			idx, err := idxVal.AsInt()
			if err != nil {
				return err
			}
			i := int(idx)
			if i < 0 {
				i = n + i
			}
			if i < 0 || i >= n {
				return fmt.Errorf("%w: INDEX index %d out of range", ErrIndexOutOfRange, idx)
			}
			result = append(result, sl[i])
		}
		return vm.stack.Push(SliceValue(result))
	default:
		return fmt.Errorf("%w: INDEX requires Int or Slice index, got %s", ErrTypeMismatch, args[1].Typ())
	}
}

// execITEM 取字典/对象/模块静态成员（ITEM，opcode 42）。
// 当前 Dict 是占位类型，总是返回 NilValue()。
func execITEM(vm *VM, _ *InstrFrame) error {
	// 取2个实参但占位实现直接忽略，返回 nil
	_, err := vm.getArgs(2)
	if err != nil {
		return err
	}
	return vm.stack.Push(NilValue())
}

// execSET 设置字典键值（SET，opcode 43）。
// 占位实现：Dict 未完整实现，返回 NilValue()。
func execSET(vm *VM, _ *InstrFrame) error {
	_, err := vm.getArgs(3)
	if err != nil {
		return err
	}
	return vm.stack.Push(NilValue())
}

// execCALL 调用对象/模块方法（CALL，opcode 44）。
// 附参=额外实参数量（ULEB128）；实参1=目标对象，实参2=方法名，剩余=方法实参。
// 占位实现：TypeObject 总返回 NilValue()。
func execCALL(vm *VM, f *InstrFrame) error {
	var extra int
	if len(f.AttrParams) > 0 {
		extra = int(readULEB128Param(f.AttrParams[0]))
	}
	total := 2 + extra
	_, err := vm.getArgs(total)
	if err != nil {
		return err
	}
	return vm.stack.Push(NilValue())
}

// execSIZE 返回集合大小（SIZE，opcode 45）。
// 实参1=集合（Slice/Bytes/String/Dict），返回 IntValue（长度）。
func execSIZE(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeSlice:
		sl, _ := v.AsSlice()
		return vm.stack.Push(IntValue(int64(len(sl))))
	case TypeBytes:
		b, _ := v.AsBytes()
		return vm.stack.Push(IntValue(int64(len(b))))
	case TypeString:
		s, _ := v.AsString()
		return vm.stack.Push(IntValue(int64(len(s))))
	case TypeDict:
		// Dict 占位：大小未知，返回 0
		return vm.stack.Push(IntValue(0))
	default:
		return fmt.Errorf("%w: SIZE unsupported type %s", ErrTypeMismatch, v.Typ())
	}
}

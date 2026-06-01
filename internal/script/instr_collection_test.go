package script

import (
	"testing"
)

// TestInstrCollection 测试集合指令 [35-45]。
func TestInstrCollection(t *testing.T) {
	t.Run("SLICE_basic", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(1), IntValue(2), IntValue(3)})
		// 实参：切片 + 下标 1，附参=2（取2个）
		vm.args.Enqueue(sl)
		vm.args.Enqueue(IntValue(1))
		frames := []InstrFrame{{Op: SLICE, AttrParams: [][]byte{{2}}}}
		vm.Run(frames)
		top, err := vm.stack.Pop()
		if err != nil {
			t.Fatalf("pop error: %v", err)
		}
		got, _ := top.AsSlice()
		if len(got) != 2 {
			t.Fatalf("want 2 elements, got %d", len(got))
		}
		v0, _ := got[0].AsInt()
		v1, _ := got[1].AsInt()
		if v0 != 2 || v1 != 3 {
			t.Fatalf("want [2,3], got [%d,%d]", v0, v1)
		}
	})

	t.Run("SLICE_zero_means_all", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(10), IntValue(20), IntValue(30)})
		vm.args.Enqueue(sl)
		vm.args.Enqueue(IntValue(1))
		frames := []InstrFrame{{Op: SLICE, AttrParams: [][]byte{{0}}}} // sz=0 表示全部
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
	})

	t.Run("SLICE_negative_index", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(1), IntValue(2), IntValue(3)})
		vm.args.Enqueue(sl)
		vm.args.Enqueue(IntValue(-2)) // 从倒数第2个开始
		frames := []InstrFrame{{Op: SLICE, AttrParams: [][]byte{{1}}}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 1 {
			t.Fatalf("want 1, got %d", len(got))
		}
		v, _ := got[0].AsInt()
		if v != 2 {
			t.Fatalf("want 2, got %d", v)
		}
	})

	t.Run("REVERSE", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(1), IntValue(2), IntValue(3)})
		vm.args.Enqueue(sl)
		frames := []InstrFrame{{Op: REVERSE}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 3 {
			t.Fatalf("want 3, got %d", len(got))
		}
		v0, _ := got[0].AsInt()
		v2, _ := got[2].AsInt()
		if v0 != 3 || v2 != 1 {
			t.Fatalf("want [3,2,1], got first=%d last=%d", v0, v2)
		}
	})

	t.Run("MERGE", func(t *testing.T) {
		vm := NewVM()
		sl1 := SliceValue([]Value{IntValue(1), IntValue(2)})
		sl2 := SliceValue([]Value{IntValue(3), IntValue(4)})
		vm.args.Enqueue(sl1)
		vm.args.Enqueue(sl2)
		frames := []InstrFrame{{Op: MERGE}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 4 {
			t.Fatalf("want 4, got %d", len(got))
		}
	})

	t.Run("MERGE_empty", func(t *testing.T) {
		vm := NewVM()
		// 实参区为空
		frames := []InstrFrame{{Op: MERGE}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 0 {
			t.Fatalf("want 0, got %d", len(got))
		}
	})

	t.Run("EXTEND", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(1)})
		vm.args.Enqueue(sl)
		vm.args.Enqueue(IntValue(2))
		vm.args.Enqueue(IntValue(3))
		frames := []InstrFrame{{Op: EXTEND}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 3 {
			t.Fatalf("want 3, got %d", len(got))
		}
	})

	t.Run("SPREAD", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(10), IntValue(20), IntValue(30)})
		vm.args.Enqueue(sl)
		frames := []InstrFrame{{Op: SPREAD}}
		vm.Run(frames)
		// 栈上应有3个元素，按顺序
		v3, _ := vm.stack.Pop()
		v2, _ := vm.stack.Pop()
		v1, _ := vm.stack.Pop()
		n3, _ := v3.AsInt()
		n2, _ := v2.AsInt()
		n1, _ := v1.AsInt()
		if n1 != 10 || n2 != 20 || n3 != 30 {
			t.Fatalf("want 10,20,30, got %d,%d,%d", n1, n2, n3)
		}
	})

	t.Run("INDEX_int", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(100), IntValue(200), IntValue(300)})
		vm.args.Enqueue(sl)
		vm.args.Enqueue(IntValue(1))
		frames := []InstrFrame{{Op: INDEX}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		n, _ := top.AsInt()
		if n != 200 {
			t.Fatalf("want 200, got %d", n)
		}
	})

	t.Run("INDEX_slice", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(1), IntValue(2), IntValue(3)})
		idxSl := SliceValue([]Value{IntValue(0), IntValue(2)})
		vm.args.Enqueue(sl)
		vm.args.Enqueue(idxSl)
		frames := []InstrFrame{{Op: INDEX}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		got, _ := top.AsSlice()
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
		v0, _ := got[0].AsInt()
		v1, _ := got[1].AsInt()
		if v0 != 1 || v1 != 3 {
			t.Fatalf("want [1,3], got [%d,%d]", v0, v1)
		}
	})

	t.Run("SIZE_slice", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(1), IntValue(2), IntValue(3)})
		vm.args.Enqueue(sl)
		frames := []InstrFrame{{Op: SIZE}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		n, _ := top.AsInt()
		if n != 3 {
			t.Fatalf("want 3, got %d", n)
		}
	})

	t.Run("SIZE_bytes", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(BytesValue([]byte{1, 2, 3, 4}))
		frames := []InstrFrame{{Op: SIZE}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		n, _ := top.AsInt()
		if n != 4 {
			t.Fatalf("want 4, got %d", n)
		}
	})

	t.Run("SIZE_string", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(StringValue("hello"))
		frames := []InstrFrame{{Op: SIZE}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		n, _ := top.AsInt()
		if n != 5 {
			t.Fatalf("want 5, got %d", n)
		}
	})

	t.Run("ITEM_placeholder", func(t *testing.T) {
		vm := NewVM()
		// Dict 占位，总是返回 nil
		vm.args.Enqueue(Value{typ: TypeDict})
		vm.args.Enqueue(StringValue("key"))
		frames := []InstrFrame{{Op: ITEM}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		if top.Typ() != TypeNil {
			t.Fatalf("want Nil, got %s", top.Typ())
		}
	})

	t.Run("PACK_ints", func(t *testing.T) {
		vm := NewVM()
		sl := SliceValue([]Value{IntValue(1)})
		vm.args.Enqueue(sl)
		frames := []InstrFrame{{Op: PACK}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		b, _ := top.AsBytes()
		if len(b) != 8 {
			t.Fatalf("want 8 bytes, got %d", len(b))
		}
		// int64(1) 大端 = 0x0000000000000001
		if b[7] != 1 {
			t.Fatalf("want last byte=1, got %d", b[7])
		}
	})

	t.Run("CALL_placeholder", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(Value{typ: TypeObject})
		vm.args.Enqueue(StringValue("method"))
		frames := []InstrFrame{{Op: CALL, AttrParams: [][]byte{{0}}}}
		vm.Run(frames)
		top, _ := vm.stack.Pop()
		if top.Typ() != TypeNil {
			t.Fatalf("want Nil, got %s", top.Typ())
		}
	})
}

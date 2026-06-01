package script

import (
	"math"
	"testing"
)

// TestInstrComparison 测试比较指令 [104-111]。
func TestInstrComparison(t *testing.T) {
	t.Run("EQUAL_int_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(IntValue(42))
		vm.args.Enqueue(IntValue(42))
		vm.Run([]InstrFrame{{Op: EQUAL}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true")
		}
	})

	t.Run("EQUAL_int_false", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(IntValue(1))
		vm.args.Enqueue(IntValue(2))
		vm.Run([]InstrFrame{{Op: EQUAL}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false")
		}
	})

	t.Run("EQUAL_float_nan_not_equal", func(t *testing.T) {
		// NaN != NaN（DEC-0502）
		nan := math.NaN()
		vm := NewVM()
		vm.args.Enqueue(FloatValue(nan))
		vm.args.Enqueue(FloatValue(nan))
		vm.Run([]InstrFrame{{Op: EQUAL}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false: NaN != NaN")
		}
	})

	t.Run("EQUAL_float_plus_minus_zero", func(t *testing.T) {
		// +0.0 == -0.0（DEC-0502）
		vm := NewVM()
		vm.args.Enqueue(FloatValue(+0.0))
		vm.args.Enqueue(FloatValue(math.Copysign(0, -1)))
		vm.Run([]InstrFrame{{Op: EQUAL}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: +0.0 == -0.0")
		}
	})

	t.Run("NEQUAL", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(IntValue(1))
		vm.args.Enqueue(IntValue(2))
		vm.Run([]InstrFrame{{Op: NEQUAL}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: 1 != 2")
		}
	})

	t.Run("LT_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(IntValue(1))
		vm.args.Enqueue(IntValue(2))
		vm.Run([]InstrFrame{{Op: LT}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: 1 < 2")
		}
	})

	t.Run("LT_false", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(IntValue(2))
		vm.args.Enqueue(IntValue(1))
		vm.Run([]InstrFrame{{Op: LT}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false: 2 < 1")
		}
	})

	t.Run("GTE_string", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(StringValue("b"))
		vm.args.Enqueue(StringValue("a"))
		vm.Run([]InstrFrame{{Op: GTE}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: 'b' >= 'a'")
		}
	})

	t.Run("LT_float_nan_error", func(t *testing.T) {
		// NaN 参与排序比较应报错
		vm := NewVM()
		vm.args.Enqueue(FloatValue(math.NaN()))
		vm.args.Enqueue(FloatValue(1.0))
		state := vm.Run([]InstrFrame{{Op: LT}})
		if state == StatePassStop {
			t.Fatal("expected error state for NaN comparison")
		}
	})

	t.Run("LT_cross_type_error", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(IntValue(1))
		vm.args.Enqueue(FloatValue(1.0))
		state := vm.Run([]InstrFrame{{Op: LT}})
		if state == StatePassStop {
			t.Fatal("expected error state for cross-type comparison")
		}
	})

	t.Run("ISEFV_nan", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(FloatValue(math.NaN()))
		vm.Run([]InstrFrame{{Op: ISEFV, AttrParams: [][]byte{{0}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: NaN is NaN")
		}
	})

	t.Run("ISEFV_pos_inf", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(FloatValue(math.Inf(1)))
		vm.Run([]InstrFrame{{Op: ISEFV, AttrParams: [][]byte{{1}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: +Inf")
		}
	})

	t.Run("ISEFV_neg_inf", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(FloatValue(math.Inf(-1)))
		vm.Run([]InstrFrame{{Op: ISEFV, AttrParams: [][]byte{{2}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: -Inf")
		}
	})

	t.Run("ISEFV_normal_not_nan", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(FloatValue(3.14))
		vm.Run([]InstrFrame{{Op: ISEFV, AttrParams: [][]byte{{0}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false: 3.14 is not NaN")
		}
	})

	t.Run("WITHIN_inside", func(t *testing.T) {
		vm := NewVM()
		bounds := SliceValue([]Value{IntValue(3), IntValue(10)})
		vm.args.Enqueue(IntValue(5))
		vm.args.Enqueue(bounds)
		vm.Run([]InstrFrame{{Op: WITHIN}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: 5 in [3,10)")
		}
	})

	t.Run("WITHIN_at_min", func(t *testing.T) {
		vm := NewVM()
		bounds := SliceValue([]Value{IntValue(3), IntValue(10)})
		vm.args.Enqueue(IntValue(3))
		vm.args.Enqueue(bounds)
		vm.Run([]InstrFrame{{Op: WITHIN}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: 3 in [3,10)")
		}
	})

	t.Run("WITHIN_at_max_excluded", func(t *testing.T) {
		vm := NewVM()
		bounds := SliceValue([]Value{IntValue(3), IntValue(10)})
		vm.args.Enqueue(IntValue(10))
		vm.args.Enqueue(bounds)
		vm.Run([]InstrFrame{{Op: WITHIN}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false: 10 not in [3,10)")
		}
	})

	t.Run("WITHIN_below_min", func(t *testing.T) {
		vm := NewVM()
		bounds := SliceValue([]Value{IntValue(3), IntValue(10)})
		vm.args.Enqueue(IntValue(2))
		vm.args.Enqueue(bounds)
		vm.Run([]InstrFrame{{Op: WITHIN}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false: 2 not in [3,10)")
		}
	})

	t.Run("WITHIN_wrong_bounds_len", func(t *testing.T) {
		vm := NewVM()
		bounds := SliceValue([]Value{IntValue(3)}) // 只有一个元素
		vm.args.Enqueue(IntValue(5))
		vm.args.Enqueue(bounds)
		state := vm.Run([]InstrFrame{{Op: WITHIN}})
		if state == StatePassStop {
			t.Fatal("expected error for wrong bounds length")
		}
	})
}

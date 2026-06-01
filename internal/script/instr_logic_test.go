package script

import "testing"

// TestInstrLogic 测试逻辑指令 [112-115]。
func TestInstrLogic(t *testing.T) {
	t.Run("BOTH_true_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(BoolValue(true))
		vm.args.Enqueue(BoolValue(true))
		vm.Run([]InstrFrame{{Op: BOTH}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true")
		}
	})

	t.Run("BOTH_true_false", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(BoolValue(true))
		vm.args.Enqueue(BoolValue(false))
		vm.Run([]InstrFrame{{Op: BOTH}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false")
		}
	})

	t.Run("BOTH_false_false", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(BoolValue(false))
		vm.args.Enqueue(BoolValue(false))
		vm.Run([]InstrFrame{{Op: BOTH}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false")
		}
	})

	t.Run("EITHER_false_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(BoolValue(false))
		vm.args.Enqueue(BoolValue(true))
		vm.Run([]InstrFrame{{Op: EITHER}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true")
		}
	})

	t.Run("EITHER_false_false", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(BoolValue(false))
		vm.args.Enqueue(BoolValue(false))
		vm.Run([]InstrFrame{{Op: EITHER}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false")
		}
	})

	t.Run("EVERY_empty", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{}))
		vm.Run([]InstrFrame{{Op: EVERY}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: empty EVERY")
		}
	})

	t.Run("EVERY_all_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{BoolValue(true), BoolValue(true)}))
		vm.Run([]InstrFrame{{Op: EVERY}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true")
		}
	})

	t.Run("EVERY_has_false", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{BoolValue(true), BoolValue(false)}))
		vm.Run([]InstrFrame{{Op: EVERY}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false")
		}
	})

	t.Run("EVERY_non_bool_error", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{BoolValue(true), IntValue(1)}))
		state := vm.Run([]InstrFrame{{Op: EVERY}})
		if state == StatePassStop {
			t.Fatal("expected error for non-bool element")
		}
	})

	t.Run("SOME_n1_has_one_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{BoolValue(false), BoolValue(true)}))
		vm.Run([]InstrFrame{{Op: SOME, AttrParams: [][]byte{{1}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: at least 1 true")
		}
	})

	t.Run("SOME_n2_only_one_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{BoolValue(true), BoolValue(false)}))
		vm.Run([]InstrFrame{{Op: SOME, AttrParams: [][]byte{{2}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false: need 2 true, only 1")
		}
	})

	t.Run("SOME_n0_always_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{}))
		vm.Run([]InstrFrame{{Op: SOME, AttrParams: [][]byte{{0}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: n=0 always true")
		}
	})

	t.Run("SOME_n1_empty_false", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{}))
		vm.Run([]InstrFrame{{Op: SOME, AttrParams: [][]byte{{1}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if b {
			t.Fatal("want false: empty slice, need 1 true")
		}
	})

	t.Run("SOME_n2_two_true", func(t *testing.T) {
		vm := NewVM()
		vm.args.Enqueue(SliceValue([]Value{BoolValue(true), BoolValue(true), BoolValue(false)}))
		vm.Run([]InstrFrame{{Op: SOME, AttrParams: [][]byte{{2}}}})
		top, _ := vm.stack.Pop()
		b, _ := top.AsBool()
		if !b {
			t.Fatal("want true: 2 true elements")
		}
	})
}

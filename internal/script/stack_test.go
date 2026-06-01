package script

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestStackBasicLIFO 测试栈的 LIFO 行为。
func TestStackBasicLIFO(t *testing.T) {
	s := NewStack()
	vals := []Value{IntValue(1), IntValue(2), IntValue(3)}
	for _, v := range vals {
		if err := s.Push(v); err != nil {
			t.Fatalf("Push error: %v", err)
		}
	}
	for i := len(vals) - 1; i >= 0; i-- {
		v, err := s.Pop()
		if err != nil {
			t.Fatalf("Pop error: %v", err)
		}
		n, _ := v.AsInt()
		want, _ := vals[i].AsInt()
		if n != want {
			t.Errorf("Pop[%d] = %d, want %d", i, n, want)
		}
	}
}

// TestStackUnderflow 测试空栈弹出返回 ErrStackUnderflow。
func TestStackUnderflow(t *testing.T) {
	s := NewStack()
	if _, err := s.Pop(); err != ErrStackUnderflow {
		t.Errorf("Pop on empty stack: got %v, want ErrStackUnderflow", err)
	}
	if _, err := s.Top(); err != ErrStackUnderflow {
		t.Errorf("Top on empty stack: got %v, want ErrStackUnderflow", err)
	}
	if _, err := s.Peek(0); err != ErrStackUnderflow {
		t.Errorf("Peek on empty stack: got %v, want ErrStackUnderflow", err)
	}
}

// TestStackOverflow 测试 MaxStackHeight 边界。
// 压入 255 个值合法，压入第 256 个返回 ErrStackOverflow。
func TestStackOverflow(t *testing.T) {
	s := NewStack()
	// MaxStackHeight = 255：压入 255 个应成功
	for i := 0; i < types.MaxStackHeight; i++ {
		if err := s.Push(NilValue()); err != nil {
			t.Fatalf("Push[%d] unexpected error: %v", i, err)
		}
	}
	if s.Len() != types.MaxStackHeight {
		t.Fatalf("Len = %d, want %d", s.Len(), types.MaxStackHeight)
	}
	// 第 256 个应失败
	if err := s.Push(NilValue()); err != ErrStackOverflow {
		t.Errorf("Push at limit: got %v, want ErrStackOverflow", err)
	}
}

// TestStackItemTooLarge 测试 MaxStackItem 边界。
// 4095 字节合法，4096 字节返回 ErrStackItemTooLarge。
func TestStackItemTooLarge(t *testing.T) {
	s := NewStack()

	// MaxStackItem = 4095：恰好 4095 字节应成功
	ok4095 := BytesValue(make([]byte, types.MaxStackItem))
	if err := s.Push(ok4095); err != nil {
		t.Fatalf("Push 4095 bytes unexpected error: %v", err)
	}
	s.Clear()

	// 4096 字节应失败
	big := BytesValue(make([]byte, types.MaxStackItem+1))
	if err := s.Push(big); err != ErrStackItemTooLarge {
		t.Errorf("Push 4096 bytes: got %v, want ErrStackItemTooLarge", err)
	}
}

// TestStackTop 测试 Top 不弹出栈顶。
func TestStackTop(t *testing.T) {
	s := NewStack()
	_ = s.Push(IntValue(7))
	v, err := s.Top()
	if err != nil {
		t.Fatalf("Top error: %v", err)
	}
	n, _ := v.AsInt()
	if n != 7 {
		t.Errorf("Top = %d, want 7", n)
	}
	if s.Len() != 1 {
		t.Error("Top should not remove item")
	}
}

// TestStackPeek 测试 Peek 正负索引。
func TestStackPeek(t *testing.T) {
	s := NewStack()
	_ = s.Push(IntValue(10)) // 栈底 idx=0
	_ = s.Push(IntValue(20)) // idx=1
	_ = s.Push(IntValue(30)) // 栈顶 idx=2

	cases := []struct {
		pos  int
		want int64
	}{
		{0, 10},  // 从底部第 0 个
		{1, 20},  // 第 1 个
		{2, 30},  // 第 2 个（栈顶）
		{-1, 30}, // 栈顶
		{-2, 20}, // 次栈顶
		{-3, 10}, // 栈底
	}
	for _, tc := range cases {
		v, err := s.Peek(tc.pos)
		if err != nil {
			t.Errorf("Peek(%d) error: %v", tc.pos, err)
			continue
		}
		n, _ := v.AsInt()
		if n != tc.want {
			t.Errorf("Peek(%d) = %d, want %d", tc.pos, n, tc.want)
		}
	}

	// 越界
	if _, err := s.Peek(3); err != ErrIndexOutOfRange {
		t.Errorf("Peek(3) = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := s.Peek(-4); err != ErrIndexOutOfRange {
		t.Errorf("Peek(-4) = %v, want ErrIndexOutOfRange", err)
	}
}

// TestStackClear 测试 Clear 后栈为空。
func TestStackClear(t *testing.T) {
	s := NewStack()
	_ = s.Push(IntValue(1))
	_ = s.Push(IntValue(2))
	s.Clear()
	if s.Len() != 0 {
		t.Errorf("Len after Clear = %d, want 0", s.Len())
	}
	if _, err := s.Pop(); err != ErrStackUnderflow {
		t.Error("Pop on cleared stack should return ErrStackUnderflow")
	}
}

// TestStackItems 测试 Items 返回正确副本。
func TestStackItems(t *testing.T) {
	s := NewStack()
	_ = s.Push(IntValue(1))
	_ = s.Push(IntValue(2))
	items := s.Items()
	if len(items) != 2 {
		t.Fatalf("Items len = %d, want 2", len(items))
	}
	n0, _ := items[0].AsInt()
	n1, _ := items[1].AsInt()
	if n0 != 1 || n1 != 2 {
		t.Errorf("Items = [%d, %d], want [1, 2]", n0, n1)
	}
}

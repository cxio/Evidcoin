package script

import "github.com/cxio/evidcoin/pkg/types"

// Stack 是脚本 VM 的数据栈（LIFO）。
// 最大高度 MaxStackHeight = 255，单项最大字节数 MaxStackItem = 4095（DEC-0505）。
// 参考：docs/proposal/10.Script-System.md §3
type Stack struct {
	items []Value
}

// NewStack 创建一个空数据栈。
func NewStack() *Stack { return &Stack{} }

// Push 将值压入栈顶。
// 若栈高度达到 MaxStackHeight（255）返回 ErrStackOverflow；
// 若单项字节大小超过 MaxStackItem（4095）返回 ErrStackItemTooLarge。
func (s *Stack) Push(v Value) error {
	if len(s.items) >= types.MaxStackHeight {
		return ErrStackOverflow
	}
	if v.ByteSize() > types.MaxStackItem {
		return ErrStackItemTooLarge
	}
	s.items = append(s.items, v)
	return nil
}

// Pop 弹出并返回栈顶值。栈为空时返回 ErrStackUnderflow。
func (s *Stack) Pop() (Value, error) {
	if len(s.items) == 0 {
		return Value{}, ErrStackUnderflow
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, nil
}

// Top 返回栈顶值但不弹出。栈为空时返回 ErrStackUnderflow。
func (s *Stack) Top() (Value, error) {
	if len(s.items) == 0 {
		return Value{}, ErrStackUnderflow
	}
	return s.items[len(s.items)-1], nil
}

// Peek 返回指定位置的值（不弹出）。
// pos >= 0 时为从栈底（0=底部第一个元素）；pos < 0 时从栈顶计（-1=栈顶，-2=次栈顶）。
// 越界时返回 ErrIndexOutOfRange；空栈返回 ErrStackUnderflow。
func (s *Stack) Peek(pos int) (Value, error) {
	n := len(s.items)
	if n == 0 {
		return Value{}, ErrStackUnderflow
	}
	var idx int
	if pos >= 0 {
		idx = pos
	} else {
		idx = n + pos
	}
	if idx < 0 || idx >= n {
		return Value{}, ErrIndexOutOfRange
	}
	return s.items[idx], nil
}

// Len 返回栈中元素数量。
func (s *Stack) Len() int { return len(s.items) }

// Clear 清空栈。
func (s *Stack) Clear() { s.items = s.items[:0] }

// Items 返回栈内所有元素的副本（从栈底到栈顶顺序）。
func (s *Stack) Items() []Value {
	cp := make([]Value, len(s.items))
	copy(cp, s.items)
	return cp
}
